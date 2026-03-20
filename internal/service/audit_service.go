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

// AuditService handles audit logging operations
type AuditService struct {
	auditRepo *postgres.AuditRepository
}

// NewAuditService creates a new audit service
func NewAuditService(auditRepo *postgres.AuditRepository) *AuditService {
	return &AuditService{
		auditRepo: auditRepo,
	}
}


// LogAction creates an audit log entry
func (s *AuditService) LogAction(
	ctx context.Context,
	action domain.AuditAction,
	actorID uuid.UUID,
	actorRole domain.Role,
	tenantID *uuid.UUID,
	resourceType domain.AuditResourceType,
	resourceID uuid.UUID,
	beforeState interface{},
	afterState interface{},
	ipAddress string,
) error {
	// Create audit log
	auditLog := domain.NewAuditLog(
		tenantID,
		actorID,
		actorRole,
		action,
		resourceType,
		resourceID,
		ipAddress,
	)

	// Add before and after states if provided
	if beforeState != nil {
		auditLog.WithBeforeState(beforeState)
	}
	if afterState != nil {
		auditLog.WithAfterState(afterState)
	}

	// Calculate field-level changes if both states are provided
	if beforeState != nil && afterState != nil {
		changes := calculateChanges(beforeState, afterState)
		if len(changes) > 0 {
			auditLog.WithChanges(changes)
		}
	}

	// Create audit log entry
	if err := s.auditRepo.Create(ctx, auditLog); err != nil {
		// Log error but don't fail the operation
		// Audit logging should not break business operations
		fmt.Printf("failed to create audit log: %v\n", err)
		return err
	}

	return nil
}

// LogActionWithMetadata creates an audit log entry with additional metadata
func (s *AuditService) LogActionWithMetadata(
	ctx context.Context,
	action domain.AuditAction,
	actorID uuid.UUID,
	actorRole domain.Role,
	tenantID *uuid.UUID,
	resourceType domain.AuditResourceType,
	resourceID uuid.UUID,
	beforeState interface{},
	afterState interface{},
	ipAddress string,
	metadata map[string]interface{},
	userAgent string,
) error {
	// Create audit log
	auditLog := domain.NewAuditLog(
		tenantID,
		actorID,
		actorRole,
		action,
		resourceType,
		resourceID,
		ipAddress,
	)

	// Add states
	if beforeState != nil {
		auditLog.WithBeforeState(beforeState)
	}
	if afterState != nil {
		auditLog.WithAfterState(afterState)
	}

	// Add metadata and user agent
	if len(metadata) > 0 {
		auditLog.WithMetadata(metadata)
	}
	if userAgent != "" {
		auditLog.WithUserAgent(userAgent)
	}

	// Calculate changes
	if beforeState != nil && afterState != nil {
		changes := calculateChanges(beforeState, afterState)
		if len(changes) > 0 {
			auditLog.WithChanges(changes)
		}
	}

	// Create audit log entry
	if err := s.auditRepo.Create(ctx, auditLog); err != nil {
		fmt.Printf("failed to create audit log: %v\n", err)
		return err
	}

	return nil
}

// GetAuditLogs queries audit logs with filters and pagination
func (s *AuditService) GetAuditLogs(
	ctx context.Context,
	filters *domain.AuditFilters,
	params repository.PaginationParams,
) ([]*domain.AuditLog, *repository.PaginatedResult, error) {
	logs, pagination, err := s.auditRepo.List(ctx, filters, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get audit logs: %w", err)
	}
	return logs, pagination, nil
}

// GetAuditLog gets a specific audit log by ID
func (s *AuditService) GetAuditLog(ctx context.Context, id uuid.UUID) (*domain.AuditLog, error) {
	log, err := s.auditRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log: %w", err)
	}
	return log, nil
}

// GetResourceAuditTrail gets all audit logs for a specific resource
func (s *AuditService) GetResourceAuditTrail(
	ctx context.Context,
	resourceType domain.AuditResourceType,
	resourceID uuid.UUID,
	params repository.PaginationParams,
) ([]*domain.AuditLog, *repository.PaginatedResult, error) {
	logs, pagination, err := s.auditRepo.GetByResource(ctx, resourceType, resourceID, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get resource audit trail: %w", err)
	}
	return logs, pagination, nil
}

// GetTenantAuditLogs gets all audit logs for a specific tenant
func (s *AuditService) GetTenantAuditLogs(
	ctx context.Context,
	tenantID uuid.UUID,
	filters *domain.AuditFilters,
	params repository.PaginationParams,
) ([]*domain.AuditLog, *repository.PaginatedResult, error) {
	// Override tenant ID in filters
	if filters == nil {
		filters = &domain.AuditFilters{}
	}
	filters.TenantID = &tenantID

	logs, pagination, err := s.auditRepo.List(ctx, filters, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get tenant audit logs: %w", err)
	}
	return logs, pagination, nil
}

// GetUserAuditLogs gets all audit logs for actions performed by a specific user
func (s *AuditService) GetUserAuditLogs(
	ctx context.Context,
	userID uuid.UUID,
	params repository.PaginationParams,
) ([]*domain.AuditLog, *repository.PaginatedResult, error) {
	filters := &domain.AuditFilters{
		ActorUserID: &userID,
	}

	logs, pagination, err := s.auditRepo.List(ctx, filters, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user audit logs: %w", err)
	}
	return logs, pagination, nil
}

// GetSensitiveAuditLogs gets all sensitive audit logs (SUPER_ADMIN only)
func (s *AuditService) GetSensitiveAuditLogs(
	ctx context.Context,
	filters *domain.AuditFilters,
	params repository.PaginationParams,
) ([]*domain.AuditLog, *repository.PaginatedResult, error) {
	// Override is_sensitive in filters
	if filters == nil {
		filters = &domain.AuditFilters{}
	}
	isSensitive := true
	filters.IsSensitive = &isSensitive

	logs, pagination, err := s.auditRepo.List(ctx, filters, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get sensitive audit logs: %w", err)
	}
	return logs, pagination, nil
}

// GetAuditLogsByDateRange gets audit logs within a specific date range
func (s *AuditService) GetAuditLogsByDateRange(
	ctx context.Context,
	startDate, endDate time.Time,
	filters *domain.AuditFilters,
	params repository.PaginationParams,
) ([]*domain.AuditLog, *repository.PaginatedResult, error) {
	// Override date range in filters
	if filters == nil {
		filters = &domain.AuditFilters{}
	}
	filters.StartDate = &startDate
	filters.EndDate = &endDate

	logs, pagination, err := s.auditRepo.List(ctx, filters, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get audit logs by date range: %w", err)
	}
	return logs, pagination, nil
}

// calculateChanges compares before and after states and returns field-level changes
// This is a simplified implementation - in production, you'd use reflection or a library
func calculateChanges(before, after interface{}) map[string]interface{} {
	changes := make(map[string]interface{})

	// This is a placeholder implementation
	// In a real implementation, you would:
	// 1. Use reflection to compare struct fields
	// 2. Or use a library like go-cmp or reflect.DeepEqual
	// 3. Return only the fields that changed with their new values

	// For now, we'll return an empty map
	// The before/after states in the audit log provide full context

	return changes
}
