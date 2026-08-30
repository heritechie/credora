-- Credora engine: decision reasons and evidence tables (SQLite)
-- Stores structured decision explanations and evidence provenance.

CREATE TABLE IF NOT EXISTS decision_reasons (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    assessment_id TEXT NOT NULL REFERENCES assessments(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    value TEXT,
    threshold TEXT,
    evidence_ref TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_decision_reasons_assessment ON decision_reasons(assessment_id);

CREATE TABLE IF NOT EXISTS evidence (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    assessment_id TEXT NOT NULL REFERENCES assessments(id) ON DELETE CASCADE,
    source TEXT NOT NULL,
    field TEXT NOT NULL,
    value TEXT,
    retrieved_at TEXT NOT NULL DEFAULT (datetime('now')),
    reference TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_evidence_assessment ON evidence(assessment_id);
