-- Drop report_cards table
DROP INDEX IF EXISTS idx_report_cards_generated_at;
DROP INDEX IF EXISTS idx_report_cards_tenant_id_term_id;
DROP INDEX IF EXISTS idx_report_cards_tenant_id_student_id;
DROP INDEX IF EXISTS idx_report_cards_class_id;
DROP INDEX IF EXISTS idx_report_cards_term_id;
DROP INDEX IF EXISTS idx_report_cards_student_id;
DROP INDEX IF EXISTS idx_report_cards_tenant_id_id;
DROP INDEX IF EXISTS idx_report_cards_tenant_id;
DROP TABLE IF EXISTS report_cards CASCADE;

-- Drop progress table
DROP INDEX IF EXISTS idx_progress_tenant_id_term_id;
DROP INDEX IF EXISTS idx_progress_tenant_id_student_id;
DROP INDEX IF EXISTS idx_progress_is_flagged;
DROP INDEX IF EXISTS idx_progress_status;
DROP INDEX IF EXISTS idx_progress_class_id;
DROP INDEX IF EXISTS idx_progress_term_id;
DROP INDEX IF EXISTS idx_progress_course_id;
DROP INDEX IF EXISTS idx_progress_student_id;
DROP INDEX IF EXISTS idx_progress_tenant_id_id;
DROP INDEX IF EXISTS idx_progress_tenant_id;
DROP TABLE IF EXISTS progress CASCADE;
