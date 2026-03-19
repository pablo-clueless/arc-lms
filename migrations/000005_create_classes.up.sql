-- Create classes table
CREATE TABLE IF NOT EXISTS classes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    arm VARCHAR(20) NOT NULL,
    level VARCHAR(20) NOT NULL CHECK (level IN ('PRIMARY', 'SECONDARY')),
    capacity INTEGER CHECK (capacity >= 1 AND capacity <= 200),
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE')),
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Unique constraint: one class name+arm combination per session per tenant
    CONSTRAINT uq_classes_name_arm_session UNIQUE (tenant_id, session_id, name, arm)
);

-- Create indexes for classes
CREATE INDEX idx_classes_tenant_id ON classes(tenant_id);
CREATE INDEX idx_classes_tenant_id_id ON classes(tenant_id, id);
CREATE INDEX idx_classes_session_id ON classes(session_id);
CREATE INDEX idx_classes_status ON classes(status);
CREATE INDEX idx_classes_tenant_id_session_id ON classes(tenant_id, session_id);
CREATE INDEX idx_classes_level ON classes(level);

-- Add comments
COMMENT ON TABLE classes IS 'Student cohort groups (e.g., JSS1A, Primary 4B) for a session';
COMMENT ON COLUMN classes.arm IS 'Class division like A, B, Gold, Diamond';
COMMENT ON COLUMN classes.level IS 'School level: PRIMARY or SECONDARY';
