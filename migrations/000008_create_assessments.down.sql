-- Drop assignment_submissions table
DROP INDEX IF EXISTS idx_assignment_submissions_tenant_id_student_id;
DROP INDEX IF EXISTS idx_assignment_submissions_status;
DROP INDEX IF EXISTS idx_assignment_submissions_student_id;
DROP INDEX IF EXISTS idx_assignment_submissions_assignment_id;
DROP INDEX IF EXISTS idx_assignment_submissions_tenant_id_id;
DROP INDEX IF EXISTS idx_assignment_submissions_tenant_id;
DROP TABLE IF EXISTS assignment_submissions CASCADE;

-- Drop assignments table
DROP INDEX IF EXISTS idx_assignments_submission_deadline;
DROP INDEX IF EXISTS idx_assignments_status;
DROP INDEX IF EXISTS idx_assignments_created_by_tutor_id;
DROP INDEX IF EXISTS idx_assignments_course_id;
DROP INDEX IF EXISTS idx_assignments_tenant_id_id;
DROP INDEX IF EXISTS idx_assignments_tenant_id;
DROP TABLE IF EXISTS assignments CASCADE;

-- Drop quiz_submissions table
DROP INDEX IF EXISTS idx_quiz_submissions_tenant_id_student_id;
DROP INDEX IF EXISTS idx_quiz_submissions_status;
DROP INDEX IF EXISTS idx_quiz_submissions_student_id;
DROP INDEX IF EXISTS idx_quiz_submissions_quiz_id;
DROP INDEX IF EXISTS idx_quiz_submissions_tenant_id_id;
DROP INDEX IF EXISTS idx_quiz_submissions_tenant_id;
DROP INDEX IF EXISTS idx_quiz_submissions_one_per_student;
DROP TABLE IF EXISTS quiz_submissions CASCADE;

-- Drop quizzes table
DROP INDEX IF EXISTS idx_quizzes_availability_end;
DROP INDEX IF EXISTS idx_quizzes_availability_start;
DROP INDEX IF EXISTS idx_quizzes_status;
DROP INDEX IF EXISTS idx_quizzes_created_by_tutor_id;
DROP INDEX IF EXISTS idx_quizzes_course_id;
DROP INDEX IF EXISTS idx_quizzes_tenant_id_id;
DROP INDEX IF EXISTS idx_quizzes_tenant_id;
DROP TABLE IF EXISTS quizzes CASCADE;
