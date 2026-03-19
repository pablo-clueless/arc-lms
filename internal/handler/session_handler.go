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

// SessionHandler handles academic session HTTP requests
type SessionHandler struct {
	sessionService *service.SessionService
}

// NewSessionHandler creates a new session handler
func NewSessionHandler(sessionService *service.SessionService) *SessionHandler {
	return &SessionHandler{
		sessionService: sessionService,
	}
}

// CreateSession godoc
// @Summary Create new session
// @Description Create a new academic session (ADMIN only)
// @Tags Sessions
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.CreateSessionRequest true "Session creation data"
// @Success 201 {object} domain.Session
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 409 {object} errors.ErrorResponse
// @Router /sessions [post]
func (h *SessionHandler) CreateSession(c *gin.Context) {
	var req service.CreateSessionRequest

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

	// Get actor details
	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)

	roleValue, _ := c.Get("role")
	actorRole, _ := roleValue.(domain.Role)

	if !validator.BindAndValidate(c, &req) {
		return
	}

	// Get IP address
	ipAddress := c.ClientIP()

	// Call service
	session, err := h.sessionService.CreateSession(c.Request.Context(), tenantID, &req, actorID, actorRole, ipAddress)
	if err != nil {
		if err == repository.ErrDuplicateKey {
			errors.Conflict(c, "CONFLICT", "session with this label already exists", map[string]interface{}{"error": err.Error()})
			return
		}
		errors.BadRequest(c, "failed to create session", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, session)
}

// ListSessions godoc
// @Summary List sessions
// @Description List all sessions for tenant
// @Tags Sessions
// @Security BearerAuth
// @Produce json
// @Param status query string false "Filter by status"
// @Param year query int false "Filter by year"
// @Param search query string false "Search by label"
// @Param limit query int false "Items per page" default(20)
// @Param cursor query string false "Pagination cursor"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Router /sessions [get]
func (h *SessionHandler) ListSessions(c *gin.Context) {
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
	filters := &service.SessionFilters{}
	if statusStr := c.Query("status"); statusStr != "" {
		status := domain.SessionStatus(statusStr)
		filters.Status = &status
	}
	if search := c.Query("search"); search != "" {
		filters.SearchTerm = &search
	}

	// Build pagination params
	params := repository.PaginationParams{
		Limit:  20,
		Cursor: nil, // TODO: Parse cursor from query string
	}

	// Call service
	sessions, pagination, err := h.sessionService.ListSessions(c.Request.Context(), tenantID, filters, params)
	if err != nil {
		errors.BadRequest(c, "failed to list sessions", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions":   sessions,
		"pagination": pagination,
	})
}

// GetSession godoc
// @Summary Get session by ID
// @Description Get session details
// @Tags Sessions
// @Security BearerAuth
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} domain.Session
// @Failure 400 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /sessions/{id} [get]
func (h *SessionHandler) GetSession(c *gin.Context) {
	// Parse session ID from path
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		errors.BadRequest(c, "invalid session ID format", map[string]interface{}{"error": err.Error()})
		return
	}

	// Call service
	session, err := h.sessionService.GetSession(c.Request.Context(), id)
	if err != nil {
		errors.NotFound(c, "session not found")
		return
	}

	c.JSON(http.StatusOK, session)
}

// UpdateSession godoc
// @Summary Update session
// @Description Update session details (ADMIN only)
// @Tags Sessions
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Session ID"
// @Param request body service.UpdateSessionRequest true "Update data"
// @Success 200 {object} domain.Session
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /sessions/{id} [put]
func (h *SessionHandler) UpdateSession(c *gin.Context) {
	var req service.UpdateSessionRequest

	// Parse session ID from path
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		errors.BadRequest(c, "invalid session ID format", map[string]interface{}{"error": err.Error()})
		return
	}

	// Get actor details
	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)

	roleValue, _ := c.Get("role")
	actorRole, _ := roleValue.(domain.Role)

	if !validator.BindAndValidate(c, &req) {
		return
	}

	// Get IP address
	ipAddress := c.ClientIP()

	// Call service
	session, err := h.sessionService.UpdateSession(c.Request.Context(), id, &req, actorID, actorRole, ipAddress)
	if err != nil {
		errors.BadRequest(c, "failed to update session", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, session)
}

// DeleteSession godoc
// @Summary Delete session
// @Description Delete a session (ADMIN only, cannot delete active session)
// @Tags Sessions
// @Security BearerAuth
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /sessions/{id} [delete]
func (h *SessionHandler) DeleteSession(c *gin.Context) {
	// Parse session ID from path
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		errors.BadRequest(c, "invalid session ID format", map[string]interface{}{"error": err.Error()})
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
	if err := h.sessionService.DeleteSession(c.Request.Context(), id, actorID, actorRole, ipAddress); err != nil {
		errors.BadRequest(c, "failed to delete session", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session deleted successfully"})
}

// ActivateSession godoc
// @Summary Activate session
// @Description Activate a session (enforces BR-007: only one active per tenant)
// @Tags Sessions
// @Security BearerAuth
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} domain.Session
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /sessions/{id}/activate [post]
func (h *SessionHandler) ActivateSession(c *gin.Context) {
	// Parse session ID from path
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		errors.BadRequest(c, "invalid session ID format", map[string]interface{}{"error": err.Error()})
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
	session, err := h.sessionService.ActivateSession(c.Request.Context(), id, actorID, actorRole, ipAddress)
	if err != nil {
		errors.BadRequest(c, "failed to activate session", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, session)
}

// ArchiveSession godoc
// @Summary Archive session
// @Description Archive a session (ADMIN only)
// @Tags Sessions
// @Security BearerAuth
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} domain.Session
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /sessions/{id}/archive [post]
func (h *SessionHandler) ArchiveSession(c *gin.Context) {
	// Parse session ID from path
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		errors.BadRequest(c, "invalid session ID format", map[string]interface{}{"error": err.Error()})
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
	session, err := h.sessionService.ArchiveSession(c.Request.Context(), id, actorID, actorRole, ipAddress)
	if err != nil {
		errors.BadRequest(c, "failed to archive session", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, session)
}

// GetActiveSession godoc
// @Summary Get active session
// @Description Get the currently active session for tenant
// @Tags Sessions
// @Security BearerAuth
// @Produce json
// @Success 200 {object} domain.Session
// @Failure 404 {object} errors.ErrorResponse
// @Router /sessions/active [get]
func (h *SessionHandler) GetActiveSession(c *gin.Context) {
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

	// Call service
	session, err := h.sessionService.GetActiveSession(c.Request.Context(), tenantID)
	if err != nil {
		errors.NotFound(c, "no active session found")
		return
	}

	c.JSON(http.StatusOK, session)
}
