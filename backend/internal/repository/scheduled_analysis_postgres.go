package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

const scheduledAnalysisAdvisoryLockID int64 = 0x5343414e57415443

type ScheduledAnalysisRun struct {
	ID               string
	ScheduledFor     time.Time
	SnapshotRevision int64
	Symbols          []string
	Status           string
	StartedAt        time.Time
	FinishedAt       *time.Time
	CompletedCount   int
}

type ScheduledAnalysisResult struct {
	ID           string
	RunID        string
	Symbol       string
	GeneratedAt  time.Time
	ContextTime  int64
	Interval     string
	Status       string
	Confidence   *int
	Timing       string
	Direction    string
	AnalysisJSON json.RawMessage
	EvidenceJSON json.RawMessage
	Model        string
	ErrorCode    string
}

type AlertComparison struct {
	Timing     string  `json:"timing"`
	Direction  string  `json:"direction"`
	Interval   string  `json:"interval"`
	EntryType  string  `json:"entry_type"`
	EntryPrice float64 `json:"entry_price"`
	TakeProfit float64 `json:"take_profit"`
	StopLoss   float64 `json:"stop_loss"`
	ATR        float64 `json:"atr"`
}

type ScheduledAnalysisOutboxItem struct {
	ID          string
	ResultID    string
	Symbol      string
	Fingerprint string
	Payload     string
	Comparison  AlertComparison
	Confidence  int
	Attempts    int
}

type PostgresScheduledAnalysisRepository struct {
	db *sql.DB
}

func NewPostgresScheduledAnalysisRepository(db *sql.DB) *PostgresScheduledAnalysisRepository {
	return &PostgresScheduledAnalysisRepository{db: db}
}

func (r *PostgresScheduledAnalysisRepository) TrySchedulerLock(ctx context.Context) (func(), bool, error) {
	conn, err := r.db.Conn(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("get scan lock connection: %w", err)
	}
	var acquired bool
	if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", scheduledAnalysisAdvisoryLockID).Scan(&acquired); err != nil {
		_ = conn.Close()
		return nil, false, fmt.Errorf("acquire scan lock: %w", err)
	}
	if !acquired {
		_ = conn.Close()
		return func() {}, false, nil
	}
	var once sync.Once
	unlock := func() {
		once.Do(func() {
			_, _ = conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", scheduledAnalysisAdvisoryLockID)
			_ = conn.Close()
		})
	}
	return unlock, true, nil
}

func (r *PostgresScheduledAnalysisRepository) CreateOrGetRun(ctx context.Context, scheduledFor time.Time, revision int64, symbols []string) (ScheduledAnalysisRun, error) {
	id, err := newScheduledAnalysisID()
	if err != nil {
		return ScheduledAnalysisRun{}, err
	}
	encodedSymbols, err := json.Marshal(symbols)
	if err != nil {
		return ScheduledAnalysisRun{}, fmt.Errorf("encode scan Watchlist: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO scheduled_analysis_runs(id, scheduled_for, snapshot_revision, symbols)
		VALUES ($1, $2, $3, $4::jsonb)
		ON CONFLICT (scheduled_for) DO NOTHING
	`, id, scheduledFor.UTC(), revision, string(encodedSymbols))
	if err != nil {
		return ScheduledAnalysisRun{}, fmt.Errorf("create scan run: %w", err)
	}
	var run ScheduledAnalysisRun
	var rawSymbols []byte
	var finished sql.NullTime
	if err := r.db.QueryRowContext(ctx, `
		SELECT id::text, scheduled_for, snapshot_revision, symbols, status, started_at, finished_at, completed_count
		FROM scheduled_analysis_runs WHERE scheduled_for = $1
	`, scheduledFor.UTC()).Scan(&run.ID, &run.ScheduledFor, &run.SnapshotRevision, &rawSymbols, &run.Status, &run.StartedAt, &finished, &run.CompletedCount); err != nil {
		return ScheduledAnalysisRun{}, fmt.Errorf("read scan run: %w", err)
	}
	if err := json.Unmarshal(rawSymbols, &run.Symbols); err != nil {
		return ScheduledAnalysisRun{}, fmt.Errorf("decode saved Watchlist snapshot: %w", err)
	}
	if finished.Valid {
		run.FinishedAt = &finished.Time
	}
	return run, nil
}

func (r *PostgresScheduledAnalysisRepository) IncompleteRuns(ctx context.Context, beforeOrAt time.Time, limit int) ([]ScheduledAnalysisRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 24
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, scheduled_for, snapshot_revision, symbols, status, started_at, finished_at, completed_count
		FROM scheduled_analysis_runs
		WHERE scheduled_for <= $1 AND status IN ('running', 'partial', 'interrupted')
		  AND completed_count < jsonb_array_length(symbols)
		ORDER BY scheduled_for DESC LIMIT $2
	`, beforeOrAt.UTC(), limit)
	if err != nil {
		return nil, fmt.Errorf("list incomplete scan runs: %w", err)
	}
	defer rows.Close()
	runs := make([]ScheduledAnalysisRun, 0)
	for rows.Next() {
		var run ScheduledAnalysisRun
		var rawSymbols []byte
		var finished sql.NullTime
		if err := rows.Scan(&run.ID, &run.ScheduledFor, &run.SnapshotRevision, &rawSymbols, &run.Status, &run.StartedAt, &finished, &run.CompletedCount); err != nil {
			return nil, fmt.Errorf("scan incomplete run: %w", err)
		}
		if err := json.Unmarshal(rawSymbols, &run.Symbols); err != nil {
			return nil, fmt.Errorf("decode incomplete run Watchlist snapshot: %w", err)
		}
		if finished.Valid {
			run.FinishedAt = &finished.Time
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate incomplete runs: %w", err)
	}
	return runs, nil
}

func (r *PostgresScheduledAnalysisRepository) CompletedSymbols(ctx context.Context, runID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT symbol FROM scheduled_analysis_results WHERE run_id = $1 ORDER BY symbol`, runID)
	if err != nil {
		return nil, fmt.Errorf("list completed scan symbols: %w", err)
	}
	defer rows.Close()
	symbols := make([]string, 0)
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, fmt.Errorf("scan completed symbol: %w", err)
		}
		symbols = append(symbols, symbol)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate completed symbols: %w", err)
	}
	return symbols, nil
}

func (r *PostgresScheduledAnalysisRepository) InsertResult(ctx context.Context, result *ScheduledAnalysisResult) (bool, error) {
	if result == nil || result.RunID == "" || result.Symbol == "" || result.Interval == "" {
		return false, errors.New("scan result is incomplete")
	}
	if result.ID == "" {
		id, err := newScheduledAnalysisID()
		if err != nil {
			return false, err
		}
		result.ID = id
	}
	if result.GeneratedAt.IsZero() {
		result.GeneratedAt = time.Now().UTC()
	}
	if result.Timing == "" {
		result.Timing = "undetermined"
	}
	if len(result.AnalysisJSON) == 0 {
		result.AnalysisJSON = json.RawMessage("null")
	}
	if len(result.EvidenceJSON) == 0 {
		result.EvidenceJSON = json.RawMessage("[]")
	}
	var confidence any
	if result.Confidence != nil {
		confidence = *result.Confidence
	}
	var inserted string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO scheduled_analysis_results(
			id, run_id, symbol, generated_at, context_time, interval, result_status, confidence,
			timing, direction, analysis, evidence, model, error_code
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb, $12::jsonb, $13, $14)
		ON CONFLICT (run_id, symbol) DO NOTHING
		RETURNING id::text
	`, result.ID, result.RunID, result.Symbol, result.GeneratedAt, result.ContextTime, result.Interval, result.Status,
		confidence, result.Timing, result.Direction, string(result.AnalysisJSON), string(result.EvidenceJSON), result.Model, result.ErrorCode).Scan(&inserted)
	if errors.Is(err, sql.ErrNoRows) {
		if err := r.db.QueryRowContext(ctx, "SELECT id::text FROM scheduled_analysis_results WHERE run_id = $1 AND symbol = $2", result.RunID, result.Symbol).Scan(&result.ID); err != nil {
			return false, fmt.Errorf("read existing scan result: %w", err)
		}
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("insert scan result: %w", err)
	}
	result.ID = inserted
	return true, nil
}

func (r *PostgresScheduledAnalysisRepository) ListResults(ctx context.Context, runID string) ([]ScheduledAnalysisResult, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, run_id::text, symbol, generated_at, context_time, interval, result_status,
		       confidence, timing, direction, analysis, evidence, model, error_code
		FROM scheduled_analysis_results WHERE run_id = $1 ORDER BY symbol
	`, runID)
	if err != nil {
		return nil, fmt.Errorf("list scan results: %w", err)
	}
	defer rows.Close()
	results := make([]ScheduledAnalysisResult, 0)
	for rows.Next() {
		var result ScheduledAnalysisResult
		var confidence sql.NullInt64
		if err := rows.Scan(&result.ID, &result.RunID, &result.Symbol, &result.GeneratedAt, &result.ContextTime, &result.Interval, &result.Status,
			&confidence, &result.Timing, &result.Direction, &result.AnalysisJSON, &result.EvidenceJSON, &result.Model, &result.ErrorCode); err != nil {
			return nil, fmt.Errorf("scan saved result: %w", err)
		}
		if confidence.Valid {
			value := int(confidence.Int64)
			result.Confidence = &value
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate scan results: %w", err)
	}
	return results, nil
}

func (r *PostgresScheduledAnalysisRepository) FinishRun(ctx context.Context, runID, status string) error {
	if status != "complete" && status != "partial" && status != "interrupted" {
		return errors.New("invalid scan run status")
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE scheduled_analysis_runs AS run
		SET status = $2, finished_at = NOW(),
		    completed_count = (SELECT COUNT(*) FROM scheduled_analysis_results WHERE run_id = run.id)
		WHERE id = $1
	`, runID, status)
	if err != nil {
		return fmt.Errorf("finish scan run: %w", err)
	}
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *PostgresScheduledAnalysisRepository) QueueAlert(ctx context.Context, result ScheduledAnalysisResult, comparison AlertComparison, payload string) (bool, error) {
	if result.Status != "actionable" || result.Confidence == nil || *result.Confidence <= 70 || payload == "" {
		return false, nil
	}
	var runSlot time.Time
	if err := r.db.QueryRowContext(ctx, "SELECT scheduled_for FROM scheduled_analysis_runs WHERE id = $1", result.RunID).Scan(&runSlot); err != nil {
		return false, fmt.Errorf("load alert scan slot: %w", err)
	}
	var priorStatus string
	var priorConfidence int
	err := r.db.QueryRowContext(ctx, `
		SELECT previous.result_status, COALESCE(previous.confidence, 0)
		FROM scheduled_analysis_results AS previous
		JOIN scheduled_analysis_runs AS previous_run ON previous_run.id = previous.run_id
		WHERE previous.symbol = $1 AND previous_run.scheduled_for < $2
		ORDER BY previous_run.scheduled_for DESC LIMIT 1
	`, result.Symbol, runSlot).Scan(&priorStatus, &priorConfidence)
	requalified := errors.Is(err, sql.ErrNoRows) || (err == nil && (priorStatus != "actionable" || priorConfidence <= 70))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("read prior scan result for alert: %w", err)
	}

	var previousComparisonJSON []byte
	err = r.db.QueryRowContext(ctx, `
		SELECT comparison_data FROM scheduled_analysis_outbox
		WHERE symbol = $1 AND status = 'sent'
		ORDER BY sent_at DESC LIMIT 1
	`, result.Symbol).Scan(&previousComparisonJSON)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("read prior sent alert: %w", err)
	}
	changed := errors.Is(err, sql.ErrNoRows) || requalified
	if err == nil && !changed {
		var previous AlertComparison
		if err := json.Unmarshal(previousComparisonJSON, &previous); err != nil {
			return false, fmt.Errorf("decode prior alert comparison: %w", err)
		}
		changed = hasMaterialAlertChange(previous, comparison)
	}
	if !changed {
		return false, nil
	}

	fingerprintBytes, err := json.Marshal(comparison)
	if err != nil {
		return false, fmt.Errorf("encode alert fingerprint: %w", err)
	}
	fingerprint := fmt.Sprintf("%x", sha256.Sum256(fingerprintBytes))
	pendingRows, err := r.db.QueryContext(ctx, `
		SELECT id::text, status, comparison_data, claimed_at
		FROM scheduled_analysis_outbox
		WHERE symbol = $1 AND status IN ('pending', 'sending', 'retry')
		ORDER BY created_at DESC
	`, result.Symbol)
	if err != nil {
		return false, fmt.Errorf("check pending alert retries: %w", err)
	}
	type pendingAlert struct {
		id         string
		status     string
		comparison AlertComparison
		claimedAt  sql.NullTime
	}
	existingAlerts := make([]pendingAlert, 0)
	for pendingRows.Next() {
		var item pendingAlert
		var rawComparison []byte
		if err := pendingRows.Scan(&item.id, &item.status, &rawComparison, &item.claimedAt); err != nil {
			_ = pendingRows.Close()
			return false, fmt.Errorf("scan pending alert retry: %w", err)
		}
		if err := json.Unmarshal(rawComparison, &item.comparison); err != nil {
			_ = pendingRows.Close()
			return false, fmt.Errorf("decode pending alert comparison: %w", err)
		}
		existingAlerts = append(existingAlerts, item)
	}
	if err := pendingRows.Err(); err != nil {
		_ = pendingRows.Close()
		return false, fmt.Errorf("iterate pending alert retries: %w", err)
	}
	_ = pendingRows.Close()
	for _, pending := range existingAlerts {
		if !hasMaterialAlertChange(pending.comparison, comparison) {
			return false, nil
		}
		if pending.status == "sending" && pending.claimedAt.Valid && time.Since(pending.claimedAt.Time) < 5*time.Minute {
			return false, nil
		}
		if _, err := r.db.ExecContext(ctx, `
			UPDATE scheduled_analysis_outbox SET status = 'failed', last_error_code = 'superseded'
			WHERE id = $1 AND status IN ('pending', 'retry', 'sending')
		`, pending.id); err != nil {
			return false, fmt.Errorf("supersede stale analysis alert: %w", err)
		}
	}
	id, err := newScheduledAnalysisID()
	if err != nil {
		return false, err
	}
	comparisonJSON, err := json.Marshal(comparison)
	if err != nil {
		return false, fmt.Errorf("encode alert comparison: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO scheduled_analysis_outbox(id, result_id, symbol, fingerprint, payload, comparison_data)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb)
		ON CONFLICT (result_id) DO NOTHING
	`, id, result.ID, result.Symbol, fingerprint, payload, string(comparisonJSON))
	if err != nil {
		return false, fmt.Errorf("queue analysis alert: %w", err)
	}
	return true, nil
}

func hasMaterialAlertChange(previous, current AlertComparison) bool {
	if previous.Timing != current.Timing || previous.Direction != current.Direction || previous.Interval != current.Interval || previous.EntryType != current.EntryType {
		return true
	}
	threshold := 0.5 * current.ATR
	for _, pair := range [][2]float64{{previous.EntryPrice, current.EntryPrice}, {previous.TakeProfit, current.TakeProfit}, {previous.StopLoss, current.StopLoss}} {
		difference := math.Abs(pair[0] - pair[1])
		if (threshold <= 0 && difference > 0) || (threshold > 0 && difference >= threshold) {
			return true
		}
	}
	return false
}

func (r *PostgresScheduledAnalysisRepository) ClaimDueAlerts(ctx context.Context, limit int) ([]ScheduledAnalysisOutboxItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if _, err := r.db.ExecContext(ctx, `
		UPDATE scheduled_analysis_outbox
		SET status = 'failed', last_error_code = 'delivery_lease_expired'
		WHERE status = 'sending' AND attempts >= 6 AND claimed_at <= NOW() - INTERVAL '5 minutes'
	`); err != nil {
		return nil, fmt.Errorf("expire exhausted analysis alert leases: %w", err)
	}
	rows, err := r.db.QueryContext(ctx, `
		WITH due AS (
			SELECT outbox.id, COALESCE(result.confidence, 0)::integer AS confidence
			FROM scheduled_analysis_outbox AS outbox
			JOIN scheduled_analysis_results AS result ON result.id = outbox.result_id
			WHERE (outbox.status IN ('pending', 'retry') AND outbox.next_attempt_at <= NOW())
			   OR (outbox.status = 'sending' AND outbox.attempts < 6 AND outbox.claimed_at <= NOW() - INTERVAL '5 minutes')
			ORDER BY COALESCE(result.confidence, 0) DESC, outbox.created_at, outbox.id
			FOR UPDATE OF outbox SKIP LOCKED LIMIT $1
		)
		UPDATE scheduled_analysis_outbox AS outbox
		SET status = 'sending', attempts = attempts + 1, claimed_at = NOW()
		FROM due WHERE outbox.id = due.id
		RETURNING outbox.id::text, outbox.result_id::text, outbox.symbol, outbox.fingerprint, outbox.payload,
		          outbox.comparison_data, due.confidence, outbox.attempts
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("claim due analysis alerts: %w", err)
	}
	defer rows.Close()
	items := make([]ScheduledAnalysisOutboxItem, 0)
	for rows.Next() {
		var item ScheduledAnalysisOutboxItem
		var comparisonJSON []byte
		if err := rows.Scan(&item.ID, &item.ResultID, &item.Symbol, &item.Fingerprint, &item.Payload, &comparisonJSON, &item.Confidence, &item.Attempts); err != nil {
			return nil, fmt.Errorf("scan claimed alert: %w", err)
		}
		if err := json.Unmarshal(comparisonJSON, &item.Comparison); err != nil {
			return nil, fmt.Errorf("decode claimed alert comparison: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate claimed alerts: %w", err)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Confidence != items[j].Confidence {
			return items[i].Confidence > items[j].Confidence
		}
		return items[i].ID < items[j].ID
	})
	return items, nil
}

func (r *PostgresScheduledAnalysisRepository) MarkAlertSent(ctx context.Context, id string, httpStatus, apiCode int) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE scheduled_analysis_outbox
		SET status = 'sent', sent_at = NOW(), last_http_status = $2, last_wecom_errcode = $3, last_error_code = ''
		WHERE id = $1 AND status = 'sending'
	`, id, httpStatus, apiCode)
	if err != nil {
		return fmt.Errorf("mark analysis alert sent: %w", err)
	}
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return errors.New("analysis alert is no longer claimed")
	}
	return nil
}

func (r *PostgresScheduledAnalysisRepository) MarkAlertFailure(ctx context.Context, id string, attempts, maxAttempts, httpStatus, apiCode int, errorCode string, retryAt time.Time) error {
	if maxAttempts <= 0 {
		maxAttempts = 6
	}
	status := "retry"
	if attempts >= maxAttempts {
		status = "failed"
	}
	var statusValue any = httpStatus
	if httpStatus <= 0 {
		statusValue = nil
	}
	var apiCodeValue any = apiCode
	if apiCode <= 0 {
		apiCodeValue = nil
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE scheduled_analysis_outbox
		SET status = $2, next_attempt_at = $3, last_http_status = $4, last_wecom_errcode = $5, last_error_code = $6
		WHERE id = $1 AND status = 'sending'
	`, id, status, retryAt.UTC(), statusValue, apiCodeValue, errorCode)
	if err != nil {
		return fmt.Errorf("mark analysis alert failure: %w", err)
	}
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return errors.New("analysis alert is no longer claimed")
	}
	return nil
}

func newScheduledAnalysisID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate scan identifier: %w", err)
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	encoded := make([]byte, 36)
	hex.Encode(encoded[0:8], value[0:4])
	encoded[8] = '-'
	hex.Encode(encoded[9:13], value[4:6])
	encoded[13] = '-'
	hex.Encode(encoded[14:18], value[6:8])
	encoded[18] = '-'
	hex.Encode(encoded[19:23], value[8:10])
	encoded[23] = '-'
	hex.Encode(encoded[24:36], value[10:16])
	return string(encoded), nil
}
