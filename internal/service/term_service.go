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

// TermService handles term management operations
type TermService struct {
	termRepo       *postgres.TermRepository
	sessionRepo    *postgres.SessionRepository
	enrollmentRepo *postgres.EnrollmentRepository
	auditService   *AuditService
	billingService *BillingService // For triggering billing on activation
}

// NewTermService creates a new term service
func NewTermService(
	termRepo *postgres.TermRepository,
	sessionRepo *postgres.SessionRepository,
	enrollmentRepo *postgres.EnrollmentRepository,
	auditService *AuditService,
	billingService *BillingService,
) *TermService {
	return &TermService{
		termRepo:       termRepo,
		sessionRepo:    sessionRepo,
		enrollmentRepo: enrollmentRepo,
		auditService:   auditService,
		billingService: billingService,
	}
}

// CreateTermRequest represents term creation data
type CreateTermRequest struct {
	Ordinal              domain.TermOrdinal `json:"ordinal" validate:"required,oneof=FIRST SECOND THIRD"`
	StartDate            time.Time          `json:"start_date" validate:"required"`
	EndDate              time.Time          `json:"end_date" validate:"required"`
	Holidays             []domain.Holiday   `json:"holidays,omitempty"`
	NonInstructionalDays []time.Time        `json:"non_instructional_days,omitempty"`
	Description          *string            `json:"description,omitempty" validate:"omitempty,max=500"`
}

// UpdateTermRequest represents term update data
type UpdateTermRequest struct {
	StartDate            *time.Time         `json:"start_date,omitempty"`
	EndDate              *time.Time         `json:"end_date,omitempty"`
	Holidays             *[]domain.Holiday  `json:"holidays,omitempty"`
	NonInstructionalDays *[]time.Time       `json:"non_instructional_days,omitempty"`
	Description          *string            `json:"description,omitempty" validate:"omitempty,max=500"`
}

// TermFilters represents filters for listing terms
type TermFilters struct {
	Status  *domain.TermStatus  `json:"status,omitempty"`
	Ordinal *domain.TermOrdinal `json:"ordinal,omitempty"`
}

// CreateTerm creates a term (validates BR-002: no overlap)
func (s *TermService) CreateTerm(
	ctx context.Context,
	sessionID uuid.UUID,
	req *CreateTermRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Term, error) {
	// Get session to verify it exists and get tenant ID
	session, err := s.sessionRepo.Get(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Validate end date is after start date
	if req.EndDate.Before(req.StartDate) || req.EndDate.Equal(req.StartDate) {
		return nil, fmt.Errorf("end date must be after start date")
	}

	// Check if a term with the same ordinal already exists for this session (BR-001)
	existingTerms, _, err := s.termRepo.ListBySession(ctx, sessionID, repository.PaginationParams{Limit: 100})
	if err != nil {
		return nil, fmt.Errorf("failed to check existing terms: %w", err)
	}

	for _, existing := range existingTerms {
		if existing.Ordinal == req.Ordinal {
			return nil, fmt.Errorf("a term with ordinal %s already exists for this session", req.Ordinal)
		}
	}

	// Validate no overlap with existing terms in the session (BR-002)
	for _, existing := range existingTerms {
		if datesOverlap(req.StartDate, req.EndDate, existing.StartDate, existing.EndDate) {
			return nil, fmt.Errorf("term dates overlap with existing term (BR-002)")
		}
	}

	// Ensure session has exactly 3 terms or less (BR-001)
	if len(existingTerms) >= 3 {
		return nil, fmt.Errorf("session already has 3 terms (BR-001)")
	}

	// Create term
	term := &domain.Term{
		ID:                   uuid.New(),
		TenantID:             session.TenantID,
		SessionID:            sessionID,
		Ordinal:              req.Ordinal,
		StartDate:            req.StartDate,
		EndDate:              req.EndDate,
		Status:               domain.TermStatusDraft,
		Holidays:             req.Holidays,
		NonInstructionalDays: req.NonInstructionalDays,
		Description:          req.Description,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	if err := s.termRepo.Create(ctx, term, nil); err != nil {
		return nil, fmt.Errorf("failed to create term: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionTermCreated,
		actorID,
		actorRole,
		&session.TenantID,
		domain.AuditResourceTerm,
		term.ID,
		nil,
		term,
		ipAddress,
	)

	return term, nil
}

// GetTerm gets a term by ID
func (s *TermService) GetTerm(ctx context.Context, id uuid.UUID) (*domain.Term, error) {
	term, err := s.termRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get term: %w", err)
	}
	return term, nil
}

// UpdateTerm updates a term
func (s *TermService) UpdateTerm(
	ctx context.Context,
	id uuid.UUID,
	req *UpdateTermRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Term, error) {
	// Get existing term
	term, err := s.termRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get term: %w", err)
	}

	// Cannot update an active or completed term
	if term.IsActive() || term.IsCompleted() {
		return nil, fmt.Errorf("cannot update an active or completed term")
	}

	// Store before state for audit
	beforeState := *term

	// Update fields
	if req.StartDate != nil {
		term.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		term.EndDate = *req.EndDate
	}
	if req.Holidays != nil {
		term.Holidays = *req.Holidays
	}
	if req.NonInstructionalDays != nil {
		term.NonInstructionalDays = *req.NonInstructionalDays
	}
	if req.Description != nil {
		term.Description = req.Description
	}
	term.UpdatedAt = time.Now()

	// Validate end date is after start date
	if term.EndDate.Before(term.StartDate) || term.EndDate.Equal(term.StartDate) {
		return nil, fmt.Errorf("end date must be after start date")
	}

	// Validate no overlap with other terms in the session (BR-002)
	existingTerms, _, err := s.termRepo.ListBySession(ctx, term.SessionID, repository.PaginationParams{Limit: 100})
	if err != nil {
		return nil, fmt.Errorf("failed to check existing terms: %w", err)
	}

	for _, existing := range existingTerms {
		if existing.ID != term.ID {
			if datesOverlap(term.StartDate, term.EndDate, existing.StartDate, existing.EndDate) {
				return nil, fmt.Errorf("term dates overlap with existing term (BR-002)")
			}
		}
	}

	if err := s.termRepo.Update(ctx, term, nil); err != nil {
		return nil, fmt.Errorf("failed to update term: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionTermUpdated,
		actorID,
		actorRole,
		&term.TenantID,
		domain.AuditResourceTerm,
		term.ID,
		&beforeState,
		term,
		ipAddress,
	)

	return term, nil
}

// DeleteTerm deletes a term
func (s *TermService) DeleteTerm(
	ctx context.Context,
	id uuid.UUID,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) error {
	// Get term for audit
	term, err := s.termRepo.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get term: %w", err)
	}

	// Cannot delete active or completed term
	if term.IsActive() || term.IsCompleted() {
		return fmt.Errorf("cannot delete an active or completed term")
	}

	// Delete term
	if err := s.termRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete term: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionTermUpdated,
		actorID,
		actorRole,
		&term.TenantID,
		domain.AuditResourceTerm,
		term.ID,
		term,
		nil,
		ipAddress,
	)

	return nil
}

// ListTerms lists terms for a session
func (s *TermService) ListTerms(
	ctx context.Context,
	sessionID uuid.UUID,
	filters *TermFilters,
) ([]*domain.Term, error) {
	// TODO: Apply filters
	terms, _, err := s.termRepo.ListBySession(ctx, sessionID, repository.PaginationParams{Limit: 100})
	if err != nil {
		return nil, fmt.Errorf("failed to list terms: %w", err)
	}
	return terms, nil
}

// ActivateTerm activates a term (triggers billing via BillingService)
func (s *TermService) ActivateTerm(
	ctx context.Context,
	id uuid.UUID,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Term, error) {
	// Get term
	term, err := s.termRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get term: %w", err)
	}

	// Check if there's already an active term for this session
	activeTerm, err := s.termRepo.GetActiveTerm(ctx, term.SessionID)
	if err != nil && err != repository.ErrNotFound {
		return nil, fmt.Errorf("failed to check active term: %w", err)
	}
	if activeTerm != nil && activeTerm.ID != term.ID {
		return nil, fmt.Errorf("another term is already active for this session")
	}

	// Store before state
	beforeState := *term

	// Activate term
	term.Activate()

	if err := s.termRepo.Update(ctx, term, nil); err != nil {
		return nil, fmt.Errorf("failed to activate term: %w", err)
	}

	// Trigger billing (BR-009, BR-010)
	// Count active students for this tenant
	studentCount, err := s.enrollmentRepo.CountActiveStudents(ctx, term.TenantID, term.SessionID)
	if err != nil {
		// Don't fail activation if billing fails, but log the error
		fmt.Printf("failed to count students for billing: %v\n", err)
	} else if s.billingService != nil {
		// Generate invoice for this term
		_, err = s.billingService.GenerateTermInvoice(ctx, term.TenantID, term.SessionID, term.ID, studentCount, actorID, actorRole, ipAddress)
		if err != nil {
			// Log but don't fail
			fmt.Printf("failed to generate invoice for term %s: %v\n", term.ID, err)
		}
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionTermActivated,
		actorID,
		actorRole,
		&term.TenantID,
		domain.AuditResourceTerm,
		term.ID,
		&beforeState,
		term,
		ipAddress,
	)

	return term, nil
}

// CompleteTerm marks a term as completed
func (s *TermService) CompleteTerm(
	ctx context.Context,
	id uuid.UUID,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Term, error) {
	// Get term
	term, err := s.termRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get term: %w", err)
	}

	if !term.IsActive() {
		return nil, fmt.Errorf("only active terms can be completed")
	}

	// Store before state
	beforeState := *term

	// Complete term
	term.Complete()

	if err := s.termRepo.Update(ctx, term, nil); err != nil {
		return nil, fmt.Errorf("failed to complete term: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionTermCompleted,
		actorID,
		actorRole,
		&term.TenantID,
		domain.AuditResourceTerm,
		term.ID,
		&beforeState,
		term,
		ipAddress,
	)

	return term, nil
}

// GetActiveTerm gets the current active term for a session
func (s *TermService) GetActiveTerm(ctx context.Context, sessionID uuid.UUID) (*domain.Term, error) {
	term, err := s.termRepo.GetActiveTerm(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active term: %w", err)
	}
	return term, nil
}

// IsInstructionalDay checks if a date is an instructional day
func (s *TermService) IsInstructionalDay(ctx context.Context, termID uuid.UUID, date time.Time) (bool, error) {
	term, err := s.termRepo.Get(ctx, termID)
	if err != nil {
		return false, fmt.Errorf("failed to get term: %w", err)
	}
	return term.IsInstructionalDay(date), nil
}

// Helper function to check if date ranges overlap
func datesOverlap(start1, end1, start2, end2 time.Time) bool {
	// Two date ranges overlap if:
	// - start1 is before end2, AND
	// - end1 is after start2
	return start1.Before(end2) && end1.After(start2)
}
