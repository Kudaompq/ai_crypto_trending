package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kudaompq/ai_trending/backend/internal/repository"
)

type scheduledWatchlistStub struct {
	snapshot repository.WatchlistSnapshot
	calls    int
}

func (s *scheduledWatchlistStub) Get(context.Context) (repository.WatchlistSnapshot, error) {
	s.calls++
	return s.snapshot, nil
}

type scheduledAnalyzerStub struct {
	mu      sync.Mutex
	calls   []string
	results map[string]MarketAnalysisChatResponse
	errors  map[string]error
	started chan string
	release <-chan struct{}
}

func (s *scheduledAnalyzerStub) Chat(ctx context.Context, request MarketAnalysisChatRequest) (MarketAnalysisChatResponse, error) {
	s.mu.Lock()
	s.calls = append(s.calls, request.Symbol)
	if s.started != nil {
		s.started <- request.Symbol
	}
	s.mu.Unlock()
	if s.release != nil {
		select {
		case <-s.release:
		case <-ctx.Done():
			return MarketAnalysisChatResponse{}, ctx.Err()
		}
	}
	return s.results[request.Symbol], s.errors[request.Symbol]
}

type scheduledAnalysisStoreStub struct {
	mu            sync.Mutex
	run           repository.ScheduledAnalysisRun
	incomplete    []repository.ScheduledAnalysisRun
	completed     []string
	results       []repository.ScheduledAnalysisResult
	queued        []repository.ScheduledAnalysisResult
	queueSawAll   bool
	lockAcquired  bool
	lockAttempts  int
	finishedState string
	outboxItems   []repository.ScheduledAnalysisOutboxItem
	sentAlerts    []string
	failedAlerts  []string
	lastStatus    int
	lastAPICode   int
	lastErrorCode string
}

func (s *scheduledAnalysisStoreStub) TrySchedulerLock(context.Context) (func(), bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lockAttempts++
	if !s.lockAcquired {
		return func() {}, false, nil
	}
	s.lockAcquired = false
	return func() { s.mu.Lock(); s.lockAcquired = true; s.mu.Unlock() }, true, nil
}

func (s *scheduledAnalysisStoreStub) CreateOrGetRun(_ context.Context, slot time.Time, revision int64, symbols []string) (repository.ScheduledAnalysisRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.run.ID == "" {
		s.run = repository.ScheduledAnalysisRun{ID: "run-1", ScheduledFor: slot, SnapshotRevision: revision, Symbols: append([]string(nil), symbols...), Status: "running"}
	}
	return s.run, nil
}

func (s *scheduledAnalysisStoreStub) IncompleteRuns(context.Context, time.Time, int) ([]repository.ScheduledAnalysisRun, error) {
	return append([]repository.ScheduledAnalysisRun(nil), s.incomplete...), nil
}

func (s *scheduledAnalysisStoreStub) CompletedSymbols(context.Context, string) ([]string, error) {
	return append([]string(nil), s.completed...), nil
}

func (s *scheduledAnalysisStoreStub) InsertResult(_ context.Context, result *repository.ScheduledAnalysisResult) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if result.ID == "" {
		result.ID = "result-" + result.Symbol
	}
	s.results = append(s.results, *result)
	return true, nil
}

func (s *scheduledAnalysisStoreStub) ListResults(context.Context, string) ([]repository.ScheduledAnalysisResult, error) {
	return append([]repository.ScheduledAnalysisResult(nil), s.results...), nil
}

func (s *scheduledAnalysisStoreStub) FinishRun(_ context.Context, _ string, status string) error {
	s.finishedState = status
	return nil
}

func (s *scheduledAnalysisStoreStub) QueueAlert(_ context.Context, result repository.ScheduledAnalysisResult, _ repository.AlertComparison, _ string) (bool, error) {
	s.queued = append(s.queued, result)
	if s.run.ID != "" && len(s.results) == len(s.run.Symbols) {
		s.queueSawAll = true
	}
	return true, nil
}

func (s *scheduledAnalysisStoreStub) ClaimDueAlerts(context.Context, int) ([]repository.ScheduledAnalysisOutboxItem, error) {
	items := append([]repository.ScheduledAnalysisOutboxItem(nil), s.outboxItems...)
	s.outboxItems = nil
	return items, nil
}

func (s *scheduledAnalysisStoreStub) MarkAlertSent(_ context.Context, id string, status, apiCode int) error {
	s.sentAlerts = append(s.sentAlerts, id)
	s.lastStatus, s.lastAPICode = status, apiCode
	return nil
}

func (s *scheduledAnalysisStoreStub) MarkAlertFailure(_ context.Context, id string, _ int, _ int, status, apiCode int, errorCode string, _ time.Time) error {
	s.failedAlerts = append(s.failedAlerts, id)
	s.lastStatus, s.lastAPICode, s.lastErrorCode = status, apiCode, errorCode
	return nil
}

type marketAlertSenderStub struct {
	content string
	status  int
	err     error
}

func (s *marketAlertSenderStub) Send(_ context.Context, content string) (int, error) {
	s.content = content
	return s.status, s.err
}

func actionableChatResponse(symbol string, confidence int) MarketAnalysisChatResponse {
	return MarketAnalysisChatResponse{
		Symbol: symbol, Interval: "1h", ContextTime: 123, ReferencePrice: 100, ATR: 2,
		Plan: MarketAnalysisPlan{Status: "actionable", Timing: "right", Direction: "Long", EntryType: "market", EntryPrice: 100, TakeProfit: 110, StopLoss: 95, Confidence: confidence, Leverage: 2, Analysis: "结构确认后关注延续。"},
	}
}

func TestScheduledWatchlistScanUsesOneSnapshotPersistsEverySymbolAndQueuesOnlyStrictCandidates(t *testing.T) {
	watchlist := &scheduledWatchlistStub{snapshot: repository.WatchlistSnapshot{Symbols: []string{"BTCUSDT", "ETHUSDT", "SOLUSDT", "XRPUSDT"}, Revision: 7}}
	analyzer := &scheduledAnalyzerStub{
		results: map[string]MarketAnalysisChatResponse{"BTCUSDT": actionableChatResponse("BTCUSDT", 71), "SOLUSDT": {Symbol: "SOLUSDT", Interval: "1h", Plan: MarketAnalysisPlan{Status: "wait", Timing: "undetermined", Confidence: 70, Analysis: "等待确认"}}, "XRPUSDT": actionableChatResponse("XRPUSDT", 70)},
		errors:  map[string]error{"ETHUSDT": ErrMarketContextUnavailable},
	}
	store := &scheduledAnalysisStoreStub{lockAcquired: true}
	scanner := NewScheduledWatchlistAnalysisService(watchlist, analyzer, store, ScheduledWatchlistAnalysisConfig{MaxConcurrency: 2, SymbolTimeout: time.Second, MaxRunDuration: time.Minute, Model: "test-model"})
	if err := scanner.RunAt(context.Background(), time.Date(2026, 9, 28, 2, 13, 0, 0, time.FixedZone("UTC+8", 8*60*60))); err != nil {
		t.Fatalf("RunAt returned error: %v", err)
	}
	if watchlist.calls != 1 || store.run.SnapshotRevision != 7 || len(store.run.Symbols) != 4 || len(store.results) != 4 {
		t.Fatalf("snapshot calls=%d run=%+v results=%+v", watchlist.calls, store.run, store.results)
	}
	if store.finishedState != "partial" {
		t.Fatalf("run status=%q, want partial for one per-symbol error", store.finishedState)
	}
	if len(store.queued) != 1 || store.queued[0].Symbol != "BTCUSDT" || store.queued[0].Confidence == nil || *store.queued[0].Confidence != 71 {
		t.Fatalf("queued alerts=%+v; must only queue actionable confidence >70", store.queued)
	}
	if !store.queueSawAll {
		t.Fatal("alert filtering began before every symbol result was persisted")
	}
	for _, result := range store.results {
		if result.Interval != "1h" {
			t.Fatalf("unexpected scan interval: %+v", result)
		}
		if result.Symbol == "ETHUSDT" && (result.Status != "failed" || result.ErrorCode == "") {
			t.Fatalf("failed symbol lacks failure record: %+v", result)
		}
	}
}

func TestNextScheduledAnalysisHourUsesUTCAndSkipsCurrentPartialHour(t *testing.T) {
	now := time.Date(2026, 9, 28, 10, 42, 30, 0, time.FixedZone("UTC+8", 8*60*60))
	got := nextScheduledAnalysisHour(now)
	want := time.Date(2026, 9, 28, 3, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("next UTC slot=%s, want %s", got, want)
	}
}

func TestScheduledAnalysisOutboxMarksOnlySuccessfulDeliveryAsSent(t *testing.T) {
	item := repository.ScheduledAnalysisOutboxItem{ID: "alert-1", Attempts: 1, Payload: "BTCUSDT signal"}
	store := &scheduledAnalysisStoreStub{outboxItems: []repository.ScheduledAnalysisOutboxItem{item}}
	sender := &marketAlertSenderStub{status: 200}
	scanner := NewScheduledWatchlistAnalysisService(nil, nil, store, ScheduledWatchlistAnalysisConfig{AlertSender: sender})
	if err := scanner.dispatchDueAlerts(context.Background()); err != nil {
		t.Fatalf("dispatch success: %v", err)
	}
	if len(store.sentAlerts) != 1 || store.sentAlerts[0] != item.ID || len(store.failedAlerts) != 0 || !strings.Contains(sender.content, "BTCUSDT signal") {
		t.Fatalf("sent=%v failed=%v content=%q", store.sentAlerts, store.failedAlerts, sender.content)
	}

	store = &scheduledAnalysisStoreStub{outboxItems: []repository.ScheduledAnalysisOutboxItem{item}}
	sender = &marketAlertSenderStub{status: 200, err: &WeComDeliveryError{Code: "wecom_rejected", HTTPStatus: 200, APICode: 93000}}
	scanner = NewScheduledWatchlistAnalysisService(nil, nil, store, ScheduledWatchlistAnalysisConfig{AlertSender: sender})
	if err := scanner.dispatchDueAlerts(context.Background()); err != nil {
		t.Fatalf("dispatch failure should persist retry state: %v", err)
	}
	if len(store.sentAlerts) != 0 || len(store.failedAlerts) != 1 || store.lastAPICode != 93000 || store.lastErrorCode != "wecom_rejected" {
		t.Fatalf("failed delivery was marked sent or lost response code: sent=%v failed=%v api=%d code=%s", store.sentAlerts, store.failedAlerts, store.lastAPICode, store.lastErrorCode)
	}
}

func TestScheduledWatchlistScanDoesNotOverlapAndResumesOnlyUnfinishedSymbols(t *testing.T) {
	startGate := make(chan string, 2)
	release := make(chan struct{})
	watchlist := &scheduledWatchlistStub{snapshot: repository.WatchlistSnapshot{Symbols: []string{"BTCUSDT", "ETHUSDT"}, Revision: 3}}
	analyzer := &scheduledAnalyzerStub{results: map[string]MarketAnalysisChatResponse{"BTCUSDT": actionableChatResponse("BTCUSDT", 70), "ETHUSDT": actionableChatResponse("ETHUSDT", 60)}, started: startGate, release: release}
	store := &scheduledAnalysisStoreStub{lockAcquired: true}
	scanner := NewScheduledWatchlistAnalysisService(watchlist, analyzer, store, ScheduledWatchlistAnalysisConfig{MaxConcurrency: 1, SymbolTimeout: time.Second, MaxRunDuration: time.Minute})
	completed := make(chan error, 1)
	go func() { completed <- scanner.RunAt(context.Background(), time.Date(2026, 9, 28, 3, 0, 0, 0, time.UTC)) }()
	select {
	case <-startGate:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("first scan did not enter analysis")
	}
	if err := scanner.RunAt(context.Background(), time.Date(2026, 9, 28, 4, 0, 0, 0, time.UTC)); !errors.Is(err, ErrScheduledAnalysisAlreadyRunning) {
		close(release)
		t.Fatalf("overlapping scan error=%v", err)
	}
	close(release)
	if err := <-completed; err != nil {
		t.Fatalf("first scan error: %v", err)
	}

	resumeRun := repository.ScheduledAnalysisRun{ID: "run-resume", ScheduledFor: time.Date(2026, 9, 28, 1, 0, 0, 0, time.UTC), SnapshotRevision: 4, Symbols: []string{"BTCUSDT", "ETHUSDT"}, Status: "interrupted"}
	resumeStore := &scheduledAnalysisStoreStub{lockAcquired: true, incomplete: []repository.ScheduledAnalysisRun{resumeRun}, completed: []string{"BTCUSDT"}}
	resumeAnalyzer := &scheduledAnalyzerStub{results: map[string]MarketAnalysisChatResponse{"ETHUSDT": actionableChatResponse("ETHUSDT", 65)}}
	resumeScanner := NewScheduledWatchlistAnalysisService(watchlist, resumeAnalyzer, resumeStore, ScheduledWatchlistAnalysisConfig{MaxConcurrency: 1, SymbolTimeout: time.Second, MaxRunDuration: time.Minute})
	if err := resumeScanner.ResumeIncomplete(context.Background(), time.Date(2026, 9, 28, 5, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("ResumeIncomplete returned error: %v", err)
	}
	if len(resumeAnalyzer.calls) != 1 || resumeAnalyzer.calls[0] != "ETHUSDT" || watchlist.calls != 1 {
		t.Fatalf("resumed calls=%v Watchlist reads=%d", resumeAnalyzer.calls, watchlist.calls)
	}
}
