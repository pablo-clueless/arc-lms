-- Create progress table
CREATE TABLE IF NOT EXISTS progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    term_id UUID NOT NULL REFERENCES terms(id) ON DELETE CASCADE,
    class_id UUID NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'IN_PROGRESS' CHECK (status IN ('IN_PROGRESS', 'COMPLETED', 'FLAGGED')),
    quiz_scores JSONB NOT NULL DEFAULT '[]'::jsonb,
    assignment_scores JSONB NOT NULL DEFAULT '[]'::jsonb,
    examination_score INTEGER,
    grade JSONB,
    attendance JSONB NOT NULL DEFAULT '{}'::jsonb,
    tutor_remarks TEXT,
    principal_remarks TEXT,
    class_position INTEGER,
    is_flagged BOOLEAN NOT NULL DEFAULT FALSE,
    flag_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,

    -- Unique constraint: one progress record per student per course per term
    CONSTRAINT uq_progress_student_course_term UNIQUE (tenant_id, student_id, course_id, term_id)
);

-- Create indexes for progress
CREATE INDEX idx_progress_tenant_id ON progress(tenant_id);
CREATE INDEX idx_progress_tenant_id_id ON progress(tenant_id, id);
CREATE INDEX idx_progress_student_id ON progress(student_id);
CREATE INDEX idx_progress_course_id ON progress(course_id);
CREATE INDEX idx_progress_term_id ON progress(term_id);
CREATE INDEX idx_progress_class_id ON progress(class_id);
CREATE INDEX idx_progress_status ON progress(status);
CREATE INDEX idx_progress_is_flagged ON progress(is_flagged) WHERE is_flagged = TRUE;
CREATE INDEX idx_progress_tenant_id_student_id ON progress(tenant_id, student_id);
CREATE INDEX idx_progress_tenant_id_term_id ON progress(tenant_id, term_id);

-- Create report_cards table
CREATE TABLE IF NOT EXISTS report_cards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    term_id UUID NOT NULL REFERENCES terms(id) ON DELETE CASCADE,
    class_id UUID NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    course_progresses JSONB NOT NULL,
    overall_percentage NUMERIC(5, 2) NOT NULL,
    overall_grade VARCHAR(2) NOT NULL,
    class_position INTEGER NOT NULL CHECK (class_position >= 1),
    total_students INTEGER NOT NULL CHECK (total_students >= 1),
    principal_remarks TEXT,
    next_term_begins TIMESTAMPTZ,
    generated_at TIMESTAMPTZ NOT NULL,
    generated_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    pdf_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Unique constraint: one report card per student per term
    CONSTRAINT uq_report_cards_student_term UNIQUE (tenant_id, student_id, term_id)
);

-- Create indexes for report_cards
CREATE INDEX idx_report_cards_tenant_id ON report_cards(tenant_id);
CREATE INDEX idx_report_cards_tenant_id_id ON report_cards(tenant_id, id);
CREATE INDEX idx_report_cards_student_id ON report_cards(student_id);
CREATE INDEX idx_report_cards_term_id ON report_cards(term_id);
CREATE INDEX idx_report_cards_class_id ON report_cards(class_id);
CREATE INDEX idx_report_cards_tenant_id_student_id ON report_cards(tenant_id, student_id);
CREATE INDEX idx_report_cards_tenant_id_term_id ON report_cards(tenant_id, term_id);
CREATE INDEX idx_report_cards_generated_at ON report_cards(generated_at DESC);

-- Add comments
COMMENT ON TABLE progress IS 'Student academic standing per course per term';
COMMENT ON COLUMN progress.quiz_scores IS 'JSONB array of quiz scores';
COMMENT ON COLUMN progress.assignment_scores IS 'JSONB array of assignment scores';
COMMENT ON COLUMN progress.grade IS 'JSONB object containing computed grade breakdown';
COMMENT ON COLUMN progress.attendance IS 'JSONB object with total_periods, periods_attended, periods_absent, percentage';
COMMENT ON TABLE report_cards IS 'Complete academic report for a student for a term';
COMMENT ON COLUMN report_cards.course_progresses IS 'JSONB array of all progress records for the term';
