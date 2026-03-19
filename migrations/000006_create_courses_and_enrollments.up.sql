-- Create courses table
CREATE TABLE IF NOT EXISTS courses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    class_id UUID NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    term_id UUID NOT NULL REFERENCES terms(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    subject_code VARCHAR(20) NOT NULL,
    description TEXT,
    assigned_tutor_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE', 'DRAFT')),
    max_periods_per_week INTEGER CHECK (max_periods_per_week >= 1 AND max_periods_per_week <= 20),
    custom_grade_weighting JSONB,
    materials JSONB NOT NULL DEFAULT '[]'::jsonb,
    syllabus TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for courses
CREATE INDEX idx_courses_tenant_id ON courses(tenant_id);
CREATE INDEX idx_courses_tenant_id_id ON courses(tenant_id, id);
CREATE INDEX idx_courses_session_id ON courses(session_id);
CREATE INDEX idx_courses_class_id ON courses(class_id);
CREATE INDEX idx_courses_term_id ON courses(term_id);
CREATE INDEX idx_courses_assigned_tutor_id ON courses(assigned_tutor_id);
CREATE INDEX idx_courses_status ON courses(status);
CREATE INDEX idx_courses_tenant_id_term_id ON courses(tenant_id, term_id);

-- Create enrollments table
CREATE TABLE IF NOT EXISTS enrollments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    class_id UUID NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'TRANSFERRED', 'WITHDRAWN', 'SUSPENDED')),
    enrollment_date TIMESTAMPTZ NOT NULL,
    withdrawal_date TIMESTAMPTZ,
    withdrawal_reason TEXT,
    transferred_to_class_id UUID REFERENCES classes(id) ON DELETE SET NULL,
    transfer_date TIMESTAMPTZ,
    transfer_reason TEXT,
    suspension_date TIMESTAMPTZ,
    suspension_reason TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create unique index for one enrollment per student per session (BR-003)
CREATE UNIQUE INDEX idx_enrollments_one_per_student_per_session
ON enrollments(tenant_id, student_id, session_id);

-- Create indexes for enrollments
CREATE INDEX idx_enrollments_tenant_id ON enrollments(tenant_id);
CREATE INDEX idx_enrollments_tenant_id_id ON enrollments(tenant_id, id);
CREATE INDEX idx_enrollments_student_id ON enrollments(student_id);
CREATE INDEX idx_enrollments_class_id ON enrollments(class_id);
CREATE INDEX idx_enrollments_session_id ON enrollments(session_id);
CREATE INDEX idx_enrollments_status ON enrollments(status);
CREATE INDEX idx_enrollments_tenant_id_student_id ON enrollments(tenant_id, student_id);
CREATE INDEX idx_enrollments_tenant_id_class_id ON enrollments(tenant_id, class_id);

-- Add comments
COMMENT ON TABLE courses IS 'Subjects taught within a class for a term';
COMMENT ON COLUMN courses.custom_grade_weighting IS 'Optional JSONB overriding tenant default (continuous_assessment, examination)';
COMMENT ON COLUMN courses.materials IS 'JSONB array of material URLs';
COMMENT ON TABLE enrollments IS 'Student enrollment in a class for a session';
COMMENT ON INDEX idx_enrollments_one_per_student_per_session IS 'BR-003: One enrollment per student per session';
