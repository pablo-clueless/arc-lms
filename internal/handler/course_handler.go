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

// CourseHandler handles course HTTP requests
type CourseHandler struct {
	courseService *service.CourseService
}

// NewCourseHandler creates a new course handler
func NewCourseHandler(courseService *service.CourseService) *CourseHandler {
	return &CourseHandler{
		courseService: courseService,
	}
}

// CreateCourse godoc
// @Summary Create new course
// @Description Create a new course and assign tutor (ADMIN only)
// @Tags Courses
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.CreateCourseRequest true "Course creation data"
// @Success 201 {object} domain.Course
// @Failure 400 {object} errors.ErrorResponse
// @Router /courses [post]
func (h *CourseHandler) CreateCourse(c *gin.Context) {
	var req service.CreateCourseRequest

	tenantIDValue, _ := c.Get("tenant_id")
	tenantID, _ := tenantIDValue.(uuid.UUID)
	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)
	actorRole, _ := GetRoleFromContext(c)

	if !validator.BindAndValidate(c, &req) {
		return
	}

	course, err := h.courseService.CreateCourse(c.Request.Context(), tenantID, &req, actorID, actorRole, c.ClientIP())
	if err != nil {
		errors.BadRequest(c, "failed to create course", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, course)
}

// ListCourses godoc
// @Summary List courses
// @Description List all courses for tenant
// @Tags Courses
// @Security BearerAuth
// @Produce json
// @Param class_id query string false "Filter by class"
// @Param term_id query string false "Filter by term"
// @Param tutor_id query string false "Filter by tutor"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Router /courses [get]
func (h *CourseHandler) ListCourses(c *gin.Context) {
	tenantIDValue, _ := c.Get("tenant_id")
	tenantID, _ := tenantIDValue.(uuid.UUID)

	var classID, termID, tutorID *uuid.UUID
	if cid := c.Query("class_id"); cid != "" {
		id, _ := uuid.Parse(cid)
		classID = &id
	}
	if tid := c.Query("term_id"); tid != "" {
		id, _ := uuid.Parse(tid)
		termID = &id
	}
	if tuid := c.Query("tutor_id"); tuid != "" {
		id, _ := uuid.Parse(tuid)
		tutorID = &id
	}

	courses, pagination, err := h.courseService.ListCourses(c.Request.Context(), tenantID, classID, termID, tutorID, repository.PaginationParams{Limit: 20})
	if err != nil {
		errors.BadRequest(c, "failed to list courses", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"courses": courses, "pagination": pagination})
}

// GetCourse godoc
// @Summary Get course by ID
// @Description Get course details
// @Tags Courses
// @Security BearerAuth
// @Produce json
// @Param id path string true "Course ID"
// @Success 200 {object} domain.Course
// @Failure 404 {object} errors.ErrorResponse
// @Router /courses/{id} [get]
func (h *CourseHandler) GetCourse(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid course ID", map[string]interface{}{"error": err.Error()})
		return
	}

	course, err := h.courseService.GetCourse(c.Request.Context(), id)
	if err != nil {
		errors.NotFound(c, "course not found")
		return
	}

	c.JSON(http.StatusOK, course)
}

// UpdateCourse godoc
// @Summary Update course
// @Description Update course details (ADMIN only)
// @Tags Courses
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Course ID"
// @Param request body service.UpdateCourseRequest true "Update data"
// @Success 200 {object} domain.Course
// @Failure 400 {object} errors.ErrorResponse
// @Router /courses/{id} [put]
func (h *CourseHandler) UpdateCourse(c *gin.Context) {
	var req service.UpdateCourseRequest

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid course ID", map[string]interface{}{"error": err.Error()})
		return
	}

	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)
	actorRole, _ := GetRoleFromContext(c)

	if !validator.BindAndValidate(c, &req) {
		return
	}

	course, err := h.courseService.UpdateCourse(c.Request.Context(), id, &req, actorID, actorRole, c.ClientIP())
	if err != nil {
		errors.BadRequest(c, "failed to update course", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, course)
}

// DeleteCourse godoc
// @Summary Delete course
// @Description Delete a course (ADMIN only)
// @Tags Courses
// @Security BearerAuth
// @Produce json
// @Param id path string true "Course ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} errors.ErrorResponse
// @Router /courses/{id} [delete]
func (h *CourseHandler) DeleteCourse(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid course ID", map[string]interface{}{"error": err.Error()})
		return
	}

	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)
	actorRole, _ := GetRoleFromContext(c)

	if err := h.courseService.DeleteCourse(c.Request.Context(), id, actorID, actorRole, c.ClientIP()); err != nil {
		errors.BadRequest(c, "failed to delete course", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Course deleted successfully"})
}

// ReassignTutor godoc
// @Summary Reassign tutor to course
// @Description Reassign a course to a different tutor (ADMIN only, FR-ACA-005)
// @Tags Courses
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Course ID"
// @Param request body service.ReassignTutorRequest true "Reassignment data"
// @Success 200 {object} domain.Course
// @Failure 400 {object} errors.ErrorResponse
// @Router /courses/{id}/reassign-tutor [post]
func (h *CourseHandler) ReassignTutor(c *gin.Context) {
	var req service.ReassignTutorRequest

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid course ID", map[string]interface{}{"error": err.Error()})
		return
	}

	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)
	actorRole, _ := GetRoleFromContext(c)

	if !validator.BindAndValidate(c, &req) {
		return
	}

	course, err := h.courseService.ReassignTutor(c.Request.Context(), id, &req, actorID, actorRole, c.ClientIP())
	if err != nil {
		errors.BadRequest(c, "failed to reassign tutor", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, course)
}
