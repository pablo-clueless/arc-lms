package handler

import (
	"net/http"

	"arc-lms/internal/pkg/errors"
	"arc-lms/internal/pkg/validator"
	"arc-lms/internal/repository"
	"arc-lms/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ClassHandler handles class HTTP requests
type ClassHandler struct {
	classService *service.ClassService
}

// NewClassHandler creates a new class handler
func NewClassHandler(classService *service.ClassService) *ClassHandler {
	return &ClassHandler{
		classService: classService,
	}
}

// CreateClass godoc
// @Summary Create new class
// @Description Create a new class for a session (ADMIN only)
// @Tags Classes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.CreateClassRequest true "Class creation data"
// @Success 201 {object} domain.Class
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Router /classes [post]
func (h *ClassHandler) CreateClass(c *gin.Context) {
	var req service.CreateClassRequest

	// Get tenant ID from JWT
	tenantIDValue, exists := c.Get("tenant_id")
	if !exists {
		errors.Forbidden(c, "tenant context required")
		return
	}

	tenantID, _ := tenantIDValue.(uuid.UUID)
	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)
	actorRole, _ := GetRoleFromContext(c)

	if !validator.BindAndValidate(c, &req) {
		return
	}

	class, err := h.classService.CreateClass(c.Request.Context(), tenantID, &req, actorID, actorRole, c.ClientIP())
	if err != nil {
		errors.BadRequest(c, "failed to create class", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, class)
}

// ListClasses godoc
// @Summary List classes
// @Description List all classes for tenant
// @Tags Classes
// @Security BearerAuth
// @Produce json
// @Param session_id query string false "Filter by session"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Router /classes [get]
func (h *ClassHandler) ListClasses(c *gin.Context) {
	tenantIDValue, _ := c.Get("tenant_id")
	tenantID, _ := tenantIDValue.(uuid.UUID)

	var sessionID *uuid.UUID
	if sidStr := c.Query("session_id"); sidStr != "" {
		sid, err := uuid.Parse(sidStr)
		if err == nil {
			sessionID = &sid
		}
	}

	classes, pagination, err := h.classService.ListClasses(c.Request.Context(), tenantID, sessionID, repository.PaginationParams{Limit: 20})
	if err != nil {
		errors.BadRequest(c, "failed to list classes", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"classes": classes, "pagination": pagination})
}

// GetClass godoc
// @Summary Get class by ID
// @Description Get class details
// @Tags Classes
// @Security BearerAuth
// @Produce json
// @Param id path string true "Class ID"
// @Success 200 {object} domain.Class
// @Failure 404 {object} errors.ErrorResponse
// @Router /classes/{id} [get]
func (h *ClassHandler) GetClass(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid class ID", map[string]interface{}{"error": err.Error()})
		return
	}

	class, err := h.classService.GetClass(c.Request.Context(), id)
	if err != nil {
		errors.NotFound(c, "class not found")
		return
	}

	c.JSON(http.StatusOK, class)
}

// UpdateClass godoc
// @Summary Update class
// @Description Update class details (ADMIN only)
// @Tags Classes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Class ID"
// @Param request body service.UpdateClassRequest true "Update data"
// @Success 200 {object} domain.Class
// @Failure 400 {object} errors.ErrorResponse
// @Router /classes/{id} [put]
func (h *ClassHandler) UpdateClass(c *gin.Context) {
	var req service.UpdateClassRequest

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid class ID", map[string]interface{}{"error": err.Error()})
		return
	}

	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)
	actorRole, _ := GetRoleFromContext(c)

	if !validator.BindAndValidate(c, &req) {
		return
	}

	class, err := h.classService.UpdateClass(c.Request.Context(), id, &req, actorID, actorRole, c.ClientIP())
	if err != nil {
		errors.BadRequest(c, "failed to update class", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, class)
}

// DeleteClass godoc
// @Summary Delete class
// @Description Delete a class (ADMIN only)
// @Tags Classes
// @Security BearerAuth
// @Produce json
// @Param id path string true "Class ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} errors.ErrorResponse
// @Router /classes/{id} [delete]
func (h *ClassHandler) DeleteClass(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid class ID", map[string]interface{}{"error": err.Error()})
		return
	}

	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)
	actorRole, _ := GetRoleFromContext(c)

	if err := h.classService.DeleteClass(c.Request.Context(), id, actorID, actorRole, c.ClientIP()); err != nil {
		errors.BadRequest(c, "failed to delete class", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Class deleted successfully"})
}
