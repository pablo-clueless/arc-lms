package handler

import (
	"net/http"
	"time"

	"arc-lms/internal/domain"
	"arc-lms/internal/pkg/errors"
	"arc-lms/internal/pkg/validator"
	"arc-lms/internal/repository"
	"arc-lms/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MeetingHandler handles meeting HTTP requests
type MeetingHandler struct {
	meetingService *service.MeetingService
}

// NewMeetingHandler creates a new meeting handler
func NewMeetingHandler(meetingService *service.MeetingService) *MeetingHandler {
	return &MeetingHandler{
		meetingService: meetingService,
	}
}

// ListMeetings godoc
// @Summary List meetings
// @Tags Meetings
// @Security BearerAuth
// @Produce json
// @Param status query string false "Filter by status (SCHEDULED, LIVE, ENDED, CANCELLED)"
// @Param cursor query string false "Pagination cursor"
// @Param limit query int false "Number of results"
// @Success 200 {object} map[string]interface{}
// @Router /meetings [get]
func (h *MeetingHandler) ListMeetings(c *gin.Context) {
	tenantID, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	var status *domain.MeetingStatus
	if statusStr := c.Query("status"); statusStr != "" {
		s := domain.MeetingStatus(statusStr)
		status = &s
	}

	params := repository.PaginationParams{Limit: 50, SortOrder: "DESC"}
	if cursorStr := c.Query("cursor"); cursorStr != "" {
		cursor, _ := uuid.Parse(cursorStr)
		params.Cursor = &cursor
	}

	var meetings []*domain.Meeting
	var pagination *repository.PaginatedResult
	var err error

	// Students see only upcoming meetings for their class
	if role == domain.RoleStudent {
		upcoming, err := h.meetingService.ListUpcomingMeetingsForStudent(c.Request.Context(), userID, 50)
		if err != nil {
			errors.InternalError(c, "failed to list meetings")
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": upcoming})
		return
	}

	// Tutors see their own meetings, admins see all
	if role == domain.RoleTutor {
		meetings, pagination, err = h.meetingService.ListMeetingsByTutor(c.Request.Context(), userID, params)
	} else {
		meetings, pagination, err = h.meetingService.ListMeetings(c.Request.Context(), tenantID, status, params)
	}

	if err != nil {
		errors.InternalError(c, "failed to list meetings")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": meetings, "pagination": pagination})
}

// ScheduleMeeting godoc
// @Summary Schedule a new meeting
// @Tags Meetings
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.ScheduleMeetingRequest true "Meeting data"
// @Success 201 {object} domain.Meeting
// @Router /meetings [post]
func (h *MeetingHandler) ScheduleMeeting(c *gin.Context) {
	tenantID, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	var req service.ScheduleMeetingRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	meeting, err := h.meetingService.ScheduleMeeting(
		c.Request.Context(),
		tenantID,
		userID,
		&req,
		c.ClientIP(),
	)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusCreated, meeting)
}

// GetMeeting godoc
// @Summary Get a meeting
// @Tags Meetings
// @Security BearerAuth
// @Produce json
// @Param id path string true "Meeting ID"
// @Success 200 {object} domain.Meeting
// @Router /meetings/{id} [get]
func (h *MeetingHandler) GetMeeting(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid meeting ID", nil)
		return
	}

	meeting, err := h.meetingService.GetMeeting(c.Request.Context(), id)
	if err != nil {
		errors.NotFound(c, "meeting not found")
		return
	}

	c.JSON(http.StatusOK, meeting)
}

// UpdateMeeting godoc
// @Summary Update a meeting
// @Tags Meetings
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Meeting ID"
// @Param request body service.UpdateMeetingRequest true "Meeting data"
// @Success 200 {object} domain.Meeting
// @Router /meetings/{id} [put]
func (h *MeetingHandler) UpdateMeeting(c *gin.Context) {
	_, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid meeting ID", nil)
		return
	}

	var req service.UpdateMeetingRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	meeting, err := h.meetingService.UpdateMeeting(
		c.Request.Context(),
		id,
		userID,
		&req,
		c.ClientIP(),
	)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, meeting)
}

// StartMeeting godoc
// @Summary Start a meeting
// @Tags Meetings
// @Security BearerAuth
// @Param id path string true "Meeting ID"
// @Success 200 {object} domain.Meeting
// @Router /meetings/{id}/start [post]
func (h *MeetingHandler) StartMeeting(c *gin.Context) {
	_, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid meeting ID", nil)
		return
	}

	meeting, err := h.meetingService.StartMeeting(
		c.Request.Context(),
		id,
		userID,
		c.ClientIP(),
	)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, meeting)
}

// EndMeeting godoc
// @Summary End a meeting
// @Tags Meetings
// @Security BearerAuth
// @Param id path string true "Meeting ID"
// @Success 200 {object} domain.Meeting
// @Router /meetings/{id}/end [post]
func (h *MeetingHandler) EndMeeting(c *gin.Context) {
	_, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid meeting ID", nil)
		return
	}

	meeting, err := h.meetingService.EndMeeting(
		c.Request.Context(),
		id,
		userID,
		c.ClientIP(),
	)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, meeting)
}

// CancelMeeting godoc
// @Summary Cancel a meeting
// @Tags Meetings
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Meeting ID"
// @Param request body service.CancelMeetingRequest true "Cancellation reason"
// @Success 200 {object} domain.Meeting
// @Router /meetings/{id}/cancel [post]
func (h *MeetingHandler) CancelMeeting(c *gin.Context) {
	_, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid meeting ID", nil)
		return
	}

	var req service.CancelMeetingRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	meeting, err := h.meetingService.CancelMeeting(
		c.Request.Context(),
		id,
		userID,
		role,
		&req,
		c.ClientIP(),
	)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, meeting)
}

// JoinMeeting godoc
// @Summary Get meeting join URL
// @Tags Meetings
// @Security BearerAuth
// @Param id path string true "Meeting ID"
// @Success 200 {object} map[string]interface{}
// @Router /meetings/{id}/join [get]
func (h *MeetingHandler) JoinMeeting(c *gin.Context) {
	_, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid meeting ID", nil)
		return
	}

	joinURL, err := h.meetingService.GetMeetingJoinURL(c.Request.Context(), id, userID)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	// Record participant join
	_ = h.meetingService.RecordParticipantJoin(c.Request.Context(), id, userID)

	c.JSON(http.StatusOK, gin.H{"join_url": joinURL})
}

// RecordParticipantJoin godoc
// @Summary Record participant join event
// @Tags Meetings
// @Security BearerAuth
// @Param id path string true "Meeting ID"
// @Success 200 {object} map[string]interface{}
// @Router /meetings/{id}/participants/join [post]
func (h *MeetingHandler) RecordParticipantJoin(c *gin.Context) {
	_, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid meeting ID", nil)
		return
	}

	if err := h.meetingService.RecordParticipantJoin(c.Request.Context(), id, userID); err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "participant join recorded"})
}

// RecordParticipantLeaveRequest represents a request to record participant leave
type RecordParticipantLeaveRequest struct {
	DurationMinutes int `json:"duration_minutes" validate:"required,min=0"`
}

// RecordParticipantLeave godoc
// @Summary Record participant leave event
// @Tags Meetings
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Meeting ID"
// @Param request body RecordParticipantLeaveRequest true "Duration data"
// @Success 200 {object} map[string]interface{}
// @Router /meetings/{id}/participants/leave [post]
func (h *MeetingHandler) RecordParticipantLeave(c *gin.Context) {
	_, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid meeting ID", nil)
		return
	}

	var req RecordParticipantLeaveRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	if err := h.meetingService.RecordParticipantLeave(c.Request.Context(), id, userID, req.DurationMinutes); err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "participant leave recorded"})
}

// AddRecording godoc
// @Summary Add recording to a meeting
// @Tags Meetings
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Meeting ID"
// @Param request body service.AddRecordingRequest true "Recording data"
// @Success 200 {object} domain.Meeting
// @Router /meetings/{id}/recording [post]
func (h *MeetingHandler) AddRecording(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid meeting ID", nil)
		return
	}

	var req service.AddRecordingRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	meeting, err := h.meetingService.AddRecording(c.Request.Context(), id, &req)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, meeting)
}

// ListMeetingsByClass godoc
// @Summary List meetings for a class
// @Tags Meetings
// @Security BearerAuth
// @Produce json
// @Param class_id path string true "Class ID"
// @Param cursor query string false "Pagination cursor"
// @Param limit query int false "Number of results"
// @Success 200 {object} map[string]interface{}
// @Router /meetings/class/{class_id} [get]
func (h *MeetingHandler) ListMeetingsByClass(c *gin.Context) {
	classID, err := uuid.Parse(c.Param("class_id"))
	if err != nil {
		errors.BadRequest(c, "invalid class ID", nil)
		return
	}

	params := repository.PaginationParams{Limit: 50, SortOrder: "DESC"}
	if cursorStr := c.Query("cursor"); cursorStr != "" {
		cursor, _ := uuid.Parse(cursorStr)
		params.Cursor = &cursor
	}

	meetings, pagination, err := h.meetingService.ListMeetingsByClass(c.Request.Context(), classID, params)
	if err != nil {
		errors.InternalError(c, "failed to list meetings")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": meetings, "pagination": pagination})
}

// ListUpcomingMeetings godoc
// @Summary List upcoming meetings
// @Tags Meetings
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Number of results (default 10)"
// @Success 200 {object} map[string]interface{}
// @Router /meetings/upcoming [get]
func (h *MeetingHandler) ListUpcomingMeetings(c *gin.Context) {
	tenantID, _, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if _, err := c.GetQuery("limit"); err {
			limit = 10
		}
	}

	meetings, err := h.meetingService.ListUpcomingMeetings(c.Request.Context(), tenantID, limit)
	if err != nil {
		errors.InternalError(c, "failed to list meetings")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": meetings})
}

// ListLiveMeetings godoc
// @Summary List currently live meetings
// @Tags Meetings
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /meetings/live [get]
func (h *MeetingHandler) ListLiveMeetings(c *gin.Context) {
	tenantID, _, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	meetings, err := h.meetingService.ListLiveMeetings(c.Request.Context(), tenantID)
	if err != nil {
		errors.InternalError(c, "failed to list meetings")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": meetings})
}

// GetMeetingStatistics godoc
// @Summary Get meeting statistics
// @Tags Meetings
// @Security BearerAuth
// @Produce json
// @Param start_date query string true "Start date (RFC3339)"
// @Param end_date query string true "End date (RFC3339)"
// @Success 200 {object} postgres.MeetingStatistics
// @Router /meetings/statistics [get]
func (h *MeetingHandler) GetMeetingStatistics(c *gin.Context) {
	tenantID, _, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	if startDateStr == "" || endDateStr == "" {
		errors.BadRequest(c, "start_date and end_date are required", nil)
		return
	}

	startDate, err := time.Parse(time.RFC3339, startDateStr)
	if err != nil {
		errors.BadRequest(c, "invalid start_date format", nil)
		return
	}

	endDate, err := time.Parse(time.RFC3339, endDateStr)
	if err != nil {
		errors.BadRequest(c, "invalid end_date format", nil)
		return
	}

	stats, err := h.meetingService.GetMeetingStatistics(c.Request.Context(), tenantID, startDate, endDate)
	if err != nil {
		errors.InternalError(c, "failed to get statistics")
		return
	}

	c.JSON(http.StatusOK, stats)
}

// Helper method to get tenant and user IDs from context
// For SuperAdmins, tenant_id may not exist - they are platform-level users
func (h *MeetingHandler) getTenantAndUserID(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
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
func (h *MeetingHandler) getUserRole(c *gin.Context) domain.Role {
	role, ok := GetRoleFromContext(c)
	if !ok {
		return domain.RoleStudent
	}
	return role
}
