-- Create emails table
CREATE TABLE IF NOT EXISTS emails (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    sender_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    subject VARCHAR(200) NOT NULL,
    body TEXT NOT NULL,
    html_body TEXT,
    recipient_scope VARCHAR(20) NOT NULL CHECK (recipient_scope IN ('ALL_USERS', 'ALL_TUTORS', 'ALL_STUDENTS', 'CLASS', 'COURSE', 'SPECIFIC_USERS')),
    target_class_id UUID REFERENCES classes(id) ON DELETE SET NULL,
    target_course_id UUID REFERENCES courses(id) ON DELETE SET NULL,
    specific_user_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    recipients JSONB NOT NULL DEFAULT '[]'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'SCHEDULED', 'SENDING', 'SENT', 'FAILED', 'CANCELLED')),
    scheduled_for TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    attachment_urls JSONB NOT NULL DEFAULT '[]'::jsonb,
    total_recipients INTEGER NOT NULL DEFAULT 0,
    success_count INTEGER NOT NULL DEFAULT 0,
    failure_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cancelled_at TIMESTAMPTZ
);

-- Create indexes for emails
CREATE INDEX idx_emails_tenant_id ON emails(tenant_id);
CREATE INDEX idx_emails_tenant_id_id ON emails(tenant_id, id);
CREATE INDEX idx_emails_sender_id ON emails(sender_id);
CREATE INDEX idx_emails_status ON emails(status);
CREATE INDEX idx_emails_scheduled_for ON emails(scheduled_for) WHERE scheduled_for IS NOT NULL;
CREATE INDEX idx_emails_created_at ON emails(created_at DESC);

-- Add comments
COMMENT ON TABLE emails IS 'Email communications with recipient targeting and delivery tracking';
COMMENT ON COLUMN emails.recipients IS 'JSONB array of delivery recipient objects with user_id, email, status, sent_at, etc.';
COMMENT ON COLUMN emails.specific_user_ids IS 'JSONB array of UUID strings when recipient_scope is SPECIFIC_USERS';
