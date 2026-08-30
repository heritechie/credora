-- Credora engine: assessments table (PostgreSQL)
-- Stores the core assessment lifecycle and input data.

CREATE TABLE IF NOT EXISTS assessments (
    id TEXT PRIMARY KEY,
    applicant_id TEXT NOT NULL,
    applicant_name TEXT NOT NULL DEFAULT '',
    applicant_age INTEGER NOT NULL DEFAULT 0,
    application_id TEXT,
    application_requested_amount BIGINT,
    application_purpose TEXT,
    score JSONB,
    status TEXT NOT NULL DEFAULT 'PENDING',
    policy_id TEXT NOT NULL DEFAULT '',
    policy_version INTEGER NOT NULL DEFAULT 0,
    decision_outcome TEXT,
    decision_outputs JSONB,
    decision_policy_id TEXT NOT NULL DEFAULT '',
    decision_policy_version INTEGER NOT NULL DEFAULT 0,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_assessments_status ON assessments(status);
CREATE INDEX IF NOT EXISTS idx_assessments_created_at ON assessments(created_at);
