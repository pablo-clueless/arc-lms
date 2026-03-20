package jobs

import (
	"context"
	"fmt"
	"log"
	"time"

	"arc-lms/internal/domain"
	"arc-lms/internal/pkg/email"
	"arc-lms/internal/repository/postgres"
)

// ScheduledEmailJob sends emails that are scheduled for delivery
type ScheduledEmailJob struct {
	communicationRepo *postgres.CommunicationRepository
	emailService      *email.EmailService
	logger            *log.Logger
	interval          time.Duration
}

// NewScheduledEmailJob creates a new scheduled email job
func NewScheduledEmailJob(
	communicationRepo *postgres.CommunicationRepository,
	emailService *email.EmailService,
	logger *log.Logger,
) *ScheduledEmailJob {
	if logger == nil {
		logger = log.Default()
	}
	return &ScheduledEmailJob{
		communicationRepo: communicationRepo,
		emailService:      emailService,
		logger:            logger,
		interval:          1 * time.Minute, // Check every minute
	}
}

// Name returns the job name
func (j *ScheduledEmailJob) Name() string {
	return "scheduled-email-sender"
}

// Interval returns how often the job should run
func (j *ScheduledEmailJob) Interval() time.Duration {
	return j.interval
}

// Run executes the job
func (j *ScheduledEmailJob) Run(ctx context.Context) error {
	j.logger.Println("[ScheduledEmailJob] Checking for scheduled emails")

	if j.emailService == nil || !j.emailService.IsConfigured() {
		j.logger.Println("[ScheduledEmailJob] Email service not configured, skipping")
		return nil
	}

	// Get emails scheduled for now or earlier
	emails, err := j.communicationRepo.ListScheduled(ctx, time.Now())
	if err != nil {
		return fmt.Errorf("failed to list scheduled emails: %w", err)
	}

	if len(emails) == 0 {
		return nil
	}

	j.logger.Printf("[ScheduledEmailJob] Processing %d scheduled emails", len(emails))

	var successCount, failCount int

	for _, emailMsg := range emails {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Mark as sending
		emailMsg.Status = domain.CommunicationStatusSending
		emailMsg.UpdatedAt = time.Now()

		if err := j.communicationRepo.Update(ctx, emailMsg); err != nil {
			j.logger.Printf("[ScheduledEmailJob] Failed to update email %s status: %v", emailMsg.ID, err)
			failCount++
			continue
		}

		// Process each recipient
		if err := j.sendEmail(ctx, emailMsg); err != nil {
			j.logger.Printf("[ScheduledEmailJob] Failed to send email %s: %v", emailMsg.ID, err)

			// Mark as failed
			emailMsg.Status = domain.CommunicationStatusFailed
			emailMsg.UpdatedAt = time.Now()
			j.communicationRepo.Update(ctx, emailMsg)

			failCount++
			continue
		}

		successCount++
	}

	j.logger.Printf("[ScheduledEmailJob] Completed: %d sent, %d failed", successCount, failCount)
	return nil
}

// sendEmail sends an email to all recipients using the email service
func (j *ScheduledEmailJob) sendEmail(ctx context.Context, emailMsg *domain.Email) error {
	now := time.Now()
	var lastError error

	for i := range emailMsg.Recipients {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		recipient := &emailMsg.Recipients[i]

		// Send using the email service
		htmlBody := ""
		if emailMsg.HTMLBody != nil {
			htmlBody = *emailMsg.HTMLBody
		}

		err := j.emailService.Send(
			recipient.Email,
			emailMsg.Subject,
			emailMsg.Body,
			htmlBody,
		)

		if err != nil {
			j.logger.Printf("[ScheduledEmailJob] Failed to send to %s: %v", recipient.Email, err)
			recipient.Status = domain.CommunicationStatusFailed
			recipient.FailedAt = &now
			reason := err.Error()
			recipient.FailureReason = &reason
			emailMsg.FailureCount++
			lastError = err
		} else {
			recipient.Status = domain.CommunicationStatusSent
			recipient.SentAt = &now
			emailMsg.SuccessCount++
		}
	}

	// Update email status based on results
	if emailMsg.SuccessCount > 0 {
		if emailMsg.FailureCount > 0 {
			emailMsg.Status = domain.CommunicationStatusPartial
		} else {
			emailMsg.Status = domain.CommunicationStatusSent
		}
		emailMsg.SentAt = &now
	} else {
		emailMsg.Status = domain.CommunicationStatusFailed
	}
	emailMsg.UpdatedAt = now

	if err := j.communicationRepo.Update(ctx, emailMsg); err != nil {
		return fmt.Errorf("failed to update email after sending: %w", err)
	}

	j.logger.Printf("[ScheduledEmailJob] Email %s: %d sent, %d failed",
		emailMsg.ID, emailMsg.SuccessCount, emailMsg.FailureCount)

	// Return error only if all failed
	if emailMsg.SuccessCount == 0 && lastError != nil {
		return lastError
	}

	return nil
}
