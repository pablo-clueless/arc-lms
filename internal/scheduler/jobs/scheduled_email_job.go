package jobs

import (
	"context"
	"fmt"
	"log"
	"time"

	"arc-lms/internal/domain"
	"arc-lms/internal/repository/postgres"
)

// ScheduledEmailJob sends emails that are scheduled for delivery
type ScheduledEmailJob struct {
	communicationRepo *postgres.CommunicationRepository
	logger            *log.Logger
	interval          time.Duration
}

// NewScheduledEmailJob creates a new scheduled email job
func NewScheduledEmailJob(
	communicationRepo *postgres.CommunicationRepository,
	logger *log.Logger,
) *ScheduledEmailJob {
	if logger == nil {
		logger = log.Default()
	}
	return &ScheduledEmailJob{
		communicationRepo: communicationRepo,
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

	// Get emails scheduled for now or earlier
	emails, err := j.communicationRepo.ListScheduled(ctx, time.Now())
	if err != nil {
		return fmt.Errorf("failed to list scheduled emails: %w", err)
	}

	if len(emails) == 0 {
		j.logger.Println("[ScheduledEmailJob] No scheduled emails to process")
		return nil
	}

	j.logger.Printf("[ScheduledEmailJob] Processing %d scheduled emails", len(emails))

	var successCount, failCount int

	for _, email := range emails {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Mark as sending
		email.Status = domain.CommunicationStatusSending
		email.UpdatedAt = time.Now()

		if err := j.communicationRepo.Update(ctx, email); err != nil {
			j.logger.Printf("[ScheduledEmailJob] Failed to update email %s status: %v", email.ID, err)
			failCount++
			continue
		}

		// Process each recipient
		if err := j.sendEmail(ctx, email); err != nil {
			j.logger.Printf("[ScheduledEmailJob] Failed to send email %s: %v", email.ID, err)
			failCount++
			continue
		}

		successCount++
	}

	j.logger.Printf("[ScheduledEmailJob] Completed: %d sent, %d failed", successCount, failCount)
	return nil
}

// sendEmail sends an email to all recipients
func (j *ScheduledEmailJob) sendEmail(ctx context.Context, email *domain.Email) error {
	// In a real implementation, this would integrate with an email service provider
	// like SendGrid, AWS SES, Mailgun, Postmark, etc.

	now := time.Now()

	for i := range email.Recipients {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Simulate sending - in production, call the email provider API
		// Example with SendGrid:
		// err := sendgrid.Send(email.Recipients[i].Email, email.Subject, email.Body, email.HTMLBody)

		// For now, mark all as successful
		email.Recipients[i].Status = domain.CommunicationStatusSent
		email.Recipients[i].SentAt = &now
		email.SuccessCount++

		// In real implementation, handle failures:
		// if err != nil {
		//     email.Recipients[i].Status = domain.CommunicationStatusFailed
		//     email.Recipients[i].FailedAt = &now
		//     reason := err.Error()
		//     email.Recipients[i].FailureReason = &reason
		//     email.FailureCount++
		// }
	}

	// Update email status to sent
	email.Status = domain.CommunicationStatusSent
	email.SentAt = &now
	email.UpdatedAt = now

	if err := j.communicationRepo.Update(ctx, email); err != nil {
		return fmt.Errorf("failed to update email after sending: %w", err)
	}

	j.logger.Printf("[ScheduledEmailJob] Sent email %s to %d recipients", email.ID, len(email.Recipients))
	return nil
}

// EmailProvider interface for email service providers
type EmailProvider interface {
	Send(to, subject, textBody, htmlBody string) error
	SendBatch(recipients []string, subject, textBody, htmlBody string) error
}

// MockEmailProvider is a mock implementation for testing
type MockEmailProvider struct{}

func (p *MockEmailProvider) Send(to, subject, textBody, htmlBody string) error {
	return nil
}

func (p *MockEmailProvider) SendBatch(recipients []string, subject, textBody, htmlBody string) error {
	return nil
}
