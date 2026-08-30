-- Credora engine: decision reasons and evidence tables (PostgreSQL)
-- Stores structured decision explanations and evidence provenance.

CREATE TABLE IF NOT EXISTS decision_reasons (
    id SERIAL PRIMARY KEY,
    assessment_id TEXT NOT NULL REFERENCES assessments(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    value JSONB,
    threshold JSONB,
    evidence_ref TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_decision_reasons_assessment ON decision_reasons(assessment_id);

CREATE TABLE IF NOT EXISTS evidence (
    id SERIAL PRIMARY KEY,
    assessment_id TEXT NOT NULL REFERENCES assessments(id) ON DELETE CASCADE,
    source TEXT NOT NULL,
    field TEXT NOT NULL,
    value JSONB,
    retrieved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reference TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_evidence_assessment ON evidence(assessment_id);
