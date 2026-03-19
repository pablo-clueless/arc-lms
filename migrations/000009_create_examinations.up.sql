-- Create examinations table
CREATE TABLE IF NOT EXISTS examinations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    term_id UUID NOT NULL REFERENCES terms(id) ON DELETE CASCADE,
    created_by_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    title VARCHAR(200) NOT NULL,
    instructions TEXT NOT NULL,
    questions JSONB NOT NULL,
    total_marks INTEGER NOT NULL CHECK (total_marks >= 1),
    duration INTEGER NOT NULL CHECK (duration >= 30 AND duration <= 300),
    window_start TIMESTAMPTZ NOT NULL,
    window_end TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'SCHEDULED', 'IN_PROGRESS', 'COMPLETED', 'ARCHIVED')),
    results_published BOOLEAN NOT NULL DEFAULT FALSE,
    results_published_at TIMESTAMPTZ,
    results_published_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    scheduled_at TIMESTAMPTZ,

    -- Check constraint: window_end must be after window_start
    CONSTRAINT chk_examinations_window_order CHECK (window_end > window_start)
);

-- Create indexes for examinations
CREATE INDEX idx_examinations_tenant_id ON examinations(tenant_id);
CREATE INDEX idx_examinations_tenant_id_id ON examinations(tenant_id, id);
CREATE INDEX idx_examinations_course_id ON examinations(course_id);
CREATE INDEX idx_examinations_term_id ON examinations(term_id);
CREATE INDEX idx_examinations_created_by_id ON examinations(created_by_id);
CREATE INDEX idx_examinations_status ON examinations(status);
CREATE INDEX idx_examinations_window_start ON examinations(window_start);
CREATE INDEX idx_examinations_window_end ON examinations(window_end);

-- Create examination_submissions table
CREATE TABLE IF NOT EXISTS examination_submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    examination_id UUID NOT NULL REFERENCES examinations(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'NOT_STARTED' CHECK (status IN ('NOT_STARTED', 'IN_PROGRESS', 'SUBMITTED', 'GRADED', 'PUBLISHED')),
    started_at TIMESTAMPTZ,
    submitted_at TIMESTAMPTZ,
    auto_submitted BOOLEAN NOT NULL DEFAULT FALSE,
    answers JSONB NOT NULL DEFAULT '[]'::jsonb,
    score INTEGER,
    percentage NUMERIC(5, 2),
    is_auto_graded BOOLEAN NOT NULL DEFAULT FALSE,
    feedback TEXT,
    integrity_events JSONB NOT NULL DEFAULT '[]'::jsonb,
    ip_address VARCHAR(45) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    graded_at TIMESTAMPTZ,
    graded_by UUID REFERENCES users(id) ON DELETE SET NULL,
    results_published_to_student BOOLEAN NOT NULL DEFAULT FALSE,

    -- Unique constraint: one submission per student per examination
    CONSTRAINT uq_examination_submissions_student_exam UNIQUE (tenant_id, examination_id, student_id)
);

-- Create indexes for examination_submissions
CREATE INDEX idx_examination_submissions_tenant_id ON examination_submissions(tenant_id);
CREATE INDEX idx_examination_submissions_tenant_id_id ON examination_submissions(tenant_id, id);
CREATE INDEX idx_examination_submissions_examination_id ON examination_submissions(examination_id);
CREATE INDEX idx_examination_submissions_student_id ON examination_submissions(student_id);
CREATE INDEX idx_examination_submissions_status ON examination_submissions(status);
CREATE INDEX idx_examination_submissions_tenant_id_student_id ON examination_submissions(tenant_id, student_id);

-- Add comments
COMMENT ON TABLE examinations IS 'Formal end-of-term assessments with integrity monitoring';
COMMENT ON COLUMN examinations.questions IS 'JSONB array of examination question objects';
COMMENT ON TABLE examination_submissions IS 'Student examination attempts with integrity tracking';
COMMENT ON COLUMN examination_submissions.integrity_events IS 'JSONB array of integrity events (tab switches, focus loss, etc.)';
