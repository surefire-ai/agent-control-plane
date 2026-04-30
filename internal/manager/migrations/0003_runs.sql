CREATE TABLE IF NOT EXISTS runs (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL REFERENCES tenants(id),
    workspace_id TEXT NOT NULL REFERENCES workspaces(id),
    agent_id TEXT NOT NULL REFERENCES agents(id),
    evaluation_id TEXT NOT NULL DEFAULT '',
    agent_revision TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending',
    runtime_engine TEXT NOT NULL DEFAULT 'eino',
    runner_class TEXT NOT NULL DEFAULT 'adk',
    started_at TEXT NOT NULL DEFAULT '',
    completed_at TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    trace_ref TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
