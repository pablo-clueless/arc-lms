-- Drop billing_adjustments table
DROP INDEX IF EXISTS idx_billing_adjustments_created_at;
DROP INDEX IF EXISTS idx_billing_adjustments_applied_by;
DROP INDEX IF EXISTS idx_billing_adjustments_type;
DROP INDEX IF EXISTS idx_billing_adjustments_invoice_id;
DROP INDEX IF EXISTS idx_billing_adjustments_tenant_id;
DROP TABLE IF EXISTS billing_adjustments CASCADE;

-- Drop invoices table
DROP INDEX IF EXISTS idx_invoices_tenant_id_due_date;
DROP INDEX IF EXISTS idx_invoices_tenant_id_status;
DROP INDEX IF EXISTS idx_invoices_due_date;
DROP INDEX IF EXISTS idx_invoices_status;
DROP INDEX IF EXISTS idx_invoices_invoice_number;
DROP INDEX IF EXISTS idx_invoices_term_id;
DROP INDEX IF EXISTS idx_invoices_subscription_id;
DROP INDEX IF EXISTS idx_invoices_tenant_id_id;
DROP INDEX IF EXISTS idx_invoices_tenant_id;
DROP TABLE IF EXISTS invoices CASCADE;

-- Drop subscriptions table
DROP INDEX IF EXISTS idx_subscriptions_tenant_id_status;
DROP INDEX IF EXISTS idx_subscriptions_status;
DROP INDEX IF EXISTS idx_subscriptions_session_id;
DROP INDEX IF EXISTS idx_subscriptions_tenant_id_id;
DROP INDEX IF EXISTS idx_subscriptions_tenant_id;
DROP TABLE IF EXISTS subscriptions CASCADE;
