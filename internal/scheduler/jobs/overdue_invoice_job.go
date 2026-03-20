package jobs

import (
	"context"
	"fmt"
	"log"
	"time"

	"arc-lms/internal/domain"
	"arc-lms/internal/repository"
	"arc-lms/internal/repository/postgres"
)

// OverdueInvoiceJob processes overdue invoices and updates subscription statuses
type OverdueInvoiceJob struct {
	invoiceRepo      *postgres.InvoiceRepository
	subscriptionRepo *postgres.SubscriptionRepository
	tenantRepo       *postgres.TenantRepository
	notificationRepo *postgres.NotificationRepository
	logger           *log.Logger
	interval         time.Duration
	gracePeriodDays  int
	suspensionDays   int
}

// NewOverdueInvoiceJob creates a new overdue invoice job
func NewOverdueInvoiceJob(
	invoiceRepo *postgres.InvoiceRepository,
	subscriptionRepo *postgres.SubscriptionRepository,
	tenantRepo *postgres.TenantRepository,
	notificationRepo *postgres.NotificationRepository,
	logger *log.Logger,
) *OverdueInvoiceJob {
	if logger == nil {
		logger = log.Default()
	}
	return &OverdueInvoiceJob{
		invoiceRepo:      invoiceRepo,
		subscriptionRepo: subscriptionRepo,
		tenantRepo:       tenantRepo,
		notificationRepo: notificationRepo,
		logger:           logger,
		interval:         1 * time.Hour, // Run every hour
		gracePeriodDays:  14,
		suspensionDays:   30,
	}
}

// Name returns the job name
func (j *OverdueInvoiceJob) Name() string {
	return "overdue-invoice-processor"
}

// Interval returns how often the job should run
func (j *OverdueInvoiceJob) Interval() time.Duration {
	return j.interval
}

// Run executes the job
func (j *OverdueInvoiceJob) Run(ctx context.Context) error {
	j.logger.Println("[OverdueInvoiceJob] Starting overdue invoice processing")

	// Step 1: Mark pending invoices past due date as overdue
	markedCount, err := j.invoiceRepo.MarkOverdueInvoices(ctx)
	if err != nil {
		return fmt.Errorf("failed to mark overdue invoices: %w", err)
	}
	j.logger.Printf("[OverdueInvoiceJob] Marked %d invoices as overdue", markedCount)

	// Step 2: Get all overdue invoices for further processing
	params := repository.PaginationParams{Limit: 100}
	overdueInvoices, err := j.invoiceRepo.GetOverdueInvoices(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to get overdue invoices: %w", err)
	}

	now := time.Now()
	var flaggedCount, suspendedCount int

	for _, invoice := range overdueInvoices {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		daysOverdue := invoice.DaysOverdue(now)

		// After grace period: Flag subscription as payment overdue
		if daysOverdue >= j.gracePeriodDays && j.subscriptionRepo != nil {
			if err := j.subscriptionRepo.MarkAsOverdue(ctx, invoice.TenantID); err != nil {
				j.logger.Printf("[OverdueInvoiceJob] Failed to mark subscription as overdue for tenant %s: %v", invoice.TenantID, err)
			} else {
				flaggedCount++
				// Send notification to admin
				j.sendOverdueNotification(ctx, invoice, daysOverdue)
			}
		}

		// After suspension threshold: Suspend subscription and tenant
		if daysOverdue >= j.suspensionDays && j.subscriptionRepo != nil {
			subscription, err := j.subscriptionRepo.GetByTenant(ctx, invoice.TenantID)
			if err != nil {
				j.logger.Printf("[OverdueInvoiceJob] Failed to get subscription for tenant %s: %v", invoice.TenantID, err)
				continue
			}

			if subscription != nil && !subscription.IsSuspended() {
				if err := j.subscriptionRepo.Suspend(ctx, subscription.ID); err != nil {
					j.logger.Printf("[OverdueInvoiceJob] Failed to suspend subscription %s: %v", subscription.ID, err)
				} else {
					suspendedCount++

					// Suspend tenant if tenant repo is available
					if j.tenantRepo != nil {
						reason := fmt.Sprintf("Automatic suspension due to unpaid invoice %s (overdue %d days)", invoice.InvoiceNumber, daysOverdue)
						if err := j.tenantRepo.Suspend(ctx, invoice.TenantID, reason, nil); err != nil {
							j.logger.Printf("[OverdueInvoiceJob] Failed to suspend tenant %s: %v", invoice.TenantID, err)
						}
					}

					// Send suspension notification
					j.sendSuspensionNotification(ctx, invoice)
				}
			}
		}
	}

	j.logger.Printf("[OverdueInvoiceJob] Completed: %d flagged as overdue, %d suspended", flaggedCount, suspendedCount)
	return nil
}

// sendOverdueNotification sends a notification about an overdue invoice
func (j *OverdueInvoiceJob) sendOverdueNotification(ctx context.Context, invoice *domain.Invoice, daysOverdue int) {
	if j.notificationRepo == nil {
		return
	}

	// TODO: Get admin users for this tenant and send notifications
	// This would require a user repository to look up admins
}

// sendSuspensionNotification sends a notification about account suspension
func (j *OverdueInvoiceJob) sendSuspensionNotification(ctx context.Context, invoice *domain.Invoice) {
	if j.notificationRepo == nil {
		return
	}

	// TODO: Get admin users for this tenant and send notifications
	// This would require a user repository to look up admins
}
