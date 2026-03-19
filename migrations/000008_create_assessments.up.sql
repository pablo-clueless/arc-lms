-- Create quizzes table
CREATE TABLE IF NOT EXISTS quizzes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    created_by_tutor_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    title VARCHAR(200) NOT NULL,
    instructions TEXT NOT NULL,
    questions JSONB NOT NULL,
    total_marks INTEGER NOT NULL CHECK (total_marks >= 1),
    time_limit INTEGER NOT NULL CHECK (time_limit >= 1 AND time_limit <= 300),
    availability_start TIMESTAMPTZ NOT NULL,
    availability_end TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'PUBLISHED', 'ARCHIVED')),
    show_before_window BOOLEAN NOT NULL DEFAULT FALSE,
    allow_retake BOOLEAN NOT NULL DEFAULT FALSE,
    passing_percentage INTEGER CHECK (passing_percentage >= 0 AND passing_percentage <= 100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ,

    -- Check constraint: availability_end must be after availability_start
    CONSTRAINT chk_quizzes_availability_order CHECK (availability_end > availability_start)
);

-- Create indexes for quizzes
CREATE INDEX idx_quizzes_tenant_id ON quizzes(tenant_id);
CREATE INDEX idx_quizzes_tenant_id_id ON quizzes(tenant_id, id);
CREATE INDEX idx_quizzes_course_id ON quizzes(course_id);
CREATE INDEX idx_quizzes_created_by_tutor_id ON quizzes(created_by_tutor_id);
CREATE INDEX idx_quizzes_status ON quizzes(status);
CREATE INDEX idx_quizzes_availability_start ON quizzes(availability_start);
CREATE INDEX idx_quizzes_availability_end ON quizzes(availability_end);

-- Create quiz_submissions table
CREATE TABLE IF NOT EXISTS quiz_submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    quiz_id UUID NOT NULL REFERENCES quizzes(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'NOT_STARTED' CHECK (status IN ('NOT_STARTED', 'IN_PROGRESS', 'SUBMITTED', 'LATE', 'GRADED')),
    started_at TIMESTAMPTZ,
    submitted_at TIMESTAMPTZ,
    answers JSONB NOT NULL DEFAULT '[]'::jsonb,
    score INTEGER,
    percentage NUMERIC(5, 2),
    is_auto_graded BOOLEAN NOT NULL DEFAULT FALSE,
    feedback TEXT,
    ip_address VARCHAR(45),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    graded_at TIMESTAMPTZ,
    graded_by UUID REFERENCES users(id) ON DELETE SET NULL
);

-- Create unique index for one quiz attempt per student (BR-018)
CREATE UNIQUE INDEX idx_quiz_submissions_one_per_student
ON quiz_submissions(tenant_id, quiz_id, student_id);

-- Create indexes for quiz_submissions
CREATE INDEX idx_quiz_submissions_tenant_id ON quiz_submissions(tenant_id);
CREATE INDEX idx_quiz_submissions_tenant_id_id ON quiz_submissions(tenant_id, id);
CREATE INDEX idx_quiz_submissions_quiz_id ON quiz_submissions(quiz_id);
CREATE INDEX idx_quiz_submissions_student_id ON quiz_submissions(student_id);
CREATE INDEX idx_quiz_submissions_status ON quiz_submissions(status);
CREATE INDEX idx_quiz_submissions_tenant_id_student_id ON quiz_submissions(tenant_id, student_id);

-- Create assignments table
CREATE TABLE IF NOT EXISTS assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    created_by_tutor_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    title VARCHAR(200) NOT NULL,
    description TEXT NOT NULL,
    attachment_urls JSONB NOT NULL DEFAULT '[]'::jsonb,
    max_marks INTEGER NOT NULL CHECK (max_marks >= 1 AND max_marks <= 100),
    submission_deadline TIMESTAMPTZ NOT NULL,
    allow_late_submission BOOLEAN NOT NULL DEFAULT FALSE,
    hard_cutoff_date TIMESTAMPTZ,
    allowed_file_formats JSONB NOT NULL DEFAULT '[]'::jsonb,
    max_file_size INTEGER NOT NULL CHECK (max_file_size >= 1 AND max_file_size <= 104857600),
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'PUBLISHED', 'ARCHIVED')),
    questions JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ
);

-- Create indexes for assignments
CREATE INDEX idx_assignments_tenant_id ON assignments(tenant_id);
CREATE INDEX idx_assignments_tenant_id_id ON assignments(tenant_id, id);
CREATE INDEX idx_assignments_course_id ON assignments(course_id);
CREATE INDEX idx_assignments_created_by_tutor_id ON assignments(created_by_tutor_id);
CREATE INDEX idx_assignments_status ON assignments(status);
CREATE INDEX idx_assignments_submission_deadline ON assignments(submission_deadline);

-- Create assignment_submissions table
CREATE TABLE IF NOT EXISTS assignment_submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    assignment_id UUID NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'NOT_STARTED' CHECK (status IN ('NOT_STARTED', 'IN_PROGRESS', 'SUBMITTED', 'LATE', 'GRADED')),
    submitted_at TIMESTAMPTZ,
    is_late BOOLEAN NOT NULL DEFAULT FALSE,
    file_urls JSONB NOT NULL DEFAULT '[]'::jsonb,
    answer_text TEXT,
    score INTEGER,
    feedback TEXT,
    ip_address VARCHAR(45),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    graded_at TIMESTAMPTZ,
    graded_by UUID REFERENCES users(id) ON DELETE SET NULL,

    -- Unique constraint: one submission per student per assignment
    CONSTRAINT uq_assignment_submissions_student_assignment UNIQUE (tenant_id, assignment_id, student_id)
);

-- Create indexes for assignment_submissions
CREATE INDEX idx_assignment_submissions_tenant_id ON assignment_submissions(tenant_id);
CREATE INDEX idx_assignment_submissions_tenant_id_id ON assignment_submissions(tenant_id, id);
CREATE INDEX idx_assignment_submissions_assignment_id ON assignment_submissions(assignment_id);
CREATE INDEX idx_assignment_submissions_student_id ON assignment_submissions(student_id);
CREATE INDEX idx_assignment_submissions_status ON assignment_submissions(status);
CREATE INDEX idx_assignment_submissions_tenant_id_student_id ON assignment_submissions(tenant_id, student_id);

-- Add comments
COMMENT ON TABLE quizzes IS 'Formative assessments with auto-grading capabilities';
COMMENT ON COLUMN quizzes.questions IS 'JSONB array of question objects';
COMMENT ON TABLE quiz_submissions IS 'Student quiz attempts';
COMMENT ON INDEX idx_quiz_submissions_one_per_student IS 'BR-018: One quiz attempt per student';
COMMENT ON TABLE assignments IS 'Task-based assessments with file uploads';
COMMENT ON TABLE assignment_submissions IS 'Student assignment submissions';
