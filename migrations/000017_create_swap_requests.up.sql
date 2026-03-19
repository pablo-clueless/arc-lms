-- Create period_swap_requests table
CREATE TABLE IF NOT EXISTS period_swap_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    requesting_period_id UUID NOT NULL REFERENCES periods(id) ON DELETE CASCADE,
    target_period_id UUID NOT NULL REFERENCES periods(id) ON DELETE CASCADE,
    requesting_tutor_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    target_tutor_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED', 'ESCALATED', 'CANCELLED')),
    reason TEXT,
    rejection_reason TEXT,
    escalation_reason TEXT,
    admin_override_reason TEXT,
    admin_override_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    responded_at TIMESTAMPTZ,
    escalated_at TIMESTAMPTZ,

    -- Check constraint: requesting and target periods must be different
    CONSTRAINT chk_swap_requests_different_periods CHECK (requesting_period_id != target_period_id),

    -- Check constraint: requesting and target tutors must be different
    CONSTRAINT chk_swap_requests_different_tutors CHECK (requesting_tutor_id != target_tutor_id)
);

-- Create indexes for period_swap_requests
CREATE INDEX idx_period_swap_requests_tenant_id ON period_swap_requests(tenant_id);
CREATE INDEX idx_period_swap_requests_tenant_id_id ON period_swap_requests(tenant_id, id);
CREATE INDEX idx_period_swap_requests_requesting_period_id ON period_swap_requests(requesting_period_id);
CREATE INDEX idx_period_swap_requests_target_period_id ON period_swap_requests(target_period_id);
CREATE INDEX idx_period_swap_requests_requesting_tutor_id ON period_swap_requests(requesting_tutor_id);
CREATE INDEX idx_period_swap_requests_target_tutor_id ON period_swap_requests(target_tutor_id);
CREATE INDEX idx_period_swap_requests_status ON period_swap_requests(status);
CREATE INDEX idx_period_swap_requests_tenant_id_status ON period_swap_requests(tenant_id, status);
CREATE INDEX idx_period_swap_requests_tenant_id_requesting_tutor_id ON period_swap_requests(tenant_id, requesting_tutor_id);
CREATE INDEX idx_period_swap_requests_tenant_id_target_tutor_id ON period_swap_requests(tenant_id, target_tutor_id);

-- Add comments
COMMENT ON TABLE period_swap_requests IS 'Period swap requests between tutors';
COMMENT ON COLUMN period_swap_requests.admin_override_by IS 'Admin who forced approval/override';
COMMENT ON CONSTRAINT chk_swap_requests_different_periods ON period_swap_requests IS 'Cannot swap a period with itself';
COMMENT ON CONSTRAINT chk_swap_requests_different_tutors ON period_swap_requests IS 'Cannot swap periods with yourself';
