package handler

import (
	"net/http"
	"strconv"
	"time"

	"arc-lms/internal/domain"
	"arc-lms/internal/pkg/errors"
	"arc-lms/internal/pkg/validator"
	"arc-lms/internal/repository"
	"arc-lms/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ProgressHandler handles progress HTTP requests
type ProgressHandler struct {
	progressService *service.ProgressService
}

// NewProgressHandler creates a new progress handler
func NewProgressHandler(progressService *service.ProgressService) *ProgressHandler {
	return &ProgressHandler{
		progressService: progressService,
	}
}

// GetStudentProgress godoc
// @Summary Get a student's progress for all courses
// @Tags Progress
// @Security BearerAuth
// @Produce json
// @Param student_id path string true "Student ID"
// @Param page query int false "Page number"
// @Param limit query int false "Number of results"
// @Success 200 {object} map[string]interface{}
// @Router /progress/students/{student_id} [get]
func (h *ProgressHandler) GetStudentProgress(c *gin.Context) {
	_, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		errors.BadRequest(c, "invalid student ID", nil)
		return
	}

	role := h.getUserRole(c)

	// Students can only view their own progress
	if role == domain.RoleStudent && userID != studentID {
		errors.Forbidden(c, "you can only view your own progress")
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

	progresses, pagination, err := h.progressService.ListStudentProgress(c.Request.Context(), studentID, params)
	if err != nil {
		errors.InternalError(c, "failed to get student progress")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": progresses, "pagination": pagination})
}

// GetCourseProgress godoc
// @Summary Get progress for all students in a course
// @Tags Progress
// @Security BearerAuth
// @Produce json
// @Param course_id path string true "Course ID"
// @Param page query int false "Page number"
// @Param limit query int false "Number of results"
// @Success 200 {object} map[string]interface{}
// @Router /progress/courses/{course_id} [get]
func (h *ProgressHandler) GetCourseProgress(c *gin.Context) {
	courseID, err := uuid.Parse(c.Param("course_id"))
	if err != nil {
		errors.BadRequest(c, "invalid course ID", nil)
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

	progresses, pagination, err := h.progressService.ListCourseProgress(c.Request.Context(), courseID, params)
	if err != nil {
		errors.InternalError(c, "failed to get course progress")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": progresses, "pagination": pagination})
}

// GetClassProgress godoc
// @Summary Get progress for all students in a class
// @Tags Progress
// @Security BearerAuth
// @Produce json
// @Param class_id path string true "Class ID"
// @Param page query int false "Page number"
// @Param limit query int false "Number of results"
// @Success 200 {object} map[string]interface{}
// @Router /progress/classes/{class_id} [get]
func (h *ProgressHandler) GetClassProgress(c *gin.Context) {
	classID, err := uuid.Parse(c.Param("class_id"))
	if err != nil {
		errors.BadRequest(c, "invalid class ID", nil)
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

	progresses, pagination, err := h.progressService.ListClassProgress(c.Request.Context(), classID, params)
	if err != nil {
		errors.InternalError(c, "failed to get class progress")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": progresses, "pagination": pagination})
}

// GetProgress godoc
// @Summary Get a specific progress record
// @Tags Progress
// @Security BearerAuth
// @Produce json
// @Param id path string true "Progress ID"
// @Success 200 {object} domain.Progress
// @Router /progress/{id} [get]
func (h *ProgressHandler) GetProgress(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid progress ID", nil)
		return
	}

	progress, err := h.progressService.GetProgress(c.Request.Context(), id)
	if err != nil {
		errors.NotFound(c, "progress not found")
		return
	}

	c.JSON(http.StatusOK, progress)
}

// ListFlaggedStudents godoc
// @Summary List all flagged students
// @Tags Progress
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Number of results"
// @Success 200 {object} map[string]interface{}
// @Router /progress/flagged [get]
func (h *ProgressHandler) ListFlaggedStudents(c *gin.Context) {
	tenantID, _, ok := h.getTenantAndUserID(c)
	if !ok {
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

	progresses, pagination, err := h.progressService.ListFlaggedStudents(c.Request.Context(), tenantID, params)
	if err != nil {
		errors.InternalError(c, "failed to list flagged students")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": progresses, "pagination": pagination})
}

// ComputeGrades godoc
// @Summary Compute grades for all students in a course
// @Tags Progress
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param course_id path string true "Course ID"
// @Param term_id query string true "Term ID"
// @Param request body service.GradeWeighting false "Grade weighting (optional)"
// @Success 200 {object} map[string]interface{}
// @Router /progress/courses/{course_id}/compute-grades [post]
func (h *ProgressHandler) ComputeGrades(c *gin.Context) {
	courseID, err := uuid.Parse(c.Param("course_id"))
	if err != nil {
		errors.BadRequest(c, "invalid course ID", nil)
		return
	}

	termIDStr := c.Query("term_id")
	if termIDStr == "" {
		errors.BadRequest(c, "term_id is required", nil)
		return
	}
	termID, err := uuid.Parse(termIDStr)
	if err != nil {
		errors.BadRequest(c, "invalid term ID", nil)
		return
	}

	weighting := service.DefaultGradeWeighting()
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&weighting); err != nil {
			// Use default weighting if parsing fails
			weighting = service.DefaultGradeWeighting()
		}
	}

	progresses, err := h.progressService.ComputeAllGradesForCourse(c.Request.Context(), courseID, termID, weighting)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": progresses, "count": len(progresses)})
}

// ComputeClassPositions godoc
// @Summary Compute class positions for all students
// @Tags Progress
// @Security BearerAuth
// @Param class_id path string true "Class ID"
// @Param term_id query string true "Term ID"
// @Success 200 {object} map[string]interface{}
// @Router /progress/classes/{class_id}/compute-positions [post]
func (h *ProgressHandler) ComputeClassPositions(c *gin.Context) {
	classID, err := uuid.Parse(c.Param("class_id"))
	if err != nil {
		errors.BadRequest(c, "invalid class ID", nil)
		return
	}

	termIDStr := c.Query("term_id")
	if termIDStr == "" {
		errors.BadRequest(c, "term_id is required", nil)
		return
	}
	termID, err := uuid.Parse(termIDStr)
	if err != nil {
		errors.BadRequest(c, "invalid term ID", nil)
		return
	}

	if err := h.progressService.ComputeClassPositions(c.Request.Context(), classID, termID); err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "class positions computed successfully"})
}

// MarkAttendance godoc
// @Summary Mark attendance for a period
// @Tags Progress
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param course_id path string true "Course ID"
// @Param request body service.MarkAttendanceRequest true "Attendance data"
// @Success 200 {object} map[string]interface{}
// @Router /progress/courses/{course_id}/attendance [post]
func (h *ProgressHandler) MarkAttendance(c *gin.Context) {
	tenantID, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	courseID, err := uuid.Parse(c.Param("course_id"))
	if err != nil {
		errors.BadRequest(c, "invalid course ID", nil)
		return
	}

	var req MarkAttendanceHandlerRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	serviceReq := &service.MarkAttendanceRequest{
		PeriodID: req.PeriodID,
		Date:     req.Date,
		Present:  req.Present,
		Absent:   req.Absent,
	}

	if err := h.progressService.MarkAttendance(
		c.Request.Context(),
		tenantID,
		courseID,
		req.TermID,
		req.ClassID,
		serviceReq,
		userID,
	); err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "attendance marked successfully"})
}

// MarkAttendanceHandlerRequest is the handler-specific request for marking attendance
type MarkAttendanceHandlerRequest struct {
	PeriodID uuid.UUID   `json:"period_id" validate:"required,uuid"`
	TermID   uuid.UUID   `json:"term_id" validate:"required,uuid"`
	ClassID  uuid.UUID   `json:"class_id" validate:"required,uuid"`
	Date     time.Time   `json:"date" validate:"required"`
	Present  []uuid.UUID `json:"present" validate:"required"`
	Absent   []uuid.UUID `json:"absent" validate:"required"`
}

// AddTutorRemarks godoc
// @Summary Add tutor remarks to a progress record
// @Tags Progress
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Progress ID"
// @Param request body service.AddTutorRemarksRequest true "Remarks"
// @Success 200 {object} domain.Progress
// @Router /progress/{id}/tutor-remarks [post]
func (h *ProgressHandler) AddTutorRemarks(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid progress ID", nil)
		return
	}

	var req service.AddTutorRemarksRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	progress, err := h.progressService.AddTutorRemarks(c.Request.Context(), id, &req)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, progress)
}

// AddPrincipalRemarks godoc
// @Summary Add principal remarks to a progress record
// @Tags Progress
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Progress ID"
// @Param request body service.AddPrincipalRemarksRequest true "Remarks"
// @Success 200 {object} domain.Progress
// @Router /progress/{id}/principal-remarks [post]
func (h *ProgressHandler) AddPrincipalRemarks(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid progress ID", nil)
		return
	}

	var req service.AddPrincipalRemarksRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	progress, err := h.progressService.AddPrincipalRemarks(c.Request.Context(), id, &req)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, progress)
}

// UnflagProgress godoc
// @Summary Remove flag from a progress record
// @Tags Progress
// @Security BearerAuth
// @Param id path string true "Progress ID"
// @Success 200 {object} domain.Progress
// @Router /progress/{id}/unflag [post]
func (h *ProgressHandler) UnflagProgress(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid progress ID", nil)
		return
	}

	progress, err := h.progressService.UnflagProgress(c.Request.Context(), id)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, progress)
}

// GetCourseStatistics godoc
// @Summary Get statistics for a course
// @Tags Progress
// @Security BearerAuth
// @Produce json
// @Param course_id path string true "Course ID"
// @Param term_id query string true "Term ID"
// @Success 200 {object} postgres.CourseStatistics
// @Router /progress/courses/{course_id}/statistics [get]
func (h *ProgressHandler) GetCourseStatistics(c *gin.Context) {
	courseID, err := uuid.Parse(c.Param("course_id"))
	if err != nil {
		errors.BadRequest(c, "invalid course ID", nil)
		return
	}

	termIDStr := c.Query("term_id")
	if termIDStr == "" {
		errors.BadRequest(c, "term_id is required", nil)
		return
	}
	termID, err := uuid.Parse(termIDStr)
	if err != nil {
		errors.BadRequest(c, "invalid term ID", nil)
		return
	}

	stats, err := h.progressService.GetCourseStatistics(c.Request.Context(), courseID, termID)
	if err != nil {
		errors.InternalError(c, "failed to get course statistics")
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetClassStatistics godoc
// @Summary Get statistics for a class
// @Tags Progress
// @Security BearerAuth
// @Produce json
// @Param class_id path string true "Class ID"
// @Param term_id query string true "Term ID"
// @Success 200 {object} postgres.ClassStatistics
// @Router /progress/classes/{class_id}/statistics [get]
func (h *ProgressHandler) GetClassStatistics(c *gin.Context) {
	classID, err := uuid.Parse(c.Param("class_id"))
	if err != nil {
		errors.BadRequest(c, "invalid class ID", nil)
		return
	}

	termIDStr := c.Query("term_id")
	if termIDStr == "" {
		errors.BadRequest(c, "term_id is required", nil)
		return
	}
	termID, err := uuid.Parse(termIDStr)
	if err != nil {
		errors.BadRequest(c, "invalid term ID", nil)
		return
	}

	stats, err := h.progressService.GetClassStatistics(c.Request.Context(), classID, termID)
	if err != nil {
		errors.InternalError(c, "failed to get class statistics")
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ===================== Report Card Handlers =====================

// GenerateReportCard godoc
// @Summary Generate a report card for a student
// @Tags Report Cards
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param student_id path string true "Student ID"
// @Param request body GenerateReportCardHandlerRequest true "Report card data"
// @Success 201 {object} domain.ReportCard
// @Router /progress/report-cards/students/{student_id} [post]
func (h *ProgressHandler) GenerateReportCard(c *gin.Context) {
	tenantID, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		errors.BadRequest(c, "invalid student ID", nil)
		return
	}

	var req GenerateReportCardHandlerRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	weighting := service.DefaultGradeWeighting()
	if req.CAWeight != nil && req.ExamWeight != nil {
		weighting = service.GradeWeighting{
			ContinuousAssessmentWeight: *req.CAWeight,
			ExaminationWeight:          *req.ExamWeight,
		}
	}

	serviceReq := &service.GenerateReportCardRequest{
		StudentID:        studentID,
		TermID:           req.TermID,
		PrincipalRemarks: req.PrincipalRemarks,
		NextTermBegins:   req.NextTermBegins,
	}

	reportCard, err := h.progressService.GenerateReportCard(
		c.Request.Context(),
		tenantID,
		serviceReq,
		userID,
		weighting,
		c.ClientIP(),
	)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusCreated, reportCard)
}

// GenerateReportCardHandlerRequest is the handler-specific request for generating a report card
type GenerateReportCardHandlerRequest struct {
	TermID           uuid.UUID  `json:"term_id" validate:"required,uuid"`
	PrincipalRemarks *string    `json:"principal_remarks,omitempty" validate:"omitempty,max=1000"`
	NextTermBegins   *time.Time `json:"next_term_begins,omitempty"`
	CAWeight         *int       `json:"ca_weight,omitempty" validate:"omitempty,min=0,max=100"`
	ExamWeight       *int       `json:"exam_weight,omitempty" validate:"omitempty,min=0,max=100"`
}

// GenerateClassReportCards godoc
// @Summary Generate report cards for all students in a class
// @Tags Report Cards
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param class_id path string true "Class ID"
// @Param request body GenerateClassReportCardsRequest true "Report card data"
// @Success 201 {object} map[string]interface{}
// @Router /progress/report-cards/classes/{class_id} [post]
func (h *ProgressHandler) GenerateClassReportCards(c *gin.Context) {
	tenantID, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	classID, err := uuid.Parse(c.Param("class_id"))
	if err != nil {
		errors.BadRequest(c, "invalid class ID", nil)
		return
	}

	var req GenerateClassReportCardsRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	weighting := service.DefaultGradeWeighting()
	if req.CAWeight != nil && req.ExamWeight != nil {
		weighting = service.GradeWeighting{
			ContinuousAssessmentWeight: *req.CAWeight,
			ExaminationWeight:          *req.ExamWeight,
		}
	}

	reportCards, err := h.progressService.GenerateClassReportCards(
		c.Request.Context(),
		tenantID,
		classID,
		req.TermID,
		req.PrincipalRemarks,
		req.NextTermBegins,
		userID,
		weighting,
		c.ClientIP(),
	)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": reportCards, "count": len(reportCards)})
}

// GenerateClassReportCardsRequest is the request for generating class report cards
type GenerateClassReportCardsRequest struct {
	TermID           uuid.UUID  `json:"term_id" validate:"required,uuid"`
	PrincipalRemarks *string    `json:"principal_remarks,omitempty" validate:"omitempty,max=1000"`
	NextTermBegins   *time.Time `json:"next_term_begins,omitempty"`
	CAWeight         *int       `json:"ca_weight,omitempty" validate:"omitempty,min=0,max=100"`
	ExamWeight       *int       `json:"exam_weight,omitempty" validate:"omitempty,min=0,max=100"`
}

// GetReportCard godoc
// @Summary Get a report card by ID
// @Tags Report Cards
// @Security BearerAuth
// @Produce json
// @Param id path string true "Report Card ID"
// @Success 200 {object} domain.ReportCard
// @Router /progress/report-cards/{id} [get]
func (h *ProgressHandler) GetReportCard(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid report card ID", nil)
		return
	}

	reportCard, err := h.progressService.GetReportCard(c.Request.Context(), id)
	if err != nil {
		errors.NotFound(c, "report card not found")
		return
	}

	c.JSON(http.StatusOK, reportCard)
}

// GetStudentReportCard godoc
// @Summary Get a student's report card for a term
// @Tags Report Cards
// @Security BearerAuth
// @Produce json
// @Param student_id path string true "Student ID"
// @Param term_id query string true "Term ID"
// @Success 200 {object} domain.ReportCard
// @Router /progress/report-cards/students/{student_id} [get]
func (h *ProgressHandler) GetStudentReportCard(c *gin.Context) {
	_, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		errors.BadRequest(c, "invalid student ID", nil)
		return
	}

	role := h.getUserRole(c)

	// Students can only view their own report cards
	if role == domain.RoleStudent && userID != studentID {
		errors.Forbidden(c, "you can only view your own report cards")
		return
	}

	termIDStr := c.Query("term_id")
	if termIDStr == "" {
		// List all report cards for the student
		reportCards, err := h.progressService.ListStudentReportCards(c.Request.Context(), studentID)
		if err != nil {
			errors.InternalError(c, "failed to list report cards")
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": reportCards})
		return
	}

	termID, err := uuid.Parse(termIDStr)
	if err != nil {
		errors.BadRequest(c, "invalid term ID", nil)
		return
	}

	reportCard, err := h.progressService.GetStudentReportCard(c.Request.Context(), studentID, termID)
	if err != nil {
		errors.NotFound(c, "report card not found")
		return
	}

	c.JSON(http.StatusOK, reportCard)
}

// UpdateReportCardRemarks godoc
// @Summary Update remarks on a report card
// @Tags Report Cards
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Report Card ID"
// @Param request body service.UpdateReportCardRemarksRequest true "Remarks"
// @Success 200 {object} domain.ReportCard
// @Router /progress/report-cards/{id}/remarks [put]
func (h *ProgressHandler) UpdateReportCardRemarks(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid report card ID", nil)
		return
	}

	var req service.UpdateReportCardRemarksRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	reportCard, err := h.progressService.UpdateReportCardRemarks(c.Request.Context(), id, &req)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, reportCard)
}

// Helper method to get tenant and user IDs from context
// For SuperAdmins, tenant_id may not exist - they are platform-level users
func (h *ProgressHandler) getTenantAndUserID(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
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
func (h *ProgressHandler) getUserRole(c *gin.Context) domain.Role {
	role, ok := GetRoleFromContext(c)
	if !ok {
		return domain.RoleStudent
	}
	return role
}
