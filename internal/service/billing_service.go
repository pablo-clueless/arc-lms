package service

import (
	"context"
	"fmt"
	"time"

	"arc-lms/internal/domain"
	"arc-lms/internal/repository/postgres"
	"arc-lms/internal/repository"

	"github.com/google/uuid"
)

// BillingService handles billing and subscription operations
type BillingService struct {
	invoiceRepo      repository.InvoiceRepository
	subscriptionRepo repository.SubscriptionRepository
	tenantRepo       *postgres.TenantRepository
	auditService     *AuditService
}

// NewBillingService creates a new billing service
func NewBillingService(
	invoiceRepo repository.InvoiceRepository,
	subscriptionRepo repository.SubscriptionRepository,
	tenantRepo *postgres.TenantRepository,
	auditService *AuditService,
) *BillingService {
	return &BillingService{
		invoiceRepo:      invoiceRepo,
		subscriptionRepo: subscriptionRepo,
		tenantRepo:       tenantRepo,
		auditService:     auditService,
	}
}

// GenerateTermInvoice generates an invoice for a term activation (BR-009, BR-010)
func (s *BillingService) GenerateTermInvoice(
	ctx context.Context,
	tenantID uuid.UUID,
	termID uuid.UUID,
	studentCount int,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Invoice, error) {
	// Get tenant for billing information
	tenant, err := s.tenantRepo.Get(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Get or create subscription for this tenant
	subscription, err := s.getOrCreateSubscription(ctx, tenantID, actorID, actorRole, ipAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Calculate invoice amount (BR-010: NGN 500 per student per term)
	// Store in Kobo (1 NGN = 100 Kobo)
	pricePerStudentPerTermKobo := 50000 // NGN 500 * 100
	totalAmountKobo := studentCount * pricePerStudentPerTermKobo

	// Generate invoice number (format: INV-YYYY-NNNN)
	invoiceNumber := fmt.Sprintf("INV-%d-%04d", time.Now().Year(), time.Now().Unix()%10000)

	// Create line items
	lineItems := []domain.InvoiceLineItem{
		{
			Description: fmt.Sprintf("Term subscription for %d students", studentCount),
			Quantity:    studentCount,
			UnitPrice:   pricePerStudentPerTermKobo,
			Amount:      totalAmountKobo,
		},
	}

	// Calculate due date based on tenant configuration
	gracePeriod := tenant.Configuration.InvoiceGracePeriod
	if gracePeriod == 0 {
		gracePeriod = 14 // Default 14 days
	}
	dueDate := time.Now().Add(time.Duration(gracePeriod) * 24 * time.Hour)

	// Create invoice
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
		TaxAmount:      0, // No tax for now
		DiscountAmount: 0,
		TotalAmount:    totalAmountKobo,
		StudentCount:   studentCount,
		DueDate:        dueDate,
		IssuedDate:     time.Now(),
		BillingEmail:   tenant.BillingContact.Email,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.invoiceRepo.Create(ctx, invoice); err != nil {
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
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Subscription, error) {
	// TODO: Implement subscription repository
	return nil, fmt.Errorf("subscription management not yet implemented")
}
