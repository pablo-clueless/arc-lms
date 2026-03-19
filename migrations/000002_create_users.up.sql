-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL CHECK (role IN ('SUPER_ADMIN', 'ADMIN', 'TUTOR', 'STUDENT')),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    middle_name VARCHAR(100),
    profile_photo TEXT,
    phone VARCHAR(20),
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE', 'DEACTIVATED', 'PENDING')),
    permissions JSONB NOT NULL DEFAULT '[]'::jsonb,
    notification_preferences JSONB NOT NULL DEFAULT '[]'::jsonb,
    last_login_at TIMESTAMPTZ,
    password_reset_token VARCHAR(255),
    password_reset_expiry TIMESTAMPTZ,
    invitation_token VARCHAR(255),
    invitation_expiry TIMESTAMPTZ,
    deactivated_at TIMESTAMPTZ,
    deactivation_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Check constraint: SUPER_ADMIN must have NULL tenant_id
    CONSTRAINT chk_super_admin_no_tenant CHECK (
        (role = 'SUPER_ADMIN' AND tenant_id IS NULL) OR
        (role != 'SUPER_ADMIN' AND tenant_id IS NOT NULL)
    )
);

-- Create indexes for users
CREATE INDEX idx_users_tenant_id ON users(tenant_id);
CREATE INDEX idx_users_tenant_id_id ON users(tenant_id, id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_tenant_id_role ON users(tenant_id, role);
CREATE INDEX idx_users_password_reset_token ON users(password_reset_token) WHERE password_reset_token IS NOT NULL;
CREATE INDEX idx_users_invitation_token ON users(invitation_token) WHERE invitation_token IS NOT NULL;

-- Add comments
COMMENT ON TABLE users IS 'Platform users with RBAC support (SUPER_ADMIN, ADMIN, TUTOR, STUDENT)';
COMMENT ON COLUMN users.tenant_id IS 'NULL for SUPER_ADMIN, required for all other roles';
COMMENT ON COLUMN users.permissions IS 'Array of permission strings like ["tenant:create", "user:update"]';
COMMENT ON CONSTRAINT chk_super_admin_no_tenant ON users IS 'BR-001: SUPER_ADMIN must have null tenant_id';
