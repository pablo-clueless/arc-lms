package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"arc-lms/internal/repository"
	"arc-lms/internal/service"
)

// GuardianHandler handles guardian-related HTTP requests
type GuardianHandler struct {
	guardianService *service.GuardianService
}

// NewGuardianHandler creates a new guardian handler
func NewGuardianHandler(guardianService *service.GuardianService) *GuardianHandler {
	return &GuardianHandler{
		guardianService: guardianService,
	}
}

// CreateAndLinkGuardian creates a new guardian user and links them to a student
// @Summary Create and link guardian
// @Description Create a new guardian user and link them to a student in one operation
// @Tags Guardians
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.CreateAndLinkGuardianRequest true "Create and link guardian request"
// @Success 201 {object} service.CreateAndLinkGuardianResponse
// @Failure 400 {object} errors.ErrorResponse
// @Router /guardians/create-and-link [post]
func (h *GuardianHandler) CreateAndLinkGuardian(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant context required"})
		return
	}

	var req service.CreateAndLinkGuardianRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	actorID, _ := c.Get("user_id")
	actorRole, _ := GetRoleFromContext(c)

	result, err := h.guardianService.CreateAndLinkGuardian(
		c.Request.Context(),
		tenantID.(uuid.UUID),
		&req,
		actorID.(uuid.UUID),
		actorRole,
		c.ClientIP(),
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// LinkWard links a guardian to a student (ward)
// @Summary Link ward to guardian
// @Description Link a student (ward) to a parent/guardian
// @Tags Guardians
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param guardian_id path string true "Guardian User ID"
// @Param request body service.LinkWardRequest true "Link ward request"
// @Success 201 {object} domain.GuardianWithDetails
// @Failure 400 {object} errors.ErrorResponse
// @Router /guardians/{guardian_id}/wards [post]
func (h *GuardianHandler) LinkWard(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant context required"})
		return
	}

	guardianID, err := uuid.Parse(c.Param("guardian_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid guardian ID"})
		return
	}

	var req service.LinkWardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	actorID, _ := c.Get("user_id")
	actorRole, _ := GetRoleFromContext(c)

	result, err := h.guardianService.LinkWard(
		c.Request.Context(),
		tenantID.(uuid.UUID),
		guardianID,
		&req,
		actorID.(uuid.UUID),
		actorRole,
		c.ClientIP(),
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// UnlinkWard removes a guardian-ward relationship
// @Summary Unlink ward from guardian
// @Description Remove a student (ward) from a parent/guardian
// @Tags Guardians
// @Security BearerAuth
// @Produce json
// @Param id path string true "Guardian Relationship ID"
// @Success 204 "No Content"
// @Failure 400 {object} errors.ErrorResponse
// @Router /guardians/{id} [delete]
func (h *GuardianHandler) UnlinkWard(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant context required"})
		return
	}

	relationshipID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid relationship ID"})
		return
	}

	actorID, _ := c.Get("user_id")
	actorRole, _ := GetRoleFromContext(c)

	err = h.guardianService.UnlinkWard(
		c.Request.Context(),
		tenantID.(uuid.UUID),
		relationshipID,
		actorID.(uuid.UUID),
		actorRole,
		c.ClientIP(),
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetMyWards returns all wards for the current guardian (parent)
// @Summary Get my wards
// @Description Get all students (wards) linked to the current parent/guardian
// @Tags Guardians
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Router /guardians/my-wards [get]
func (h *GuardianHandler) GetMyWards(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
		return
	}

	params := repository.PaginationParams{
		Page:      1,
		Limit:     20,
		SortOrder: "DESC",
	}

	wards, pagination, err := h.guardianService.GetWards(
		c.Request.Context(),
		userID.(uuid.UUID),
		params,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       wards,
		"pagination": pagination,
	})
}

// GetWardProgress returns progress for a specific ward
// @Summary Get ward progress
// @Description Get academic progress for a specific ward (student)
// @Tags Guardians
// @Security BearerAuth
// @Produce json
// @Param student_id path string true "Student ID"
// @Success 200 {object} service.WardProgressResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Router /guardians/wards/{student_id}/progress [get]
func (h *GuardianHandler) GetWardProgress(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
		return
	}

	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student ID"})
		return
	}

	result, err := h.guardianService.GetWardProgress(
		c.Request.Context(),
		userID.(uuid.UUID),
		studentID,
	)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetWardInvoices returns invoices for a specific ward
// @Summary Get ward invoices
// @Description Get billing invoices for a specific ward (student)
// @Tags Guardians
// @Security BearerAuth
// @Produce json
// @Param student_id path string true "Student ID"
// @Success 200 {object} service.WardInvoicesResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Router /guardians/wards/{student_id}/invoices [get]
func (h *GuardianHandler) GetWardInvoices(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found in context"})
		return
	}

	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student ID"})
		return
	}

	result, err := h.guardianService.GetWardInvoices(
		c.Request.Context(),
		userID.(uuid.UUID),
		studentID,
	)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetStudentGuardians returns all guardians for a student (admin view)
// @Summary Get student guardians
// @Description Get all guardians linked to a specific student
// @Tags Guardians
// @Security BearerAuth
// @Produce json
// @Param student_id path string true "Student ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Router /guardians/students/{student_id} [get]
func (h *GuardianHandler) GetStudentGuardians(c *gin.Context) {
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid student ID"})
		return
	}

	params := repository.PaginationParams{
		Page:      1,
		Limit:     20,
		SortOrder: "DESC",
	}

	guardians, pagination, err := h.guardianService.GetGuardians(
		c.Request.Context(),
		studentID,
		params,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       guardians,
		"pagination": pagination,
	})
}

// ListRelationships lists all guardian relationships (admin view)
// @Summary List guardian relationships
// @Description List all guardian-student relationships in the tenant
// @Tags Guardians
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Router /guardians [get]
func (h *GuardianHandler) ListRelationships(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant context required"})
		return
	}

	params := repository.PaginationParams{
		Page:      1,
		Limit:     20,
		SortOrder: "DESC",
	}

	relationships, pagination, err := h.guardianService.ListRelationships(
		c.Request.Context(),
		tenantID.(uuid.UUID),
		params,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       relationships,
		"pagination": pagination,
	})
}
