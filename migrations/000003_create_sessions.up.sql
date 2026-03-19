-- Create sessions table
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    label VARCHAR(20) NOT NULL,
    start_year INTEGER NOT NULL CHECK (start_year >= 2000 AND start_year <= 2100),
    end_year INTEGER NOT NULL CHECK (end_year >= 2000 AND end_year <= 2100),
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('ACTIVE', 'ARCHIVED', 'DRAFT')),
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at TIMESTAMPTZ,

    -- Check constraint: end_year must be greater than start_year
    CONSTRAINT chk_sessions_year_order CHECK (end_year > start_year)
);

-- Create unique partial index for one active session per tenant (BR-007)
CREATE UNIQUE INDEX idx_sessions_one_active_per_tenant
ON sessions(tenant_id)
WHERE status = 'ACTIVE';

-- Create indexes for sessions
CREATE INDEX idx_sessions_tenant_id ON sessions(tenant_id);
CREATE INDEX idx_sessions_tenant_id_id ON sessions(tenant_id, id);
CREATE INDEX idx_sessions_status ON sessions(status);
CREATE INDEX idx_sessions_tenant_id_status ON sessions(tenant_id, status);

-- Add comments
COMMENT ON TABLE sessions IS 'Academic sessions (e.g., 2025/2026) - one active per tenant';
COMMENT ON INDEX idx_sessions_one_active_per_tenant IS 'BR-007: Ensures only one active session per tenant';
