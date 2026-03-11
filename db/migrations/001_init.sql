CREATE TABLE IF NOT EXISTS jobs (
    job_id TEXT PRIMARY KEY,
    app_id TEXT NOT NULL,
    state TEXT NOT NULL,
    payload JSONB NOT NULL,
    result JSONB,
    error TEXT,
    attempts INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS jobs_state_created_idx ON jobs (state, created_at);
