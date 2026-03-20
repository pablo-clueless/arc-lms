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

// ExaminationHandler handles examination HTTP requests
type ExaminationHandler struct {
	examinationService *service.ExaminationService
}

// NewExaminationHandler creates a new examination handler
func NewExaminationHandler(examinationService *service.ExaminationService) *ExaminationHandler {
	return &ExaminationHandler{
		examinationService: examinationService,
	}
}

// ListExaminations godoc
// @Summary List examinations
// @Tags Examinations
// @Security BearerAuth
// @Produce json
// @Param course_id query string false "Filter by course ID"
// @Param term_id query string false "Filter by term ID"
// @Param status query string false "Filter by status (DRAFT, SCHEDULED, IN_PROGRESS, COMPLETED)"
// @Param cursor query string false "Pagination cursor"
// @Param limit query int false "Number of results"
// @Success 200 {object} map[string]interface{}
// @Router /examinations [get]
func (h *ExaminationHandler) ListExaminations(c *gin.Context) {
	tenantID, _, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	var courseID *uuid.UUID
	if courseIDStr := c.Query("course_id"); courseIDStr != "" {
		parsed, err := uuid.Parse(courseIDStr)
		if err != nil {
			errors.BadRequest(c, "invalid course_id", nil)
			return
		}
		courseID = &parsed
	}

	var termID *uuid.UUID
	if termIDStr := c.Query("term_id"); termIDStr != "" {
		parsed, err := uuid.Parse(termIDStr)
		if err != nil {
			errors.BadRequest(c, "invalid term_id", nil)
			return
		}
		termID = &parsed
	}

	var status *domain.ExaminationStatus
	if statusStr := c.Query("status"); statusStr != "" {
		s := domain.ExaminationStatus(statusStr)
		status = &s
	}

	params := repository.PaginationParams{Limit: 50, SortOrder: "DESC"}
	if cursorStr := c.Query("cursor"); cursorStr != "" {
		cursor, _ := uuid.Parse(cursorStr)
		params.Cursor = &cursor
	}

	examinations, pagination, err := h.examinationService.ListExaminations(c.Request.Context(), tenantID, courseID, termID, status, params)
	if err != nil {
		errors.InternalError(c, "failed to list examinations")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": examinations, "pagination": pagination})
}

// CreateExamination godoc
// @Summary Create an examination
// @Tags Examinations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.CreateExaminationRequest true "Examination data"
// @Success 201 {object} domain.Examination
// @Router /examinations [post]
func (h *ExaminationHandler) CreateExamination(c *gin.Context) {
	tenantID, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	var req service.CreateExaminationRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	examination, err := h.examinationService.CreateExamination(
		c.Request.Context(),
		tenantID,
		&req,
		userID,
		role,
		c.ClientIP(),
	)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusCreated, examination)
}

// GetExamination godoc
// @Summary Get an examination
// @Tags Examinations
// @Security BearerAuth
// @Produce json
// @Param id path string true "Examination ID"
// @Success 200 {object} domain.Examination
// @Router /examinations/{id} [get]
func (h *ExaminationHandler) GetExamination(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid examination ID", nil)
		return
	}

	examination, err := h.examinationService.GetExamination(c.Request.Context(), id)
	if err != nil {
		errors.NotFound(c, "examination not found")
		return
	}

	c.JSON(http.StatusOK, examination)
}

// UpdateExamination godoc
// @Summary Update an examination
// @Tags Examinations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Examination ID"
// @Param request body service.UpdateExaminationRequest true "Examination data"
// @Success 200 {object} domain.Examination
// @Router /examinations/{id} [put]
func (h *ExaminationHandler) UpdateExamination(c *gin.Context) {
	_, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid examination ID", nil)
		return
	}

	var req service.UpdateExaminationRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	examination, err := h.examinationService.UpdateExamination(
		c.Request.Context(),
		id,
		&req,
		userID,
		role,
		c.ClientIP(),
	)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, examination)
}

// DeleteExamination godoc
// @Summary Delete an examination
// @Tags Examinations
// @Security BearerAuth
// @Param id path string true "Examination ID"
// @Success 200 {object} map[string]interface{}
// @Router /examinations/{id} [delete]
func (h *ExaminationHandler) DeleteExamination(c *gin.Context) {
	_, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid examination ID", nil)
		return
	}

	if err := h.examinationService.DeleteExamination(c.Request.Context(), id, userID, role, c.ClientIP()); err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "examination deleted"})
}

// ScheduleExamination godoc
// @Summary Schedule an examination (move from DRAFT to SCHEDULED)
// @Tags Examinations
// @Security BearerAuth
// @Param id path string true "Examination ID"
// @Success 200 {object} domain.Examination
// @Router /examinations/{id}/schedule [post]
func (h *ExaminationHandler) ScheduleExamination(c *gin.Context) {
	_, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid examination ID", nil)
		return
	}

	examination, err := h.examinationService.ScheduleExamination(c.Request.Context(), id, userID, role, c.ClientIP())
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, examination)
}

// StartExamination godoc
// @Summary Start an examination attempt (for students)
// @Tags Examinations
// @Security BearerAuth
// @Param id path string true "Examination ID"
// @Success 200 {object} domain.ExaminationSubmission
// @Router /examinations/{id}/start [post]
func (h *ExaminationHandler) StartExamination(c *gin.Context) {
	_, studentID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	examID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid examination ID", nil)
		return
	}

	submission, err := h.examinationService.StartExamination(c.Request.Context(), examID, studentID, c.ClientIP())
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, submission)
}

// SaveAnswer godoc
// @Summary Save an answer during examination (progressive saving)
// @Tags Examinations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Examination ID"
// @Param submission_id path string true "Submission ID"
// @Param request body service.SaveAnswerRequest true "Answer data"
// @Success 200 {object} domain.ExaminationSubmission
// @Router /examinations/{id}/submissions/{submission_id}/answers [post]
func (h *ExaminationHandler) SaveAnswer(c *gin.Context) {
	_, studentID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	submissionID, err := uuid.Parse(c.Param("submission_id"))
	if err != nil {
		errors.BadRequest(c, "invalid submission ID", nil)
		return
	}

	var req service.SaveAnswerRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	submission, err := h.examinationService.SaveAnswer(c.Request.Context(), submissionID, &req, studentID)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, submission)
}

// RecordIntegrityEvent godoc
// @Summary Record an integrity event during examination (tab switch, focus loss, etc.)
// @Tags Examinations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Examination ID"
// @Param submission_id path string true "Submission ID"
// @Param request body service.RecordIntegrityEventRequest true "Event data"
// @Success 200 {object} map[string]interface{}
// @Router /examinations/{id}/submissions/{submission_id}/integrity-events [post]
func (h *ExaminationHandler) RecordIntegrityEvent(c *gin.Context) {
	_, studentID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	submissionID, err := uuid.Parse(c.Param("submission_id"))
	if err != nil {
		errors.BadRequest(c, "invalid submission ID", nil)
		return
	}

	var req service.RecordIntegrityEventRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	if err := h.examinationService.RecordIntegrityEvent(c.Request.Context(), submissionID, &req, studentID); err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "integrity event recorded"})
}

// SubmitExamination godoc
// @Summary Submit an examination
// @Tags Examinations
// @Security BearerAuth
// @Param id path string true "Examination ID"
// @Param submission_id path string true "Submission ID"
// @Success 200 {object} domain.ExaminationSubmission
// @Router /examinations/{id}/submissions/{submission_id}/submit [post]
func (h *ExaminationHandler) SubmitExamination(c *gin.Context) {
	_, studentID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	submissionID, err := uuid.Parse(c.Param("submission_id"))
	if err != nil {
		errors.BadRequest(c, "invalid submission ID", nil)
		return
	}

	submission, err := h.examinationService.SubmitExamination(c.Request.Context(), submissionID, studentID, c.ClientIP())
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, submission)
}

// GradeSubmission godoc
// @Summary Grade an examination submission (manual grading)
// @Tags Examinations
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Examination ID"
// @Param submission_id path string true "Submission ID"
// @Param request body GradeSubmissionRequest true "Grading data"
// @Success 200 {object} domain.ExaminationSubmission
// @Router /examinations/{id}/submissions/{submission_id}/grade [post]
func (h *ExaminationHandler) GradeSubmission(c *gin.Context) {
	_, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	submissionID, err := uuid.Parse(c.Param("submission_id"))
	if err != nil {
		errors.BadRequest(c, "invalid submission ID", nil)
		return
	}

	var req GradeSubmissionRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	submission, err := h.examinationService.GradeSubmission(
		c.Request.Context(),
		submissionID,
		req.Grades,
		req.Feedback,
		userID,
		role,
		c.ClientIP(),
	)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, submission)
}

// GradeSubmissionRequest represents the request body for grading a submission
type GradeSubmissionRequest struct {
	Grades   []service.GradeAnswerRequest `json:"grades" validate:"required,min=1,dive"`
	Feedback *string                      `json:"feedback,omitempty"`
}

// PublishResults godoc
// @Summary Publish examination results
// @Tags Examinations
// @Security BearerAuth
// @Param id path string true "Examination ID"
// @Success 200 {object} domain.Examination
// @Router /examinations/{id}/publish-results [post]
func (h *ExaminationHandler) PublishResults(c *gin.Context) {
	_, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	examID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid examination ID", nil)
		return
	}

	examination, err := h.examinationService.PublishResults(c.Request.Context(), examID, userID, role, c.ClientIP())
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, examination)
}

// GetSubmission godoc
// @Summary Get an examination submission
// @Tags Examinations
// @Security BearerAuth
// @Produce json
// @Param id path string true "Examination ID"
// @Param submission_id path string true "Submission ID"
// @Success 200 {object} domain.ExaminationSubmission
// @Router /examinations/{id}/submissions/{submission_id} [get]
func (h *ExaminationHandler) GetSubmission(c *gin.Context) {
	submissionID, err := uuid.Parse(c.Param("submission_id"))
	if err != nil {
		errors.BadRequest(c, "invalid submission ID", nil)
		return
	}

	submission, err := h.examinationService.GetSubmission(c.Request.Context(), submissionID)
	if err != nil {
		errors.NotFound(c, "submission not found")
		return
	}

	c.JSON(http.StatusOK, submission)
}

// GetMySubmission godoc
// @Summary Get my submission for an examination
// @Tags Examinations
// @Security BearerAuth
// @Produce json
// @Param id path string true "Examination ID"
// @Success 200 {object} domain.ExaminationSubmission
// @Router /examinations/{id}/my-submission [get]
func (h *ExaminationHandler) GetMySubmission(c *gin.Context) {
	_, studentID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	examID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid examination ID", nil)
		return
	}

	submission, err := h.examinationService.GetStudentSubmission(c.Request.Context(), studentID, examID)
	if err != nil {
		errors.NotFound(c, "submission not found")
		return
	}

	c.JSON(http.StatusOK, submission)
}

// ListSubmissions godoc
// @Summary List examination submissions
// @Tags Examinations
// @Security BearerAuth
// @Produce json
// @Param id path string true "Examination ID"
// @Param cursor query string false "Pagination cursor"
// @Param limit query int false "Number of results"
// @Success 200 {object} map[string]interface{}
// @Router /examinations/{id}/submissions [get]
func (h *ExaminationHandler) ListSubmissions(c *gin.Context) {
	examID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid examination ID", nil)
		return
	}

	params := repository.PaginationParams{Limit: 50, SortOrder: "DESC"}
	if cursorStr := c.Query("cursor"); cursorStr != "" {
		cursor, _ := uuid.Parse(cursorStr)
		params.Cursor = &cursor
	}

	submissions, pagination, err := h.examinationService.ListSubmissions(c.Request.Context(), examID, params)
	if err != nil {
		errors.InternalError(c, "failed to list submissions")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": submissions, "pagination": pagination})
}

// GetPendingGradingSubmissions godoc
// @Summary Get submissions pending manual grading
// @Tags Examinations
// @Security BearerAuth
// @Produce json
// @Param id path string true "Examination ID"
// @Success 200 {object} map[string]interface{}
// @Router /examinations/{id}/submissions/pending-grading [get]
func (h *ExaminationHandler) GetPendingGradingSubmissions(c *gin.Context) {
	examID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid examination ID", nil)
		return
	}

	submissions, err := h.examinationService.GetPendingGradingSubmissions(c.Request.Context(), examID)
	if err != nil {
		errors.InternalError(c, "failed to get pending submissions")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": submissions})
}

// GetExaminationStats godoc
// @Summary Get examination statistics
// @Tags Examinations
// @Security BearerAuth
// @Produce json
// @Param id path string true "Examination ID"
// @Success 200 {object} service.ExaminationStats
// @Router /examinations/{id}/stats [get]
func (h *ExaminationHandler) GetExaminationStats(c *gin.Context) {
	examID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid examination ID", nil)
		return
	}

	stats, err := h.examinationService.GetExaminationStats(c.Request.Context(), examID)
	if err != nil {
		errors.InternalError(c, "failed to get examination stats")
		return
	}

	c.JSON(http.StatusOK, stats)
}

// Helper method to get tenant and user IDs from context
// For SuperAdmins, tenant_id may not exist - they are platform-level users
func (h *ExaminationHandler) getTenantAndUserID(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	// Get user ID first (required for all users)
	userIDValue, exists := c.Get("user_id")
	if !exists {
		errors.Unauthorized(c, "user not found in token")
		return uuid.Nil, uuid.Nil, false
	}
	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		errors.BadRequest(c, "invalid user ID format", nil)
		return uuid.Nil, uuid.Nil, false
	}

	// Check if user is SuperAdmin - they don't have tenant_id
	role, _ := GetRoleFromContext(c)
	if role == domain.RoleSuperAdmin {
		return uuid.Nil, userID, true
	}

	// For non-SuperAdmin users, tenant_id is required
	tenantIDValue, exists := c.Get("tenant_id")
	if !exists {
		errors.Unauthorized(c, "tenant not found in token")
		return uuid.Nil, uuid.Nil, false
	}
	tenantID, ok := tenantIDValue.(uuid.UUID)
	if !ok {
		errors.BadRequest(c, "invalid tenant ID format", nil)
		return uuid.Nil, uuid.Nil, false
	}

	return tenantID, userID, true
}

// Helper method to get user role from context
func (h *ExaminationHandler) getUserRole(c *gin.Context) domain.Role {
	role, ok := GetRoleFromContext(c)
	if !ok {
		return domain.RoleStudent
	}
	return role
}
