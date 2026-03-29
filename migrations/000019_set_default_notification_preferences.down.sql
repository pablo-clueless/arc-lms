-- Revert notification preferences to empty array
-- Note: This will remove any user customizations made after the migration

UPDATE users
SET notification_preferences = '[]'::jsonb,
updated_at = NOW();
