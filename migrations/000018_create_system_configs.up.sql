-- Create system_configs table for platform-wide configuration
CREATE TABLE IF NOT EXISTS system_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key VARCHAR(100) NOT NULL UNIQUE,
    value JSONB NOT NULL,
    description TEXT,
    category VARCHAR(50) NOT NULL DEFAULT 'general',
    is_sensitive BOOLEAN NOT NULL DEFAULT false,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX idx_system_configs_key ON system_configs(key);
CREATE INDEX idx_system_configs_category ON system_configs(category);

-- Add comments
COMMENT ON TABLE system_configs IS 'Platform-wide system configuration settings managed by SUPER_ADMIN';
COMMENT ON COLUMN system_configs.key IS 'Unique configuration key (e.g., platform.name, maintenance.enabled)';
COMMENT ON COLUMN system_configs.value IS 'JSON value for the configuration';
COMMENT ON COLUMN system_configs.category IS 'Category for grouping configs (general, billing, email, security, features)';
COMMENT ON COLUMN system_configs.is_sensitive IS 'If true, value should be masked in API responses';

-- Insert default system configurations
INSERT INTO system_configs (key, value, description, category, is_sensitive) VALUES
    ('platform.name', '"Arc LMS"', 'Platform display name', 'general', false),
    ('platform.version', '"1.0.0"', 'Current platform version', 'general', false),
    ('platform.support_email', '"support@arc-lms.com"', 'Platform support email address', 'general', false),
    ('platform.support_url', '"https://arc-lms.com/support"', 'Platform support URL', 'general', false),
    ('maintenance.enabled', 'false', 'Enable maintenance mode', 'maintenance', false),
    ('maintenance.message', '"The platform is currently undergoing scheduled maintenance. Please try again later."', 'Message shown during maintenance', 'maintenance', false),
    ('maintenance.allowed_ips', '[]', 'IP addresses allowed during maintenance', 'maintenance', false),
    ('billing.currency', '"NGN"', 'Default billing currency', 'billing', false),
    ('billing.student_fee_per_term', '500', 'Fee per student per term in default currency', 'billing', false),
    ('billing.grace_period_days', '14', 'Default invoice grace period in days', 'billing', false),
    ('billing.late_fee_percentage', '5', 'Late payment fee percentage', 'billing', false),
    ('email.from_name', '"Arc LMS"', 'Default sender name for emails', 'email', false),
    ('email.from_address', '"noreply@arc-lms.com"', 'Default sender email address', 'email', false),
    ('email.daily_limit_per_tenant', '1000', 'Maximum emails per tenant per day', 'email', false),
    ('security.max_login_attempts', '5', 'Maximum failed login attempts before lockout', 'security', false),
    ('security.lockout_duration_minutes', '30', 'Account lockout duration in minutes', 'security', false),
    ('security.password_min_length', '8', 'Minimum password length', 'security', false),
    ('security.session_timeout_minutes', '60', 'Session inactivity timeout in minutes', 'security', false),
    ('security.jwt_access_token_ttl_minutes', '15', 'JWT access token TTL in minutes', 'security', false),
    ('security.jwt_refresh_token_ttl_days', '30', 'JWT refresh token TTL in days', 'security', false),
    ('features.websocket_enabled', 'true', 'Enable WebSocket real-time notifications', 'features', false),
    ('features.email_notifications_enabled', 'true', 'Enable email notifications', 'features', false),
    ('features.meeting_recording_enabled', 'true', 'Enable meeting recording feature', 'features', false),
    ('features.examination_proctoring_enabled', 'true', 'Enable examination integrity monitoring', 'features', false),
    ('rate_limit.requests_per_minute_tenant', '1000', 'Rate limit per tenant per minute', 'rate_limit', false),
    ('rate_limit.requests_per_minute_superadmin', '2000', 'Rate limit for superadmin per minute', 'rate_limit', false),
    ('rate_limit.burst_multiplier', '2', 'Burst rate multiplier', 'rate_limit', false),
    ('defaults.timezone', '"Africa/Lagos"', 'Default timezone for new tenants', 'defaults', false),
    ('defaults.period_duration_minutes', '45', 'Default class period duration', 'defaults', false),
    ('defaults.daily_period_limit', '8', 'Default max periods per day', 'defaults', false);
