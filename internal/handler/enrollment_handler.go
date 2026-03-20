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

// EnrollmentHandler handles enrollment HTTP requests
type EnrollmentHandler struct {
	enrollmentService *service.EnrollmentService
}

// NewEnrollmentHandler creates a new enrollment handler
func NewEnrollmentHandler(enrollmentService *service.EnrollmentService) *EnrollmentHandler {
	return &EnrollmentHandler{
		enrollmentService: enrollmentService,
	}
}

// EnrollStudent godoc
// @Summary Enroll student
// @Description Enroll a student in a class (ADMIN only, enforces BR-003)
// @Tags Enrollments
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.EnrollStudentRequest true "Enrollment data"
// @Success 201 {object} domain.Enrollment
// @Failure 400 {object} errors.ErrorResponse
// @Router /enrollments [post]
func (h *EnrollmentHandler) EnrollStudent(c *gin.Context) {
	var req service.EnrollStudentRequest

	tenantIDValue, _ := c.Get("tenant_id")
	tenantID, _ := tenantIDValue.(uuid.UUID)
	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)
	actorRole, _ := GetRoleFromContext(c)

	if !validator.BindAndValidate(c, &req) {
		return
	}

	enrollment, err := h.enrollmentService.EnrollStudent(c.Request.Context(), tenantID, &req, actorID, actorRole, c.ClientIP())
	if err != nil {
		errors.BadRequest(c, "failed to enroll student", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, enrollment)
}

// ListEnrollments godoc
// @Summary List enrollments
// @Description List all enrollments for tenant
// @Tags Enrollments
// @Security BearerAuth
// @Produce json
// @Param class_id query string false "Filter by class"
// @Param session_id query string false "Filter by session"
// @Param status query string false "Filter by status"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Router /enrollments [get]
func (h *EnrollmentHandler) ListEnrollments(c *gin.Context) {
	tenantIDValue, _ := c.Get("tenant_id")
	tenantID, _ := tenantIDValue.(uuid.UUID)

	var classID, sessionID *uuid.UUID
	var status *domain.EnrollmentStatus

	if cid := c.Query("class_id"); cid != "" {
		id, _ := uuid.Parse(cid)
		classID = &id
	}
	if sid := c.Query("session_id"); sid != "" {
		id, _ := uuid.Parse(sid)
		sessionID = &id
	}
	if statusStr := c.Query("status"); statusStr != "" {
		s := domain.EnrollmentStatus(statusStr)
		status = &s
	}

	enrollments, pagination, err := h.enrollmentService.ListEnrollments(c.Request.Context(), tenantID, classID, sessionID, status, repository.PaginationParams{Limit: 20})
	if err != nil {
		errors.BadRequest(c, "failed to list enrollments", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"enrollments": enrollments, "pagination": pagination})
}

// GetEnrollment godoc
// @Summary Get enrollment by ID
// @Description Get enrollment details
// @Tags Enrollments
// @Security BearerAuth
// @Produce json
// @Param id path string true "Enrollment ID"
// @Success 200 {object} domain.Enrollment
// @Failure 404 {object} errors.ErrorResponse
// @Router /enrollments/{id} [get]
func (h *EnrollmentHandler) GetEnrollment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid enrollment ID", map[string]interface{}{"error": err.Error()})
		return
	}

	enrollment, err := h.enrollmentService.GetEnrollment(c.Request.Context(), id)
	if err != nil {
		errors.NotFound(c, "enrollment not found")
		return
	}

	c.JSON(http.StatusOK, enrollment)
}

// TransferStudent godoc
// @Summary Transfer student
// @Description Transfer a student to a different class (ADMIN only, FR-ACA-006)
// @Tags Enrollments
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Enrollment ID"
// @Param request body service.TransferStudentRequest true "Transfer data"
// @Success 200 {object} domain.Enrollment
// @Failure 400 {object} errors.ErrorResponse
// @Router /enrollments/{id}/transfer [post]
func (h *EnrollmentHandler) TransferStudent(c *gin.Context) {
	var req service.TransferStudentRequest

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid enrollment ID", map[string]interface{}{"error": err.Error()})
		return
	}

	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)
	actorRole, _ := GetRoleFromContext(c)

	if !validator.BindAndValidate(c, &req) {
		return
	}

	enrollment, err := h.enrollmentService.TransferStudent(c.Request.Context(), id, &req, actorID, actorRole, c.ClientIP())
	if err != nil {
		errors.BadRequest(c, "failed to transfer student", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, enrollment)
}

// WithdrawStudent godoc
// @Summary Withdraw student
// @Description Withdraw a student from their class (ADMIN only)
// @Tags Enrollments
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Enrollment ID"
// @Param request body map[string]string true "Withdrawal reason"
// @Success 200 {object} domain.Enrollment
// @Failure 400 {object} errors.ErrorResponse
// @Router /enrollments/{id}/withdraw [post]
func (h *EnrollmentHandler) WithdrawStudent(c *gin.Context) {
	var req struct {
		Reason string `json:"reason" validate:"required,min=10,max=500"`
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid enrollment ID", map[string]interface{}{"error": err.Error()})
		return
	}

	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)
	actorRole, _ := GetRoleFromContext(c)

	if !validator.BindAndValidate(c, &req) {
		return
	}

	enrollment, err := h.enrollmentService.WithdrawStudent(c.Request.Context(), id, req.Reason, actorID, actorRole, c.ClientIP())
	if err != nil {
		errors.BadRequest(c, "failed to withdraw student", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, enrollment)
}

// SuspendEnrollment godoc
// @Summary Suspend enrollment
// @Description Suspend a student's enrollment (ADMIN only)
// @Tags Enrollments
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Enrollment ID"
// @Param request body map[string]string true "Suspension reason"
// @Success 200 {object} domain.Enrollment
// @Failure 400 {object} errors.ErrorResponse
// @Router /enrollments/{id}/suspend [post]
func (h *EnrollmentHandler) SuspendEnrollment(c *gin.Context) {
	var req struct {
		Reason string `json:"reason" validate:"required,min=10,max=500"`
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid enrollment ID", map[string]interface{}{"error": err.Error()})
		return
	}

	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)
	actorRole, _ := GetRoleFromContext(c)

	if !validator.BindAndValidate(c, &req) {
		return
	}

	enrollment, err := h.enrollmentService.SuspendEnrollment(c.Request.Context(), id, req.Reason, actorID, actorRole, c.ClientIP())
	if err != nil {
		errors.BadRequest(c, "failed to suspend enrollment", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, enrollment)
}

// ReactivateEnrollment godoc
// @Summary Reactivate enrollment
// @Description Reactivate a suspended enrollment (ADMIN only)
// @Tags Enrollments
// @Security BearerAuth
// @Produce json
// @Param id path string true "Enrollment ID"
// @Success 200 {object} domain.Enrollment
// @Failure 400 {object} errors.ErrorResponse
// @Router /enrollments/{id}/reactivate [post]
func (h *EnrollmentHandler) ReactivateEnrollment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid enrollment ID", map[string]interface{}{"error": err.Error()})
		return
	}

	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)
	actorRole, _ := GetRoleFromContext(c)

	enrollment, err := h.enrollmentService.ReactivateEnrollment(c.Request.Context(), id, actorID, actorRole, c.ClientIP())
	if err != nil {
		errors.BadRequest(c, "failed to reactivate enrollment", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, enrollment)
}
