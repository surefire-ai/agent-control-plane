CREATE TABLE IF NOT EXISTS provider_accounts (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL REFERENCES tenants(id),
    workspace_id TEXT NOT NULL DEFAULT '',
    provider TEXT NOT NULL,
    display_name TEXT NOT NULL,
    family TEXT NOT NULL DEFAULT '',
    base_url TEXT NOT NULL DEFAULT '',
    credential_ref TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    domestic BOOLEAN NOT NULL DEFAULT false,
    supports_json_schema BOOLEAN NOT NULL DEFAULT false,
    supports_tool_calling BOOLEAN NOT NULL DEFAULT false,
    capabilities JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, workspace_id, provider)
);
