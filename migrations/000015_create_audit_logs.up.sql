-- Create audit_logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    actor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    actor_role VARCHAR(20) NOT NULL CHECK (actor_role IN ('SUPER_ADMIN', 'ADMIN', 'TUTOR', 'STUDENT')),
    action VARCHAR(50) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id UUID NOT NULL,
    resource_name VARCHAR(200),
    before_state JSONB,
    after_state JSONB,
    changes JSONB,
    ip_address VARCHAR(45) NOT NULL,
    user_agent TEXT,
    metadata JSONB,
    is_sensitive BOOLEAN NOT NULL DEFAULT FALSE,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for audit_logs
CREATE INDEX idx_audit_logs_tenant_id ON audit_logs(tenant_id);
CREATE INDEX idx_audit_logs_actor_user_id ON audit_logs(actor_user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_resource_type ON audit_logs(resource_type);
CREATE INDEX idx_audit_logs_resource_id ON audit_logs(resource_id);
CREATE INDEX idx_audit_logs_is_sensitive ON audit_logs(is_sensitive) WHERE is_sensitive = TRUE;
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp DESC);
CREATE INDEX idx_audit_logs_tenant_id_timestamp ON audit_logs(tenant_id, timestamp DESC);
CREATE INDEX idx_audit_logs_tenant_id_resource_type ON audit_logs(tenant_id, resource_type);
CREATE INDEX idx_audit_logs_tenant_id_actor_user_id ON audit_logs(tenant_id, actor_user_id);

-- Add comments
COMMENT ON TABLE audit_logs IS 'Immutable audit trail for all system actions';
COMMENT ON COLUMN audit_logs.tenant_id IS 'NULL for platform-level actions (e.g., tenant creation)';
COMMENT ON COLUMN audit_logs.before_state IS 'JSONB snapshot of resource before change';
COMMENT ON COLUMN audit_logs.after_state IS 'JSONB snapshot of resource after change';
COMMENT ON COLUMN audit_logs.changes IS 'JSONB object with field-level changes';
COMMENT ON COLUMN audit_logs.is_sensitive IS 'TRUE for sensitive actions (billing, suspension, role changes)';
