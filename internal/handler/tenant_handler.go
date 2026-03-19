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

// TenantHandler handles tenant HTTP requests
type TenantHandler struct {
	tenantService *service.TenantService
	validator     *validator.Validator
}

// NewTenantHandler creates a new tenant handler
func NewTenantHandler(tenantService *service.TenantService) *TenantHandler {
	return &TenantHandler{
		tenantService: tenantService,
		validator:     validator.New(),
	}
}

// CreateTenant godoc
// @Summary Create new tenant
// @Description Create a new tenant with principal ADMIN (SUPER_ADMIN only)
// @Tags Tenants
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.CreateTenantRequest true "Tenant creation data"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 409 {object} errors.ErrorResponse
// @Router /tenants [post]
func (h *TenantHandler) CreateTenant(c *gin.Context) {
	var req service.CreateTenantRequest

	// Get actor details from JWT
	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)

	roleValue, _ := c.Get("role")
	actorRole, _ := roleValue.(domain.Role)

	// Bind JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("invalid request body", err))
		return
	}

	// Validate request
	if err := h.validator.Validate(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.ValidationError(err))
		return
	}

	// Get IP address
	ipAddress := c.ClientIP()

	// Call service
	tenant, admin, err := h.tenantService.CreateTenant(c.Request.Context(), &req, actorID, actorRole, ipAddress)
	if err != nil {
		if err == repository.ErrDuplicateKey {
			c.JSON(http.StatusConflict, errors.Conflict("tenant or admin email already exists", err))
			return
		}
		c.JSON(http.StatusBadRequest, errors.BadRequest("failed to create tenant", err))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"tenant": tenant,
		"admin":  admin,
	})
}

// ListTenants godoc
// @Summary List tenants
// @Description List all tenants with optional filters (SUPER_ADMIN only)
// @Tags Tenants
// @Security BearerAuth
// @Produce json
// @Param status query string false "Filter by status"
// @Param school_type query string false "Filter by school type"
// @Param search query string false "Search by name"
// @Param limit query int false "Items per page" default(20)
// @Param cursor query string false "Pagination cursor"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Router /tenants [get]
func (h *TenantHandler) ListTenants(c *gin.Context) {
	// Build filters
	filters := &service.TenantFilters{}
	if statusStr := c.Query("status"); statusStr != "" {
		status := domain.TenantStatus(statusStr)
		filters.Status = &status
	}
	if typeStr := c.Query("school_type"); typeStr != "" {
		schoolType := domain.SchoolType(typeStr)
		filters.SchoolType = &schoolType
	}
	if search := c.Query("search"); search != "" {
		filters.SearchTerm = &search
	}

	// Build pagination params
	params := repository.PaginationParams{
		Limit:  20, // Default limit
		Cursor: c.Query("cursor"),
	}

	// Call service
	tenants, pagination, err := h.tenantService.ListTenants(c.Request.Context(), filters, params)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("failed to list tenants", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tenants":    tenants,
		"pagination": pagination,
	})
}

// GetTenant godoc
// @Summary Get tenant by ID
// @Description Get tenant details by ID (SUPER_ADMIN only)
// @Tags Tenants
// @Security BearerAuth
// @Produce json
// @Param id path string true "Tenant ID"
// @Success 200 {object} domain.Tenant
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /tenants/{id} [get]
func (h *TenantHandler) GetTenant(c *gin.Context) {
	// Parse tenant ID from path
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("invalid tenant ID format", err))
		return
	}

	// Call service
	tenant, err := h.tenantService.GetTenant(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.NotFound("tenant not found", err))
		return
	}

	c.JSON(http.StatusOK, tenant)
}

// UpdateTenant godoc
// @Summary Update tenant
// @Description Update tenant details (SUPER_ADMIN only)
// @Tags Tenants
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Tenant ID"
// @Param request body service.UpdateTenantRequest true "Update data"
// @Success 200 {object} domain.Tenant
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /tenants/{id} [put]
func (h *TenantHandler) UpdateTenant(c *gin.Context) {
	var req service.UpdateTenantRequest

	// Parse tenant ID from path
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("invalid tenant ID format", err))
		return
	}

	// Get actor details
	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)

	roleValue, _ := c.Get("role")
	actorRole, _ := roleValue.(domain.Role)

	// Bind JSON request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("invalid request body", err))
		return
	}

	// Validate request
	if err := h.validator.Validate(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.ValidationError(err))
		return
	}

	// Get IP address
	ipAddress := c.ClientIP()

	// Call service
	tenant, err := h.tenantService.UpdateTenant(c.Request.Context(), id, &req, actorID, actorRole, ipAddress)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("failed to update tenant", err))
		return
	}

	c.JSON(http.StatusOK, tenant)
}

// DeleteTenant godoc
// @Summary Delete tenant
// @Description Delete a tenant (cascading deletion) (SUPER_ADMIN only)
// @Tags Tenants
// @Security BearerAuth
// @Produce json
// @Param id path string true "Tenant ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /tenants/{id} [delete]
func (h *TenantHandler) DeleteTenant(c *gin.Context) {
	// Parse tenant ID from path
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("invalid tenant ID format", err))
		return
	}

	// Get actor details
	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)

	roleValue, _ := c.Get("role")
	actorRole, _ := roleValue.(domain.Role)

	// Get IP address
	ipAddress := c.ClientIP()

	// Call service
	if err := h.tenantService.DeleteTenant(c.Request.Context(), id, actorID, actorRole, ipAddress); err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("failed to delete tenant", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tenant deleted successfully"})
}

// SuspendTenant godoc
// @Summary Suspend tenant
// @Description Suspend a tenant for non-payment or policy violations (SUPER_ADMIN only)
// @Tags Tenants
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Tenant ID"
// @Param request body map[string]string true "Suspension reason"
// @Success 200 {object} domain.Tenant
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /tenants/{id}/suspend [post]
func (h *TenantHandler) SuspendTenant(c *gin.Context) {
	// Parse tenant ID from path
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("invalid tenant ID format", err))
		return
	}

	// Get reason from request body
	var req struct {
		Reason string `json:"reason" validate:"required,min=10,max=500"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("invalid request body", err))
		return
	}

	if err := h.validator.Validate(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.ValidationError(err))
		return
	}

	// Get actor details
	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)

	roleValue, _ := c.Get("role")
	actorRole, _ := roleValue.(domain.Role)

	// Get IP address
	ipAddress := c.ClientIP()

	// Call service
	tenant, err := h.tenantService.SuspendTenant(c.Request.Context(), id, req.Reason, actorID, actorRole, ipAddress)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("failed to suspend tenant", err))
		return
	}

	c.JSON(http.StatusOK, tenant)
}

// ReactivateTenant godoc
// @Summary Reactivate tenant
// @Description Reactivate a suspended tenant (SUPER_ADMIN only)
// @Tags Tenants
// @Security BearerAuth
// @Produce json
// @Param id path string true "Tenant ID"
// @Success 200 {object} domain.Tenant
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /tenants/{id}/reactivate [post]
func (h *TenantHandler) ReactivateTenant(c *gin.Context) {
	// Parse tenant ID from path
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("invalid tenant ID format", err))
		return
	}

	// Get actor details
	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)

	roleValue, _ := c.Get("role")
	actorRole, _ := roleValue.(domain.Role)

	// Get IP address
	ipAddress := c.ClientIP()

	// Call service
	tenant, err := h.tenantService.ReactivateTenant(c.Request.Context(), id, actorID, actorRole, ipAddress)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("failed to reactivate tenant", err))
		return
	}

	c.JSON(http.StatusOK, tenant)
}

// GetTenantConfiguration godoc
// @Summary Get tenant configuration
// @Description Get tenant configuration (ADMIN can access own tenant, SUPER_ADMIN can access any)
// @Tags Tenants
// @Security BearerAuth
// @Produce json
// @Param id path string true "Tenant ID"
// @Success 200 {object} domain.TenantConfiguration
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /tenants/{id}/configuration [get]
func (h *TenantHandler) GetTenantConfiguration(c *gin.Context) {
	// Parse tenant ID from path
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("invalid tenant ID format", err))
		return
	}

	// Call service
	config, err := h.tenantService.GetTenantConfiguration(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.NotFound("tenant not found", err))
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateTenantConfiguration godoc
// @Summary Update tenant configuration
// @Description Update tenant configuration (ADMIN for own tenant, SUPER_ADMIN for any)
// @Tags Tenants
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Tenant ID"
// @Param request body domain.TenantConfiguration true "Configuration data"
// @Success 200 {object} domain.TenantConfiguration
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /tenants/{id}/configuration [put]
func (h *TenantHandler) UpdateTenantConfiguration(c *gin.Context) {
	var config domain.TenantConfiguration

	// Parse tenant ID from path
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("invalid tenant ID format", err))
		return
	}

	// Get actor details
	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)

	roleValue, _ := c.Get("role")
	actorRole, _ := roleValue.(domain.Role)

	// Bind JSON request
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("invalid request body", err))
		return
	}

	// Validate request
	if err := h.validator.Validate(&config); err != nil {
		c.JSON(http.StatusBadRequest, errors.ValidationError(err))
		return
	}

	// Get IP address
	ipAddress := c.ClientIP()

	// Call service
	updatedConfig, err := h.tenantService.UpdateTenantConfiguration(c.Request.Context(), id, &config, actorID, actorRole, ipAddress)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("failed to update configuration", err))
		return
	}

	c.JSON(http.StatusOK, updatedConfig)
}
