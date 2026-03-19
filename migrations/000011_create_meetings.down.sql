-- Drop meetings table
DROP INDEX IF EXISTS idx_meetings_tenant_id_scheduled_start;
DROP INDEX IF EXISTS idx_meetings_scheduled_start;
DROP INDEX IF EXISTS idx_meetings_status;
DROP INDEX IF EXISTS idx_meetings_host_tutor_id;
DROP INDEX IF EXISTS idx_meetings_course_id;
DROP INDEX IF EXISTS idx_meetings_class_id;
DROP INDEX IF EXISTS idx_meetings_tenant_id_id;
DROP INDEX IF EXISTS idx_meetings_tenant_id;
DROP TABLE IF EXISTS meetings CASCADE;
