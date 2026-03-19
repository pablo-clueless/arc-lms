-- Drop enrollments table
DROP INDEX IF EXISTS idx_enrollments_tenant_id_class_id;
DROP INDEX IF EXISTS idx_enrollments_tenant_id_student_id;
DROP INDEX IF EXISTS idx_enrollments_status;
DROP INDEX IF EXISTS idx_enrollments_session_id;
DROP INDEX IF EXISTS idx_enrollments_class_id;
DROP INDEX IF EXISTS idx_enrollments_student_id;
DROP INDEX IF EXISTS idx_enrollments_tenant_id_id;
DROP INDEX IF EXISTS idx_enrollments_tenant_id;
DROP INDEX IF EXISTS idx_enrollments_one_per_student_per_session;
DROP TABLE IF EXISTS enrollments CASCADE;

-- Drop courses table
DROP INDEX IF EXISTS idx_courses_tenant_id_term_id;
DROP INDEX IF EXISTS idx_courses_status;
DROP INDEX IF EXISTS idx_courses_assigned_tutor_id;
DROP INDEX IF EXISTS idx_courses_term_id;
DROP INDEX IF EXISTS idx_courses_class_id;
DROP INDEX IF EXISTS idx_courses_session_id;
DROP INDEX IF EXISTS idx_courses_tenant_id_id;
DROP INDEX IF EXISTS idx_courses_tenant_id;
DROP TABLE IF EXISTS courses CASCADE;
