//go:build integration

package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/kudaompq/ai_trending/backend/internal/database"
	"github.com/kudaompq/ai_trending/backend/internal/repository"
)

type scheduledIntegrationAnalyzer struct {
	calls map[string]int
}

func (a *scheduledIntegrationAnalyzer) Chat(_ context.Context, request MarketAnalysisChatRequest) (MarketAnalysisChatResponse, error) {
	a.calls[request.Symbol]++
	call := a.calls[request.Symbol]
	switch request.Symbol {
	case "BTCUSDT":
		if call <= 2 {
			return actionableChatResponse("BTCUSDT", 82), nil
		}
		changed := actionableChatResponse("BTCUSDT", 84)
		changed.Plan.Timing = "left"
		return changed, nil
	case "ETHUSDT":
		if call == 1 {
			return actionableChatResponse("ETHUSDT", 70), nil
		}
		return MarketAnalysisChatResponse{
			Symbol: "ETHUSDT", Interval: "1h", ContextTime: 123,
			Plan: MarketAnalysisPlan{Status: "wait", Timing: "undetermined", Confidence: 95, Analysis: "等待结构确认。"},
		}, nil
	default:
		return MarketAnalysisChatResponse{}, ErrMarketContextUnavailable
	}
}

func TestScheduledWatchlistAnalysisEndToEndWithPostgresAndLocalWeCom(t *testing.T) {
	db := openScheduledAnalysisIntegrationDB(t)
	ctx := context.Background()
	store := repository.NewPostgresScheduledAnalysisRepository(db)
	var requestCount atomic.Int32
	var lastMessage atomic.Value
	wecom := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		var payload struct {
			MessageType string `json:"msgtype"`
			Text        struct {
				Content string `json:"content"`
			} `json:"text"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Errorf("decode local WeCom request: %v", err)
		}
		if request.Method != http.MethodPost || payload.MessageType != "text" {
			t.Errorf("unexpected WeCom method=%q payload=%+v", request.Method, payload)
		}
		lastMessage.Store(payload.Text.Content)
		switch requestCount.Add(1) {
		case 2:
			_, _ = fmt.Fprint(w, `{"errcode":93000,"errmsg":"simulated temporary rejection"}`)
		default:
			_, _ = fmt.Fprint(w, `{"errcode":0}`)
		}
	}))
	defer wecom.Close()
	watchlist := &scheduledWatchlistStub{snapshot: repository.WatchlistSnapshot{Symbols: []string{"BTCUSDT", "ETHUSDT", "XRPUSDT"}, Revision: 11}}
	analyzer := &scheduledIntegrationAnalyzer{calls: make(map[string]int)}
	scanner := NewScheduledWatchlistAnalysisService(watchlist, analyzer, store, ScheduledWatchlistAnalysisConfig{
		MaxConcurrency: 1, SymbolTimeout: time.Second, MaxRunDuration: time.Minute, Model: "integration-model",
		AlertSender: NewWeComNotifierWithClient(wecom.URL, wecom.Client(), 0),
	})
	slots := []time.Time{
		time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 9, 28, 11, 0, 0, 0, time.UTC),
		time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC),
	}
	for i, slot := range slots {
		if err := scanner.RunAt(ctx, slot); err != nil {
			t.Fatalf("RunAt slot %s: %v", slot, err)
		}
		if i == 0 && requestCount.Load() != 1 {
			t.Fatalf("first qualified result produced %d WeCom requests, want 1", requestCount.Load())
		}
		if i == 1 && requestCount.Load() != 1 {
			t.Fatalf("stable strategy repeated WeCom notification: requests=%d", requestCount.Load())
		}
		if i == 2 && requestCount.Load() != 2 {
			t.Fatalf("changed strategy should make one rejected delivery: requests=%d", requestCount.Load())
		}
	}
	var resultCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM scheduled_analysis_results").Scan(&resultCount); err != nil {
		t.Fatal(err)
	}
	if resultCount != 9 {
		t.Fatalf("persisted %d per-symbol results, want 9 across three scans", resultCount)
	}
	var sentCount, retryCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FILTER (WHERE status = 'sent'), COUNT(*) FILTER (WHERE status = 'retry') FROM scheduled_analysis_outbox`).Scan(&sentCount, &retryCount); err != nil {
		t.Fatal(err)
	}
	if sentCount != 1 || retryCount != 1 {
		t.Fatalf("outbox sent=%d retry=%d; want one delivered and one retry", sentCount, retryCount)
	}
	if content, _ := lastMessage.Load().(string); !strings.Contains(content, "BTCUSDT") || !strings.Contains(content, "置信度：84/100") || !strings.Contains(content, "左侧") {
		t.Fatalf("unexpected alert content: %q", content)
	}

	if _, err := db.ExecContext(ctx, "UPDATE scheduled_analysis_outbox SET next_attempt_at = NOW() - INTERVAL '1 second' WHERE status = 'retry'"); err != nil {
		t.Fatal(err)
	}
	if err := scanner.RunAt(ctx, slots[2]); err != nil {
		t.Fatalf("retry due outbox on completed scan: %v", err)
	}
	if requestCount.Load() != 3 {
		t.Fatalf("retry requests=%d, want one successful retry", requestCount.Load())
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM scheduled_analysis_outbox WHERE status = 'sent'`).Scan(&sentCount); err != nil {
		t.Fatal(err)
	}
	if sentCount != 2 {
		t.Fatalf("successful outbox count=%d, want 2 after retry", sentCount)
	}
	var lowConfidence, waitCount, failedCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FILTER (WHERE result_status = 'actionable' AND confidence = 70), COUNT(*) FILTER (WHERE result_status = 'wait'), COUNT(*) FILTER (WHERE result_status = 'failed') FROM scheduled_analysis_results`).Scan(&lowConfidence, &waitCount, &failedCount); err != nil {
		t.Fatal(err)
	}
	if lowConfidence != 1 || waitCount != 2 || failedCount != 3 {
		t.Fatalf("low-confidence=%d wait=%d failed=%d", lowConfidence, waitCount, failedCount)
	}
}

func openScheduledAnalysisIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("WATCHLIST_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("WATCHLIST_TEST_DATABASE_URL must point to an isolated test database")
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse isolated database URL: %v", err)
	}
	if !strings.HasPrefix(strings.TrimPrefix(parsed.Path, "/"), "watchlist_test") {
		t.Fatalf("refusing to reset non-test database %q", parsed.Path)
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open isolated PostgreSQL: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("connect isolated PostgreSQL: %v", err)
	}
	if _, err := db.ExecContext(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public"); err != nil {
		t.Fatalf("reset isolated schema: %v", err)
	}
	if err := database.ApplyMigrations(ctx, db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return db
}
