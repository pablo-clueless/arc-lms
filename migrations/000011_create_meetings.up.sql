-- Create meetings table
CREATE TABLE IF NOT EXISTS meetings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    class_id UUID NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    course_id UUID REFERENCES courses(id) ON DELETE SET NULL,
    host_tutor_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    scheduled_start TIMESTAMPTZ NOT NULL,
    estimated_duration INTEGER NOT NULL CHECK (estimated_duration >= 15 AND estimated_duration <= 240),
    status VARCHAR(20) NOT NULL DEFAULT 'SCHEDULED' CHECK (status IN ('SCHEDULED', 'LIVE', 'ENDED', 'CANCELLED')),
    provider VARCHAR(20) NOT NULL CHECK (provider IN ('DAILY', 'ZOOM', 'JITSI', 'CUSTOM')),
    meeting_url TEXT NOT NULL,
    provider_meeting_id VARCHAR(255) NOT NULL,
    access_code VARCHAR(50),
    actual_start_time TIMESTAMPTZ,
    actual_end_time TIMESTAMPTZ,
    recording_url TEXT,
    recording_expires_at TIMESTAMPTZ,
    participant_events JSONB NOT NULL DEFAULT '[]'::jsonb,
    cancellation_reason TEXT,
    cancelled_by UUID REFERENCES users(id) ON DELETE SET NULL,
    cancelled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for meetings
CREATE INDEX idx_meetings_tenant_id ON meetings(tenant_id);
CREATE INDEX idx_meetings_tenant_id_id ON meetings(tenant_id, id);
CREATE INDEX idx_meetings_class_id ON meetings(class_id);
CREATE INDEX idx_meetings_course_id ON meetings(course_id);
CREATE INDEX idx_meetings_host_tutor_id ON meetings(host_tutor_id);
CREATE INDEX idx_meetings_status ON meetings(status);
CREATE INDEX idx_meetings_scheduled_start ON meetings(scheduled_start);
CREATE INDEX idx_meetings_tenant_id_scheduled_start ON meetings(tenant_id, scheduled_start DESC);

-- Add comments
COMMENT ON TABLE meetings IS 'Virtual class sessions with video conferencing integration';
COMMENT ON COLUMN meetings.participant_events IS 'JSONB array of join/leave events with user_id, event_type, timestamp, duration';
COMMENT ON COLUMN meetings.estimated_duration IS 'Duration in minutes';
