-- Remove indexes
DROP INDEX IF EXISTS idx_quizzes_class_id;
DROP INDEX IF EXISTS idx_assignments_class_id;
DROP INDEX IF EXISTS idx_examinations_class_id;

-- Remove class_id columns
ALTER TABLE quizzes DROP COLUMN IF EXISTS class_id;
ALTER TABLE assignments DROP COLUMN IF EXISTS class_id;
ALTER TABLE examinations DROP COLUMN IF EXISTS class_id;
