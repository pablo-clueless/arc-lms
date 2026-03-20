package handler

import (
	"net/http"

	"arc-lms/internal/domain"
	"arc-lms/internal/pkg/errors"
	"arc-lms/internal/pkg/validator"
	"arc-lms/internal/repository"
	"arc-lms/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SystemConfigHandler handles system config HTTP requests
type SystemConfigHandler struct {
	configService *service.SystemConfigService
}

// NewSystemConfigHandler creates a new system config handler
func NewSystemConfigHandler(configService *service.SystemConfigService) *SystemConfigHandler {
	return &SystemConfigHandler{
		configService: configService,
	}
}

// CreateSystemConfig godoc
// @Summary Create a new system config
// @Description Create a new platform system configuration (SUPER_ADMIN only)
// @Tags SystemConfig
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.CreateSystemConfigRequest true "System config data"
// @Success 201 {object} domain.SystemConfig
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 409 {object} errors.ErrorResponse
// @Router /system-configs [post]
func (h *SystemConfigHandler) CreateSystemConfig(c *gin.Context) {
	var req service.CreateSystemConfigRequest

	// Get actor details from JWT
	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)

	actorRole, ok := GetRoleFromContext(c)
	if !ok {
		errors.Unauthorized(c, "invalid role in token")
		return
	}

	// Only SUPER_ADMIN can access
	if actorRole != domain.RoleSuperAdmin {
		errors.Forbidden(c, "only SUPER_ADMIN can manage system configs")
		return
	}

	if !validator.BindAndValidate(c, &req) {
		return
	}

	ipAddress := c.ClientIP()

	config, err := h.configService.CreateSystemConfig(c.Request.Context(), &req, actorID, actorRole, ipAddress)
	if err != nil {
		if err == repository.ErrDuplicateKey {
			errors.Conflict(c, "CONFLICT", "config with this key already exists", nil)
			return
		}
		errors.BadRequest(c, "failed to create system config", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, config)
}

// ListSystemConfigs godoc
// @Summary List system configs
// @Description List all platform system configurations (SUPER_ADMIN only)
// @Tags SystemConfig
// @Security BearerAuth
// @Produce json
// @Param category query string false "Filter by category"
// @Param limit query int false "Items per page" default(50)
// @Param mask_sensitive query bool false "Mask sensitive values" default(true)
// @Success 200 {object} map[string]interface{}
// @Failure 403 {object} errors.ErrorResponse
// @Router /system-configs [get]
func (h *SystemConfigHandler) ListSystemConfigs(c *gin.Context) {
	actorRole, ok := GetRoleFromContext(c)
	if !ok {
		errors.Unauthorized(c, "invalid role in token")
		return
	}

	// Only SUPER_ADMIN can access
	if actorRole != domain.RoleSuperAdmin {
		errors.Forbidden(c, "only SUPER_ADMIN can view system configs")
		return
	}

	// Build filters
	filters := &service.SystemConfigFilters{}
	if categoryStr := c.Query("category"); categoryStr != "" {
		category := domain.SystemConfigCategory(categoryStr)
		filters.Category = &category
	}

	// Build pagination params
	params := repository.PaginationParams{
		Limit:     50,
		SortOrder: "ASC",
	}

	// Check if sensitive values should be masked
	maskSensitive := c.DefaultQuery("mask_sensitive", "true") == "true"

	configs, pagination, err := h.configService.ListSystemConfigs(c.Request.Context(), filters, params, actorRole, maskSensitive)
	if err != nil {
		errors.BadRequest(c, "failed to list system configs", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"configs":    configs,
		"pagination": pagination,
	})
}

// GetSystemConfig godoc
// @Summary Get system config by ID
// @Description Get a specific system configuration by ID (SUPER_ADMIN only)
// @Tags SystemConfig
// @Security BearerAuth
// @Produce json
// @Param id path string true "Config ID"
// @Success 200 {object} domain.SystemConfig
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /system-configs/{id} [get]
func (h *SystemConfigHandler) GetSystemConfig(c *gin.Context) {
	actorRole, ok := GetRoleFromContext(c)
	if !ok {
		errors.Unauthorized(c, "invalid role in token")
		return
	}

	// Only SUPER_ADMIN can access
	if actorRole != domain.RoleSuperAdmin {
		errors.Forbidden(c, "only SUPER_ADMIN can view system configs")
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		errors.BadRequest(c, "invalid config ID format", nil)
		return
	}

	config, err := h.configService.GetSystemConfig(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			errors.NotFound(c, "system config not found")
			return
		}
		errors.BadRequest(c, "failed to get system config", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// GetSystemConfigByKey godoc
// @Summary Get system config by key
// @Description Get a specific system configuration by key (SUPER_ADMIN only)
// @Tags SystemConfig
// @Security BearerAuth
// @Produce json
// @Param key path string true "Config key"
// @Success 200 {object} domain.SystemConfig
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /system-configs/key/{key} [get]
func (h *SystemConfigHandler) GetSystemConfigByKey(c *gin.Context) {
	actorRole, ok := GetRoleFromContext(c)
	if !ok {
		errors.Unauthorized(c, "invalid role in token")
		return
	}

	// Only SUPER_ADMIN can access
	if actorRole != domain.RoleSuperAdmin {
		errors.Forbidden(c, "only SUPER_ADMIN can view system configs")
		return
	}

	key := c.Param("key")
	if key == "" {
		errors.BadRequest(c, "config key is required", nil)
		return
	}

	config, err := h.configService.GetSystemConfigByKey(c.Request.Context(), key)
	if err != nil {
		if err == repository.ErrNotFound {
			errors.NotFound(c, "system config not found")
			return
		}
		errors.BadRequest(c, "failed to get system config", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateSystemConfig godoc
// @Summary Update system config
// @Description Update a system configuration (SUPER_ADMIN only)
// @Tags SystemConfig
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Config ID"
// @Param request body service.UpdateSystemConfigRequest true "Update data"
// @Success 200 {object} domain.SystemConfig
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /system-configs/{id} [put]
func (h *SystemConfigHandler) UpdateSystemConfig(c *gin.Context) {
	var req service.UpdateSystemConfigRequest

	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)

	actorRole, ok := GetRoleFromContext(c)
	if !ok {
		errors.Unauthorized(c, "invalid role in token")
		return
	}

	// Only SUPER_ADMIN can access
	if actorRole != domain.RoleSuperAdmin {
		errors.Forbidden(c, "only SUPER_ADMIN can update system configs")
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		errors.BadRequest(c, "invalid config ID format", nil)
		return
	}

	if !validator.BindAndValidate(c, &req) {
		return
	}

	ipAddress := c.ClientIP()

	config, err := h.configService.UpdateSystemConfig(c.Request.Context(), id, &req, actorID, actorRole, ipAddress)
	if err != nil {
		if err == repository.ErrNotFound {
			errors.NotFound(c, "system config not found")
			return
		}
		errors.BadRequest(c, "failed to update system config", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// DeleteSystemConfig godoc
// @Summary Delete system config
// @Description Delete a system configuration (SUPER_ADMIN only)
// @Tags SystemConfig
// @Security BearerAuth
// @Produce json
// @Param id path string true "Config ID"
// @Success 200 {object} map[string]string
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /system-configs/{id} [delete]
func (h *SystemConfigHandler) DeleteSystemConfig(c *gin.Context) {
	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)

	actorRole, ok := GetRoleFromContext(c)
	if !ok {
		errors.Unauthorized(c, "invalid role in token")
		return
	}

	// Only SUPER_ADMIN can access
	if actorRole != domain.RoleSuperAdmin {
		errors.Forbidden(c, "only SUPER_ADMIN can delete system configs")
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		errors.BadRequest(c, "invalid config ID format", nil)
		return
	}

	ipAddress := c.ClientIP()

	if err := h.configService.DeleteSystemConfig(c.Request.Context(), id, actorID, actorRole, ipAddress); err != nil {
		if err == repository.ErrNotFound {
			errors.NotFound(c, "system config not found")
			return
		}
		errors.BadRequest(c, "failed to delete system config", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "system config deleted successfully"})
}

// ListSystemConfigsByCategory godoc
// @Summary List configs by category
// @Description List all system configurations for a specific category (SUPER_ADMIN only)
// @Tags SystemConfig
// @Security BearerAuth
// @Produce json
// @Param category path string true "Config category"
// @Param mask_sensitive query bool false "Mask sensitive values" default(true)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Router /system-configs/category/{category} [get]
func (h *SystemConfigHandler) ListSystemConfigsByCategory(c *gin.Context) {
	actorRole, ok := GetRoleFromContext(c)
	if !ok {
		errors.Unauthorized(c, "invalid role in token")
		return
	}

	// Only SUPER_ADMIN can access
	if actorRole != domain.RoleSuperAdmin {
		errors.Forbidden(c, "only SUPER_ADMIN can view system configs")
		return
	}

	categoryStr := c.Param("category")
	category := domain.SystemConfigCategory(categoryStr)

	if !domain.IsValidCategory(category) {
		errors.BadRequest(c, "invalid category", map[string]interface{}{
			"valid_categories": domain.ValidCategories(),
		})
		return
	}

	maskSensitive := c.DefaultQuery("mask_sensitive", "true") == "true"

	configs, err := h.configService.ListSystemConfigsByCategory(c.Request.Context(), category, actorRole, maskSensitive)
	if err != nil {
		errors.BadRequest(c, "failed to list configs by category", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"category": category,
		"configs":  configs,
	})
}

// BulkUpdateSystemConfigs godoc
// @Summary Bulk update configs
// @Description Update multiple system configurations at once (SUPER_ADMIN only)
// @Tags SystemConfig
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body []service.BulkUpdateConfigRequest true "Bulk update data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Router /system-configs/bulk [put]
func (h *SystemConfigHandler) BulkUpdateSystemConfigs(c *gin.Context) {
	var req []service.BulkUpdateConfigRequest

	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)

	actorRole, ok := GetRoleFromContext(c)
	if !ok {
		errors.Unauthorized(c, "invalid role in token")
		return
	}

	// Only SUPER_ADMIN can access
	if actorRole != domain.RoleSuperAdmin {
		errors.Forbidden(c, "only SUPER_ADMIN can update system configs")
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		errors.BadRequest(c, "invalid request body", map[string]interface{}{"error": err.Error()})
		return
	}

	if len(req) == 0 {
		errors.BadRequest(c, "at least one config update is required", nil)
		return
	}

	if len(req) > 50 {
		errors.BadRequest(c, "maximum 50 configs can be updated at once", nil)
		return
	}

	ipAddress := c.ClientIP()

	if err := h.configService.BulkUpdateSystemConfigs(c.Request.Context(), req, actorID, actorRole, ipAddress); err != nil {
		errors.BadRequest(c, "failed to bulk update configs", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "configs updated successfully",
		"count":   len(req),
	})
}

// GetCategories godoc
// @Summary Get all config categories
// @Description Get list of all available config categories (SUPER_ADMIN only)
// @Tags SystemConfig
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 403 {object} errors.ErrorResponse
// @Router /system-configs/categories [get]
func (h *SystemConfigHandler) GetCategories(c *gin.Context) {
	actorRole, ok := GetRoleFromContext(c)
	if !ok {
		errors.Unauthorized(c, "invalid role in token")
		return
	}

	// Only SUPER_ADMIN can access
	if actorRole != domain.RoleSuperAdmin {
		errors.Forbidden(c, "only SUPER_ADMIN can view categories")
		return
	}

	categories, err := h.configService.GetCategories(c.Request.Context(), actorRole)
	if err != nil {
		errors.BadRequest(c, "failed to get categories", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"categories": categories,
	})
}
