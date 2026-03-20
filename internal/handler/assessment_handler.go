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

// AssessmentHandler handles assessment HTTP requests
type AssessmentHandler struct {
	assessmentService *service.AssessmentService
}

// NewAssessmentHandler creates a new assessment handler
func NewAssessmentHandler(assessmentService *service.AssessmentService) *AssessmentHandler {
	return &AssessmentHandler{
		assessmentService: assessmentService,
	}
}

// === Quiz Handlers ===

// ListQuizzes godoc
// @Summary List quizzes for a course
// @Tags Quizzes
// @Security BearerAuth
// @Produce json
// @Param course_id query string true "Course ID"
// @Param status query string false "Filter by status (DRAFT, PUBLISHED, ARCHIVED)"
// @Param cursor query string false "Pagination cursor"
// @Param limit query int false "Number of results"
// @Success 200 {object} map[string]interface{}
// @Router /courses/{course_id}/quizzes [get]
func (h *AssessmentHandler) ListQuizzes(c *gin.Context) {
	courseIDStr := c.Param("id")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		errors.BadRequest(c, "invalid course ID", nil)
		return
	}

	var status *domain.AssessmentStatus
	if statusStr := c.Query("status"); statusStr != "" {
		s := domain.AssessmentStatus(statusStr)
		status = &s
	}

	params := repository.PaginationParams{Limit: 50, SortOrder: "DESC"}
	if cursorStr := c.Query("cursor"); cursorStr != "" {
		cursor, _ := uuid.Parse(cursorStr)
		params.Cursor = &cursor
	}

	quizzes, pagination, err := h.assessmentService.ListQuizzesByCourse(c.Request.Context(), courseID, status, params)
	if err != nil {
		errors.InternalError(c, "failed to list quizzes")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": quizzes, "pagination": pagination})
}

// CreateQuiz godoc
// @Summary Create a quiz
// @Tags Quizzes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.CreateQuizRequest true "Quiz data"
// @Success 201 {object} domain.Quiz
// @Router /courses/{course_id}/quizzes [post]
func (h *AssessmentHandler) CreateQuiz(c *gin.Context) {
	tenantID, tutorID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	var req service.CreateQuizRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	// Override course ID from path
	courseIDStr := c.Param("id")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		errors.BadRequest(c, "invalid course ID", nil)
		return
	}
	req.CourseID = courseID

	quiz, err := h.assessmentService.CreateQuiz(c.Request.Context(), tenantID, tutorID, &req)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusCreated, quiz)
}

// GetQuiz godoc
// @Summary Get a quiz
// @Tags Quizzes
// @Security BearerAuth
// @Produce json
// @Param id path string true "Quiz ID"
// @Success 200 {object} domain.Quiz
// @Router /courses/{course_id}/quizzes/{id} [get]
func (h *AssessmentHandler) GetQuiz(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid quiz ID", nil)
		return
	}

	quiz, err := h.assessmentService.GetQuiz(c.Request.Context(), id)
	if err != nil {
		errors.NotFound(c, "quiz not found")
		return
	}

	c.JSON(http.StatusOK, quiz)
}

// UpdateQuiz godoc
// @Summary Update a quiz
// @Tags Quizzes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Quiz ID"
// @Param request body service.CreateQuizRequest true "Quiz data"
// @Success 200 {object} domain.Quiz
// @Router /courses/{course_id}/quizzes/{id} [put]
func (h *AssessmentHandler) UpdateQuiz(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid quiz ID", nil)
		return
	}

	var req service.CreateQuizRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	quiz, err := h.assessmentService.UpdateQuiz(c.Request.Context(), id, &req)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, quiz)
}

// DeleteQuiz godoc
// @Summary Delete a quiz
// @Tags Quizzes
// @Security BearerAuth
// @Param id path string true "Quiz ID"
// @Success 200 {object} map[string]interface{}
// @Router /courses/{course_id}/quizzes/{id} [delete]
func (h *AssessmentHandler) DeleteQuiz(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid quiz ID", nil)
		return
	}

	if err := h.assessmentService.DeleteQuiz(c.Request.Context(), id); err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "quiz deleted"})
}

// PublishQuiz godoc
// @Summary Publish a quiz
// @Tags Quizzes
// @Security BearerAuth
// @Param id path string true "Quiz ID"
// @Success 200 {object} domain.Quiz
// @Router /courses/{course_id}/quizzes/{id}/publish [post]
func (h *AssessmentHandler) PublishQuiz(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid quiz ID", nil)
		return
	}

	quiz, err := h.assessmentService.PublishQuiz(c.Request.Context(), id)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, quiz)
}

// StartQuiz godoc
// @Summary Start a quiz attempt
// @Tags Quizzes
// @Security BearerAuth
// @Param id path string true "Quiz ID"
// @Success 200 {object} domain.QuizSubmission
// @Router /courses/{course_id}/quizzes/{id}/start [post]
func (h *AssessmentHandler) StartQuiz(c *gin.Context) {
	_, studentID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	quizID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid quiz ID", nil)
		return
	}

	submission, err := h.assessmentService.StartQuiz(c.Request.Context(), quizID, studentID, c.ClientIP())
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, submission)
}

// SubmitQuiz godoc
// @Summary Submit quiz answers
// @Tags Quizzes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Quiz ID"
// @Param request body service.SubmitQuizRequest true "Answers"
// @Success 200 {object} domain.QuizSubmission
// @Router /courses/{course_id}/quizzes/{id}/submit [post]
func (h *AssessmentHandler) SubmitQuiz(c *gin.Context) {
	_, studentID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	quizID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid quiz ID", nil)
		return
	}

	// Get the student's submission
	quiz, err := h.assessmentService.GetQuiz(c.Request.Context(), quizID)
	if err != nil {
		errors.NotFound(c, "quiz not found")
		return
	}

	// Find student's submission
	submissions, _, _ := h.assessmentService.ListQuizSubmissions(c.Request.Context(), quizID, repository.PaginationParams{Limit: 100})
	var submissionID uuid.UUID
	for _, s := range submissions {
		if s.StudentID == studentID {
			submissionID = s.ID
			break
		}
	}

	if submissionID == uuid.Nil {
		errors.BadRequest(c, "no quiz attempt found, start the quiz first", nil)
		return
	}

	var req service.SubmitQuizRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	submission, err := h.assessmentService.SubmitQuiz(c.Request.Context(), submissionID, &req)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	// Suppress unused variable
	_ = quiz

	c.JSON(http.StatusOK, submission)
}

// GradeQuiz godoc
// @Summary Grade a quiz submission
// @Tags Quizzes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Quiz ID"
// @Param submission_id path string true "Submission ID"
// @Param request body service.GradeQuizRequest true "Grading data"
// @Success 200 {object} domain.QuizSubmission
// @Router /courses/{course_id}/quizzes/{id}/submissions/{submission_id}/grade [post]
func (h *AssessmentHandler) GradeQuiz(c *gin.Context) {
	_, tutorID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	submissionID, err := uuid.Parse(c.Param("submission_id"))
	if err != nil {
		errors.BadRequest(c, "invalid submission ID", nil)
		return
	}

	var req service.GradeQuizRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	submission, err := h.assessmentService.GradeQuiz(c.Request.Context(), submissionID, tutorID, &req)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, submission)
}

// ListQuizSubmissions godoc
// @Summary List quiz submissions
// @Tags Quizzes
// @Security BearerAuth
// @Produce json
// @Param id path string true "Quiz ID"
// @Success 200 {object} map[string]interface{}
// @Router /courses/{course_id}/quizzes/{id}/submissions [get]
func (h *AssessmentHandler) ListQuizSubmissions(c *gin.Context) {
	quizID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid quiz ID", nil)
		return
	}

	params := repository.PaginationParams{Limit: 50, SortOrder: "DESC"}
	submissions, pagination, err := h.assessmentService.ListQuizSubmissions(c.Request.Context(), quizID, params)
	if err != nil {
		errors.InternalError(c, "failed to list submissions")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": submissions, "pagination": pagination})
}

// === Assignment Handlers ===

// ListAssignments godoc
// @Summary List assignments for a course
// @Tags Assignments
// @Security BearerAuth
// @Produce json
// @Param course_id query string true "Course ID"
// @Success 200 {object} map[string]interface{}
// @Router /courses/{course_id}/assignments [get]
func (h *AssessmentHandler) ListAssignments(c *gin.Context) {
	courseIDStr := c.Param("id")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		errors.BadRequest(c, "invalid course ID", nil)
		return
	}

	var status *domain.AssessmentStatus
	if statusStr := c.Query("status"); statusStr != "" {
		s := domain.AssessmentStatus(statusStr)
		status = &s
	}

	params := repository.PaginationParams{Limit: 50, SortOrder: "DESC"}
	assignments, pagination, err := h.assessmentService.ListAssignmentsByCourse(c.Request.Context(), courseID, status, params)
	if err != nil {
		errors.InternalError(c, "failed to list assignments")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": assignments, "pagination": pagination})
}

// CreateAssignment godoc
// @Summary Create an assignment
// @Tags Assignments
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.CreateAssignmentRequest true "Assignment data"
// @Success 201 {object} domain.Assignment
// @Router /courses/{course_id}/assignments [post]
func (h *AssessmentHandler) CreateAssignment(c *gin.Context) {
	tenantID, tutorID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	var req service.CreateAssignmentRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	courseIDStr := c.Param("id")
	courseID, err := uuid.Parse(courseIDStr)
	if err != nil {
		errors.BadRequest(c, "invalid course ID", nil)
		return
	}
	req.CourseID = courseID

	assignment, err := h.assessmentService.CreateAssignment(c.Request.Context(), tenantID, tutorID, &req)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusCreated, assignment)
}

// GetAssignment godoc
// @Summary Get an assignment
// @Tags Assignments
// @Security BearerAuth
// @Produce json
// @Param id path string true "Assignment ID"
// @Success 200 {object} domain.Assignment
// @Router /courses/{course_id}/assignments/{id} [get]
func (h *AssessmentHandler) GetAssignment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid assignment ID", nil)
		return
	}

	assignment, err := h.assessmentService.GetAssignment(c.Request.Context(), id)
	if err != nil {
		errors.NotFound(c, "assignment not found")
		return
	}

	c.JSON(http.StatusOK, assignment)
}

// UpdateAssignment godoc
// @Summary Update an assignment
// @Tags Assignments
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Assignment ID"
// @Param request body service.CreateAssignmentRequest true "Assignment data"
// @Success 200 {object} domain.Assignment
// @Router /courses/{course_id}/assignments/{id} [put]
func (h *AssessmentHandler) UpdateAssignment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid assignment ID", nil)
		return
	}

	var req service.CreateAssignmentRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	assignment, err := h.assessmentService.UpdateAssignment(c.Request.Context(), id, &req)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, assignment)
}

// DeleteAssignment godoc
// @Summary Delete an assignment
// @Tags Assignments
// @Security BearerAuth
// @Param id path string true "Assignment ID"
// @Success 200 {object} map[string]interface{}
// @Router /courses/{course_id}/assignments/{id} [delete]
func (h *AssessmentHandler) DeleteAssignment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid assignment ID", nil)
		return
	}

	if err := h.assessmentService.DeleteAssignment(c.Request.Context(), id); err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "assignment deleted"})
}

// PublishAssignment godoc
// @Summary Publish an assignment
// @Tags Assignments
// @Security BearerAuth
// @Param id path string true "Assignment ID"
// @Success 200 {object} domain.Assignment
// @Router /courses/{course_id}/assignments/{id}/publish [post]
func (h *AssessmentHandler) PublishAssignment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid assignment ID", nil)
		return
	}

	assignment, err := h.assessmentService.PublishAssignment(c.Request.Context(), id)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, assignment)
}

// SubmitAssignment godoc
// @Summary Submit an assignment
// @Tags Assignments
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Assignment ID"
// @Param request body service.SubmitAssignmentRequest true "Submission data"
// @Success 200 {object} domain.AssignmentSubmission
// @Router /courses/{course_id}/assignments/{id}/submit [post]
func (h *AssessmentHandler) SubmitAssignment(c *gin.Context) {
	_, studentID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	assignmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid assignment ID", nil)
		return
	}

	var req service.SubmitAssignmentRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	submission, err := h.assessmentService.SubmitAssignment(c.Request.Context(), assignmentID, studentID, &req, c.ClientIP())
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, submission)
}

// GradeAssignment godoc
// @Summary Grade an assignment submission
// @Tags Assignments
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Assignment ID"
// @Param submission_id path string true "Submission ID"
// @Param request body service.GradeAssignmentRequest true "Grading data"
// @Success 200 {object} domain.AssignmentSubmission
// @Router /courses/{course_id}/assignments/{id}/submissions/{submission_id}/grade [post]
func (h *AssessmentHandler) GradeAssignment(c *gin.Context) {
	_, tutorID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	submissionID, err := uuid.Parse(c.Param("submission_id"))
	if err != nil {
		errors.BadRequest(c, "invalid submission ID", nil)
		return
	}

	var req service.GradeAssignmentRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	submission, err := h.assessmentService.GradeAssignment(c.Request.Context(), submissionID, tutorID, &req)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, submission)
}

// ListAssignmentSubmissions godoc
// @Summary List assignment submissions
// @Tags Assignments
// @Security BearerAuth
// @Produce json
// @Param id path string true "Assignment ID"
// @Success 200 {object} map[string]interface{}
// @Router /courses/{course_id}/assignments/{id}/submissions [get]
func (h *AssessmentHandler) ListAssignmentSubmissions(c *gin.Context) {
	assignmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid assignment ID", nil)
		return
	}

	params := repository.PaginationParams{Limit: 50, SortOrder: "DESC"}
	submissions, pagination, err := h.assessmentService.ListAssignmentSubmissions(c.Request.Context(), assignmentID, params)
	if err != nil {
		errors.InternalError(c, "failed to list submissions")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": submissions, "pagination": pagination})
}

// Helper method to get tenant and user IDs from context
// For SuperAdmins, tenant_id may not exist - they are platform-level users
func (h *AssessmentHandler) getTenantAndUserID(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
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
