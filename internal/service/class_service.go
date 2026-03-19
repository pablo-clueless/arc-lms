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

// ClassService handles class management operations
type ClassService struct {
	classRepo    *postgres.ClassRepository
	sessionRepo  *postgres.SessionRepository
	auditService *AuditService
}

// NewClassService creates a new class service
func NewClassService(
	classRepo *postgres.ClassRepository,
	sessionRepo *postgres.SessionRepository,
	auditService *AuditService,
) *ClassService {
	return &ClassService{
		classRepo:    classRepo,
		sessionRepo:  sessionRepo,
		auditService: auditService,
	}
}

// CreateClassRequest represents class creation data
type CreateClassRequest struct {
	SessionID   uuid.UUID         `json:"session_id" validate:"required,uuid"`
	Name        string            `json:"name" validate:"required,min=1,max=100"`
	Arm         *string           `json:"arm,omitempty" validate:"omitempty,max=50"`
	SchoolLevel domain.SchoolType `json:"school_level" validate:"required,oneof=PRIMARY SECONDARY"`
	Capacity    *int              `json:"capacity,omitempty" validate:"omitempty,min=1,max=200"`
	Description *string           `json:"description,omitempty" validate:"omitempty,max=500"`
}

// UpdateClassRequest represents class update data
type UpdateClassRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Arm         *string `json:"arm,omitempty" validate:"omitempty,max=50"`
	Capacity    *int    `json:"capacity,omitempty" validate:"omitempty,min=1,max=200"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=500"`
}

// CreateClass creates a new class
func (s *ClassService) CreateClass(
	ctx context.Context,
	tenantID uuid.UUID,
	req *CreateClassRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Class, error) {
	// Verify session exists and belongs to tenant
	session, err := s.sessionRepo.Get(ctx, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if session.TenantID != tenantID {
		return nil, fmt.Errorf("session does not belong to this tenant")
	}

	// Create class
	arm := ""
	if req.Arm != nil {
		arm = *req.Arm
	}

	class := &domain.Class{
		ID:          uuid.New(),
		TenantID:    tenantID,
		SessionID:   req.SessionID,
		Name:        req.Name,
		Arm:         arm,
		Level:       domain.SchoolLevel(req.SchoolLevel),
		Status:      domain.ClassStatusActive,
		Capacity:    req.Capacity,
		Description: req.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.classRepo.Create(ctx, class, nil); err != nil {
		return nil, fmt.Errorf("failed to create class: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionClassCreated,
		actorID,
		actorRole,
		&tenantID,
		domain.AuditResourceClass,
		class.ID,
		nil,
		class,
		ipAddress,
	)

	return class, nil
}

// GetClass gets a class by ID
func (s *ClassService) GetClass(ctx context.Context, id uuid.UUID) (*domain.Class, error) {
	class, err := s.classRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get class: %w", err)
	}
	return class, nil
}

// UpdateClass updates a class
func (s *ClassService) UpdateClass(
	ctx context.Context,
	id uuid.UUID,
	req *UpdateClassRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Class, error) {
	// Get existing class
	class, err := s.classRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get class: %w", err)
	}

	// Store before state for audit
	beforeState := *class

	// Update fields
	if req.Name != nil {
		class.Name = *req.Name
	}
	if req.Arm != nil {
		class.Arm = *req.Arm
	}
	if req.Capacity != nil {
		class.Capacity = req.Capacity
	}
	if req.Description != nil {
		class.Description = req.Description
	}
	class.UpdatedAt = time.Now()

	if err := s.classRepo.Update(ctx, class, nil); err != nil {
		return nil, fmt.Errorf("failed to update class: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionClassUpdated,
		actorID,
		actorRole,
		&class.TenantID,
		domain.AuditResourceClass,
		class.ID,
		&beforeState,
		class,
		ipAddress,
	)

	return class, nil
}

// DeleteClass deletes a class
func (s *ClassService) DeleteClass(
	ctx context.Context,
	id uuid.UUID,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) error {
	// Get class for audit
	class, err := s.classRepo.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get class: %w", err)
	}

	// Delete class
	if err := s.classRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete class: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionClassDeleted,
		actorID,
		actorRole,
		&class.TenantID,
		domain.AuditResourceClass,
		class.ID,
		class,
		nil,
		ipAddress,
	)

	return nil
}

// ListClasses lists classes with filters and pagination
func (s *ClassService) ListClasses(
	ctx context.Context,
	tenantID uuid.UUID,
	sessionID *uuid.UUID,
	params repository.PaginationParams,
) ([]*domain.Class, *repository.PaginatedResult, error) {
	var classes []*domain.Class
	var err error

	if sessionID != nil {
		classes, err = s.classRepo.ListBySession(ctx, *sessionID, params)
	} else {
		classes, err = s.classRepo.ListByTenant(ctx, tenantID, params)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to list classes: %w", err)
	}

	// Build pagination result
	ids := make([]uuid.UUID, len(classes))
	for i, class := range classes {
		ids[i] = class.ID
	}
	pagination := repository.BuildPaginatedResult(ids, params.Limit)

	return classes, &pagination, nil
}
