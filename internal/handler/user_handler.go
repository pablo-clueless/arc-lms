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

// UserHandler handles user HTTP requests
type UserHandler struct {
	userService *service.UserService
	validator   *validator.Validator
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
		validator:   validator.New(),
	}
}

// GetMe godoc
// @Summary Get current user profile
// @Description Get authenticated user's profile
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} domain.User
// @Failure 401 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /users/me [get]
func (h *UserHandler) GetMe(c *gin.Context) {
	// Get user ID from JWT claims (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, errors.Unauthorized("user not authenticated", nil))
		return
	}

	id, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, errors.BadRequest("invalid user ID format", nil))
		return
	}

	// Get user
	user, err := h.userService.GetUser(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.NotFound("user not found", err))
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateMe godoc
// @Summary Update current user profile
// @Description Update authenticated user's own profile
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.UpdateUserRequest true "Update data"
// @Success 200 {object} domain.User
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Router /users/me [put]
func (h *UserHandler) UpdateMe(c *gin.Context) {
	var req service.UpdateUserRequest

	// Get user ID from JWT
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, errors.Unauthorized("user not authenticated", nil))
		return
	}

	id, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, errors.BadRequest("invalid user ID format", nil))
		return
	}

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

	// Get role from context
	roleValue, _ := c.Get("role")
	role, _ := roleValue.(domain.Role)

	// Get IP address
	ipAddress := c.ClientIP()

	// Call service
	user, err := h.userService.UpdateUser(c.Request.Context(), id, &req, id, role, ipAddress)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("failed to update user", err))
		return
	}

	c.JSON(http.StatusOK, user)
}

// InviteUser godoc
// @Summary Invite a new user
// @Description Send invitation email to a new user (ADMIN only)
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.InviteUserRequest true "Invitation data"
// @Success 201 {object} domain.User
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 409 {object} errors.ErrorResponse
// @Router /users/invite [post]
func (h *UserHandler) InviteUser(c *gin.Context) {
	var req service.InviteUserRequest

	// Get tenant ID from JWT
	tenantIDValue, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusForbidden, errors.Forbidden("tenant context required", nil))
		return
	}

	tenantID, ok := tenantIDValue.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, errors.BadRequest("invalid tenant ID format", nil))
		return
	}

	// Get actor user ID and role
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
	user, err := h.userService.InviteUser(c.Request.Context(), tenantID, &req, actorID, actorRole, ipAddress)
	if err != nil {
		if err == repository.ErrDuplicateKey {
			c.JSON(http.StatusConflict, errors.Conflict("user with this email already exists", err))
			return
		}
		c.JSON(http.StatusBadRequest, errors.BadRequest("failed to invite user", err))
		return
	}

	c.JSON(http.StatusCreated, user)
}

// ListUsers godoc
// @Summary List users
// @Description List all users in tenant with optional filters
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param role query string false "Filter by role"
// @Param status query string false "Filter by status"
// @Param search query string false "Search by name or email"
// @Param limit query int false "Items per page" default(20)
// @Param cursor query string false "Pagination cursor"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Router /users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	// Get tenant ID from JWT
	tenantIDValue, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusForbidden, errors.Forbidden("tenant context required", nil))
		return
	}

	tenantID, ok := tenantIDValue.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, errors.BadRequest("invalid tenant ID format", nil))
		return
	}

	// Build filters
	filters := &service.UserFilters{}
	if roleStr := c.Query("role"); roleStr != "" {
		role := domain.Role(roleStr)
		filters.Role = &role
	}
	if statusStr := c.Query("status"); statusStr != "" {
		status := domain.UserStatus(statusStr)
		filters.Status = &status
	}
	if search := c.Query("search"); search != "" {
		filters.SearchTerm = &search
	}

	// Build pagination params
	params := repository.PaginationParams{
		Limit:  20, // Default limit
		Cursor: c.Query("cursor"),
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		// Parse limit (omitted for brevity - add proper parsing)
		params.Limit = 20
	}

	// Call service
	users, pagination, err := h.userService.ListUsers(c.Request.Context(), tenantID, filters, params)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("failed to list users", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users":      users,
		"pagination": pagination,
	})
}

// GetUser godoc
// @Summary Get user by ID
// @Description Get user details by ID (ADMIN can access any user in tenant)
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} domain.User
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	// Parse user ID from path
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("invalid user ID format", err))
		return
	}

	// Get requesting user's details
	requestingUserID, _ := c.Get("user_id")
	requestingID, _ := requestingUserID.(uuid.UUID)

	roleValue, _ := c.Get("role")
	role, _ := roleValue.(domain.Role)

	// Check authorization: user can access their own profile or ADMIN can access any in tenant
	if id != requestingID && role != domain.RoleAdmin && role != domain.RoleSuperAdmin {
		c.JSON(http.StatusForbidden, errors.Forbidden("insufficient permissions", nil))
		return
	}

	// Get user
	user, err := h.userService.GetUser(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, errors.NotFound("user not found", err))
		return
	}

	// For non-SUPER_ADMIN, verify user is in same tenant
	if role != domain.RoleSuperAdmin {
		tenantIDValue, _ := c.Get("tenant_id")
		requestingTenantID, _ := tenantIDValue.(uuid.UUID)

		if user.TenantID == nil || *user.TenantID != requestingTenantID {
			c.JSON(http.StatusForbidden, errors.Forbidden("cannot access user from different tenant", nil))
			return
		}
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUser godoc
// @Summary Update user
// @Description Update user profile (ADMIN only)
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body service.UpdateUserRequest true "Update data"
// @Success 200 {object} domain.User
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	var req service.UpdateUserRequest

	// Parse user ID from path
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("invalid user ID format", err))
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
	user, err := h.userService.UpdateUser(c.Request.Context(), id, &req, actorID, actorRole, ipAddress)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("failed to update user", err))
		return
	}

	c.JSON(http.StatusOK, user)
}

// DeactivateUser godoc
// @Summary Deactivate user
// @Description Deactivate a user account (ADMIN only)
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body map[string]string true "Deactivation reason"
// @Success 200 {object} domain.User
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /users/{id}/deactivate [post]
func (h *UserHandler) DeactivateUser(c *gin.Context) {
	// Parse user ID from path
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("invalid user ID format", err))
		return
	}

	// Get reason from request body
	var req struct {
		Reason string `json:"reason" validate:"required,min=5,max=500"`
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
	user, err := h.userService.DeactivateUser(c.Request.Context(), id, req.Reason, actorID, actorRole, ipAddress)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("failed to deactivate user", err))
		return
	}

	c.JSON(http.StatusOK, user)
}

// ReactivateUser godoc
// @Summary Reactivate user
// @Description Reactivate a deactivated user account (ADMIN only)
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} domain.User
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /users/{id}/reactivate [post]
func (h *UserHandler) ReactivateUser(c *gin.Context) {
	// Parse user ID from path
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("invalid user ID format", err))
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
	user, err := h.userService.ReactivateUser(c.Request.Context(), id, actorID, actorRole, ipAddress)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("failed to reactivate user", err))
		return
	}

	c.JSON(http.StatusOK, user)
}

// ChangePassword godoc
// @Summary Change password
// @Description Change the authenticated user's password
// @Tags Users
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.ChangePasswordRequest true "Password change data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Router /users/me/password [put]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	var req service.ChangePasswordRequest

	// Get user ID from JWT
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, errors.Unauthorized("user not authenticated", nil))
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, errors.BadRequest("invalid user ID format", nil))
		return
	}

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
	if err := h.userService.ChangePassword(c.Request.Context(), userID, &req, ipAddress); err != nil {
		c.JSON(http.StatusBadRequest, errors.BadRequest("failed to change password", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}
