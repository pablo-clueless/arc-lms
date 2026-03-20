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

// TermHandler handles term HTTP requests
type TermHandler struct {
	termService *service.TermService
}

// NewTermHandler creates a new term handler
func NewTermHandler(termService *service.TermService) *TermHandler {
	return &TermHandler{
		termService: termService,
	}
}

// CreateTerm godoc
// @Summary Create new term
// @Description Create a new term for a session (ADMIN only, validates BR-001, BR-002)
// @Tags Terms
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param session_id path string true "Session ID"
// @Param request body service.CreateTermRequest true "Term creation data"
// @Success 201 {object} domain.Term
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Router /sessions/{session_id}/terms [post]
func (h *TermHandler) CreateTerm(c *gin.Context) {
	var req service.CreateTermRequest

	// Parse session ID from path
	sessionIDParam := c.Param("id")
	sessionID, err := uuid.Parse(sessionIDParam)
	if err != nil {
		errors.BadRequest(c, "invalid session ID format", map[string]interface{}{"error": err.Error()})
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
	term, err := h.termService.CreateTerm(c.Request.Context(), sessionID, &req, actorID, actorRole, ipAddress)
	if err != nil {
		errors.BadRequest(c, "failed to create term", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, term)
}

// ListTerms godoc
// @Summary List terms
// @Description List all terms for a session
// @Tags Terms
// @Security BearerAuth
// @Produce json
// @Param session_id path string true "Session ID"
// @Param status query string false "Filter by status"
// @Param ordinal query string false "Filter by ordinal (FIRST, SECOND, THIRD)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Router /sessions/{session_id}/terms [get]
func (h *TermHandler) ListTerms(c *gin.Context) {
	// Parse session ID from path
	sessionIDParam := c.Param("id")
	sessionID, err := uuid.Parse(sessionIDParam)
	if err != nil {
		errors.BadRequest(c, "invalid session ID format", map[string]interface{}{"error": err.Error()})
		return
	}

	// Build filters
	filters := &service.TermFilters{}
	if statusStr := c.Query("status"); statusStr != "" {
		status := domain.TermStatus(statusStr)
		filters.Status = &status
	}
	if ordinalStr := c.Query("ordinal"); ordinalStr != "" {
		ordinal := domain.TermOrdinal(ordinalStr)
		filters.Ordinal = &ordinal
	}

	// Call service
	terms, err := h.termService.ListTerms(c.Request.Context(), sessionID, filters)
	if err != nil {
		errors.BadRequest(c, "failed to list terms", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"terms": terms,
	})
}

// GetTerm godoc
// @Summary Get term by ID
// @Description Get term details
// @Tags Terms
// @Security BearerAuth
// @Produce json
// @Param session_id path string true "Session ID"
// @Param id path string true "Term ID"
// @Success 200 {object} domain.Term
// @Failure 400 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /sessions/{session_id}/terms/{id} [get]
func (h *TermHandler) GetTerm(c *gin.Context) {
	// Parse term ID from path
	idParam := c.Param("term_id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		errors.BadRequest(c, "invalid term ID format", map[string]interface{}{"error": err.Error()})
		return
	}

	// Call service
	term, err := h.termService.GetTerm(c.Request.Context(), id)
	if err != nil {
		errors.NotFound(c, "term not found")
		return
	}

	c.JSON(http.StatusOK, term)
}

// UpdateTerm godoc
// @Summary Update term
// @Description Update term details (ADMIN only, cannot update active/completed terms)
// @Tags Terms
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param session_id path string true "Session ID"
// @Param id path string true "Term ID"
// @Param request body service.UpdateTermRequest true "Update data"
// @Success 200 {object} domain.Term
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /sessions/{session_id}/terms/{id} [put]
func (h *TermHandler) UpdateTerm(c *gin.Context) {
	var req service.UpdateTermRequest

	// Parse term ID from path
	idParam := c.Param("term_id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		errors.BadRequest(c, "invalid term ID format", map[string]interface{}{"error": err.Error()})
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
	term, err := h.termService.UpdateTerm(c.Request.Context(), id, &req, actorID, actorRole, ipAddress)
	if err != nil {
		errors.BadRequest(c, "failed to update term", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, term)
}

// DeleteTerm godoc
// @Summary Delete term
// @Description Delete a term (ADMIN only, cannot delete active/completed terms)
// @Tags Terms
// @Security BearerAuth
// @Produce json
// @Param session_id path string true "Session ID"
// @Param id path string true "Term ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /sessions/{session_id}/terms/{id} [delete]
func (h *TermHandler) DeleteTerm(c *gin.Context) {
	// Parse term ID from path
	idParam := c.Param("term_id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		errors.BadRequest(c, "invalid term ID format", map[string]interface{}{"error": err.Error()})
		return
	}

	// Get actor details
	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)

	actorRole, _ := GetRoleFromContext(c)

	// Get IP address
	ipAddress := c.ClientIP()

	// Call service
	if err := h.termService.DeleteTerm(c.Request.Context(), id, actorID, actorRole, ipAddress); err != nil {
		errors.BadRequest(c, "failed to delete term", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Term deleted successfully"})
}

// ActivateTerm godoc
// @Summary Activate term
// @Description Activate a term and trigger auto-billing (BR-009, BR-010: snapshots student count and generates invoice)
// @Tags Terms
// @Security BearerAuth
// @Produce json
// @Param session_id path string true "Session ID"
// @Param id path string true "Term ID"
// @Success 200 {object} domain.Term
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /sessions/{session_id}/terms/{id}/activate [post]
func (h *TermHandler) ActivateTerm(c *gin.Context) {
	// Parse term ID from path
	idParam := c.Param("term_id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		errors.BadRequest(c, "invalid term ID format", map[string]interface{}{"error": err.Error()})
		return
	}

	// Get actor details
	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)

	actorRole, _ := GetRoleFromContext(c)

	// Get IP address
	ipAddress := c.ClientIP()

	// Call service (triggers billing automatically)
	term, err := h.termService.ActivateTerm(c.Request.Context(), id, actorID, actorRole, ipAddress)
	if err != nil {
		errors.BadRequest(c, "failed to activate term", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, term)
}

// CompleteTerm godoc
// @Summary Complete term
// @Description Mark a term as completed (ADMIN only)
// @Tags Terms
// @Security BearerAuth
// @Produce json
// @Param session_id path string true "Session ID"
// @Param id path string true "Term ID"
// @Success 200 {object} domain.Term
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /sessions/{session_id}/terms/{id}/complete [post]
func (h *TermHandler) CompleteTerm(c *gin.Context) {
	// Parse term ID from path
	idParam := c.Param("term_id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		errors.BadRequest(c, "invalid term ID format", map[string]interface{}{"error": err.Error()})
		return
	}

	// Get actor details
	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)

	actorRole, _ := GetRoleFromContext(c)

	// Get IP address
	ipAddress := c.ClientIP()

	// Call service
	term, err := h.termService.CompleteTerm(c.Request.Context(), id, actorID, actorRole, ipAddress)
	if err != nil {
		errors.BadRequest(c, "failed to complete term", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, term)
}

// GetActiveTerm godoc
// @Summary Get active term
// @Description Get the currently active term for a session
// @Tags Terms
// @Security BearerAuth
// @Produce json
// @Param session_id path string true "Session ID"
// @Success 200 {object} domain.Term
// @Failure 404 {object} errors.ErrorResponse
// @Router /sessions/{session_id}/terms/active [get]
func (h *TermHandler) GetActiveTerm(c *gin.Context) {
	// Parse session ID from path
	sessionIDParam := c.Param("id")
	sessionID, err := uuid.Parse(sessionIDParam)
	if err != nil {
		errors.BadRequest(c, "invalid session ID format", map[string]interface{}{"error": err.Error()})
		return
	}

	// Call service
	term, err := h.termService.GetActiveTerm(c.Request.Context(), sessionID)
	if err != nil {
		errors.NotFound(c, "no active term found")
		return
	}

	c.JSON(http.StatusOK, term)
}
