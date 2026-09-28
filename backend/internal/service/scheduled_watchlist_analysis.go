package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/kudaompq/ai_trending/backend/internal/repository"
)

var ErrScheduledAnalysisAlreadyRunning = errors.New("scheduled analysis is already running")

type ScheduledWatchlistSource interface {
	Get(context.Context) (repository.WatchlistSnapshot, error)
}

type ScheduledMarketAnalyzer interface {
	Chat(context.Context, MarketAnalysisChatRequest) (MarketAnalysisChatResponse, error)
}

type ScheduledAnalysisStore interface {
	TrySchedulerLock(context.Context) (func(), bool, error)
	CreateOrGetRun(context.Context, time.Time, int64, []string) (repository.ScheduledAnalysisRun, error)
	IncompleteRuns(context.Context, time.Time, int) ([]repository.ScheduledAnalysisRun, error)
	CompletedSymbols(context.Context, string) ([]string, error)
	InsertResult(context.Context, *repository.ScheduledAnalysisResult) (bool, error)
	ListResults(context.Context, string) ([]repository.ScheduledAnalysisResult, error)
	FinishRun(context.Context, string, string) error
	QueueAlert(context.Context, repository.ScheduledAnalysisResult, repository.AlertComparison, string) (bool, error)
	ClaimDueAlerts(context.Context, int) ([]repository.ScheduledAnalysisOutboxItem, error)
	MarkAlertSent(context.Context, string, int, int) error
	MarkAlertFailure(context.Context, string, int, int, int, int, string, time.Time) error
}

type MarketAlertSender interface {
	Send(context.Context, string) (int, error)
}

type ScheduledWatchlistAnalysisConfig struct {
	MaxConcurrency int
	SymbolTimeout  time.Duration
	MaxRunDuration time.Duration
	Model          string
	AlertSender    MarketAlertSender
	Now            func() time.Time
}

type ScheduledWatchlistAnalysisService struct {
	watchlist ScheduledWatchlistSource
	analyzer  ScheduledMarketAnalyzer
	store     ScheduledAnalysisStore
	config    ScheduledWatchlistAnalysisConfig
	runMu     sync.Mutex
}

func NewScheduledWatchlistAnalysisService(watchlist ScheduledWatchlistSource, analyzer ScheduledMarketAnalyzer, store ScheduledAnalysisStore, config ScheduledWatchlistAnalysisConfig) *ScheduledWatchlistAnalysisService {
	if config.MaxConcurrency <= 0 || config.MaxConcurrency > repository.MaxWatchlistSymbols {
		config.MaxConcurrency = 4
	}
	if config.SymbolTimeout <= 0 || config.SymbolTimeout > 10*time.Minute {
		config.SymbolTimeout = 2 * time.Minute
	}
	if config.MaxRunDuration <= 0 || config.MaxRunDuration > 55*time.Minute {
		config.MaxRunDuration = 50 * time.Minute
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	return &ScheduledWatchlistAnalysisService{watchlist: watchlist, analyzer: analyzer, store: store, config: config}
}

func ScheduledWatchlistAnalysisEnabledFromEnv() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("SCHEDULED_ANALYSIS_ENABLED")), "true")
}

func (s *ScheduledWatchlistAnalysisService) RunAt(ctx context.Context, scheduledFor time.Time) error {
	if !s.runMu.TryLock() {
		return ErrScheduledAnalysisAlreadyRunning
	}
	defer s.runMu.Unlock()
	if s.watchlist == nil || s.analyzer == nil || s.store == nil {
		return errors.New("scheduled analysis dependencies are not configured")
	}
	unlock, acquired, err := s.store.TrySchedulerLock(ctx)
	if err != nil {
		return err
	}
	if !acquired {
		return ErrScheduledAnalysisAlreadyRunning
	}
	defer unlock()

	snapshot, err := s.watchlist.Get(ctx)
	if err != nil {
		return fmt.Errorf("read Watchlist snapshot: %w", err)
	}
	slot := scheduledFor.UTC().Truncate(time.Hour)
	run, err := s.store.CreateOrGetRun(ctx, slot, snapshot.Revision, append([]string(nil), snapshot.Symbols...))
	if err != nil {
		return err
	}
	if run.Status == "complete" {
		return s.dispatchDueAlerts(ctx)
	}
	return s.processRun(ctx, run)
}

func (s *ScheduledWatchlistAnalysisService) ResumeIncomplete(ctx context.Context, beforeOrAt time.Time) error {
	if !s.runMu.TryLock() {
		return ErrScheduledAnalysisAlreadyRunning
	}
	defer s.runMu.Unlock()
	if s.analyzer == nil || s.store == nil {
		return errors.New("scheduled analysis dependencies are not configured")
	}
	unlock, acquired, err := s.store.TrySchedulerLock(ctx)
	if err != nil {
		return err
	}
	if !acquired {
		return ErrScheduledAnalysisAlreadyRunning
	}
	defer unlock()
	runs, err := s.store.IncompleteRuns(ctx, beforeOrAt.UTC(), 100)
	if err != nil {
		return err
	}
	for _, run := range runs {
		if err := s.processRun(ctx, run); err != nil {
			return err
		}
	}
	return s.dispatchDueAlerts(ctx)
}

func (s *ScheduledWatchlistAnalysisService) Start(ctx context.Context) {
	if s == nil {
		return
	}
	if err := s.ResumeIncomplete(ctx, s.config.Now()); err != nil && !errors.Is(err, ErrScheduledAnalysisAlreadyRunning) && !errors.Is(err, context.Canceled) {
		logScheduledAnalysisError("resume incomplete scans", err)
	}
	next := nextScheduledAnalysisHour(s.config.Now())
	for {
		delay := time.Until(next)
		if delay < 0 {
			next = nextScheduledAnalysisHour(s.config.Now())
			delay = time.Until(next)
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			if err := s.RunAt(ctx, next); err != nil && !errors.Is(err, ErrScheduledAnalysisAlreadyRunning) && !errors.Is(err, context.Canceled) {
				logScheduledAnalysisError("hourly scan", err)
			}
			next = nextScheduledAnalysisHour(s.config.Now())
		}
	}
}

func nextScheduledAnalysisHour(now time.Time) time.Time {
	return now.UTC().Truncate(time.Hour).Add(time.Hour)
}

func (s *ScheduledWatchlistAnalysisService) processRun(parent context.Context, run repository.ScheduledAnalysisRun) error {
	ctx, cancel := context.WithTimeout(parent, s.config.MaxRunDuration)
	defer cancel()
	completed, err := s.store.CompletedSymbols(ctx, run.ID)
	if err != nil {
		return err
	}
	done := make(map[string]struct{}, len(completed))
	for _, symbol := range completed {
		done[symbol] = struct{}{}
	}
	jobs := make(chan string, len(run.Symbols))
	var wg sync.WaitGroup
	var resultMu sync.Mutex
	anyFailed := false
	var firstStoreErr error
	workers := s.config.MaxConcurrency
	if workers > len(run.Symbols) {
		workers = len(run.Symbols)
	}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for symbol := range jobs {
				_, alreadyDone := done[symbol]
				if alreadyDone {
					continue
				}
				result := s.analyzeSymbol(ctx, run.ID, symbol)
				persistCtx, persistCancel := context.WithTimeout(context.WithoutCancel(parent), 10*time.Second)
				_, saveErr := s.store.InsertResult(persistCtx, &result)
				persistCancel()
				resultMu.Lock()
				if saveErr != nil && firstStoreErr == nil {
					firstStoreErr = saveErr
				}
				if result.Status == "failed" || saveErr != nil {
					anyFailed = true
				}
				resultMu.Unlock()
			}
		}()
	}
	for _, symbol := range run.Symbols {
		if _, alreadyDone := done[symbol]; !alreadyDone {
			jobs <- symbol
		}
	}
	close(jobs)
	wg.Wait()
	if firstStoreErr != nil {
		persistCtx, persistCancel := context.WithTimeout(context.WithoutCancel(parent), 10*time.Second)
		defer persistCancel()
		_ = s.store.FinishRun(persistCtx, run.ID, "interrupted")
		return fmt.Errorf("persist one or more scan results: %w", firstStoreErr)
	}
	if err := ctx.Err(); err != nil {
		anyFailed = true
	}
	persistCtx, persistCancel := context.WithTimeout(context.WithoutCancel(parent), 15*time.Second)
	defer persistCancel()
	results, err := s.store.ListResults(persistCtx, run.ID)
	if err != nil {
		return err
	}
	if len(results) != len(run.Symbols) {
		anyFailed = true
	}
	for _, result := range results {
		if result.Status == "failed" {
			anyFailed = true
		}
		if result.Status != "actionable" || result.Confidence == nil || *result.Confidence <= 70 {
			continue
		}
		var plan MarketAnalysisPlan
		if err := json.Unmarshal(result.AnalysisJSON, &plan); err != nil {
			anyFailed = true
			continue
		}
		var evidence struct {
			ATR float64 `json:"atr"`
		}
		_ = json.Unmarshal(result.EvidenceJSON, &evidence)
		comparison := repository.AlertComparison{
			Timing: plan.Timing, Direction: plan.Direction, Interval: result.Interval, EntryType: plan.EntryType,
			EntryPrice: plan.EntryPrice, TakeProfit: plan.TakeProfit, StopLoss: plan.StopLoss,
		}
		if isFinite(evidence.ATR) && evidence.ATR > 0 {
			comparison.ATR = evidence.ATR
		}
		if _, err := s.store.QueueAlert(persistCtx, result, comparison, formatMarketAlert(result, plan)); err != nil {
			anyFailed = true
			logScheduledAnalysisError("queue alert", err)
		}
	}
	runStatus := "complete"
	if anyFailed {
		runStatus = "partial"
	}
	if err := s.store.FinishRun(persistCtx, run.ID, runStatus); err != nil {
		return err
	}
	if parent.Err() == nil {
		if err := s.dispatchDueAlerts(parent); err != nil {
			logScheduledAnalysisError("deliver alerts", err)
		}
	}
	return nil
}

func (s *ScheduledWatchlistAnalysisService) analyzeSymbol(ctx context.Context, runID, symbol string) repository.ScheduledAnalysisResult {
	now := s.config.Now().UTC()
	result := repository.ScheduledAnalysisResult{
		RunID: runID, Symbol: symbol, GeneratedAt: now, Interval: "1h", Status: "failed",
		Timing: "undetermined", AnalysisJSON: json.RawMessage("null"), EvidenceJSON: json.RawMessage("[]"), Model: s.config.Model,
	}
	requestCtx, cancel := context.WithTimeout(ctx, s.config.SymbolTimeout)
	defer cancel()
	response, err := s.analyzer.Chat(requestCtx, MarketAnalysisChatRequest{Symbol: symbol, Interval: "1h", Message: "比较当前证据下的左侧与右侧交易机会，只返回置信度最高的一套；证据不足时返回等待确认。请使用成交量和 OI 摘要，并按需调用只读工具核实。"})
	if err != nil {
		result.ErrorCode = scheduledAnalysisErrorCode(err)
		return result
	}
	if response.Symbol != symbol || response.Interval != "1h" || response.ContextTime <= 0 || response.Plan.Status != "actionable" && response.Plan.Status != "wait" {
		result.ErrorCode = "invalid_analysis_result"
		return result
	}
	planJSON, err := json.Marshal(response.Plan)
	if err != nil {
		result.ErrorCode = "invalid_analysis_result"
		return result
	}
	evidenceJSON, err := json.Marshal(struct {
		ATR           float64                           `json:"atr"`
		VolumeSummary MarketAnalysisVolumeSummary       `json:"volume_summary"`
		OpenInterest  MarketAnalysisOpenInterestSummary `json:"open_interest"`
		Tools         []MarketAnalysisToolEvidence      `json:"tool_evidence,omitempty"`
	}{response.ATR, response.VolumeSummary, response.OpenInterest, response.ToolEvidence})
	if err != nil {
		result.ErrorCode = "invalid_analysis_result"
		return result
	}
	result.Status = response.Plan.Status
	result.ContextTime = response.ContextTime
	confidence := response.Plan.Confidence
	result.Confidence = &confidence
	result.Timing = response.Plan.Timing
	result.Direction = response.Plan.Direction
	result.AnalysisJSON = planJSON
	result.EvidenceJSON = evidenceJSON
	return result
}

func scheduledAnalysisErrorCode(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, ErrMarketContextUnavailable):
		return "market_context_unavailable"
	case errors.Is(err, ErrChatProviderNotConfigured):
		return "provider_not_configured"
	case errors.Is(err, ErrChatProviderFailed):
		return "provider_failed"
	default:
		return "analysis_failed"
	}
}

func formatMarketAlert(result repository.ScheduledAnalysisResult, plan MarketAnalysisPlan) string {
	timing := "暂不可判定"
	if plan.Timing == "left" {
		timing = "左侧"
	} else if plan.Timing == "right" {
		timing = "右侧"
	}
	analysis := strings.TrimSpace(plan.Analysis)
	if utf8.RuneCountInString(analysis) > 160 {
		runes := []rune(analysis)
		analysis = string(runes[:160])
	}
	entryType := "限价"
	if plan.EntryType == "market" {
		entryType = "市价"
	}
	return fmt.Sprintf("📊 %s · %s %s\n周期：%s · 置信度：%d/100（信号强弱，非胜率）\n入场：%s %s\n止盈：%s\n止损：%s\n依据：%s\n行情数据时间：%s",
		result.Symbol, timing, plan.Direction, result.Interval, plan.Confidence, entryType, formatAlertPrice(plan.EntryPrice),
		formatAlertPrice(plan.TakeProfit), formatAlertPrice(plan.StopLoss), analysis, time.UnixMilli(result.ContextTime).UTC().Format("2006-01-02 15:04:05 UTC"))
}

func formatAlertPrice(price float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.10f", price), "0"), ".")
}

func (s *ScheduledWatchlistAnalysisService) dispatchDueAlerts(ctx context.Context) error {
	if s.config.AlertSender == nil {
		return nil
	}
	for {
		items, err := s.store.ClaimDueAlerts(ctx, 100)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		for _, group := range packAlertItems(items, 2048) {
			status, sendErr := s.config.AlertSender.Send(ctx, group.body)
			for _, item := range group.items {
				if sendErr == nil {
					if err := s.store.MarkAlertSent(ctx, item.ID, status, 0); err != nil {
						return err
					}
					continue
				}
				retry := time.Duration(1<<minInt(item.Attempts-1, 10)) * time.Minute
				if retry > 6*time.Hour {
					retry = 6 * time.Hour
				}
				if err := s.store.MarkAlertFailure(ctx, item.ID, item.Attempts, 6, status, wecomAPICode(sendErr), wecomErrorCode(sendErr), s.config.Now().Add(retry)); err != nil {
					return err
				}
			}
		}
	}
}

type alertItemGroup struct {
	items []repository.ScheduledAnalysisOutboxItem
	body  string
}

func packAlertItems(items []repository.ScheduledAnalysisOutboxItem, maxBytes int) []alertItemGroup {
	if maxBytes <= 0 {
		maxBytes = 2048
	}
	items = append([]repository.ScheduledAnalysisOutboxItem(nil), items...)
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Confidence > items[j].Confidence
	})
	groups := make([]alertItemGroup, 0)
	for _, item := range items {
		block := item.Payload
		if len(block) > maxBytes {
			block = truncateUTF8(block, maxBytes)
		}
		if len(groups) == 0 || len(groups[len(groups)-1].body)+2+len(block) > maxBytes {
			groups = append(groups, alertItemGroup{})
		}
		last := &groups[len(groups)-1]
		if last.body != "" {
			last.body += "\n\n"
		}
		last.body += block
		last.items = append(last.items, item)
	}
	return groups
}

func truncateUTF8(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	for maxBytes > 0 && !utf8.ValidString(value[:maxBytes]) {
		maxBytes--
	}
	return value[:maxBytes]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func logScheduledAnalysisError(action string, err error) {
	if err != nil {
		log.Printf("scheduled analysis %s failed: %s", action, wecomErrorCode(err))
	}
}
