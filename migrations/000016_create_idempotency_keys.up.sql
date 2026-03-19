-- Create idempotency_keys table
CREATE TABLE IF NOT EXISTS idempotency_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key VARCHAR(255) NOT NULL UNIQUE,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    request_method VARCHAR(10) NOT NULL,
    request_path TEXT NOT NULL,
    request_body JSONB,
    response_status_code INTEGER,
    response_body JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

-- Create indexes for idempotency_keys
CREATE INDEX idx_idempotency_keys_key ON idempotency_keys(key);
CREATE INDEX idx_idempotency_keys_tenant_id ON idempotency_keys(tenant_id);
CREATE INDEX idx_idempotency_keys_user_id ON idempotency_keys(user_id);
CREATE INDEX idx_idempotency_keys_expires_at ON idempotency_keys(expires_at);
CREATE INDEX idx_idempotency_keys_created_at ON idempotency_keys(created_at);

-- Add comments
COMMENT ON TABLE idempotency_keys IS 'Idempotency keys for preventing duplicate operations';
COMMENT ON COLUMN idempotency_keys.key IS 'Unique idempotency key from client (e.g., UUID)';
COMMENT ON COLUMN idempotency_keys.expires_at IS 'Keys expire after 24 hours for cleanup';
COMMENT ON COLUMN idempotency_keys.response_body IS 'Cached response for replay to duplicate requests';
