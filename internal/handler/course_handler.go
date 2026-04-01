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

// ==================== Course Content Handlers ====================

// CreateContent godoc
// @Summary Create course content
// @Description Add new content to a course (TUTOR/ADMIN)
// @Tags Course Content
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Course ID"
// @Param request body service.CreateContentRequest true "Content data"
// @Success 201 {object} domain.CourseContent
// @Failure 400 {object} errors.ErrorResponse
// @Router /courses/{id}/contents [post]
func (h *CourseHandler) CreateContent(c *gin.Context) {
	var req service.CreateContentRequest

	courseID, err := uuid.Parse(c.Param("id"))
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

	content, err := h.courseService.CreateContent(c.Request.Context(), courseID, &req, actorID, actorRole, c.ClientIP())
	if err != nil {
		errors.BadRequest(c, "failed to create content", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, content)
}

// ListContents godoc
// @Summary List course contents
// @Description List all content items for a course
// @Tags Course Content
// @Security BearerAuth
// @Produce json
// @Param id path string true "Course ID"
// @Param type query string false "Filter by content type (TEXT, VIDEO, IMAGE, AUDIO, DOCUMENT, LINK)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Router /courses/{id}/contents [get]
func (h *CourseHandler) ListContents(c *gin.Context) {
	courseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid course ID", map[string]interface{}{"error": err.Error()})
		return
	}

	var contentType *string
	if ct := c.Query("type"); ct != "" {
		contentType = &ct
	}

	params := repository.PaginationParams{Limit: 100, SortOrder: "ASC"}

	var contents interface{}
	var pagination interface{}
	var listErr error

	if contentType != nil {
		ct := *contentType
		contents, pagination, listErr = h.courseService.ListContents(c.Request.Context(), courseID, (*domain.ContentType)(&ct), params)
	} else {
		contents, pagination, listErr = h.courseService.ListContents(c.Request.Context(), courseID, nil, params)
	}

	if listErr != nil {
		errors.BadRequest(c, "failed to list contents", map[string]interface{}{"error": listErr.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": contents, "pagination": pagination})
}

// GetContent godoc
// @Summary Get content by ID
// @Description Get a specific content item
// @Tags Course Content
// @Security BearerAuth
// @Produce json
// @Param id path string true "Course ID"
// @Param content_id path string true "Content ID"
// @Success 200 {object} domain.CourseContent
// @Failure 404 {object} errors.ErrorResponse
// @Router /courses/{id}/contents/{content_id} [get]
func (h *CourseHandler) GetContent(c *gin.Context) {
	contentID, err := uuid.Parse(c.Param("content_id"))
	if err != nil {
		errors.BadRequest(c, "invalid content ID", map[string]interface{}{"error": err.Error()})
		return
	}

	content, err := h.courseService.GetContent(c.Request.Context(), contentID)
	if err != nil {
		errors.NotFound(c, "content not found")
		return
	}

	c.JSON(http.StatusOK, content)
}

// UpdateContent godoc
// @Summary Update content
// @Description Update a content item (TUTOR/ADMIN)
// @Tags Course Content
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Course ID"
// @Param content_id path string true "Content ID"
// @Param request body service.UpdateContentRequest true "Update data"
// @Success 200 {object} domain.CourseContent
// @Failure 400 {object} errors.ErrorResponse
// @Router /courses/{id}/contents/{content_id} [put]
func (h *CourseHandler) UpdateContent(c *gin.Context) {
	var req service.UpdateContentRequest

	contentID, err := uuid.Parse(c.Param("content_id"))
	if err != nil {
		errors.BadRequest(c, "invalid content ID", map[string]interface{}{"error": err.Error()})
		return
	}

	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)
	actorRole, _ := GetRoleFromContext(c)

	if !validator.BindAndValidate(c, &req) {
		return
	}

	content, err := h.courseService.UpdateContent(c.Request.Context(), contentID, &req, actorID, actorRole, c.ClientIP())
	if err != nil {
		errors.BadRequest(c, "failed to update content", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, content)
}

// DeleteContent godoc
// @Summary Delete content
// @Description Delete a content item (TUTOR/ADMIN)
// @Tags Course Content
// @Security BearerAuth
// @Produce json
// @Param id path string true "Course ID"
// @Param content_id path string true "Content ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} errors.ErrorResponse
// @Router /courses/{id}/contents/{content_id} [delete]
func (h *CourseHandler) DeleteContent(c *gin.Context) {
	contentID, err := uuid.Parse(c.Param("content_id"))
	if err != nil {
		errors.BadRequest(c, "invalid content ID", map[string]interface{}{"error": err.Error()})
		return
	}

	actorIDValue, _ := c.Get("user_id")
	actorID, _ := actorIDValue.(uuid.UUID)
	actorRole, _ := GetRoleFromContext(c)

	if err := h.courseService.DeleteContent(c.Request.Context(), contentID, actorID, actorRole, c.ClientIP()); err != nil {
		errors.BadRequest(c, "failed to delete content", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Content deleted successfully"})
}

// ReorderContents godoc
// @Summary Reorder course contents
// @Description Reorder content items for a course (TUTOR/ADMIN)
// @Tags Course Content
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Course ID"
// @Param request body service.ReorderContentRequest true "New order of content IDs"
// @Success 200 {object} map[string]string
// @Failure 400 {object} errors.ErrorResponse
// @Router /courses/{id}/contents/reorder [post]
func (h *CourseHandler) ReorderContents(c *gin.Context) {
	var req service.ReorderContentRequest

	courseID, err := uuid.Parse(c.Param("id"))
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

	if err := h.courseService.ReorderContents(c.Request.Context(), courseID, &req, actorID, actorRole, c.ClientIP()); err != nil {
		errors.BadRequest(c, "failed to reorder contents", map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contents reordered successfully"})
}

// GetCourseWithContents godoc
// @Summary Get course with all contents
// @Description Get course details including all content items
// @Tags Courses
// @Security BearerAuth
// @Produce json
// @Param id path string true "Course ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} errors.ErrorResponse
// @Router /courses/{id}/full [get]
func (h *CourseHandler) GetCourseWithContents(c *gin.Context) {
	courseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid course ID", map[string]interface{}{"error": err.Error()})
		return
	}

	course, contents, err := h.courseService.GetCourseWithContents(c.Request.Context(), courseID)
	if err != nil {
		errors.NotFound(c, "course not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"course": course, "contents": contents})
}
