package service

import (
	"context"
	"fmt"
	"time"

	"arc-lms/internal/domain"
	"arc-lms/internal/repository"
	"arc-lms/internal/repository/postgres"

	"github.com/google/uuid"
)

// BillingService handles billing and subscription operations
type BillingService struct {
	invoiceRepo      *postgres.InvoiceRepository
	subscriptionRepo *postgres.SubscriptionRepository
	tenantRepo       *postgres.TenantRepository
	enrollmentRepo   *postgres.EnrollmentRepository
	auditService     *AuditService
}

// NewBillingService creates a new billing service
func NewBillingService(
	invoiceRepo *postgres.InvoiceRepository,
	subscriptionRepo *postgres.SubscriptionRepository,
	tenantRepo *postgres.TenantRepository,
	enrollmentRepo *postgres.EnrollmentRepository,
	auditService *AuditService,
) *BillingService {
	return &BillingService{
		invoiceRepo:      invoiceRepo,
		subscriptionRepo: subscriptionRepo,
		tenantRepo:       tenantRepo,
		enrollmentRepo:   enrollmentRepo,
		auditService:     auditService,
	}
}

// Constants for billing (in Kobo: 1 NGN = 100 Kobo)
const (
	PricePerStudentPerTermKobo = 50000 // NGN 500 * 100
	DefaultGracePeriodDays     = 14
	SuspensionThresholdDays    = 30
)

// GenerateTermInvoice generates an invoice for a term activation (FR-BIL-001, FR-BIL-002, FR-BIL-003)
func (s *BillingService) GenerateTermInvoice(
	ctx context.Context,
	tenantID uuid.UUID,
	sessionID uuid.UUID,
	termID uuid.UUID,
	studentCount int,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Invoice, error) {
	if s.invoiceRepo == nil || s.tenantRepo == nil {
		return nil, fmt.Errorf("billing service not fully configured")
	}

	// Get tenant for billing information
	tenant, err := s.tenantRepo.Get(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Get or create subscription for this tenant/session
	subscription, err := s.getOrCreateSubscription(ctx, tenantID, sessionID, actorID, actorRole, ipAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Calculate invoice amount (FR-BIL-001: NGN 500 per student per term)
	totalAmountKobo := studentCount * PricePerStudentPerTermKobo

	// Generate invoice number
	invoiceNumber, err := s.invoiceRepo.GetNextInvoiceNumber(ctx, tenantID)
	if err != nil {
		// Fallback to timestamp-based number
		invoiceNumber = fmt.Sprintf("INV-%d-%04d", time.Now().Year(), time.Now().Unix()%10000)
	}

	// Create line items
	lineItems := []domain.InvoiceLineItem{
		{
			Description: fmt.Sprintf("Term subscription for %d students @ NGN 500/student", studentCount),
			Quantity:    studentCount,
			UnitPrice:   PricePerStudentPerTermKobo,
			Amount:      totalAmountKobo,
		},
	}

	// Calculate due date based on tenant configuration (FR-BIL-006)
	gracePeriod := tenant.Configuration.InvoiceGracePeriod
	if gracePeriod == 0 {
		gracePeriod = DefaultGracePeriodDays
	}
	dueDate := time.Now().Add(time.Duration(gracePeriod) * 24 * time.Hour)

	now := time.Now()
	invoice := &domain.Invoice{
		ID:             uuid.New(),
		TenantID:       tenantID,
		SubscriptionID: subscription.ID,
		TermID:         termID,
		InvoiceNumber:  invoiceNumber,
		Status:         domain.InvoiceStatusPending,
		Currency:       domain.CurrencyNGN,
		LineItems:      lineItems,
		SubtotalAmount: totalAmountKobo,
		TaxAmount:      0,
		DiscountAmount: 0,
		TotalAmount:    totalAmountKobo,
		StudentCount:   studentCount, // FR-BIL-002: Snapshot student count
		DueDate:        dueDate,
		IssuedDate:     now,
		BillingEmail:   tenant.BillingContact.Email,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.invoiceRepo.Create(ctx, invoice, nil); err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionInvoiceGenerated,
		actorID,
		actorRole,
		&tenantID,
		domain.AuditResourceInvoice,
		invoice.ID,
		nil,
		invoice,
		ipAddress,
	)

	return invoice, nil
}

// getOrCreateSubscription gets or creates a subscription for a tenant
func (s *BillingService) getOrCreateSubscription(
	ctx context.Context,
	tenantID uuid.UUID,
	sessionID uuid.UUID,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Subscription, error) {
	if s.subscriptionRepo == nil {
		// Return a minimal subscription for backwards compatibility
		return &domain.Subscription{
			ID:                     uuid.New(),
			TenantID:               tenantID,
			SessionID:              sessionID,
			Status:                 domain.SubscriptionStatusActive,
			PricePerStudentPerTerm: PricePerStudentPerTermKobo,
			Currency:               domain.CurrencyNGN,
			StartDate:              time.Now(),
			CreatedAt:              time.Now(),
			UpdatedAt:              time.Now(),
		}, nil
	}

	// Try to get existing subscription for this session
	subscription, err := s.subscriptionRepo.GetByTenantAndSession(ctx, tenantID, sessionID)
	if err == nil {
		return subscription, nil
	}

	// Create new subscription
	now := time.Now()
	subscription = &domain.Subscription{
		ID:                     uuid.New(),
		TenantID:               tenantID,
		SessionID:              sessionID,
		Status:                 domain.SubscriptionStatusActive,
		PricePerStudentPerTerm: PricePerStudentPerTermKobo,
		Currency:               domain.CurrencyNGN,
		StartDate:              now,
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	if err := s.subscriptionRepo.Create(ctx, subscription, nil); err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	return subscription, nil
}

// GetInvoice retrieves an invoice by ID
func (s *BillingService) GetInvoice(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	return s.invoiceRepo.Get(ctx, id)
}

// ListInvoices lists invoices with optional filtering
func (s *BillingService) ListInvoices(
	ctx context.Context,
	tenantID *uuid.UUID,
	status *domain.InvoiceStatus,
	params repository.PaginationParams,
) ([]*domain.Invoice, *repository.PaginatedResult, error) {
	var invoices []*domain.Invoice
	var err error

	if tenantID != nil {
		invoices, err = s.invoiceRepo.ListByTenant(ctx, *tenantID, status, params)
	} else {
		// SuperAdmin can see all invoices
		invoices, err = s.invoiceRepo.ListAll(ctx, status, params)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to list invoices: %w", err)
	}

	ids := make([]uuid.UUID, len(invoices))
	for i, inv := range invoices {
		ids[i] = inv.ID
	}
	pagination := repository.BuildPaginatedResult(ids, params.Limit)

	return invoices, &pagination, nil
}

// MarkInvoicePaidRequest represents a request to mark an invoice as paid
type MarkInvoicePaidRequest struct {
	PaymentMethod    domain.PaymentMethod `json:"payment_method" validate:"required,oneof=BANK_TRANSFER CARD USSD MANUAL"`
	PaymentReference string               `json:"payment_reference" validate:"required,min=3,max=100"`
}

// MarkInvoiceAsPaid marks an invoice as paid (FR-BIL-005)
func (s *BillingService) MarkInvoiceAsPaid(
	ctx context.Context,
	invoiceID uuid.UUID,
	req *MarkInvoicePaidRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Invoice, error) {
	invoice, err := s.invoiceRepo.Get(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	if invoice.IsPaid() {
		return nil, fmt.Errorf("invoice is already paid")
	}

	if invoice.IsVoided() {
		return nil, fmt.Errorf("cannot mark a voided invoice as paid")
	}

	beforeState := *invoice
	invoice.MarkAsPaid(req.PaymentMethod, req.PaymentReference)

	if err := s.invoiceRepo.Update(ctx, invoice, nil); err != nil {
		return nil, fmt.Errorf("failed to update invoice: %w", err)
	}

	// If subscription was overdue, reactivate it
	if s.subscriptionRepo != nil {
		subscription, err := s.subscriptionRepo.Get(ctx, invoice.SubscriptionID)
		if err == nil && subscription.IsOverdue() {
			subscription.Reactivate()
			_ = s.subscriptionRepo.Update(ctx, subscription, nil)
		}
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionInvoicePaid,
		actorID,
		actorRole,
		&invoice.TenantID,
		domain.AuditResourceInvoice,
		invoice.ID,
		&beforeState,
		invoice,
		ipAddress,
	)

	return invoice, nil
}

// DisputeInvoiceRequest represents a request to dispute an invoice
type DisputeInvoiceRequest struct {
	Reason string `json:"reason" validate:"required,min=10,max=1000"`
}

// DisputeInvoice marks an invoice as disputed
func (s *BillingService) DisputeInvoice(
	ctx context.Context,
	invoiceID uuid.UUID,
	req *DisputeInvoiceRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Invoice, error) {
	invoice, err := s.invoiceRepo.Get(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	if invoice.IsPaid() || invoice.IsVoided() {
		return nil, fmt.Errorf("cannot dispute a paid or voided invoice")
	}

	beforeState := *invoice
	invoice.Dispute(req.Reason)

	if err := s.invoiceRepo.Update(ctx, invoice, nil); err != nil {
		return nil, fmt.Errorf("failed to update invoice: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionInvoiceGenerated, // TODO: Add specific dispute action
		actorID,
		actorRole,
		&invoice.TenantID,
		domain.AuditResourceInvoice,
		invoice.ID,
		&beforeState,
		invoice,
		ipAddress,
	)

	return invoice, nil
}

// VoidInvoiceRequest represents a request to void an invoice
type VoidInvoiceRequest struct {
	Reason string `json:"reason" validate:"required,min=10,max=500"`
}

// VoidInvoice voids an invoice (SuperAdmin only)
func (s *BillingService) VoidInvoice(
	ctx context.Context,
	invoiceID uuid.UUID,
	req *VoidInvoiceRequest,
	actorID uuid.UUID,
	ipAddress string,
) (*domain.Invoice, error) {
	invoice, err := s.invoiceRepo.Get(ctx, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	if invoice.IsPaid() {
		return nil, fmt.Errorf("cannot void a paid invoice")
	}

	beforeState := *invoice
	invoice.Void(req.Reason)

	if err := s.invoiceRepo.Update(ctx, invoice, nil); err != nil {
		return nil, fmt.Errorf("failed to update invoice: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionInvoiceGenerated, // TODO: Add specific void action
		actorID,
		domain.RoleSuperAdmin,
		&invoice.TenantID,
		domain.AuditResourceInvoice,
		invoice.ID,
		&beforeState,
		invoice,
		ipAddress,
	)

	return invoice, nil
}

// ApplyBillingAdjustmentRequest represents a request to apply a billing adjustment (FR-BIL-008)
type ApplyBillingAdjustmentRequest struct {
	TenantID  uuid.UUID  `json:"tenant_id" validate:"required,uuid"`
	InvoiceID *uuid.UUID `json:"invoice_id,omitempty" validate:"omitempty,uuid"`
	Type      string     `json:"type" validate:"required,oneof=CREDIT DISCOUNT CHARGE REFUND"`
	Amount    int        `json:"amount" validate:"required"` // in Kobo
	Reason    string     `json:"reason" validate:"required,min=10,max=1000"`
}

// ApplyBillingAdjustment applies a billing adjustment (SuperAdmin only - FR-BIL-008)
func (s *BillingService) ApplyBillingAdjustment(
	ctx context.Context,
	req *ApplyBillingAdjustmentRequest,
	actorID uuid.UUID,
	ipAddress string,
) (*domain.BillingAdjustment, error) {
	adjustment := &domain.BillingAdjustment{
		ID:        uuid.New(),
		TenantID:  req.TenantID,
		InvoiceID: req.InvoiceID,
		Type:      req.Type,
		Amount:    req.Amount,
		Currency:  domain.CurrencyNGN,
		Reason:    req.Reason,
		AppliedBy: actorID,
		CreatedAt: time.Now(),
	}

	if err := s.invoiceRepo.CreateBillingAdjustment(ctx, adjustment); err != nil {
		return nil, fmt.Errorf("failed to create billing adjustment: %w", err)
	}

	// If adjustment is for a specific invoice, apply it
	if req.InvoiceID != nil && (req.Type == "CREDIT" || req.Type == "DISCOUNT") {
		if err := s.invoiceRepo.ApplyAdjustmentToInvoice(ctx, *req.InvoiceID, req.Amount); err != nil {
			// Log error but don't fail - adjustment is recorded
			fmt.Printf("failed to apply adjustment to invoice: %v\n", err)
		}
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionBillingAdjustmentApplied,
		actorID,
		domain.RoleSuperAdmin,
		&req.TenantID,
		domain.AuditResourceInvoice,
		adjustment.ID,
		nil,
		adjustment,
		ipAddress,
	)

	return adjustment, nil
}

// ListBillingAdjustments lists billing adjustments for a tenant
func (s *BillingService) ListBillingAdjustments(
	ctx context.Context,
	tenantID uuid.UUID,
	params repository.PaginationParams,
) ([]*domain.BillingAdjustment, error) {
	return s.invoiceRepo.ListBillingAdjustments(ctx, tenantID, params)
}

// GetBillingMetrics retrieves platform-wide billing metrics (SuperAdmin)
func (s *BillingService) GetBillingMetrics(ctx context.Context) (*postgres.BillingMetrics, error) {
	return s.invoiceRepo.GetBillingMetrics(ctx)
}

// GetTenantBillingStats retrieves billing statistics for a specific tenant
func (s *BillingService) GetTenantBillingStats(ctx context.Context, tenantID uuid.UUID) (*postgres.TenantBillingStats, error) {
	return s.invoiceRepo.GetTenantBillingStats(ctx, tenantID)
}

// GetUpcomingInvoices retrieves invoices due within the specified days
func (s *BillingService) GetUpcomingInvoices(ctx context.Context, withinDays int, params repository.PaginationParams) ([]*domain.Invoice, error) {
	return s.invoiceRepo.GetUpcomingInvoices(ctx, withinDays, params)
}

// GetOverdueInvoices retrieves overdue invoices
func (s *BillingService) GetOverdueInvoices(ctx context.Context, params repository.PaginationParams) ([]*domain.Invoice, error) {
	return s.invoiceRepo.GetOverdueInvoices(ctx, params)
}

// ProcessOverdueInvoices processes overdue invoices (FR-BIL-006)
// This should be called by a background job
func (s *BillingService) ProcessOverdueInvoices(ctx context.Context) error {
	// Mark pending invoices past due date as overdue
	_, err := s.invoiceRepo.MarkOverdueInvoices(ctx)
	if err != nil {
		return fmt.Errorf("failed to mark overdue invoices: %w", err)
	}

	// Get all overdue invoices
	params := repository.PaginationParams{Limit: 100}
	overdueInvoices, err := s.invoiceRepo.GetOverdueInvoices(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to get overdue invoices: %w", err)
	}

	now := time.Now()
	for _, invoice := range overdueInvoices {
		daysOverdue := invoice.DaysOverdue(now)

		// Mark subscription as overdue after grace period
		if daysOverdue >= DefaultGracePeriodDays && s.subscriptionRepo != nil {
			_ = s.subscriptionRepo.MarkAsOverdue(ctx, invoice.TenantID)
		}

		// Suspend subscription after secondary threshold
		if daysOverdue >= SuspensionThresholdDays && s.subscriptionRepo != nil {
			subscription, err := s.subscriptionRepo.GetByTenant(ctx, invoice.TenantID)
			if err == nil && !subscription.IsSuspended() {
				_ = s.subscriptionRepo.Suspend(ctx, subscription.ID)

				// TODO: Also suspend the tenant
			}
		}
	}

	return nil
}

// GetSubscription retrieves a subscription by ID
func (s *BillingService) GetSubscription(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	if s.subscriptionRepo == nil {
		return nil, fmt.Errorf("subscription repository not configured")
	}
	return s.subscriptionRepo.Get(ctx, id)
}

// GetTenantSubscription retrieves the active subscription for a tenant
func (s *BillingService) GetTenantSubscription(ctx context.Context, tenantID uuid.UUID) (*domain.Subscription, error) {
	if s.subscriptionRepo == nil {
		return nil, fmt.Errorf("subscription repository not configured")
	}
	return s.subscriptionRepo.GetByTenant(ctx, tenantID)
}

// ListSubscriptions lists subscriptions with optional status filter
func (s *BillingService) ListSubscriptions(
	ctx context.Context,
	status *domain.SubscriptionStatus,
	params repository.PaginationParams,
) ([]*domain.Subscription, *repository.PaginatedResult, error) {
	if s.subscriptionRepo == nil {
		return nil, nil, fmt.Errorf("subscription repository not configured")
	}

	subscriptions, err := s.subscriptionRepo.ListByStatus(ctx, status, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	ids := make([]uuid.UUID, len(subscriptions))
	for i, sub := range subscriptions {
		ids[i] = sub.ID
	}
	pagination := repository.BuildPaginatedResult(ids, params.Limit)

	return subscriptions, &pagination, nil
}

// CancelSubscription cancels a subscription
func (s *BillingService) CancelSubscription(
	ctx context.Context,
	subscriptionID uuid.UUID,
	reason string,
	actorID uuid.UUID,
	ipAddress string,
) (*domain.Subscription, error) {
	if s.subscriptionRepo == nil {
		return nil, fmt.Errorf("subscription repository not configured")
	}

	subscription, err := s.subscriptionRepo.Get(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	beforeState := *subscription
	subscription.Cancel(reason)

	if err := s.subscriptionRepo.Update(ctx, subscription, nil); err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionSubscriptionCancelled,
		actorID,
		domain.RoleSuperAdmin,
		&subscription.TenantID,
		domain.AuditResourceTenant,
		subscription.ID,
		&beforeState,
		subscription,
		ipAddress,
	)

	return subscription, nil
}

// ReactivateSubscription reactivates a suspended subscription
func (s *BillingService) ReactivateSubscription(
	ctx context.Context,
	subscriptionID uuid.UUID,
	actorID uuid.UUID,
	ipAddress string,
) (*domain.Subscription, error) {
	if s.subscriptionRepo == nil {
		return nil, fmt.Errorf("subscription repository not configured")
	}

	subscription, err := s.subscriptionRepo.Get(ctx, subscriptionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	if subscription.Status == domain.SubscriptionStatusActive {
		return nil, fmt.Errorf("subscription is already active")
	}

	if subscription.Status == domain.SubscriptionStatusCancelled {
		return nil, fmt.Errorf("cannot reactivate a cancelled subscription")
	}

	beforeState := *subscription
	subscription.Reactivate()

	if err := s.subscriptionRepo.Update(ctx, subscription, nil); err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionSubscriptionCancelled, // TODO: Add reactivation action
		actorID,
		domain.RoleSuperAdmin,
		&subscription.TenantID,
		domain.AuditResourceTenant,
		subscription.ID,
		&beforeState,
		subscription,
		ipAddress,
	)

	return subscription, nil
}

// GetSubscriptionStatistics retrieves subscription statistics
func (s *BillingService) GetSubscriptionStatistics(ctx context.Context) (*postgres.SubscriptionStatistics, error) {
	if s.subscriptionRepo == nil {
		return nil, fmt.Errorf("subscription repository not configured")
	}
	return s.subscriptionRepo.GetSubscriptionStatistics(ctx)
}
