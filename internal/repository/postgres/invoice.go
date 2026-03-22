package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"arc-lms/internal/domain"
	"arc-lms/internal/repository"
)

// InvoiceRepository handles database operations for invoices
type InvoiceRepository struct {
	*repository.BaseRepository
}

// NewInvoiceRepository creates a new invoice repository
func NewInvoiceRepository(db *sql.DB) *InvoiceRepository {
	return &InvoiceRepository{
		BaseRepository: repository.NewBaseRepository(db),
	}
}

// BillingMetrics contains aggregated billing statistics
type BillingMetrics struct {
	TotalRevenue     int64             `json:"total_revenue"`      // Total paid amount in Kobo
	MRR              int64             `json:"mrr"`                // Monthly Recurring Revenue in Kobo
	UpcomingPayments int64             `json:"upcoming_payments"`  // Total amount due within 30 days
	LatePayments     int64             `json:"late_payments"`      // Total overdue amount
	UpcomingCount    int               `json:"upcoming_count"`     // Number of upcoming invoices
	LateCount        int               `json:"late_count"`         // Number of late invoices
	RecentInvoices   []*domain.Invoice `json:"recent_invoices"`    // Last 5 invoices
}

// GetBillingMetrics calculates platform-wide billing metrics for SuperAdmin dashboard
func (r *InvoiceRepository) GetBillingMetrics(ctx context.Context) (*BillingMetrics, error) {
	metrics := &BillingMetrics{}

	// Get total revenue (sum of all paid invoices)
	totalRevenueQuery := `
		SELECT COALESCE(SUM(total_amount), 0)
		FROM invoices
		WHERE status = 'PAID'
	`
	err := r.GetDB().QueryRowContext(ctx, totalRevenueQuery).Scan(&metrics.TotalRevenue)
	if err != nil {
		return nil, fmt.Errorf("failed to get total revenue: %w", err)
	}

	// Calculate MRR (Monthly Recurring Revenue)
	// MRR = (Total revenue from last 12 months) / 12
	mrrQuery := `
		SELECT COALESCE(SUM(total_amount), 0) / 12
		FROM invoices
		WHERE status = 'PAID'
			AND paid_at >= NOW() - INTERVAL '12 months'
	`
	err = r.GetDB().QueryRowContext(ctx, mrrQuery).Scan(&metrics.MRR)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate MRR: %w", err)
	}

	// Get upcoming payments (pending invoices due within 30 days)
	upcomingQuery := `
		SELECT COALESCE(SUM(total_amount), 0), COUNT(*)
		FROM invoices
		WHERE status = 'PENDING'
			AND due_date BETWEEN NOW() AND NOW() + INTERVAL '30 days'
	`
	err = r.GetDB().QueryRowContext(ctx, upcomingQuery).Scan(&metrics.UpcomingPayments, &metrics.UpcomingCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get upcoming payments: %w", err)
	}

	// Get late payments (overdue invoices)
	lateQuery := `
		SELECT COALESCE(SUM(total_amount), 0), COUNT(*)
		FROM invoices
		WHERE status = 'OVERDUE'
			OR (status = 'PENDING' AND due_date < NOW())
	`
	err = r.GetDB().QueryRowContext(ctx, lateQuery).Scan(&metrics.LatePayments, &metrics.LateCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get late payments: %w", err)
	}

	// Get recent invoices (last 5)
	recentQuery := `
		SELECT
			id, tenant_id, subscription_id, term_id, invoice_number,
			status, currency, line_items, subtotal_amount, tax_amount,
			discount_amount, total_amount, student_count, due_date,
			issued_date, paid_at, payment_method, payment_reference,
			notes, billing_email, pdf_url, created_at, updated_at,
			disputed_at, dispute_reason, voided_at, void_reason
		FROM invoices
		ORDER BY created_at DESC
		LIMIT 5
	`
	rows, err := r.GetDB().QueryContext(ctx, recentQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent invoices: %w", err)
	}
	defer rows.Close()

	metrics.RecentInvoices = make([]*domain.Invoice, 0)
	for rows.Next() {
		invoice, err := r.scanInvoiceFromRows(rows)
		if err != nil {
			return nil, err
		}
		metrics.RecentInvoices = append(metrics.RecentInvoices, invoice)
	}

	return metrics, nil
}

// GetUpcomingInvoices retrieves invoices due within the specified days
func (r *InvoiceRepository) GetUpcomingInvoices(ctx context.Context, withinDays int, params repository.PaginationParams) ([]*domain.Invoice, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	whereClause := "WHERE status = 'PENDING' AND due_date BETWEEN NOW() AND NOW() + $1 * INTERVAL '1 day'"

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM invoices %s", whereClause)
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, withinDays).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT
			id, tenant_id, subscription_id, term_id, invoice_number,
			status, currency, line_items, subtotal_amount, tax_amount,
			discount_amount, total_amount, student_count, due_date,
			issued_date, paid_at, payment_method, payment_reference,
			notes, billing_email, pdf_url, created_at, updated_at,
			disputed_at, dispute_reason, voided_at, void_reason
		FROM invoices
		%s
		ORDER BY due_date %s
		LIMIT $2 OFFSET $3
	`, whereClause, params.SortOrder)

	invoices, err := r.queryInvoices(ctx, query, withinDays, params.Limit, params.Offset())
	if err != nil {
		return nil, 0, err
	}

	return invoices, total, nil
}

// GetOverdueInvoices retrieves overdue invoices
func (r *InvoiceRepository) GetOverdueInvoices(ctx context.Context, params repository.PaginationParams) ([]*domain.Invoice, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	whereClause := "WHERE status = 'OVERDUE' OR (status = 'PENDING' AND due_date < NOW())"

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM invoices %s", whereClause)
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT
			id, tenant_id, subscription_id, term_id, invoice_number,
			status, currency, line_items, subtotal_amount, tax_amount,
			discount_amount, total_amount, student_count, due_date,
			issued_date, paid_at, payment_method, payment_reference,
			notes, billing_email, pdf_url, created_at, updated_at,
			disputed_at, dispute_reason, voided_at, void_reason
		FROM invoices
		%s
		ORDER BY due_date %s
		LIMIT $1 OFFSET $2
	`, whereClause, params.SortOrder)

	invoices, err := r.queryInvoices(ctx, query, params.Limit, params.Offset())
	if err != nil {
		return nil, 0, err
	}

	return invoices, total, nil
}

// Create creates a new invoice
func (r *InvoiceRepository) Create(ctx context.Context, invoice *domain.Invoice, tx *sql.Tx) error {
	lineItemsJSON, err := json.Marshal(invoice.LineItems)
	if err != nil {
		return fmt.Errorf("failed to marshal line items: %w", err)
	}

	query := `
		INSERT INTO invoices (
			id, tenant_id, subscription_id, term_id, invoice_number,
			status, currency, line_items, subtotal_amount, tax_amount,
			discount_amount, total_amount, student_count, due_date,
			issued_date, billing_email, notes, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	_, err = execer.ExecContext(ctx, query,
		invoice.ID,
		invoice.TenantID,
		invoice.SubscriptionID,
		invoice.TermID,
		invoice.InvoiceNumber,
		invoice.Status,
		invoice.Currency,
		lineItemsJSON,
		invoice.SubtotalAmount,
		invoice.TaxAmount,
		invoice.DiscountAmount,
		invoice.TotalAmount,
		invoice.StudentCount,
		invoice.DueDate,
		invoice.IssuedDate,
		invoice.BillingEmail,
		repository.ToNullString(invoice.Notes),
		invoice.CreatedAt,
		invoice.UpdatedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// Get retrieves an invoice by ID
func (r *InvoiceRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	query := `
		SELECT
			id, tenant_id, subscription_id, term_id, invoice_number,
			status, currency, line_items, subtotal_amount, tax_amount,
			discount_amount, total_amount, student_count, due_date,
			issued_date, paid_at, payment_method, payment_reference,
			notes, billing_email, pdf_url, created_at, updated_at,
			disputed_at, dispute_reason, voided_at, void_reason
		FROM invoices
		WHERE id = $1
	`

	return r.scanInvoice(r.GetDB().QueryRowContext(ctx, query, id))
}

// Update updates an invoice
func (r *InvoiceRepository) Update(ctx context.Context, invoice *domain.Invoice, tx *sql.Tx) error {
	query := `
		UPDATE invoices
		SET
			status = $2,
			paid_at = $3,
			payment_method = $4,
			payment_reference = $5,
			disputed_at = $6,
			dispute_reason = $7,
			voided_at = $8,
			void_reason = $9,
			updated_at = $10
		WHERE id = $1
	`

	var paidAt, disputedAt, voidedAt sql.NullTime
	var paymentMethod, paymentRef, disputeReason, voidReason sql.NullString

	if invoice.PaidAt != nil {
		paidAt = sql.NullTime{Time: *invoice.PaidAt, Valid: true}
	}
	if invoice.PaymentMethod != nil {
		paymentMethod = sql.NullString{String: string(*invoice.PaymentMethod), Valid: true}
	}
	if invoice.PaymentReference != nil {
		paymentRef = sql.NullString{String: *invoice.PaymentReference, Valid: true}
	}
	if invoice.DisputedAt != nil {
		disputedAt = sql.NullTime{Time: *invoice.DisputedAt, Valid: true}
	}
	if invoice.DisputeReason != nil {
		disputeReason = sql.NullString{String: *invoice.DisputeReason, Valid: true}
	}
	if invoice.VoidedAt != nil {
		voidedAt = sql.NullTime{Time: *invoice.VoidedAt, Valid: true}
	}
	if invoice.VoidReason != nil {
		voidReason = sql.NullString{String: *invoice.VoidReason, Valid: true}
	}

	execer := repository.GetExecer(r.GetDB(), tx)
	result, err := execer.ExecContext(ctx, query,
		invoice.ID,
		invoice.Status,
		paidAt,
		paymentMethod,
		paymentRef,
		disputedAt,
		disputeReason,
		voidedAt,
		voidReason,
		invoice.UpdatedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// ListByTenant retrieves invoices for a tenant
func (r *InvoiceRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, status *domain.InvoiceStatus, params repository.PaginationParams) ([]*domain.Invoice, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	// Build WHERE clause
	whereClause := "WHERE tenant_id = $1"
	args := []interface{}{tenantID}
	argIndex := 2

	if status != nil {
		whereClause += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *status)
		argIndex++
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM invoices %s", whereClause)
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT
			id, tenant_id, subscription_id, term_id, invoice_number,
			status, currency, line_items, subtotal_amount, tax_amount,
			discount_amount, total_amount, student_count, due_date,
			issued_date, paid_at, payment_method, payment_reference,
			notes, billing_email, pdf_url, created_at, updated_at,
			disputed_at, dispute_reason, voided_at, void_reason
		FROM invoices
		%s
		ORDER BY created_at %s
		LIMIT $%d OFFSET $%d
	`, whereClause, params.SortOrder, argIndex, argIndex+1)

	args = append(args, params.Limit, params.Offset())

	invoices, err := r.queryInvoices(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}

	return invoices, total, nil
}

// scanInvoice scans an invoice from a database row
func (r *InvoiceRepository) scanInvoice(row *sql.Row) (*domain.Invoice, error) {
	var invoice domain.Invoice
	var paidAt, disputedAt, voidedAt sql.NullTime
	var paymentMethod, paymentRef, notes, pdfURL, disputeReason, voidReason sql.NullString
	var lineItemsJSON []byte

	err := row.Scan(
		&invoice.ID,
		&invoice.TenantID,
		&invoice.SubscriptionID,
		&invoice.TermID,
		&invoice.InvoiceNumber,
		&invoice.Status,
		&invoice.Currency,
		&lineItemsJSON,
		&invoice.SubtotalAmount,
		&invoice.TaxAmount,
		&invoice.DiscountAmount,
		&invoice.TotalAmount,
		&invoice.StudentCount,
		&invoice.DueDate,
		&invoice.IssuedDate,
		&paidAt,
		&paymentMethod,
		&paymentRef,
		&notes,
		&invoice.BillingEmail,
		&pdfURL,
		&invoice.CreatedAt,
		&invoice.UpdatedAt,
		&disputedAt,
		&disputeReason,
		&voidedAt,
		&voidReason,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	if paidAt.Valid {
		invoice.PaidAt = &paidAt.Time
	}
	if paymentMethod.Valid {
		pm := domain.PaymentMethod(paymentMethod.String)
		invoice.PaymentMethod = &pm
	}
	if paymentRef.Valid {
		invoice.PaymentReference = &paymentRef.String
	}
	if notes.Valid {
		invoice.Notes = &notes.String
	}
	if pdfURL.Valid {
		invoice.PDFUrl = &pdfURL.String
	}
	if disputedAt.Valid {
		invoice.DisputedAt = &disputedAt.Time
	}
	if disputeReason.Valid {
		invoice.DisputeReason = &disputeReason.String
	}
	if voidedAt.Valid {
		invoice.VoidedAt = &voidedAt.Time
	}
	if voidReason.Valid {
		invoice.VoidReason = &voidReason.String
	}

	if err := json.Unmarshal(lineItemsJSON, &invoice.LineItems); err != nil {
		return nil, fmt.Errorf("failed to unmarshal line items: %w", err)
	}

	return &invoice, nil
}

// scanInvoiceFromRows scans an invoice from a Rows object
func (r *InvoiceRepository) scanInvoiceFromRows(rows *sql.Rows) (*domain.Invoice, error) {
	var invoice domain.Invoice
	var paidAt, disputedAt, voidedAt sql.NullTime
	var paymentMethod, paymentRef, notes, pdfURL, disputeReason, voidReason sql.NullString
	var lineItemsJSON []byte

	err := rows.Scan(
		&invoice.ID,
		&invoice.TenantID,
		&invoice.SubscriptionID,
		&invoice.TermID,
		&invoice.InvoiceNumber,
		&invoice.Status,
		&invoice.Currency,
		&lineItemsJSON,
		&invoice.SubtotalAmount,
		&invoice.TaxAmount,
		&invoice.DiscountAmount,
		&invoice.TotalAmount,
		&invoice.StudentCount,
		&invoice.DueDate,
		&invoice.IssuedDate,
		&paidAt,
		&paymentMethod,
		&paymentRef,
		&notes,
		&invoice.BillingEmail,
		&pdfURL,
		&invoice.CreatedAt,
		&invoice.UpdatedAt,
		&disputedAt,
		&disputeReason,
		&voidedAt,
		&voidReason,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	if paidAt.Valid {
		invoice.PaidAt = &paidAt.Time
	}
	if paymentMethod.Valid {
		pm := domain.PaymentMethod(paymentMethod.String)
		invoice.PaymentMethod = &pm
	}
	if paymentRef.Valid {
		invoice.PaymentReference = &paymentRef.String
	}
	if notes.Valid {
		invoice.Notes = &notes.String
	}
	if pdfURL.Valid {
		invoice.PDFUrl = &pdfURL.String
	}
	if disputedAt.Valid {
		invoice.DisputedAt = &disputedAt.Time
	}
	if disputeReason.Valid {
		invoice.DisputeReason = &disputeReason.String
	}
	if voidedAt.Valid {
		invoice.VoidedAt = &voidedAt.Time
	}
	if voidReason.Valid {
		invoice.VoidReason = &voidReason.String
	}

	if err := json.Unmarshal(lineItemsJSON, &invoice.LineItems); err != nil {
		return nil, fmt.Errorf("failed to unmarshal line items: %w", err)
	}

	return &invoice, nil
}

// queryInvoices executes a query and returns a list of invoices
func (r *InvoiceRepository) queryInvoices(ctx context.Context, query string, args ...interface{}) ([]*domain.Invoice, error) {
	rows, err := r.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, repository.ParseError(err)
	}
	defer rows.Close()

	invoices := make([]*domain.Invoice, 0)
	for rows.Next() {
		invoice, err := r.scanInvoiceFromRows(rows)
		if err != nil {
			return nil, err
		}
		invoices = append(invoices, invoice)
	}

	if err := rows.Err(); err != nil {
		return nil, repository.ParseError(err)
	}

	return invoices, nil
}

// MarkOverdueInvoices marks pending invoices past their due date as overdue
func (r *InvoiceRepository) MarkOverdueInvoices(ctx context.Context) (int64, error) {
	query := `
		UPDATE invoices
		SET status = 'OVERDUE', updated_at = $1
		WHERE status = 'PENDING' AND due_date < NOW()
	`

	result, err := r.GetDB().ExecContext(ctx, query, time.Now())
	if err != nil {
		return 0, repository.ParseError(err)
	}

	return result.RowsAffected()
}

// GetByTerm retrieves an invoice for a specific term
func (r *InvoiceRepository) GetByTerm(ctx context.Context, termID uuid.UUID) (*domain.Invoice, error) {
	query := `
		SELECT
			id, tenant_id, subscription_id, term_id, invoice_number,
			status, currency, line_items, subtotal_amount, tax_amount,
			discount_amount, total_amount, student_count, due_date,
			issued_date, paid_at, payment_method, payment_reference,
			notes, billing_email, pdf_url, created_at, updated_at,
			disputed_at, dispute_reason, voided_at, void_reason
		FROM invoices
		WHERE term_id = $1
	`

	return r.scanInvoice(r.GetDB().QueryRowContext(ctx, query, termID))
}

// ListAll retrieves all invoices across all tenants (for SuperAdmin)
func (r *InvoiceRepository) ListAll(ctx context.Context, status *domain.InvoiceStatus, params repository.PaginationParams) ([]*domain.Invoice, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if status != nil {
		whereClause += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *status)
		argIndex++
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM invoices %s", whereClause)
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT
			id, tenant_id, subscription_id, term_id, invoice_number,
			status, currency, line_items, subtotal_amount, tax_amount,
			discount_amount, total_amount, student_count, due_date,
			issued_date, paid_at, payment_method, payment_reference,
			notes, billing_email, pdf_url, created_at, updated_at,
			disputed_at, dispute_reason, voided_at, void_reason
		FROM invoices
		%s
		ORDER BY created_at %s
		LIMIT $%d OFFSET $%d
	`, whereClause, params.SortOrder, argIndex, argIndex+1)

	args = append(args, params.Limit, params.Offset())

	invoices, err := r.queryInvoices(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}

	return invoices, total, nil
}

// CreateBillingAdjustment creates a billing adjustment record
func (r *InvoiceRepository) CreateBillingAdjustment(ctx context.Context, adjustment *domain.BillingAdjustment) error {
	query := `
		INSERT INTO billing_adjustments (
			id, tenant_id, invoice_id, type, amount, currency,
			reason, applied_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.GetDB().ExecContext(ctx, query,
		adjustment.ID,
		adjustment.TenantID,
		adjustment.InvoiceID,
		adjustment.Type,
		adjustment.Amount,
		adjustment.Currency,
		adjustment.Reason,
		adjustment.AppliedBy,
		adjustment.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create billing adjustment: %w", err)
	}

	return nil
}

// ListBillingAdjustments retrieves billing adjustments for a tenant
func (r *InvoiceRepository) ListBillingAdjustments(ctx context.Context, tenantID uuid.UUID, params repository.PaginationParams) ([]*domain.BillingAdjustment, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM billing_adjustments WHERE tenant_id = $1"
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count billing adjustments: %w", err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT
			id, tenant_id, invoice_id, type, amount, currency,
			reason, applied_by, created_at
		FROM billing_adjustments
		WHERE tenant_id = $1
		ORDER BY created_at %s
		LIMIT $2 OFFSET $3
	`, params.SortOrder)

	rows, err := r.GetDB().QueryContext(ctx, query, tenantID, params.Limit, params.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list billing adjustments: %w", err)
	}
	defer rows.Close()

	var adjustments []*domain.BillingAdjustment
	for rows.Next() {
		var adj domain.BillingAdjustment
		var invoiceID sql.NullString

		err := rows.Scan(
			&adj.ID,
			&adj.TenantID,
			&invoiceID,
			&adj.Type,
			&adj.Amount,
			&adj.Currency,
			&adj.Reason,
			&adj.AppliedBy,
			&adj.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan billing adjustment: %w", err)
		}

		if invoiceID.Valid {
			id, _ := uuid.Parse(invoiceID.String)
			adj.InvoiceID = &id
		}

		adjustments = append(adjustments, &adj)
	}

	return adjustments, total, nil
}

// ApplyAdjustmentToInvoice applies an adjustment to an invoice's total
func (r *InvoiceRepository) ApplyAdjustmentToInvoice(ctx context.Context, invoiceID uuid.UUID, adjustmentAmount int) error {
	query := `
		UPDATE invoices SET
			discount_amount = discount_amount + $1,
			total_amount = subtotal_amount + tax_amount - (discount_amount + $1),
			updated_at = $2
		WHERE id = $3 AND status IN ('PENDING', 'OVERDUE')
	`

	result, err := r.GetDB().ExecContext(ctx, query, adjustmentAmount, time.Now(), invoiceID)
	if err != nil {
		return fmt.Errorf("failed to apply adjustment: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("invoice not found or cannot be adjusted")
	}

	return nil
}

// GetTenantBillingStats retrieves billing statistics for a tenant
func (r *InvoiceRepository) GetTenantBillingStats(ctx context.Context, tenantID uuid.UUID) (*TenantBillingStats, error) {
	query := `
		SELECT
			COUNT(*) as total_invoices,
			COUNT(*) FILTER (WHERE status = 'PENDING') as pending_count,
			COUNT(*) FILTER (WHERE status = 'PAID') as paid_count,
			COUNT(*) FILTER (WHERE status = 'OVERDUE') as overdue_count,
			COALESCE(SUM(total_amount) FILTER (WHERE status = 'PAID'), 0) as total_paid,
			COALESCE(SUM(total_amount) FILTER (WHERE status IN ('PENDING', 'OVERDUE')), 0) as total_outstanding
		FROM invoices
		WHERE tenant_id = $1
	`

	var stats TenantBillingStats
	stats.TenantID = tenantID

	err := r.GetDB().QueryRowContext(ctx, query, tenantID).Scan(
		&stats.TotalInvoices,
		&stats.PendingCount,
		&stats.PaidCount,
		&stats.OverdueCount,
		&stats.TotalPaid,
		&stats.TotalOutstanding,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get billing stats: %w", err)
	}

	return &stats, nil
}

// TenantBillingStats represents billing statistics for a tenant
type TenantBillingStats struct {
	TenantID         uuid.UUID `json:"tenant_id"`
	TotalInvoices    int       `json:"total_invoices"`
	PendingCount     int       `json:"pending_count"`
	PaidCount        int       `json:"paid_count"`
	OverdueCount     int       `json:"overdue_count"`
	TotalPaid        int64     `json:"total_paid"`        // in Kobo
	TotalOutstanding int64     `json:"total_outstanding"` // in Kobo
}

// GetInvoicesByDateRange retrieves invoices within a date range
func (r *InvoiceRepository) GetInvoicesByDateRange(ctx context.Context, tenantID *uuid.UUID, startDate, endDate time.Time, params repository.PaginationParams) ([]*domain.Invoice, error) {
	query := `
		SELECT
			id, tenant_id, subscription_id, term_id, invoice_number,
			status, currency, line_items, subtotal_amount, tax_amount,
			discount_amount, total_amount, student_count, due_date,
			issued_date, paid_at, payment_method, payment_reference,
			notes, billing_email, pdf_url, created_at, updated_at,
			disputed_at, dispute_reason, voided_at, void_reason
		FROM invoices
		WHERE created_at BETWEEN $1 AND $2
	`

	args := []interface{}{startDate, endDate}
	argIndex := 3

	if tenantID != nil {
		query += fmt.Sprintf(" AND tenant_id = $%d", argIndex)
		args = append(args, *tenantID)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	if params.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", params.Limit)
	}

	return r.queryInvoices(ctx, query, args...)
}

// GetNextInvoiceNumber generates the next invoice number for a tenant
func (r *InvoiceRepository) GetNextInvoiceNumber(ctx context.Context, tenantID uuid.UUID) (string, error) {
	year := time.Now().Year()
	prefix := fmt.Sprintf("INV-%d-", year)

	query := `
		SELECT COALESCE(MAX(CAST(SUBSTRING(invoice_number FROM '[0-9]+$') AS INTEGER)), 0) + 1
		FROM invoices
		WHERE tenant_id = $1 AND invoice_number LIKE $2
	`

	var nextNum int
	err := r.GetDB().QueryRowContext(ctx, query, tenantID, prefix+"%").Scan(&nextNum)
	if err != nil {
		return "", fmt.Errorf("failed to get next invoice number: %w", err)
	}

	return fmt.Sprintf("%s%04d", prefix, nextNum), nil
}
