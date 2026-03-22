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

// CommunicationService handles email communication operations
type CommunicationService struct {
	communicationRepo *postgres.CommunicationRepository
	userRepo          *postgres.UserRepository
	classRepo         *postgres.ClassRepository
	courseRepo        *postgres.CourseRepository
	auditService      *AuditService
}

// NewCommunicationService creates a new communication service
func NewCommunicationService(
	communicationRepo *postgres.CommunicationRepository,
	userRepo *postgres.UserRepository,
	classRepo *postgres.ClassRepository,
	courseRepo *postgres.CourseRepository,
	auditService *AuditService,
) *CommunicationService {
	return &CommunicationService{
		communicationRepo: communicationRepo,
		userRepo:          userRepo,
		classRepo:         classRepo,
		courseRepo:        courseRepo,
		auditService:      auditService,
	}
}

// ComposeEmailRequest represents a request to compose an email
type ComposeEmailRequest struct {
	Subject         string              `json:"subject" validate:"required,min=3,max=200"`
	Body            string              `json:"body" validate:"required,min=10"`
	HTMLBody        *string             `json:"html_body,omitempty"`
	RecipientScope  domain.RecipientScope `json:"recipient_scope" validate:"required,oneof=ALL_USERS ALL_TUTORS ALL_STUDENTS CLASS COURSE SPECIFIC_USERS"`
	TargetClassID   *uuid.UUID          `json:"target_class_id,omitempty" validate:"omitempty,uuid"`
	TargetCourseID  *uuid.UUID          `json:"target_course_id,omitempty" validate:"omitempty,uuid"`
	SpecificUserIDs []uuid.UUID         `json:"specific_user_ids,omitempty"`
	AttachmentURLs  []string            `json:"attachment_urls,omitempty"`
	ScheduledFor    *time.Time          `json:"scheduled_for,omitempty"`
	SendImmediately bool                `json:"send_immediately"`
}

// ComposeEmail creates a new email (draft, scheduled, or sends immediately)
func (s *CommunicationService) ComposeEmail(
	ctx context.Context,
	tenantID uuid.UUID,
	senderID uuid.UUID,
	role domain.Role,
	req *ComposeEmailRequest,
	ipAddress string,
) (*domain.Email, error) {
	// Validate sender permissions based on role
	if err := s.validateSenderPermissions(ctx, senderID, role, req); err != nil {
		return nil, err
	}

	// Get recipients based on scope
	recipients, err := s.communicationRepo.GetRecipientsForScope(
		ctx,
		tenantID,
		req.RecipientScope,
		req.TargetClassID,
		req.TargetCourseID,
		req.SpecificUserIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve recipients: %w", err)
	}

	if len(recipients) == 0 {
		return nil, fmt.Errorf("no recipients found for the specified scope")
	}

	now := time.Now()
	email := &domain.Email{
		ID:              uuid.New(),
		TenantID:        tenantID,
		SenderID:        senderID,
		Subject:         req.Subject,
		Body:            req.Body,
		HTMLBody:        req.HTMLBody,
		RecipientScope:  req.RecipientScope,
		TargetClassID:   req.TargetClassID,
		TargetCourseID:  req.TargetCourseID,
		SpecificUserIDs: req.SpecificUserIDs,
		Recipients:      recipients,
		AttachmentURLs:  req.AttachmentURLs,
		TotalRecipients: len(recipients),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Determine status based on request
	if req.SendImmediately {
		email.Status = domain.CommunicationStatusSending
	} else if req.ScheduledFor != nil {
		if req.ScheduledFor.Before(now) {
			return nil, fmt.Errorf("scheduled time must be in the future")
		}
		email.Status = domain.CommunicationStatusScheduled
		email.ScheduledFor = req.ScheduledFor
	} else {
		email.Status = domain.CommunicationStatusDraft
	}

	if err := s.communicationRepo.Create(ctx, email); err != nil {
		return nil, fmt.Errorf("failed to create email: %w", err)
	}

	// If sending immediately, trigger the send process
	if req.SendImmediately {
		go s.processEmailSend(context.Background(), email)
	}

	// Audit log
	action := domain.AuditActionEmailScheduled
	if req.SendImmediately {
		action = domain.AuditActionEmailSent
	}

	_ = s.auditService.LogAction(
		ctx,
		action,
		senderID,
		role,
		&tenantID,
		domain.AuditResourceEmail,
		email.ID,
		nil,
		email,
		ipAddress,
	)

	return email, nil
}

// validateSenderPermissions validates that the sender can send to the specified scope
func (s *CommunicationService) validateSenderPermissions(
	ctx context.Context,
	senderID uuid.UUID,
	role domain.Role,
	req *ComposeEmailRequest,
) error {
	// Students cannot send emails
	if role == domain.RoleStudent {
		return fmt.Errorf("students are not allowed to send emails")
	}

	// Admins and Super Admins can send to any scope
	if role == domain.RoleAdmin || role == domain.RoleSuperAdmin {
		return nil
	}

	// Tutors can only send to CLASS or COURSE scope for their assigned classes/courses
	if role == domain.RoleTutor {
		if req.RecipientScope != domain.RecipientScopeClass && req.RecipientScope != domain.RecipientScopeCourse {
			return fmt.Errorf("tutors can only send emails to their assigned classes or courses")
		}

		hasAccess, err := s.communicationRepo.ValidateTutorAccess(
			ctx,
			senderID,
			req.RecipientScope,
			req.TargetClassID,
			req.TargetCourseID,
		)
		if err != nil {
			return fmt.Errorf("failed to validate tutor access: %w", err)
		}
		if !hasAccess {
			return fmt.Errorf("you do not have access to send emails to this scope")
		}
	}

	return nil
}

// processEmailSend processes the actual email sending (simulated)
func (s *CommunicationService) processEmailSend(ctx context.Context, email *domain.Email) {
	// In a real implementation, this would integrate with an email service provider
	// like SendGrid, AWS SES, Mailgun, etc.

	for i := range email.Recipients {
		// Simulate sending - in production, call the email provider API
		email.Recipients[i].Status = domain.CommunicationStatusSent
		now := time.Now()
		email.Recipients[i].SentAt = &now
		email.SuccessCount++
	}

	email.Status = domain.CommunicationStatusSent
	now := time.Now()
	email.SentAt = &now
	email.UpdatedAt = now

	_ = s.communicationRepo.Update(ctx, email)
}

// GetEmail retrieves an email by ID
func (s *CommunicationService) GetEmail(ctx context.Context, id uuid.UUID) (*domain.Email, error) {
	return s.communicationRepo.Get(ctx, id)
}

// ListEmails lists emails for a tenant
func (s *CommunicationService) ListEmails(
	ctx context.Context,
	tenantID uuid.UUID,
	senderID *uuid.UUID,
	status *domain.CommunicationStatus,
	role domain.Role,
	params repository.PaginationParams,
) ([]*domain.Email, *repository.PaginatedResult, error) {
	var emails []*domain.Email
	var total int
	var err error

	// Tutors can only see their own emails
	if role == domain.RoleTutor && senderID != nil {
		emails, total, err = s.communicationRepo.ListBySender(ctx, *senderID, params)
	} else {
		emails, total, err = s.communicationRepo.ListByTenant(ctx, tenantID, status, params)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to list emails: %w", err)
	}

	pagination := repository.BuildPaginatedResult(total, params)

	return emails, &pagination, nil
}

// CancelEmail cancels a scheduled or draft email
func (s *CommunicationService) CancelEmail(
	ctx context.Context,
	emailID uuid.UUID,
	userID uuid.UUID,
	role domain.Role,
	ipAddress string,
) (*domain.Email, error) {
	email, err := s.communicationRepo.Get(ctx, emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to get email: %w", err)
	}

	// Only the sender or an admin can cancel
	if email.SenderID != userID && role != domain.RoleAdmin && role != domain.RoleSuperAdmin {
		return nil, fmt.Errorf("you can only cancel your own emails")
	}

	if !email.CanCancel() {
		return nil, fmt.Errorf("this email cannot be cancelled (status: %s)", email.Status)
	}

	email.Cancel()

	if err := s.communicationRepo.Update(ctx, email); err != nil {
		return nil, fmt.Errorf("failed to cancel email: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionEmailCancelled,
		userID,
		role,
		&email.TenantID,
		domain.AuditResourceEmail,
		email.ID,
		nil,
		email,
		ipAddress,
	)

	return email, nil
}

// DeleteEmail deletes a draft email
func (s *CommunicationService) DeleteEmail(
	ctx context.Context,
	emailID uuid.UUID,
	userID uuid.UUID,
	role domain.Role,
) error {
	email, err := s.communicationRepo.Get(ctx, emailID)
	if err != nil {
		return fmt.Errorf("failed to get email: %w", err)
	}

	// Only the sender or an admin can delete
	if email.SenderID != userID && role != domain.RoleAdmin && role != domain.RoleSuperAdmin {
		return fmt.Errorf("you can only delete your own emails")
	}

	if !email.IsDraft() {
		return fmt.Errorf("only draft emails can be deleted")
	}

	return s.communicationRepo.Delete(ctx, emailID)
}

// ScheduleEmail schedules a draft email for future delivery
func (s *CommunicationService) ScheduleEmail(
	ctx context.Context,
	emailID uuid.UUID,
	userID uuid.UUID,
	role domain.Role,
	scheduledFor time.Time,
	ipAddress string,
) (*domain.Email, error) {
	email, err := s.communicationRepo.Get(ctx, emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to get email: %w", err)
	}

	// Only the sender or an admin can schedule
	if email.SenderID != userID && role != domain.RoleAdmin && role != domain.RoleSuperAdmin {
		return nil, fmt.Errorf("you can only schedule your own emails")
	}

	if !email.IsDraft() {
		return nil, fmt.Errorf("only draft emails can be scheduled")
	}

	if scheduledFor.Before(time.Now()) {
		return nil, fmt.Errorf("scheduled time must be in the future")
	}

	email.Status = domain.CommunicationStatusScheduled
	email.ScheduledFor = &scheduledFor
	email.UpdatedAt = time.Now()

	if err := s.communicationRepo.Update(ctx, email); err != nil {
		return nil, fmt.Errorf("failed to schedule email: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionEmailScheduled,
		userID,
		role,
		&email.TenantID,
		domain.AuditResourceEmail,
		email.ID,
		nil,
		email,
		ipAddress,
	)

	return email, nil
}

// SendEmailNow sends a draft or scheduled email immediately
func (s *CommunicationService) SendEmailNow(
	ctx context.Context,
	emailID uuid.UUID,
	userID uuid.UUID,
	role domain.Role,
	ipAddress string,
) (*domain.Email, error) {
	email, err := s.communicationRepo.Get(ctx, emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to get email: %w", err)
	}

	// Only the sender or an admin can send
	if email.SenderID != userID && role != domain.RoleAdmin && role != domain.RoleSuperAdmin {
		return nil, fmt.Errorf("you can only send your own emails")
	}

	if email.Status != domain.CommunicationStatusDraft && email.Status != domain.CommunicationStatusScheduled {
		return nil, fmt.Errorf("this email cannot be sent (status: %s)", email.Status)
	}

	email.Status = domain.CommunicationStatusSending
	email.UpdatedAt = time.Now()

	if err := s.communicationRepo.Update(ctx, email); err != nil {
		return nil, fmt.Errorf("failed to update email status: %w", err)
	}

	// Process the send asynchronously
	go s.processEmailSend(context.Background(), email)

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionEmailSent,
		userID,
		role,
		&email.TenantID,
		domain.AuditResourceEmail,
		email.ID,
		nil,
		email,
		ipAddress,
	)

	return email, nil
}

// SendEmailToCoTutorsRequest represents a request to send email to co-tutors
type SendEmailToCoTutorsRequest struct {
	ClassID        uuid.UUID `json:"class_id" validate:"required,uuid"`
	Subject        string    `json:"subject" validate:"required,min=3,max=200"`
	Body           string    `json:"body" validate:"required,min=10"`
	HTMLBody       *string   `json:"html_body,omitempty"`
	AttachmentURLs []string  `json:"attachment_urls,omitempty"`
}

// SendEmailToCoTutors sends an email to co-tutors in the same class
func (s *CommunicationService) SendEmailToCoTutors(
	ctx context.Context,
	tenantID uuid.UUID,
	senderID uuid.UUID,
	req *SendEmailToCoTutorsRequest,
	ipAddress string,
) (*domain.Email, error) {
	// Validate tutor is assigned to this class
	hasAccess, err := s.communicationRepo.ValidateTutorAccess(
		ctx,
		senderID,
		domain.RecipientScopeClass,
		&req.ClassID,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to validate access: %w", err)
	}
	if !hasAccess {
		return nil, fmt.Errorf("you are not assigned to this class")
	}

	// Get co-tutors
	recipients, err := s.communicationRepo.GetCoTutorsForClass(ctx, req.ClassID, senderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get co-tutors: %w", err)
	}

	if len(recipients) == 0 {
		return nil, fmt.Errorf("no co-tutors found in this class")
	}

	now := time.Now()
	email := &domain.Email{
		ID:              uuid.New(),
		TenantID:        tenantID,
		SenderID:        senderID,
		Subject:         req.Subject,
		Body:            req.Body,
		HTMLBody:        req.HTMLBody,
		RecipientScope:  domain.RecipientScopeClass,
		TargetClassID:   &req.ClassID,
		Recipients:      recipients,
		Status:          domain.CommunicationStatusSending,
		AttachmentURLs:  req.AttachmentURLs,
		TotalRecipients: len(recipients),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.communicationRepo.Create(ctx, email); err != nil {
		return nil, fmt.Errorf("failed to create email: %w", err)
	}

	// Process send asynchronously
	go s.processEmailSend(context.Background(), email)

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionEmailSent,
		senderID,
		domain.RoleTutor,
		&tenantID,
		domain.AuditResourceEmail,
		email.ID,
		nil,
		email,
		ipAddress,
	)

	return email, nil
}

// GetEmailStatistics retrieves email statistics for a tenant
func (s *CommunicationService) GetEmailStatistics(
	ctx context.Context,
	tenantID uuid.UUID,
	startDate, endDate time.Time,
) (*postgres.EmailStatistics, error) {
	return s.communicationRepo.GetEmailStatistics(ctx, tenantID, startDate, endDate)
}

// SearchEmails searches emails by subject or body
func (s *CommunicationService) SearchEmails(
	ctx context.Context,
	tenantID uuid.UUID,
	searchTerm string,
	params repository.PaginationParams,
) ([]*domain.Email, *repository.PaginatedResult, error) {
	if len(searchTerm) < 3 {
		return nil, nil, fmt.Errorf("search term must be at least 3 characters")
	}
	emails, total, err := s.communicationRepo.SearchEmails(ctx, tenantID, searchTerm, params)
	if err != nil {
		return nil, nil, err
	}
	pagination := repository.BuildPaginatedResult(total, params)
	return emails, &pagination, nil
}

// ProcessScheduledEmails processes emails that are due to be sent
// This would be called by a background job/scheduler
func (s *CommunicationService) ProcessScheduledEmails(ctx context.Context) error {
	emails, err := s.communicationRepo.ListScheduled(ctx, time.Now())
	if err != nil {
		return fmt.Errorf("failed to list scheduled emails: %w", err)
	}

	for _, email := range emails {
		email.Status = domain.CommunicationStatusSending
		email.UpdatedAt = time.Now()

		if err := s.communicationRepo.Update(ctx, email); err != nil {
			continue // Log error and continue with next email
		}

		go s.processEmailSend(context.Background(), email)
	}

	return nil
}

// GetRecipientPreview returns a preview of recipients for a scope
func (s *CommunicationService) GetRecipientPreview(
	ctx context.Context,
	tenantID uuid.UUID,
	scope domain.RecipientScope,
	targetClassID *uuid.UUID,
	targetCourseID *uuid.UUID,
	specificUserIDs []uuid.UUID,
) ([]domain.DeliveryRecipient, int, error) {
	recipients, err := s.communicationRepo.GetRecipientsForScope(
		ctx,
		tenantID,
		scope,
		targetClassID,
		targetCourseID,
		specificUserIDs,
	)
	if err != nil {
		return nil, 0, err
	}

	// Return first 10 as preview with total count
	preview := recipients
	if len(recipients) > 10 {
		preview = recipients[:10]
	}

	return preview, len(recipients), nil
}
