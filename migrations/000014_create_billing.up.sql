-- Create subscriptions table
CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'PAYMENT_OVERDUE', 'SUSPENDED', 'CANCELLED')),
    price_per_student_per_term BIGINT NOT NULL CHECK (price_per_student_per_term >= 1),
    currency VARCHAR(3) NOT NULL DEFAULT 'NGN' CHECK (currency = 'NGN'),
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    cancellation_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Unique constraint: one subscription per tenant per session
    CONSTRAINT uq_subscriptions_tenant_session UNIQUE (tenant_id, session_id)
);

-- Create indexes for subscriptions
CREATE INDEX idx_subscriptions_tenant_id ON subscriptions(tenant_id);
CREATE INDEX idx_subscriptions_tenant_id_id ON subscriptions(tenant_id, id);
CREATE INDEX idx_subscriptions_session_id ON subscriptions(session_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_subscriptions_tenant_id_status ON subscriptions(tenant_id, status);

-- Create invoices table
CREATE TABLE IF NOT EXISTS invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    term_id UUID NOT NULL REFERENCES terms(id) ON DELETE CASCADE,
    invoice_number VARCHAR(50) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'PAID', 'OVERDUE', 'DISPUTED', 'VOIDED')),
    currency VARCHAR(3) NOT NULL DEFAULT 'NGN' CHECK (currency = 'NGN'),
    line_items JSONB NOT NULL,
    subtotal_amount BIGINT NOT NULL CHECK (subtotal_amount >= 0),
    tax_amount BIGINT NOT NULL DEFAULT 0 CHECK (tax_amount >= 0),
    discount_amount BIGINT NOT NULL DEFAULT 0 CHECK (discount_amount >= 0),
    total_amount BIGINT NOT NULL CHECK (total_amount >= 0),
    student_count INTEGER NOT NULL CHECK (student_count >= 0),
    due_date TIMESTAMPTZ NOT NULL,
    issued_date TIMESTAMPTZ NOT NULL,
    paid_at TIMESTAMPTZ,
    payment_method VARCHAR(20) CHECK (payment_method IN ('BANK_TRANSFER', 'CARD', 'USSD', 'MANUAL')),
    payment_reference VARCHAR(100),
    notes TEXT,
    billing_email VARCHAR(255) NOT NULL,
    pdf_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    disputed_at TIMESTAMPTZ,
    dispute_reason TEXT,
    voided_at TIMESTAMPTZ,
    void_reason TEXT
);

-- Create indexes for invoices
CREATE INDEX idx_invoices_tenant_id ON invoices(tenant_id);
CREATE INDEX idx_invoices_tenant_id_id ON invoices(tenant_id, id);
CREATE INDEX idx_invoices_subscription_id ON invoices(subscription_id);
CREATE INDEX idx_invoices_term_id ON invoices(term_id);
CREATE INDEX idx_invoices_invoice_number ON invoices(invoice_number);
CREATE INDEX idx_invoices_status ON invoices(status);
CREATE INDEX idx_invoices_due_date ON invoices(due_date);
CREATE INDEX idx_invoices_tenant_id_status ON invoices(tenant_id, status);
CREATE INDEX idx_invoices_tenant_id_due_date ON invoices(tenant_id, due_date);

-- Create billing_adjustments table
CREATE TABLE IF NOT EXISTS billing_adjustments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_id UUID REFERENCES invoices(id) ON DELETE SET NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('CREDIT', 'DISCOUNT', 'CHARGE', 'REFUND')),
    amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'NGN' CHECK (currency = 'NGN'),
    reason TEXT NOT NULL,
    applied_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for billing_adjustments
CREATE INDEX idx_billing_adjustments_tenant_id ON billing_adjustments(tenant_id);
CREATE INDEX idx_billing_adjustments_invoice_id ON billing_adjustments(invoice_id);
CREATE INDEX idx_billing_adjustments_type ON billing_adjustments(type);
CREATE INDEX idx_billing_adjustments_applied_by ON billing_adjustments(applied_by);
CREATE INDEX idx_billing_adjustments_created_at ON billing_adjustments(created_at DESC);

-- Add comments
COMMENT ON TABLE subscriptions IS 'Tenant billing subscriptions per session';
COMMENT ON COLUMN subscriptions.price_per_student_per_term IS 'Price in Kobo (NGN * 100)';
COMMENT ON TABLE invoices IS 'Billing invoices generated per term';
COMMENT ON COLUMN invoices.line_items IS 'JSONB array of line item objects with description, quantity, unit_price, amount';
COMMENT ON COLUMN invoices.subtotal_amount IS 'Amount in Kobo (NGN * 100)';
COMMENT ON COLUMN invoices.tax_amount IS 'Amount in Kobo (NGN * 100)';
COMMENT ON COLUMN invoices.discount_amount IS 'Amount in Kobo (NGN * 100)';
COMMENT ON COLUMN invoices.total_amount IS 'Amount in Kobo (NGN * 100)';
COMMENT ON TABLE billing_adjustments IS 'Manual billing adjustments (credits, discounts, charges, refunds)';
COMMENT ON COLUMN billing_adjustments.amount IS 'Amount in Kobo (NGN * 100), can be negative for credits/refunds';
