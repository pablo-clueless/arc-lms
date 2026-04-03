-- Add class_id column to quizzes table
ALTER TABLE quizzes
ADD COLUMN class_id UUID NOT NULL REFERENCES classes(id) ON DELETE CASCADE;

-- Add class_id column to assignments table
ALTER TABLE assignments
ADD COLUMN class_id UUID NOT NULL REFERENCES classes(id) ON DELETE CASCADE;

-- Add class_id column to examinations table
ALTER TABLE examinations
ADD COLUMN class_id UUID NOT NULL REFERENCES classes(id) ON DELETE CASCADE;

-- Create indexes for class_id on all tables
CREATE INDEX idx_quizzes_class_id ON quizzes(class_id);
CREATE INDEX idx_assignments_class_id ON assignments(class_id);
CREATE INDEX idx_examinations_class_id ON examinations(class_id);
