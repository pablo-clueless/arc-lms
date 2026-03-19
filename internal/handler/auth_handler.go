package handler

import (
	"net/http"

	"arc-lms/internal/domain"
	"arc-lms/internal/pkg/errors"
	"arc-lms/internal/pkg/validator"
	"arc-lms/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthHandler handles authentication HTTP requests
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Login godoc
// @Summary User login
// @Description Authenticate user with email and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body service.LoginRequest true "Login credentials"
// @Success 200 {object} service.LoginResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Router /public/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req service.LoginRequest

	if !validator.BindAndValidate(c, &req) {
		return
	}

	// Get IP address
	ipAddress := c.ClientIP()

	// Call service
	response, err := h.authService.Login(c.Request.Context(), &req, ipAddress)
	if err != nil {
		errors.Unauthorized(c, "authentication failed")
		return
	}

	c.JSON(http.StatusOK, response)
}

// Register godoc
// @Summary Register new user
// @Description Create a new user account (public registration)
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body service.RegisterRequest true "Registration data"
// @Success 201 {object} domain.User
// @Failure 400 {object} errors.ErrorResponse
// @Failure 409 {object} errors.ErrorResponse
// @Router /public/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req service.RegisterRequest

	if !validator.BindAndValidate(c, &req) {
		return
	}

	// Get IP address
	ipAddress := c.ClientIP()

	// For public registration, actor is the user being created
	// In a more restricted flow, this would be called by an ADMIN
	actorID := uuid.Nil             // No actor for public registration
	actorRole := domain.RoleStudent // Default role for self-registration

	// Call service
	user, err := h.authService.Register(c.Request.Context(), &req, actorID, actorRole, ipAddress)
	if err != nil {
		errors.Conflict(c, "CONFLICT", "registration failed", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// RequestPasswordReset godoc
// @Summary Request password reset
// @Description Send password reset email to user
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body service.PasswordResetRequest true "Email address"
// @Success 200 {object} map[string]string
// @Failure 400 {object} errors.ErrorResponse
// @Router /public/auth/password-reset [post]
func (h *AuthHandler) RequestPasswordReset(c *gin.Context) {
	var req service.PasswordResetRequest

	if !validator.BindAndValidate(c, &req) {
		return
	}

	// Call service
	if err := h.authService.RequestPasswordReset(c.Request.Context(), &req); err != nil {
		// Don't expose internal errors for security
		c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a password reset link has been sent"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a password reset link has been sent"})
}

// ResetPassword godoc
// @Summary Reset password with token
// @Description Reset user password using reset token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body service.ResetPasswordRequest true "Reset token and new password"
// @Success 200 {object} map[string]string
// @Failure 400 {object} errors.ErrorResponse
// @Router /public/auth/password-reset/confirm [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req service.ResetPasswordRequest

	if !validator.BindAndValidate(c, &req) {
		return
	}

	// Get IP address
	ipAddress := c.ClientIP()

	// Call service
	if err := h.authService.ResetPassword(c.Request.Context(), &req, ipAddress); err != nil {
		errors.BadRequest(c, "password reset failed", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Generate new access token from refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body map[string]string true "Refresh token"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Router /public/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}

	if !validator.BindAndValidate(c, &req) {
		return
	}

	// Call service
	tokenPair, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		errors.Unauthorized(c, "token refresh failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"token_pair": tokenPair})
}

// ValidateInvitation godoc
// @Summary Validate invitation token
// @Description Check if invitation token is valid
// @Tags Authentication
// @Accept json
// @Produce json
// @Param token query string true "Invitation token"
// @Success 200 {object} domain.User
// @Failure 400 {object} errors.ErrorResponse
// @Router /public/auth/invitation/validate [get]
func (h *AuthHandler) ValidateInvitation(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		errors.BadRequest(c, "token is required", nil)
		return
	}

	// Call service
	user, err := h.authService.ValidateInvitation(c.Request.Context(), token)
	if err != nil {
		errors.BadRequest(c, "invitation validation failed", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// AcceptInvitation godoc
// @Summary Accept user invitation
// @Description Accept invitation and set password to activate account
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body service.AcceptInvitationRequest true "Invitation token and password"
// @Success 200 {object} domain.User
// @Failure 400 {object} errors.ErrorResponse
// @Router /public/auth/invitation/accept [post]
func (h *AuthHandler) AcceptInvitation(c *gin.Context) {
	var req service.AcceptInvitationRequest

	if !validator.BindAndValidate(c, &req) {
		return
	}

	// Get IP address
	ipAddress := c.ClientIP()

	// Call service
	user, err := h.authService.AcceptInvitation(c.Request.Context(), &req, ipAddress)
	if err != nil {
		errors.BadRequest(c, "invitation acceptance failed", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}
