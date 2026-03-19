-- Drop periods table
DROP INDEX IF EXISTS idx_periods_tenant_id_class_id;
DROP INDEX IF EXISTS idx_periods_tenant_id_tutor_id;
DROP INDEX IF EXISTS idx_periods_day_of_week;
DROP INDEX IF EXISTS idx_periods_class_id;
DROP INDEX IF EXISTS idx_periods_tutor_id;
DROP INDEX IF EXISTS idx_periods_course_id;
DROP INDEX IF EXISTS idx_periods_timetable_id;
DROP INDEX IF EXISTS idx_periods_tenant_id_id;
DROP INDEX IF EXISTS idx_periods_tenant_id;
DROP TABLE IF EXISTS periods CASCADE;

-- Drop timetables table
DROP INDEX IF EXISTS idx_timetables_tenant_id_term_id;
DROP INDEX IF EXISTS idx_timetables_status;
DROP INDEX IF EXISTS idx_timetables_term_id;
DROP INDEX IF EXISTS idx_timetables_class_id;
DROP INDEX IF EXISTS idx_timetables_tenant_id_id;
DROP INDEX IF EXISTS idx_timetables_tenant_id;
DROP TABLE IF EXISTS timetables CASCADE;
