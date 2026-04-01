-- Create course_contents table for storing course learning materials
CREATE TABLE IF NOT EXISTS course_contents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    content_type VARCHAR(20) NOT NULL CHECK (content_type IN ('TEXT', 'VIDEO', 'IMAGE', 'AUDIO', 'DOCUMENT', 'LINK')),
    content TEXT NOT NULL,
    description TEXT,
    order_index INT NOT NULL DEFAULT 0,
    duration INT, -- Duration in seconds for video/audio
    file_size BIGINT, -- File size in bytes
    mime_type VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX idx_course_contents_course_id ON course_contents(course_id);
CREATE INDEX idx_course_contents_content_type ON course_contents(content_type);
CREATE INDEX idx_course_contents_order ON course_contents(course_id, order_index);

-- Add comments
COMMENT ON TABLE course_contents IS 'Learning materials and content for courses';
COMMENT ON COLUMN course_contents.content_type IS 'Type: TEXT, VIDEO, IMAGE, AUDIO, DOCUMENT, LINK';
COMMENT ON COLUMN course_contents.content IS 'Text content or URL to resource';
COMMENT ON COLUMN course_contents.order_index IS 'Display order within the course';
COMMENT ON COLUMN course_contents.duration IS 'Duration in seconds for video/audio content';
