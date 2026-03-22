package handler

import (
	"net/http"
	"strconv"

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
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
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
		errors.Unauthorized(c, "user not authenticated")
		return
	}

	id, ok := userID.(uuid.UUID)
	if !ok {
		errors.BadRequest(c, "invalid user ID format", nil)
		return
	}

	// Get user
	user, err := h.userService.GetUser(c.Request.Context(), id)
	if err != nil {
		errors.NotFound(c, "user not found")
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
		errors.Unauthorized(c, "user not authenticated")
		return
	}

	id, ok := userID.(uuid.UUID)
	if !ok {
		errors.BadRequest(c, "invalid user ID format", nil)
		return
	}

	if !validator.BindAndValidate(c, &req) {
		return
	}

	// Get role from context
	role, _ := GetRoleFromContext(c)

	// Get IP address
	ipAddress := c.ClientIP()

	// Call service
	user, err := h.userService.UpdateUser(c.Request.Context(), id, &req, id, role, ipAddress)
	if err != nil {
		errors.BadRequest(c, "failed to update user", map[string]interface{}{"error": err.Error()})
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
		errors.Forbidden(c, "tenant context required")
		return
	}

	tenantID, ok := tenantIDValue.(uuid.UUID)
	if !ok {
		errors.BadRequest(c, "invalid tenant ID format", nil)
		return
	}

	// Get actor user ID and role
	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)

	actorRole, _ := GetRoleFromContext(c)

	if !validator.BindAndValidate(c, &req) {
		return
	}

	// Get IP address
	ipAddress := c.ClientIP()

	// Call service
	user, err := h.userService.InviteUser(c.Request.Context(), tenantID, &req, actorID, actorRole, ipAddress)
	if err != nil {
		if err == repository.ErrDuplicateKey {
			errors.Conflict(c, "CONFLICT", "user with this email already exists", map[string]interface{}{"error": err.Error()})
			return
		}
		errors.BadRequest(c, "failed to invite user", map[string]interface{}{"error": err.Error()})
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
// @Param role query string false "Filter by role (ADMIN, TUTOR, STUDENT)"
// @Param status query string false "Filter by status (ACTIVE, PENDING, DEACTIVATED)"
// @Param search query string false "Search by name or email"
// @Param class_id query string false "Filter students by class enrollment"
// @Param session_id query string false "Filter students by session enrollment"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Router /users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	// Get tenant ID from JWT
	tenantIDValue, exists := c.Get("tenant_id")
	if !exists {
		errors.Forbidden(c, "tenant context required")
		return
	}

	tenantID, ok := tenantIDValue.(uuid.UUID)
	if !ok {
		errors.BadRequest(c, "invalid tenant ID format", nil)
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
	if classIDStr := c.Query("class_id"); classIDStr != "" {
		if classID, err := uuid.Parse(classIDStr); err == nil {
			filters.ClassID = &classID
		}
	}
	if sessionIDStr := c.Query("session_id"); sessionIDStr != "" {
		if sessionID, err := uuid.Parse(sessionIDStr); err == nil {
			filters.SessionID = &sessionID
		}
	}

	// Build pagination params
	params := repository.DefaultPaginationParams()
	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			params.Page = page
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			params.Limit = limit
		}
	}

	// Call service
	users, pagination, err := h.userService.ListUsers(c.Request.Context(), tenantID, filters, params)
	if err != nil {
		errors.BadRequest(c, "failed to list users", map[string]interface{}{"error": err.Error()})
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
		errors.BadRequest(c, "invalid user ID format", map[string]interface{}{"error": err.Error()})
		return
	}

	// Get requesting user's details
	requestingUserID, _ := c.Get("user_id")
	requestingID, _ := requestingUserID.(uuid.UUID)

	role, _ := GetRoleFromContext(c)

	// Check authorization: user can access their own profile or ADMIN can access any in tenant
	if id != requestingID && role != domain.RoleAdmin && role != domain.RoleSuperAdmin {
		errors.Forbidden(c, "insufficient permissions")
		return
	}

	// Get user
	user, err := h.userService.GetUser(c.Request.Context(), id)
	if err != nil {
		errors.NotFound(c, "user not found")
		return
	}

	// For non-SUPER_ADMIN, verify user is in same tenant
	if role != domain.RoleSuperAdmin {
		tenantIDValue, _ := c.Get("tenant_id")
		requestingTenantID, _ := tenantIDValue.(uuid.UUID)

		if user.TenantID == nil || *user.TenantID != requestingTenantID {
			errors.Forbidden(c, "cannot access user from different tenant")
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
		errors.BadRequest(c, "invalid user ID format", map[string]interface{}{"error": err.Error()})
		return
	}

	// Get actor details
	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)

	actorRole, _ := GetRoleFromContext(c)

	if !validator.BindAndValidate(c, &req) {
		return
	}

	// Get IP address
	ipAddress := c.ClientIP()

	// Call service
	user, err := h.userService.UpdateUser(c.Request.Context(), id, &req, actorID, actorRole, ipAddress)
	if err != nil {
		errors.BadRequest(c, "failed to update user", map[string]interface{}{"error": err.Error()})
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
		errors.BadRequest(c, "invalid user ID format", map[string]interface{}{"error": err.Error()})
		return
	}

	// Get reason from request body
	var req struct {
		Reason string `json:"reason" validate:"required,min=5,max=500"`
	}

	if !validator.BindAndValidate(c, &req) {
		return
	}

	// Get actor details
	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)

	actorRole, _ := GetRoleFromContext(c)

	// Get IP address
	ipAddress := c.ClientIP()

	// Call service
	user, err := h.userService.DeactivateUser(c.Request.Context(), id, req.Reason, actorID, actorRole, ipAddress)
	if err != nil {
		errors.BadRequest(c, "failed to deactivate user", map[string]interface{}{"error": err.Error()})
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
		errors.BadRequest(c, "invalid user ID format", map[string]interface{}{"error": err.Error()})
		return
	}

	// Get actor details
	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)

	actorRole, _ := GetRoleFromContext(c)

	// Get IP address
	ipAddress := c.ClientIP()

	// Call service
	user, err := h.userService.ReactivateUser(c.Request.Context(), id, actorID, actorRole, ipAddress)
	if err != nil {
		errors.BadRequest(c, "failed to reactivate user", map[string]interface{}{"error": err.Error()})
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
		errors.Unauthorized(c, "user not authenticated")
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		errors.BadRequest(c, "invalid user ID format", nil)
		return
	}

	if !validator.BindAndValidate(c, &req) {
		return
	}

	// Get IP address
	ipAddress := c.ClientIP()

	// Call service
	if err := h.userService.ChangePassword(c.Request.Context(), userID, &req, ipAddress); err != nil {
		errors.BadRequest(c, "failed to change password", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// CreateSuperAdmin godoc
// @Summary Create a new SuperAdmin
// @Description Create a new SuperAdmin user (SUPER_ADMIN only)
// @Tags SuperAdmin
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.CreateSuperAdminRequest true "SuperAdmin data"
// @Success 201 {object} domain.User
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 409 {object} errors.ErrorResponse
// @Router /superadmins [post]
func (h *UserHandler) CreateSuperAdmin(c *gin.Context) {
	var req service.CreateSuperAdminRequest

	// Get actor details
	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)

	actorRole, ok := GetRoleFromContext(c)
	if !ok {
		errors.Unauthorized(c, "invalid role in token")
		return
	}

	// Only SUPER_ADMIN can create SuperAdmins
	if actorRole != domain.RoleSuperAdmin {
		errors.Forbidden(c, "only SuperAdmins can create other SuperAdmins")
		return
	}

	if !validator.BindAndValidate(c, &req) {
		return
	}

	ipAddress := c.ClientIP()

	user, err := h.userService.CreateSuperAdmin(c.Request.Context(), &req, actorID, actorRole, ipAddress)
	if err != nil {
		if err == repository.ErrDuplicateKey {
			errors.Conflict(c, "CONFLICT", "SuperAdmin with this email already exists", nil)
			return
		}
		errors.BadRequest(c, "failed to create SuperAdmin", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// ListSuperAdmins godoc
// @Summary List all SuperAdmins
// @Description List all SuperAdmin users (SUPER_ADMIN only)
// @Tags SuperAdmin
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 403 {object} errors.ErrorResponse
// @Router /superadmins [get]
func (h *UserHandler) ListSuperAdmins(c *gin.Context) {
	actorRole, ok := GetRoleFromContext(c)
	if !ok {
		errors.Unauthorized(c, "invalid role in token")
		return
	}

	// Only SUPER_ADMIN can list SuperAdmins
	if actorRole != domain.RoleSuperAdmin {
		errors.Forbidden(c, "only SuperAdmins can list other SuperAdmins")
		return
	}

	params := repository.DefaultPaginationParams()
	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			params.Page = page
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			params.Limit = limit
		}
	}

	users, pagination, err := h.userService.ListSuperAdmins(c.Request.Context(), actorRole, params)
	if err != nil {
		errors.BadRequest(c, "failed to list SuperAdmins", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users":      users,
		"pagination": pagination,
	})
}

// GetSuperAdmin godoc
// @Summary Get SuperAdmin by ID
// @Description Get SuperAdmin details by ID (SUPER_ADMIN only)
// @Tags SuperAdmin
// @Security BearerAuth
// @Produce json
// @Param id path string true "SuperAdmin ID"
// @Success 200 {object} domain.User
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /superadmins/{id} [get]
func (h *UserHandler) GetSuperAdmin(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		errors.BadRequest(c, "invalid SuperAdmin ID format", nil)
		return
	}

	actorRole, ok := GetRoleFromContext(c)
	if !ok {
		errors.Unauthorized(c, "invalid role in token")
		return
	}

	// Only SUPER_ADMIN can view SuperAdmins
	if actorRole != domain.RoleSuperAdmin {
		errors.Forbidden(c, "only SuperAdmins can view other SuperAdmins")
		return
	}

	user, err := h.userService.GetUser(c.Request.Context(), id)
	if err != nil {
		errors.NotFound(c, "SuperAdmin not found")
		return
	}

	// Verify the user is a SUPER_ADMIN
	if user.Role != domain.RoleSuperAdmin {
		errors.NotFound(c, "SuperAdmin not found")
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateSuperAdmin godoc
// @Summary Update SuperAdmin
// @Description Update SuperAdmin profile (SUPER_ADMIN only)
// @Tags SuperAdmin
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "SuperAdmin ID"
// @Param request body service.UpdateUserRequest true "Update data"
// @Success 200 {object} domain.User
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /superadmins/{id} [put]
func (h *UserHandler) UpdateSuperAdmin(c *gin.Context) {
	var req service.UpdateUserRequest

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		errors.BadRequest(c, "invalid SuperAdmin ID format", nil)
		return
	}

	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)

	actorRole, ok := GetRoleFromContext(c)
	if !ok {
		errors.Unauthorized(c, "invalid role in token")
		return
	}

	// Only SUPER_ADMIN can update SuperAdmins
	if actorRole != domain.RoleSuperAdmin {
		errors.Forbidden(c, "only SuperAdmins can update other SuperAdmins")
		return
	}

	if !validator.BindAndValidate(c, &req) {
		return
	}

	ipAddress := c.ClientIP()

	user, err := h.userService.UpdateSuperAdmin(c.Request.Context(), id, &req, actorID, actorRole, ipAddress)
	if err != nil {
		errors.BadRequest(c, "failed to update SuperAdmin", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// DeleteSuperAdmin godoc
// @Summary Delete SuperAdmin
// @Description Delete a SuperAdmin user (SUPER_ADMIN only)
// @Tags SuperAdmin
// @Security BearerAuth
// @Produce json
// @Param id path string true "SuperAdmin ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /superadmins/{id} [delete]
func (h *UserHandler) DeleteSuperAdmin(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		errors.BadRequest(c, "invalid SuperAdmin ID format", nil)
		return
	}

	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)

	actorRole, ok := GetRoleFromContext(c)
	if !ok {
		errors.Unauthorized(c, "invalid role in token")
		return
	}

	// Only SUPER_ADMIN can delete SuperAdmins
	if actorRole != domain.RoleSuperAdmin {
		errors.Forbidden(c, "only SuperAdmins can delete other SuperAdmins")
		return
	}

	ipAddress := c.ClientIP()

	if err := h.userService.DeleteSuperAdmin(c.Request.Context(), id, actorID, actorRole, ipAddress); err != nil {
		errors.BadRequest(c, "failed to delete SuperAdmin", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "SuperAdmin deleted successfully"})
}
