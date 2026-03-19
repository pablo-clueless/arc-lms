-- Create timetables table
CREATE TABLE IF NOT EXISTS timetables (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    class_id UUID NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    term_id UUID NOT NULL REFERENCES terms(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'PUBLISHED', 'ARCHIVED')),
    generated_at TIMESTAMPTZ NOT NULL,
    generated_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    published_at TIMESTAMPTZ,
    published_by UUID REFERENCES users(id) ON DELETE RESTRICT,
    generation_version INTEGER NOT NULL DEFAULT 1 CHECK (generation_version >= 1),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at TIMESTAMPTZ
);

-- Create indexes for timetables
CREATE INDEX idx_timetables_tenant_id ON timetables(tenant_id);
CREATE INDEX idx_timetables_tenant_id_id ON timetables(tenant_id, id);
CREATE INDEX idx_timetables_class_id ON timetables(class_id);
CREATE INDEX idx_timetables_term_id ON timetables(term_id);
CREATE INDEX idx_timetables_status ON timetables(status);
CREATE INDEX idx_timetables_tenant_id_term_id ON timetables(tenant_id, term_id);

-- Create periods table
CREATE TABLE IF NOT EXISTS periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    timetable_id UUID NOT NULL REFERENCES timetables(id) ON DELETE CASCADE,
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    tutor_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    class_id UUID NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    day_of_week VARCHAR(20) NOT NULL CHECK (day_of_week IN ('MONDAY', 'TUESDAY', 'WEDNESDAY', 'THURSDAY', 'FRIDAY', 'SATURDAY')),
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    period_number INTEGER NOT NULL CHECK (period_number >= 1 AND period_number <= 15),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Check constraint: end_time must be after start_time
    CONSTRAINT chk_periods_time_order CHECK (end_time > start_time)
);

-- Create EXCLUDE constraint to prevent tutor double-booking (requires btree_gist extension)
-- This prevents a tutor from having overlapping periods on the same day
ALTER TABLE periods
ADD CONSTRAINT exclude_periods_tutor_double_booking
EXCLUDE USING gist (
    tenant_id WITH =,
    tutor_id WITH =,
    day_of_week WITH =,
    tstzrange(
        (CURRENT_DATE + start_time)::timestamptz,
        (CURRENT_DATE + end_time)::timestamptz
    ) WITH &&
);

-- Create indexes for periods
CREATE INDEX idx_periods_tenant_id ON periods(tenant_id);
CREATE INDEX idx_periods_tenant_id_id ON periods(tenant_id, id);
CREATE INDEX idx_periods_timetable_id ON periods(timetable_id);
CREATE INDEX idx_periods_course_id ON periods(course_id);
CREATE INDEX idx_periods_tutor_id ON periods(tutor_id);
CREATE INDEX idx_periods_class_id ON periods(class_id);
CREATE INDEX idx_periods_day_of_week ON periods(day_of_week);
CREATE INDEX idx_periods_tenant_id_tutor_id ON periods(tenant_id, tutor_id);
CREATE INDEX idx_periods_tenant_id_class_id ON periods(tenant_id, class_id);

-- Add comments
COMMENT ON TABLE timetables IS 'Auto-generated schedules for a class within a term';
COMMENT ON COLUMN timetables.generation_version IS 'Incremented on each regeneration';
COMMENT ON TABLE periods IS 'Individual scheduled blocks in a timetable';
COMMENT ON CONSTRAINT exclude_periods_tutor_double_booking ON periods IS 'Prevents tutor from having overlapping periods on same day';
