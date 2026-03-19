-- Create terms table
CREATE TABLE IF NOT EXISTS terms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    ordinal VARCHAR(20) NOT NULL CHECK (ordinal IN ('FIRST', 'SECOND', 'THIRD')),
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'ACTIVE', 'COMPLETED')),
    holidays JSONB NOT NULL DEFAULT '[]'::jsonb,
    non_instructional_days JSONB NOT NULL DEFAULT '[]'::jsonb,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    activated_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    -- Check constraint: end_date must be after start_date
    CONSTRAINT chk_terms_date_order CHECK (end_date > start_date),

    -- Unique constraint: one term per ordinal per session
    CONSTRAINT uq_terms_session_ordinal UNIQUE (session_id, ordinal)
);

-- Create exclusion constraint to prevent overlapping terms within same tenant (requires btree_gist)
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE IF NOT EXISTS term_date_ranges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    term_id UUID NOT NULL REFERENCES terms(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    date_range TSTZRANGE NOT NULL,
    EXCLUDE USING gist (tenant_id WITH =, date_range WITH &&)
);

-- Create indexes for terms
CREATE INDEX idx_terms_tenant_id ON terms(tenant_id);
CREATE INDEX idx_terms_tenant_id_id ON terms(tenant_id, id);
CREATE INDEX idx_terms_session_id ON terms(session_id);
CREATE INDEX idx_terms_status ON terms(status);
CREATE INDEX idx_terms_tenant_id_status ON terms(tenant_id, status);
CREATE INDEX idx_terms_start_date ON terms(start_date);
CREATE INDEX idx_terms_end_date ON terms(end_date);

-- Create indexes for term_date_ranges
CREATE INDEX idx_term_date_ranges_term_id ON term_date_ranges(term_id);
CREATE INDEX idx_term_date_ranges_tenant_id ON term_date_ranges(tenant_id);

-- Add comments
COMMENT ON TABLE terms IS 'Academic terms (FIRST, SECOND, THIRD) within a session with non-overlapping dates';
COMMENT ON COLUMN terms.holidays IS 'JSONB array of holiday objects with date, name, description, is_public';
COMMENT ON COLUMN terms.non_instructional_days IS 'JSONB array of date strings for school-specific non-instructional days';
COMMENT ON TABLE term_date_ranges IS 'Helper table to enforce non-overlapping term dates within tenant';
