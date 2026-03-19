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

// SessionService handles academic session operations
type SessionService struct {
	sessionRepo  *postgres.SessionRepository
	auditService *AuditService
}

// NewSessionService creates a new session service
func NewSessionService(
	sessionRepo *postgres.SessionRepository,
	auditService *AuditService,
) *SessionService {
	return &SessionService{
		sessionRepo:  sessionRepo,
		auditService: auditService,
	}
}

// CreateSessionRequest represents session creation data
type CreateSessionRequest struct {
	Label       string  `json:"label" validate:"required,min=7,max=20"`
	StartYear   int     `json:"start_year" validate:"required,min=2000,max=2100"`
	EndYear     int     `json:"end_year" validate:"required,min=2000,max=2100"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=500"`
}

// UpdateSessionRequest represents session update data
type UpdateSessionRequest struct {
	Label       *string `json:"label,omitempty" validate:"omitempty,min=7,max=20"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=500"`
}

// SessionFilters represents filters for listing sessions
type SessionFilters struct {
	Status     *domain.SessionStatus `json:"status,omitempty"`
	Year       *int                  `json:"year,omitempty"`
	SearchTerm *string               `json:"search_term,omitempty"`
}

// CreateSession creates a new academic session
func (s *SessionService) CreateSession(
	ctx context.Context,
	tenantID uuid.UUID,
	req *CreateSessionRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Session, error) {
	// Validate end year is after start year
	if req.EndYear <= req.StartYear {
		return nil, fmt.Errorf("end year must be after start year")
	}

	// Database constraint enforces unique labels per tenant
	// Create session
	session := &domain.Session{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Label:       req.Label,
		StartYear:   req.StartYear,
		EndYear:     req.EndYear,
		Status:      domain.SessionStatusDraft,
		Description: req.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.sessionRepo.Create(ctx, session, nil); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionSessionCreated,
		actorID,
		actorRole,
		&tenantID,
		domain.AuditResourceSession,
		session.ID,
		nil,
		session,
		ipAddress,
	)

	return session, nil
}

// GetSession gets a session by ID
func (s *SessionService) GetSession(ctx context.Context, id uuid.UUID) (*domain.Session, error) {
	session, err := s.sessionRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return session, nil
}

// UpdateSession updates a session
func (s *SessionService) UpdateSession(
	ctx context.Context,
	id uuid.UUID,
	req *UpdateSessionRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Session, error) {
	// Get existing session
	session, err := s.sessionRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Store before state for audit
	beforeState := *session

	// Update fields
	if req.Label != nil {
		session.Label = *req.Label
	}
	if req.Description != nil {
		session.Description = req.Description
	}
	session.UpdatedAt = time.Now()

	if err := s.sessionRepo.Update(ctx, session, nil); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionSessionUpdated,
		actorID,
		actorRole,
		&session.TenantID,
		domain.AuditResourceSession,
		session.ID,
		&beforeState,
		session,
		ipAddress,
	)

	return session, nil
}

// DeleteSession deletes a session
func (s *SessionService) DeleteSession(
	ctx context.Context,
	id uuid.UUID,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) error {
	// Get session for audit
	session, err := s.sessionRepo.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Cannot delete active session
	if session.IsActive() {
		return fmt.Errorf("cannot delete an active session")
	}

	// Delete session
	if err := s.sessionRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionSessionUpdated,
		actorID,
		actorRole,
		&session.TenantID,
		domain.AuditResourceSession,
		session.ID,
		session,
		nil,
		ipAddress,
	)

	return nil
}

// ListSessions lists sessions with filters and pagination
func (s *SessionService) ListSessions(
	ctx context.Context,
	tenantID uuid.UUID,
	filters *SessionFilters,
	params repository.PaginationParams,
) ([]*domain.Session, *repository.PaginatedResult, error) {
	var status *domain.SessionStatus
	if filters != nil {
		status = filters.Status
	}

	sessions, err := s.sessionRepo.ListByTenant(ctx, tenantID, status, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	// Build pagination result
	ids := make([]uuid.UUID, len(sessions))
	for i, session := range sessions {
		ids[i] = session.ID
	}
	pagination := repository.BuildPaginatedResult(ids, params.Limit)

	return sessions, &pagination, nil
}

// ActivateSession activates a session (enforces BR-007: one active per tenant)
func (s *SessionService) ActivateSession(
	ctx context.Context,
	id uuid.UUID,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Session, error) {
	// Get session
	session, err := s.sessionRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Check if there's already an active session for this tenant (BR-007)
	activeSession, err := s.sessionRepo.GetActiveSession(ctx, session.TenantID)
	if err != nil && err != repository.ErrNotFound {
		return nil, fmt.Errorf("failed to check active session: %w", err)
	}
	if activeSession != nil && activeSession.ID != session.ID {
		return nil, fmt.Errorf("another session is already active for this tenant (BR-007)")
	}

	// Store before state
	beforeState := *session

	// Activate session
	session.Activate()

	if err := s.sessionRepo.Update(ctx, session, nil); err != nil {
		return nil, fmt.Errorf("failed to activate session: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionSessionUpdated,
		actorID,
		actorRole,
		&session.TenantID,
		domain.AuditResourceSession,
		session.ID,
		&beforeState,
		session,
		ipAddress,
	)

	return session, nil
}

// ArchiveSession archives a session
func (s *SessionService) ArchiveSession(
	ctx context.Context,
	id uuid.UUID,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Session, error) {
	// Get session
	session, err := s.sessionRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Store before state
	beforeState := *session

	// Archive session
	session.Archive()

	if err := s.sessionRepo.Update(ctx, session, nil); err != nil {
		return nil, fmt.Errorf("failed to archive session: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionSessionArchived,
		actorID,
		actorRole,
		&session.TenantID,
		domain.AuditResourceSession,
		session.ID,
		&beforeState,
		session,
		ipAddress,
	)

	return session, nil
}

// GetActiveSession gets the current active session for a tenant
func (s *SessionService) GetActiveSession(ctx context.Context, tenantID uuid.UUID) (*domain.Session, error) {
	session, err := s.sessionRepo.GetActiveSession(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active session: %w", err)
	}
	return session, nil
}
