CREATE TABLE IF NOT EXISTS scheduled_analysis_runs (
    id UUID PRIMARY KEY,
    scheduled_for TIMESTAMPTZ NOT NULL UNIQUE,
    snapshot_revision BIGINT NOT NULL,
    symbols JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'complete', 'partial', 'interrupted')),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    completed_count INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS scheduled_analysis_runs_status_idx
    ON scheduled_analysis_runs(status, scheduled_for);

CREATE TABLE IF NOT EXISTS scheduled_analysis_results (
    id UUID PRIMARY KEY,
    run_id UUID NOT NULL REFERENCES scheduled_analysis_runs(id) ON DELETE RESTRICT,
    symbol TEXT NOT NULL,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    context_time BIGINT NOT NULL DEFAULT 0,
    interval TEXT NOT NULL,
    result_status TEXT NOT NULL CHECK (result_status IN ('actionable', 'wait', 'failed')),
    confidence INTEGER CHECK (confidence IS NULL OR confidence BETWEEN 0 AND 100),
    timing TEXT NOT NULL DEFAULT 'undetermined' CHECK (timing IN ('left', 'right', 'undetermined')),
    direction TEXT NOT NULL DEFAULT '',
    analysis JSONB NOT NULL DEFAULT 'null'::jsonb,
    evidence JSONB NOT NULL DEFAULT '[]'::jsonb,
    model TEXT NOT NULL DEFAULT '',
    error_code TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(run_id, symbol)
);

CREATE INDEX IF NOT EXISTS scheduled_analysis_results_symbol_idx
    ON scheduled_analysis_results(symbol, created_at DESC);

CREATE TABLE IF NOT EXISTS scheduled_analysis_outbox (
    id UUID PRIMARY KEY,
    result_id UUID NOT NULL REFERENCES scheduled_analysis_results(id) ON DELETE RESTRICT,
    symbol TEXT NOT NULL,
    fingerprint TEXT NOT NULL,
    payload TEXT NOT NULL,
    comparison_data JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sending', 'retry', 'sent', 'failed')),
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    claimed_at TIMESTAMPTZ,
    last_http_status INTEGER,
    last_wecom_errcode INTEGER,
    last_error_code TEXT NOT NULL DEFAULT '',
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(result_id)
);

CREATE INDEX IF NOT EXISTS scheduled_analysis_outbox_due_idx
    ON scheduled_analysis_outbox(status, next_attempt_at, claimed_at, created_at);

CREATE INDEX IF NOT EXISTS scheduled_analysis_outbox_success_symbol_idx
    ON scheduled_analysis_outbox(symbol, sent_at DESC)
    WHERE status = 'sent';
