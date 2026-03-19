-- Create tenants table
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    school_type VARCHAR(20) NOT NULL CHECK (school_type IN ('PRIMARY', 'SECONDARY', 'COMBINED')),
    contact_email VARCHAR(255) NOT NULL,
    address TEXT NOT NULL,
    logo TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'SUSPENDED')),
    configuration JSONB NOT NULL DEFAULT '{}'::jsonb,
    billing_contact JSONB NOT NULL,
    suspension_reason TEXT,
    principal_admin_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    suspended_at TIMESTAMPTZ
);

-- Create indexes for tenants
CREATE INDEX idx_tenants_status ON tenants(status);
CREATE INDEX idx_tenants_principal_admin_id ON tenants(principal_admin_id);
CREATE INDEX idx_tenants_created_at ON tenants(created_at DESC);

-- Add comments
COMMENT ON TABLE tenants IS 'Multi-tenant schools with configuration and billing information';
COMMENT ON COLUMN tenants.configuration IS 'JSONB containing timezone, grade_weighting, attendance_threshold, etc.';
COMMENT ON COLUMN tenants.billing_contact IS 'JSONB containing name, email, and phone';
