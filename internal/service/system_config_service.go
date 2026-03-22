package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"arc-lms/internal/domain"
	"arc-lms/internal/repository"
	"arc-lms/internal/repository/postgres"
)

// SystemConfigService handles system config operations
type SystemConfigService struct {
	configRepo   *postgres.SystemConfigRepository
	auditService *AuditService
}

// NewSystemConfigService creates a new system config service
func NewSystemConfigService(
	configRepo *postgres.SystemConfigRepository,
	auditService *AuditService,
) *SystemConfigService {
	return &SystemConfigService{
		configRepo:   configRepo,
		auditService: auditService,
	}
}

// CreateSystemConfigRequest represents data for creating a new system config
type CreateSystemConfigRequest struct {
	Key         string                      `json:"key" validate:"required,min=1,max=100"`
	Value       json.RawMessage             `json:"value" validate:"required"`
	Description *string                     `json:"description,omitempty" validate:"omitempty,max=500"`
	Category    domain.SystemConfigCategory `json:"category" validate:"required"`
	IsSensitive bool                        `json:"is_sensitive"`
}

// UpdateSystemConfigRequest represents data for updating a system config
type UpdateSystemConfigRequest struct {
	Value       json.RawMessage              `json:"value,omitempty"`
	Description *string                      `json:"description,omitempty" validate:"omitempty,max=500"`
	Category    *domain.SystemConfigCategory `json:"category,omitempty"`
	IsSensitive *bool                        `json:"is_sensitive,omitempty"`
}

// BulkUpdateConfigRequest represents a single config update in a bulk operation
type BulkUpdateConfigRequest struct {
	Key   string          `json:"key" validate:"required"`
	Value json.RawMessage `json:"value" validate:"required"`
}

// SystemConfigFilters represents filters for listing configs
type SystemConfigFilters struct {
	Category *domain.SystemConfigCategory `json:"category,omitempty"`
}

// CreateSystemConfig creates a new system config (SUPER_ADMIN only)
func (s *SystemConfigService) CreateSystemConfig(
	ctx context.Context,
	req *CreateSystemConfigRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.SystemConfig, error) {
	// Verify actor is SUPER_ADMIN
	if actorRole != domain.RoleSuperAdmin {
		return nil, fmt.Errorf("only SUPER_ADMIN can create system configs")
	}

	// Validate category
	if !domain.IsValidCategory(req.Category) {
		return nil, fmt.Errorf("invalid category: %s", req.Category)
	}

	// Check if key already exists
	existing, err := s.configRepo.GetByKey(ctx, req.Key)
	if err != nil && err != repository.ErrNotFound {
		return nil, fmt.Errorf("failed to check existing config: %w", err)
	}
	if existing != nil {
		return nil, repository.ErrDuplicateKey
	}

	config := &domain.SystemConfig{
		ID:          uuid.New(),
		Key:         req.Key,
		Value:       req.Value,
		Description: req.Description,
		Category:    req.Category,
		IsSensitive: req.IsSensitive,
		CreatedBy:   &actorID,
		UpdatedBy:   &actorID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.configRepo.Create(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to create system config: %w", err)
	}

	// Log audit
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionSystemConfigCreated,
		actorID,
		actorRole,
		nil, // No tenant for system configs
		domain.AuditResourceSystemConfig,
		config.ID,
		nil,
		config,
		ipAddress,
	)

	return config, nil
}

// GetSystemConfig retrieves a system config by ID
func (s *SystemConfigService) GetSystemConfig(ctx context.Context, id uuid.UUID) (*domain.SystemConfig, error) {
	return s.configRepo.GetByID(ctx, id)
}

// GetSystemConfigByKey retrieves a system config by key
func (s *SystemConfigService) GetSystemConfigByKey(ctx context.Context, key string) (*domain.SystemConfig, error) {
	return s.configRepo.GetByKey(ctx, key)
}

// UpdateSystemConfig updates an existing system config (SUPER_ADMIN only)
func (s *SystemConfigService) UpdateSystemConfig(
	ctx context.Context,
	id uuid.UUID,
	req *UpdateSystemConfigRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.SystemConfig, error) {
	// Verify actor is SUPER_ADMIN
	if actorRole != domain.RoleSuperAdmin {
		return nil, fmt.Errorf("only SUPER_ADMIN can update system configs")
	}

	// Get existing config
	config, err := s.configRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Store old state for audit
	oldConfig := *config

	// Update fields
	if req.Value != nil {
		config.Value = req.Value
	}
	if req.Description != nil {
		config.Description = req.Description
	}
	if req.Category != nil {
		if !domain.IsValidCategory(*req.Category) {
			return nil, fmt.Errorf("invalid category: %s", *req.Category)
		}
		config.Category = *req.Category
	}
	if req.IsSensitive != nil {
		config.IsSensitive = *req.IsSensitive
	}

	config.UpdatedBy = &actorID
	config.UpdatedAt = time.Now()

	if err := s.configRepo.Update(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to update system config: %w", err)
	}

	// Log audit
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionSystemConfigUpdated,
		actorID,
		actorRole,
		nil, // No tenant for system configs
		domain.AuditResourceSystemConfig,
		config.ID,
		&oldConfig,
		config,
		ipAddress,
	)

	return config, nil
}

// DeleteSystemConfig deletes a system config (SUPER_ADMIN only)
func (s *SystemConfigService) DeleteSystemConfig(
	ctx context.Context,
	id uuid.UUID,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) error {
	// Verify actor is SUPER_ADMIN
	if actorRole != domain.RoleSuperAdmin {
		return fmt.Errorf("only SUPER_ADMIN can delete system configs")
	}

	// Get config for audit logging
	config, err := s.configRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.configRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete system config: %w", err)
	}

	// Log audit
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionSystemConfigDeleted,
		actorID,
		actorRole,
		nil, // No tenant for system configs
		domain.AuditResourceSystemConfig,
		id,
		config,
		nil,
		ipAddress,
	)

	return nil
}

// ListSystemConfigs lists system configs with optional filters (SUPER_ADMIN only)
func (s *SystemConfigService) ListSystemConfigs(
	ctx context.Context,
	filters *SystemConfigFilters,
	params repository.PaginationParams,
	actorRole domain.Role,
	maskSensitive bool,
) ([]*domain.SystemConfig, *repository.PaginatedResult, error) {
	// Verify actor is SUPER_ADMIN
	if actorRole != domain.RoleSuperAdmin {
		return nil, nil, fmt.Errorf("only SUPER_ADMIN can list system configs")
	}

	var category *domain.SystemConfigCategory
	if filters != nil {
		category = filters.Category
	}

	configs, total, err := s.configRepo.List(ctx, category, params)
	if err != nil {
		return nil, nil, err
	}

	// Mask sensitive values if requested
	if maskSensitive {
		for _, config := range configs {
			if config.IsSensitive {
				config.Value = config.MaskedValue()
			}
		}
	}

	pagination := repository.BuildPaginatedResult(total, params)
	return configs, &pagination, nil
}

// ListSystemConfigsByCategory lists configs for a specific category
func (s *SystemConfigService) ListSystemConfigsByCategory(
	ctx context.Context,
	category domain.SystemConfigCategory,
	actorRole domain.Role,
	maskSensitive bool,
) ([]*domain.SystemConfig, error) {
	// Verify actor is SUPER_ADMIN
	if actorRole != domain.RoleSuperAdmin {
		return nil, fmt.Errorf("only SUPER_ADMIN can list system configs")
	}

	if !domain.IsValidCategory(category) {
		return nil, fmt.Errorf("invalid category: %s", category)
	}

	configs, err := s.configRepo.ListByCategory(ctx, category)
	if err != nil {
		return nil, err
	}

	// Mask sensitive values if requested
	if maskSensitive {
		for _, config := range configs {
			if config.IsSensitive {
				config.Value = config.MaskedValue()
			}
		}
	}

	return configs, nil
}

// BulkUpdateSystemConfigs updates multiple configs at once (SUPER_ADMIN only)
func (s *SystemConfigService) BulkUpdateSystemConfigs(
	ctx context.Context,
	updates []BulkUpdateConfigRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) error {
	// Verify actor is SUPER_ADMIN
	if actorRole != domain.RoleSuperAdmin {
		return fmt.Errorf("only SUPER_ADMIN can update system configs")
	}

	configs := make([]*domain.SystemConfig, len(updates))
	now := time.Now()

	for i, update := range updates {
		configs[i] = &domain.SystemConfig{
			Key:       update.Key,
			Value:     update.Value,
			UpdatedBy: &actorID,
			UpdatedAt: now,
		}
	}

	if err := s.configRepo.BulkUpdate(ctx, configs); err != nil {
		return fmt.Errorf("failed to bulk update configs: %w", err)
	}

	// Log audit
	keys := make([]string, len(updates))
	for i, u := range updates {
		keys[i] = u.Key
	}

	_ = s.auditService.LogActionWithMetadata(
		ctx,
		domain.AuditActionSystemConfigBulkUpdated,
		actorID,
		actorRole,
		nil, // No tenant for system configs
		domain.AuditResourceSystemConfig,
		uuid.Nil,
		nil,
		nil,
		ipAddress,
		map[string]interface{}{
			"bulk_update": true,
			"keys":        keys,
			"count":       len(updates),
		},
		"",
	)

	return nil
}

// GetCategories returns all available config categories
func (s *SystemConfigService) GetCategories(ctx context.Context, actorRole domain.Role) ([]domain.SystemConfigCategory, error) {
	// Verify actor is SUPER_ADMIN
	if actorRole != domain.RoleSuperAdmin {
		return nil, fmt.Errorf("only SUPER_ADMIN can list categories")
	}

	return s.configRepo.GetCategories(ctx)
}

// GetConfigValue is a helper to get a config value by key (for internal use)
func (s *SystemConfigService) GetConfigValue(ctx context.Context, key string) (*domain.SystemConfig, error) {
	return s.configRepo.GetByKey(ctx, key)
}

// IsMaintenanceMode checks if maintenance mode is enabled
func (s *SystemConfigService) IsMaintenanceMode(ctx context.Context) (bool, string, error) {
	enabledConfig, err := s.configRepo.GetByKey(ctx, "maintenance.enabled")
	if err != nil {
		if err == repository.ErrNotFound {
			return false, "", nil
		}
		return false, "", err
	}

	if !enabledConfig.GetBool() {
		return false, "", nil
	}

	messageConfig, err := s.configRepo.GetByKey(ctx, "maintenance.message")
	if err != nil && err != repository.ErrNotFound {
		return true, "System is under maintenance", nil
	}

	return true, messageConfig.GetString(), nil
}
