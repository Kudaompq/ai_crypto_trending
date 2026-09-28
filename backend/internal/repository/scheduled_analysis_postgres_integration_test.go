//go:build integration

package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/kudaompq/ai_trending/backend/internal/database"
)

func TestPostgresScheduledAnalysisHistoryIsAppendOnlyAndResumeKeepsSnapshot(t *testing.T) {
	db, watchlist := newTestWatchlistRepository(t)
	initialWatchlist, err := watchlist.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	repo := NewPostgresScheduledAnalysisRepository(db)
	ctx := context.Background()
	slot := time.Date(2026, 9, 28, 1, 0, 0, 0, time.UTC)
	run, err := repo.CreateOrGetRun(ctx, slot, initialWatchlist.Revision, []string{"BTCUSDT", "ETHUSDT"})
	if err != nil {
		t.Fatalf("create scan run: %v", err)
	}
	resumed, err := repo.CreateOrGetRun(ctx, slot, initialWatchlist.Revision+1, []string{"SOLUSDT"})
	if err != nil {
		t.Fatalf("resume scan run: %v", err)
	}
	if run.ID != resumed.ID || resumed.SnapshotRevision != initialWatchlist.Revision || !equalStrings(resumed.Symbols, []string{"BTCUSDT", "ETHUSDT"}) {
		t.Fatalf("resume replaced immutable scan snapshot: first=%+v resumed=%+v", run, resumed)
	}

	actionableJSON := json.RawMessage(`{"status":"actionable","timing":"left","confidence":66}`)
	lowConfidence := 66
	if _, err := repo.InsertResult(ctx, &ScheduledAnalysisResult{RunID: run.ID, Symbol: "BTCUSDT", Interval: "1h", Status: "actionable", Confidence: &lowConfidence, Timing: "left", AnalysisJSON: actionableJSON, EvidenceJSON: json.RawMessage(`[]`)}); err != nil {
		t.Fatalf("save low confidence result: %v", err)
	}
	if _, err := repo.InsertResult(ctx, &ScheduledAnalysisResult{RunID: run.ID, Symbol: "ETHUSDT", Interval: "1h", Status: "failed", ErrorCode: "provider_failed", AnalysisJSON: json.RawMessage(`null`), EvidenceJSON: json.RawMessage(`[]`)}); err != nil {
		t.Fatalf("save failed result: %v", err)
	}
	waitConfidence := 40
	if inserted, err := repo.InsertResult(ctx, &ScheduledAnalysisResult{RunID: run.ID, Symbol: "BTCUSDT", Interval: "1h", Status: "wait", Confidence: &waitConfidence, Timing: "undetermined", AnalysisJSON: json.RawMessage(`{"status":"wait"}`), EvidenceJSON: json.RawMessage(`[]`)}); err != nil || inserted {
		t.Fatalf("duplicate insert should be idempotent: %v", err)
	}
	results, err := repo.ListResults(ctx, run.ID)
	if err != nil {
		t.Fatalf("list saved results: %v", err)
	}
	var expectedAnalysis, storedAnalysis bytes.Buffer
	_ = json.Compact(&expectedAnalysis, actionableJSON)
	_ = json.Compact(&storedAnalysis, results[0].AnalysisJSON)
	if len(results) != 2 || results[0].Symbol != "BTCUSDT" || results[0].Status != "actionable" || results[0].Confidence == nil || *results[0].Confidence != 66 || results[1].Status != "failed" || results[1].Confidence != nil {
		t.Fatalf("saved results=%+v; low confidence and failure must be retained without overwrite", results)
	}
	completed, err := repo.CompletedSymbols(ctx, run.ID)
	if err != nil || !equalStrings(completed, []string{"BTCUSDT", "ETHUSDT"}) {
		t.Fatalf("completed symbols=%v err=%v", completed, err)
	}

	if err := database.ApplyMigrations(ctx, db); err != nil {
		t.Fatalf("reapply migrations: %v", err)
	}
	afterMigration, err := watchlist.Get(ctx)
	if err != nil || !equalStrings(afterMigration.Symbols, initialWatchlist.Symbols) || afterMigration.Revision != initialWatchlist.Revision {
		t.Fatalf("watchlist changed during migration: before=%+v after=%+v err=%v", initialWatchlist, afterMigration, err)
	}
}

func TestPostgresScheduledAnalysisAlertThresholdDedupAndRequalification(t *testing.T) {
	db, _ := newTestWatchlistRepository(t)
	repo := NewPostgresScheduledAnalysisRepository(db)
	ctx := context.Background()
	comparison := AlertComparison{Timing: "right", Direction: "Long", Interval: "1h", EntryType: "market", EntryPrice: 100, TakeProfit: 110, StopLoss: 95, ATR: 1}

	first := insertIntegrationResult(t, repo, ctx, time.Date(2026, 9, 28, 1, 0, 0, 0, time.UTC), "BTCUSDT", "actionable", 80)
	queued, err := repo.QueueAlert(ctx, first, comparison, "first candidate")
	if err != nil || !queued {
		t.Fatalf("first qualified candidate queued=%v err=%v", queued, err)
	}
	claimed, err := repo.ClaimDueAlerts(ctx, 10)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claimed=%+v err=%v", claimed, err)
	}
	if err := repo.MarkAlertSent(ctx, claimed[0].ID, 200, 0); err != nil {
		t.Fatalf("mark first alert delivered: %v", err)
	}

	stable := insertIntegrationResult(t, repo, ctx, time.Date(2026, 9, 28, 2, 0, 0, 0, time.UTC), "BTCUSDT", "actionable", 83)
	queued, err = repo.QueueAlert(ctx, stable, comparison, "stable candidate")
	if err != nil || queued {
		t.Fatalf("unchanged candidate queued=%v err=%v", queued, err)
	}
	nearby := comparison
	nearby.EntryPrice += 0.49
	nearby.TakeProfit += 0.49
	nearby.StopLoss += 0.49
	smallMove := insertIntegrationResult(t, repo, ctx, time.Date(2026, 9, 28, 3, 0, 0, 0, time.UTC), "BTCUSDT", "actionable", 85)
	queued, err = repo.QueueAlert(ctx, smallMove, nearby, "small price change")
	if err != nil || queued {
		t.Fatalf("sub-threshold price change queued=%v err=%v", queued, err)
	}
	changedPlan := nearby
	changedPlan.Timing = "left"
	changed := insertIntegrationResult(t, repo, ctx, time.Date(2026, 9, 28, 4, 0, 0, 0, time.UTC), "BTCUSDT", "actionable", 82)
	queued, err = repo.QueueAlert(ctx, changed, changedPlan, "changed timing")
	if err != nil || !queued {
		t.Fatalf("material strategy change queued=%v err=%v", queued, err)
	}
	claimed, err = repo.ClaimDueAlerts(ctx, 10)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claimed material update=%+v err=%v", claimed, err)
	}
	if err := repo.MarkAlertSent(ctx, claimed[0].ID, 200, 0); err != nil {
		t.Fatal(err)
	}

	low := insertIntegrationResult(t, repo, ctx, time.Date(2026, 9, 28, 5, 0, 0, 0, time.UTC), "BTCUSDT", "actionable", 70)
	queued, err = repo.QueueAlert(ctx, low, changedPlan, "at threshold")
	if err != nil || queued {
		t.Fatalf("confidence 70 candidate queued=%v err=%v", queued, err)
	}
	requalified := insertIntegrationResult(t, repo, ctx, time.Date(2026, 9, 28, 6, 0, 0, 0, time.UTC), "BTCUSDT", "actionable", 71)
	queued, err = repo.QueueAlert(ctx, requalified, changedPlan, "requalified candidate")
	if err != nil || !queued {
		t.Fatalf("requalified candidate queued=%v err=%v", queued, err)
	}
	wait := insertIntegrationResult(t, repo, ctx, time.Date(2026, 9, 28, 7, 0, 0, 0, time.UTC), "BTCUSDT", "wait", 95)
	queued, err = repo.QueueAlert(ctx, wait, changedPlan, "waiting candidate")
	if err != nil || queued {
		t.Fatalf("wait candidate queued=%v err=%v", queued, err)
	}
}

func TestPostgresScheduledAnalysisLockIsSharedAcrossRepositoryInstances(t *testing.T) {
	db, _ := newTestWatchlistRepository(t)
	first := NewPostgresScheduledAnalysisRepository(db)
	second := NewPostgresScheduledAnalysisRepository(db)
	unlock, acquired, err := first.TrySchedulerLock(context.Background())
	if err != nil || !acquired {
		t.Fatalf("first lock acquired=%v err=%v", acquired, err)
	}
	defer unlock()
	_, acquired, err = second.TrySchedulerLock(context.Background())
	if err != nil || acquired {
		t.Fatalf("second lock acquired=%v err=%v, want held lock", acquired, err)
	}
	unlock()
	unlockAgain, acquired, err := second.TrySchedulerLock(context.Background())
	if err != nil || !acquired {
		t.Fatalf("lock after release acquired=%v err=%v", acquired, err)
	}
	unlockAgain()
}

func TestPostgresScheduledAnalysisOutboxRetriesAndOnlySuccessAffectsDedup(t *testing.T) {
	db, _ := newTestWatchlistRepository(t)
	repo := NewPostgresScheduledAnalysisRepository(db)
	ctx := context.Background()
	comparison := AlertComparison{Timing: "left", Direction: "Short", Interval: "1h", EntryType: "limit", EntryPrice: 100, TakeProfit: 90, StopLoss: 110, ATR: 2}
	result := insertIntegrationResult(t, repo, ctx, time.Date(2026, 9, 28, 1, 0, 0, 0, time.UTC), "ETHUSDT", "actionable", 79)
	queued, err := repo.QueueAlert(ctx, result, comparison, "retry candidate")
	if err != nil || !queued {
		t.Fatalf("queue candidate=%v err=%v", queued, err)
	}
	claimed, err := repo.ClaimDueAlerts(ctx, 10)
	if err != nil || len(claimed) != 1 || claimed[0].Attempts != 1 {
		t.Fatalf("first claim=%+v err=%v", claimed, err)
	}
	if err := repo.MarkAlertFailure(ctx, claimed[0].ID, claimed[0].Attempts, 6, 503, 0, "http_status", time.Now().Add(-time.Second)); err != nil {
		t.Fatalf("mark temporary failure: %v", err)
	}
	retried, err := repo.ClaimDueAlerts(ctx, 10)
	if err != nil || len(retried) != 1 || retried[0].Attempts != 2 {
		t.Fatalf("retry claim=%+v err=%v", retried, err)
	}
	if err := repo.MarkAlertSent(ctx, retried[0].ID, 200, 0); err != nil {
		t.Fatalf("mark retry delivered: %v", err)
	}
	stable := insertIntegrationResult(t, repo, ctx, time.Date(2026, 9, 28, 2, 0, 0, 0, time.UTC), "ETHUSDT", "actionable", 81)
	queued, err = repo.QueueAlert(ctx, stable, comparison, "stable after retry")
	if err != nil || queued {
		t.Fatalf("successfully retried candidate must deduplicate: queued=%v err=%v", queued, err)
	}
}

func TestPostgresScheduledAnalysisOutboxOrdersByConfidenceAndReclaimsExpiredDelivery(t *testing.T) {
	db, _ := newTestWatchlistRepository(t)
	repo := NewPostgresScheduledAnalysisRepository(db)
	ctx := context.Background()
	comparison := AlertComparison{Timing: "right", Direction: "Long", Interval: "1h", EntryType: "market", EntryPrice: 100, TakeProfit: 110, StopLoss: 95, ATR: 2}
	high := insertIntegrationResult(t, repo, ctx, time.Date(2026, 9, 28, 1, 0, 0, 0, time.UTC), "BTCUSDT", "actionable", 91)
	low := insertIntegrationResult(t, repo, ctx, time.Date(2026, 9, 28, 2, 0, 0, 0, time.UTC), "SOLUSDT", "actionable", 76)
	if queued, err := repo.QueueAlert(ctx, low, comparison, "lower confidence"); err != nil || !queued {
		t.Fatalf("queue low confidence=%v err=%v", queued, err)
	}
	if queued, err := repo.QueueAlert(ctx, high, comparison, "higher confidence"); err != nil || !queued {
		t.Fatalf("queue high confidence=%v err=%v", queued, err)
	}
	claimed, err := repo.ClaimDueAlerts(ctx, 10)
	if err != nil || len(claimed) != 2 || claimed[0].Confidence != 91 || claimed[1].Confidence != 76 {
		t.Fatalf("alerts not ordered by confidence: %+v err=%v", claimed, err)
	}
	if err := repo.MarkAlertSent(ctx, claimed[1].ID, 200, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE scheduled_analysis_outbox SET claimed_at = NOW() - INTERVAL '6 minutes' WHERE id = $1`, claimed[0].ID); err != nil {
		t.Fatalf("expire test claim: %v", err)
	}
	reclaimed, err := repo.ClaimDueAlerts(ctx, 10)
	if err != nil || len(reclaimed) != 1 || reclaimed[0].ID != claimed[0].ID || reclaimed[0].Attempts != 2 {
		t.Fatalf("expired claim recovery=%+v err=%v", reclaimed, err)
	}
	if err := repo.MarkAlertFailure(ctx, reclaimed[0].ID, reclaimed[0].Attempts, 2, 503, 0, "http_status", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if pending, err := repo.ClaimDueAlerts(ctx, 10); err != nil || len(pending) != 0 {
		t.Fatalf("exhausted retry was claimed again: %+v err=%v", pending, err)
	}
}

func insertIntegrationResult(t *testing.T, repo *PostgresScheduledAnalysisRepository, ctx context.Context, slot time.Time, symbol, status string, confidence int) ScheduledAnalysisResult {
	t.Helper()
	run, err := repo.CreateOrGetRun(ctx, slot, 1, []string{symbol})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}
	var confidenceValue *int
	if status != "failed" {
		confidenceValue = &confidence
	}
	result := &ScheduledAnalysisResult{RunID: run.ID, Symbol: symbol, Interval: "1h", Status: status, Confidence: confidenceValue, Timing: "undetermined", AnalysisJSON: json.RawMessage(`{"status":"candidate"}`), EvidenceJSON: json.RawMessage(`[]`)}
	if _, err := repo.InsertResult(ctx, result); err != nil {
		t.Fatalf("insert result: %v", err)
	}
	return *result
}
