-- Set default notification preferences for all existing users
-- All notification channels (in_app, push, email) are enabled for all event types

UPDATE users
SET notification_preferences = '[
    {"event_type": "QUIZ_PUBLISHED", "in_app_enabled": true, "push_enabled": true, "email_enabled": true},
    {"event_type": "ASSIGNMENT_PUBLISHED", "in_app_enabled": true, "push_enabled": true, "email_enabled": true},
    {"event_type": "ASSIGNMENT_DEADLINE_APPROACHING", "in_app_enabled": true, "push_enabled": true, "email_enabled": true},
    {"event_type": "EXAMINATION_SCHEDULED", "in_app_enabled": true, "push_enabled": true, "email_enabled": true},
    {"event_type": "EXAMINATION_WINDOW_OPEN", "in_app_enabled": true, "push_enabled": true, "email_enabled": true},
    {"event_type": "EXAMINATION_WINDOW_CLOSE", "in_app_enabled": true, "push_enabled": true, "email_enabled": true},
    {"event_type": "GRADE_PUBLISHED", "in_app_enabled": true, "push_enabled": true, "email_enabled": true},
    {"event_type": "TIMETABLE_PUBLISHED", "in_app_enabled": true, "push_enabled": true, "email_enabled": true},
    {"event_type": "TIMETABLE_UPDATED", "in_app_enabled": true, "push_enabled": true, "email_enabled": true},
    {"event_type": "MEETING_SCHEDULED", "in_app_enabled": true, "push_enabled": true, "email_enabled": true},
    {"event_type": "MEETING_CANCELLED", "in_app_enabled": true, "push_enabled": true, "email_enabled": true},
    {"event_type": "MEETING_STARTING", "in_app_enabled": true, "push_enabled": true, "email_enabled": true},
    {"event_type": "INVOICE_GENERATED", "in_app_enabled": true, "push_enabled": true, "email_enabled": true},
    {"event_type": "PAYMENT_OVERDUE", "in_app_enabled": true, "push_enabled": true, "email_enabled": true},
    {"event_type": "CUSTOM", "in_app_enabled": true, "push_enabled": true, "email_enabled": true}
]'::jsonb,
updated_at = NOW()
WHERE notification_preferences = '[]'::jsonb OR notification_preferences IS NULL;
