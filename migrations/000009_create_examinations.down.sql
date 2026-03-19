-- Drop examination_submissions table
DROP INDEX IF EXISTS idx_examination_submissions_tenant_id_student_id;
DROP INDEX IF EXISTS idx_examination_submissions_status;
DROP INDEX IF EXISTS idx_examination_submissions_student_id;
DROP INDEX IF EXISTS idx_examination_submissions_examination_id;
DROP INDEX IF EXISTS idx_examination_submissions_tenant_id_id;
DROP INDEX IF EXISTS idx_examination_submissions_tenant_id;
DROP TABLE IF EXISTS examination_submissions CASCADE;

-- Drop examinations table
DROP INDEX IF EXISTS idx_examinations_window_end;
DROP INDEX IF EXISTS idx_examinations_window_start;
DROP INDEX IF EXISTS idx_examinations_status;
DROP INDEX IF EXISTS idx_examinations_created_by_id;
DROP INDEX IF EXISTS idx_examinations_term_id;
DROP INDEX IF EXISTS idx_examinations_course_id;
DROP INDEX IF EXISTS idx_examinations_tenant_id_id;
DROP INDEX IF EXISTS idx_examinations_tenant_id;
DROP TABLE IF EXISTS examinations CASCADE;
