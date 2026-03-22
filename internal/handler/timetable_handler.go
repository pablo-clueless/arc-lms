package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"arc-lms/internal/domain"
	"arc-lms/internal/repository"
	"arc-lms/internal/service"
)

// TimetableHandler handles timetable-related HTTP requests
type TimetableHandler struct {
	timetableService *service.TimetableService
}

// NewTimetableHandler creates a new timetable handler
func NewTimetableHandler(timetableService *service.TimetableService) *TimetableHandler {
	return &TimetableHandler{
		timetableService: timetableService,
	}
}

// GenerateTimetable generates a new timetable for a class
// @Summary Generate timetable
// @Description Automatically generates a timetable for a class within a term
// @Tags timetables
// @Accept json
// @Produce json
// @Param request body service.GenerateTimetableRequest true "Generate timetable request"
// @Success 201 {object} service.TimetableWithPeriods
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /timetables/generate [post]
func (h *TimetableHandler) GenerateTimetable(c *gin.Context) {
	var req service.GenerateTimetableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant not found in context"})
		return
	}

	userID, _ := c.Get("user_id")
	role, _ := GetRoleFromContext(c)

	timetable, err := h.timetableService.GenerateTimetable(
		c.Request.Context(),
		tenantID.(uuid.UUID),
		&req,
		userID.(uuid.UUID),
		role,
		c.ClientIP(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, timetable)
}

// GetTimetable retrieves a timetable by ID
// @Summary Get timetable
// @Description Retrieves a timetable with all its periods
// @Tags timetables
// @Produce json
// @Param id path string true "Timetable ID"
// @Success 200 {object} service.TimetableWithPeriods
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /timetables/{id} [get]
func (h *TimetableHandler) GetTimetable(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid timetable ID"})
		return
	}

	timetable, err := h.timetableService.GetTimetable(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, timetable)
}

// ListTimetables lists timetables with optional filters
// @Summary List timetables
// @Description Lists timetables for a tenant, optionally filtered by class or term
// @Tags timetables
// @Produce json
// @Param class_id query string false "Filter by class ID"
// @Param term_id query string false "Filter by term ID"
// @Param cursor query string false "Pagination cursor"
// @Param limit query int false "Number of items per page" default(20)
// @Success 200 {object} PaginatedResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /timetables [get]
func (h *TimetableHandler) ListTimetables(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant not found in context"})
		return
	}

	var classID, termID *uuid.UUID
	if classIDStr := c.Query("class_id"); classIDStr != "" {
		id, err := uuid.Parse(classIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid class_id"})
			return
		}
		classID = &id
	}

	if termIDStr := c.Query("term_id"); termIDStr != "" {
		id, err := uuid.Parse(termIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid term_id"})
			return
		}
		termID = &id
	}

	params := repository.PaginationParams{Limit: 20, SortOrder: "DESC"}

	timetables, pagination, err := h.timetableService.ListTimetables(
		c.Request.Context(),
		tenantID.(uuid.UUID),
		classID,
		termID,
		params,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       timetables,
		"pagination": pagination,
	})
}

// PublishTimetable publishes a draft timetable
// @Summary Publish timetable
// @Description Publishes a draft timetable, making it visible to tutors and students
// @Tags timetables
// @Produce json
// @Param id path string true "Timetable ID"
// @Success 200 {object} domain.Timetable
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /timetables/{id}/publish [post]
func (h *TimetableHandler) PublishTimetable(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid timetable ID"})
		return
	}

	userID, _ := c.Get("user_id")
	role, _ := GetRoleFromContext(c)

	timetable, err := h.timetableService.PublishTimetable(
		c.Request.Context(),
		id,
		userID.(uuid.UUID),
		role,
		c.ClientIP(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, timetable)
}

// RegenerateTimetable regenerates a timetable for a class
// @Summary Regenerate timetable
// @Description Regenerates a timetable for a class, archiving the existing one
// @Tags timetables
// @Accept json
// @Produce json
// @Param request body regenerateTimetableRequest true "Regenerate request"
// @Success 201 {object} service.TimetableWithPeriods
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /timetables/regenerate [post]
func (h *TimetableHandler) RegenerateTimetable(c *gin.Context) {
	var req regenerateTimetableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant not found in context"})
		return
	}

	userID, _ := c.Get("user_id")
	role, _ := GetRoleFromContext(c)

	timetable, err := h.timetableService.RegenerateTimetable(
		c.Request.Context(),
		tenantID.(uuid.UUID),
		req.ClassID,
		req.TermID,
		req.Notes,
		userID.(uuid.UUID),
		role,
		c.ClientIP(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, timetable)
}

type regenerateTimetableRequest struct {
	ClassID uuid.UUID `json:"class_id" binding:"required"`
	TermID  uuid.UUID `json:"term_id" binding:"required"`
	Notes   *string   `json:"notes,omitempty"`
}

// GetTutorTimetable retrieves the timetable for a specific tutor
// @Summary Get tutor timetable
// @Description Retrieves all periods for a tutor in a term
// @Tags timetables
// @Produce json
// @Param tutor_id path string true "Tutor ID"
// @Param term_id query string true "Term ID"
// @Success 200 {array} domain.Period
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /timetables/tutor/{tutor_id} [get]
func (h *TimetableHandler) GetTutorTimetable(c *gin.Context) {
	tutorIDStr := c.Param("tutor_id")
	tutorID, err := uuid.Parse(tutorIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tutor ID"})
		return
	}

	termIDStr := c.Query("term_id")
	if termIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "term_id is required"})
		return
	}

	termID, err := uuid.Parse(termIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid term_id"})
		return
	}

	periods, err := h.timetableService.GetTutorTimetable(c.Request.Context(), tutorID, termID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": periods})
}

// GetClassTimetable retrieves the published timetable for a class and term
// @Summary Get class timetable
// @Description Retrieves the published timetable for a class in a term
// @Tags timetables
// @Produce json
// @Param class_id path string true "Class ID"
// @Param term_id query string true "Term ID"
// @Success 200 {object} service.TimetableWithPeriods
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /timetables/class/{class_id} [get]
func (h *TimetableHandler) GetClassTimetable(c *gin.Context) {
	classIDStr := c.Param("class_id")
	classID, err := uuid.Parse(classIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid class ID"})
		return
	}

	termIDStr := c.Query("term_id")
	if termIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "term_id is required"})
		return
	}

	termID, err := uuid.Parse(termIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid term_id"})
		return
	}

	timetable, err := h.timetableService.GetTimetableByClassAndTerm(c.Request.Context(), classID, termID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, timetable)
}

// CreateSwapRequest creates a period swap request
// @Summary Create swap request
// @Description Creates a request to swap periods between tutors
// @Tags timetables,swap-requests
// @Accept json
// @Produce json
// @Param request body service.SwapRequestInput true "Swap request"
// @Success 201 {object} domain.SwapRequest
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /timetables/swap-requests [post]
func (h *TimetableHandler) CreateSwapRequest(c *gin.Context) {
	var req service.SwapRequestInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant not found in context"})
		return
	}

	userID, _ := c.Get("user_id")
	role, _ := GetRoleFromContext(c)

	swapRequest, err := h.timetableService.CreateSwapRequest(
		c.Request.Context(),
		tenantID.(uuid.UUID),
		&req,
		userID.(uuid.UUID),
		role,
		c.ClientIP(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, swapRequest)
}

// GetSwapRequest retrieves a swap request by ID
// @Summary Get swap request
// @Description Retrieves a swap request by ID
// @Tags timetables,swap-requests
// @Produce json
// @Param id path string true "Swap request ID"
// @Success 200 {object} domain.SwapRequest
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /timetables/swap-requests/{id} [get]
func (h *TimetableHandler) GetSwapRequest(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid swap request ID"})
		return
	}

	swapRequest, err := h.timetableService.GetSwapRequest(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, swapRequest)
}

// ListSwapRequests lists swap requests
// @Summary List swap requests
// @Description Lists swap requests with optional filters
// @Tags timetables,swap-requests
// @Produce json
// @Param tutor_id query string false "Filter by tutor ID"
// @Param status query string false "Filter by status"
// @Param cursor query string false "Pagination cursor"
// @Param limit query int false "Number of items per page" default(20)
// @Success 200 {object} PaginatedResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /timetables/swap-requests [get]
func (h *TimetableHandler) ListSwapRequests(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant not found in context"})
		return
	}

	var tutorID *uuid.UUID
	if tutorIDStr := c.Query("tutor_id"); tutorIDStr != "" {
		id, err := uuid.Parse(tutorIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tutor_id"})
			return
		}
		tutorID = &id
	}

	var status *domain.SwapRequestStatus
	if statusStr := c.Query("status"); statusStr != "" {
		s := domain.SwapRequestStatus(statusStr)
		status = &s
	}

	params := repository.PaginationParams{Limit: 20, SortOrder: "DESC"}

	requests, pagination, err := h.timetableService.ListSwapRequests(
		c.Request.Context(),
		tenantID.(uuid.UUID),
		tutorID,
		status,
		params,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       requests,
		"pagination": pagination,
	})
}

// ListPendingSwapRequests lists pending swap requests for the current tutor
// @Summary List pending swap requests
// @Description Lists pending swap requests where the current user is the target tutor
// @Tags timetables,swap-requests
// @Produce json
// @Param cursor query string false "Pagination cursor"
// @Param limit query int false "Number of items per page" default(20)
// @Success 200 {object} PaginatedResponse
// @Failure 500 {object} ErrorResponse
// @Router /timetables/swap-requests/pending [get]
func (h *TimetableHandler) ListPendingSwapRequests(c *gin.Context) {
	userID, _ := c.Get("user_id")
	params := repository.PaginationParams{Limit: 20, SortOrder: "DESC"}

	requests, pagination, err := h.timetableService.ListPendingSwapRequestsForTutor(
		c.Request.Context(),
		userID.(uuid.UUID),
		params,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       requests,
		"pagination": pagination,
	})
}

// ListEscalatedSwapRequests lists escalated swap requests for admin review
// @Summary List escalated swap requests
// @Description Lists escalated swap requests that need admin review
// @Tags timetables,swap-requests
// @Produce json
// @Param cursor query string false "Pagination cursor"
// @Param limit query int false "Number of items per page" default(20)
// @Success 200 {object} PaginatedResponse
// @Failure 500 {object} ErrorResponse
// @Router /timetables/swap-requests/escalated [get]
func (h *TimetableHandler) ListEscalatedSwapRequests(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant not found in context"})
		return
	}

	params := repository.PaginationParams{Limit: 20, SortOrder: "DESC"}

	requests, pagination, err := h.timetableService.ListEscalatedSwapRequests(
		c.Request.Context(),
		tenantID.(uuid.UUID),
		params,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       requests,
		"pagination": pagination,
	})
}

// ApproveSwapRequest approves a swap request
// @Summary Approve swap request
// @Description Approves a pending swap request (by target tutor)
// @Tags timetables,swap-requests
// @Produce json
// @Param id path string true "Swap request ID"
// @Success 200 {object} domain.SwapRequest
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /timetables/swap-requests/{id}/approve [post]
func (h *TimetableHandler) ApproveSwapRequest(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid swap request ID"})
		return
	}

	userID, _ := c.Get("user_id")
	role, _ := GetRoleFromContext(c)

	swapRequest, err := h.timetableService.ApproveSwapRequest(
		c.Request.Context(),
		id,
		userID.(uuid.UUID),
		role,
		c.ClientIP(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, swapRequest)
}

// RejectSwapRequest rejects a swap request
// @Summary Reject swap request
// @Description Rejects a pending swap request (by target tutor)
// @Tags timetables,swap-requests
// @Accept json
// @Produce json
// @Param id path string true "Swap request ID"
// @Param request body rejectSwapRequest true "Rejection reason"
// @Success 200 {object} domain.SwapRequest
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /timetables/swap-requests/{id}/reject [post]
func (h *TimetableHandler) RejectSwapRequest(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid swap request ID"})
		return
	}

	var req rejectSwapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	role, _ := GetRoleFromContext(c)

	swapRequest, err := h.timetableService.RejectSwapRequest(
		c.Request.Context(),
		id,
		req.Reason,
		userID.(uuid.UUID),
		role,
		c.ClientIP(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, swapRequest)
}

type rejectSwapRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// EscalateSwapRequest escalates a rejected swap request
// @Summary Escalate swap request
// @Description Escalates a rejected swap request to admin (by requesting tutor)
// @Tags timetables,swap-requests
// @Accept json
// @Produce json
// @Param id path string true "Swap request ID"
// @Param request body escalateSwapRequest true "Escalation reason"
// @Success 200 {object} domain.SwapRequest
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /timetables/swap-requests/{id}/escalate [post]
func (h *TimetableHandler) EscalateSwapRequest(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid swap request ID"})
		return
	}

	var req escalateSwapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	role, _ := GetRoleFromContext(c)

	swapRequest, err := h.timetableService.EscalateSwapRequest(
		c.Request.Context(),
		id,
		req.Reason,
		userID.(uuid.UUID),
		role,
		c.ClientIP(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, swapRequest)
}

type escalateSwapRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// AdminOverrideSwapRequest allows admin to force approve an escalated swap
// @Summary Admin override swap request
// @Description Allows admin to force approve an escalated swap request
// @Tags timetables,swap-requests
// @Accept json
// @Produce json
// @Param id path string true "Swap request ID"
// @Param request body adminOverrideRequest true "Override reason"
// @Success 200 {object} domain.SwapRequest
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /timetables/swap-requests/{id}/override [post]
func (h *TimetableHandler) AdminOverrideSwapRequest(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid swap request ID"})
		return
	}

	var req adminOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	role, _ := GetRoleFromContext(c)

	swapRequest, err := h.timetableService.AdminOverrideSwapRequest(
		c.Request.Context(),
		id,
		req.Reason,
		userID.(uuid.UUID),
		role,
		c.ClientIP(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, swapRequest)
}

type adminOverrideRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// CancelSwapRequest cancels a swap request
// @Summary Cancel swap request
// @Description Cancels a pending or escalated swap request (by requesting tutor)
// @Tags timetables,swap-requests
// @Produce json
// @Param id path string true "Swap request ID"
// @Success 200 {object} domain.SwapRequest
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /timetables/swap-requests/{id}/cancel [post]
func (h *TimetableHandler) CancelSwapRequest(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid swap request ID"})
		return
	}

	userID, _ := c.Get("user_id")

	swapRequest, err := h.timetableService.CancelSwapRequest(
		c.Request.Context(),
		id,
		userID.(uuid.UUID),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, swapRequest)
}
