ALTER TABLE assignment_submissions
    ADD COLUMN IF NOT EXISTS answers JSONB NOT NULL DEFAULT '[]';
