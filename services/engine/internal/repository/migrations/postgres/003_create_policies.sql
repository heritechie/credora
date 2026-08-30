-- Credora engine: policies table (PostgreSQL)
-- Stores policy metadata for audit and reference purposes.
-- Executable policy logic lives in Go code (policy registry).

CREATE TABLE IF NOT EXISTS policies (
    id TEXT NOT NULL,
    version INTEGER NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, version)
);
