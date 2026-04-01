-- Create guardians table for parent-student relationships
CREATE TABLE IF NOT EXISTS guardians (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    guardian_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    relationship VARCHAR(20) NOT NULL CHECK (relationship IN ('FATHER', 'MOTHER', 'GUARDIAN', 'OTHER')),
    is_primary BOOLEAN NOT NULL DEFAULT false,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE')),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ensure unique guardian-student pair per tenant
    CONSTRAINT unique_guardian_student UNIQUE (tenant_id, guardian_id, student_id)
);

-- Create indexes
CREATE INDEX idx_guardians_tenant_id ON guardians(tenant_id);
CREATE INDEX idx_guardians_guardian_id ON guardians(guardian_id);
CREATE INDEX idx_guardians_student_id ON guardians(student_id);
CREATE INDEX idx_guardians_tenant_guardian ON guardians(tenant_id, guardian_id);
CREATE INDEX idx_guardians_tenant_student ON guardians(tenant_id, student_id);
CREATE INDEX idx_guardians_status ON guardians(status);

-- Add comments
COMMENT ON TABLE guardians IS 'Parent/guardian to student relationships';
COMMENT ON COLUMN guardians.guardian_id IS 'User ID of the parent/guardian';
COMMENT ON COLUMN guardians.student_id IS 'User ID of the student (ward)';
COMMENT ON COLUMN guardians.is_primary IS 'Primary contact for the student';
