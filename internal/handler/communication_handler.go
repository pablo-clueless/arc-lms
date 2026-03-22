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

// CommunicationHandler handles email communication HTTP requests
type CommunicationHandler struct {
	communicationService *service.CommunicationService
}

// NewCommunicationHandler creates a new communication handler
func NewCommunicationHandler(communicationService *service.CommunicationService) *CommunicationHandler {
	return &CommunicationHandler{
		communicationService: communicationService,
	}
}

// ComposeEmail godoc
// @Summary Compose and send an email
// @Description Compose a new email as draft, schedule it, or send immediately
// @Tags Communications
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.ComposeEmailRequest true "Email composition data"
// @Success 201 {object} domain.Email
// @Router /communications/emails [post]
func (h *CommunicationHandler) ComposeEmail(c *gin.Context) {
	tenantID, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	var req service.ComposeEmailRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	email, err := h.communicationService.ComposeEmail(
		c.Request.Context(),
		tenantID,
		userID,
		role,
		&req,
		c.ClientIP(),
	)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusCreated, email)
}

// ListEmails godoc
// @Summary List emails
// @Description List emails for the tenant with optional filtering
// @Tags Communications
// @Security BearerAuth
// @Produce json
// @Param status query string false "Filter by status (DRAFT, SCHEDULED, SENDING, SENT, FAILED, CANCELLED)"
// @Param page query int false "Page number"
// @Param limit query int false "Number of results"
// @Success 200 {object} map[string]interface{}
// @Router /communications/emails [get]
func (h *CommunicationHandler) ListEmails(c *gin.Context) {
	tenantID, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	var status *domain.CommunicationStatus
	if statusStr := c.Query("status"); statusStr != "" {
		s := domain.CommunicationStatus(statusStr)
		status = &s
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

	// For tutors, only show their own emails
	var senderID *uuid.UUID
	if role == domain.RoleTutor {
		senderID = &userID
	}

	emails, pagination, err := h.communicationService.ListEmails(
		c.Request.Context(),
		tenantID,
		senderID,
		status,
		role,
		params,
	)
	if err != nil {
		errors.InternalError(c, "failed to list emails")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": emails, "pagination": pagination})
}

// GetEmail godoc
// @Summary Get an email
// @Description Get email details by ID
// @Tags Communications
// @Security BearerAuth
// @Produce json
// @Param id path string true "Email ID"
// @Success 200 {object} domain.Email
// @Router /communications/emails/{id} [get]
func (h *CommunicationHandler) GetEmail(c *gin.Context) {
	_, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid email ID", nil)
		return
	}

	email, err := h.communicationService.GetEmail(c.Request.Context(), id)
	if err != nil {
		errors.NotFound(c, "email not found")
		return
	}

	// Tutors can only view their own emails
	if role == domain.RoleTutor && email.SenderID != userID {
		errors.Forbidden(c, "you can only view your own emails")
		return
	}

	c.JSON(http.StatusOK, email)
}

// CancelEmail godoc
// @Summary Cancel an email
// @Description Cancel a scheduled or draft email
// @Tags Communications
// @Security BearerAuth
// @Produce json
// @Param id path string true "Email ID"
// @Success 200 {object} domain.Email
// @Router /communications/emails/{id}/cancel [post]
func (h *CommunicationHandler) CancelEmail(c *gin.Context) {
	_, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid email ID", nil)
		return
	}

	email, err := h.communicationService.CancelEmail(
		c.Request.Context(),
		id,
		userID,
		role,
		c.ClientIP(),
	)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, email)
}

// DeleteEmail godoc
// @Summary Delete a draft email
// @Description Delete a draft email
// @Tags Communications
// @Security BearerAuth
// @Param id path string true "Email ID"
// @Success 204 "No Content"
// @Router /communications/emails/{id} [delete]
func (h *CommunicationHandler) DeleteEmail(c *gin.Context) {
	_, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid email ID", nil)
		return
	}

	if err := h.communicationService.DeleteEmail(c.Request.Context(), id, userID, role); err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.Status(http.StatusNoContent)
}

// ScheduleEmailRequest represents a request to schedule an email
type ScheduleEmailRequest struct {
	ScheduledFor time.Time `json:"scheduled_for" validate:"required"`
}

// ScheduleEmail godoc
// @Summary Schedule an email
// @Description Schedule a draft email for future delivery
// @Tags Communications
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Email ID"
// @Param request body ScheduleEmailRequest true "Schedule data"
// @Success 200 {object} domain.Email
// @Router /communications/emails/{id}/schedule [post]
func (h *CommunicationHandler) ScheduleEmail(c *gin.Context) {
	_, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid email ID", nil)
		return
	}

	var req ScheduleEmailRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	email, err := h.communicationService.ScheduleEmail(
		c.Request.Context(),
		id,
		userID,
		role,
		req.ScheduledFor,
		c.ClientIP(),
	)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, email)
}

// SendEmailNow godoc
// @Summary Send an email immediately
// @Description Send a draft or scheduled email immediately
// @Tags Communications
// @Security BearerAuth
// @Produce json
// @Param id path string true "Email ID"
// @Success 200 {object} domain.Email
// @Router /communications/emails/{id}/send [post]
func (h *CommunicationHandler) SendEmailNow(c *gin.Context) {
	_, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid email ID", nil)
		return
	}

	email, err := h.communicationService.SendEmailNow(
		c.Request.Context(),
		id,
		userID,
		role,
		c.ClientIP(),
	)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, email)
}

// SendToCoTutors godoc
// @Summary Send email to co-tutors
// @Description Send an email to other tutors in the same class
// @Tags Communications
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.SendEmailToCoTutorsRequest true "Email data"
// @Success 201 {object} domain.Email
// @Router /communications/emails/co-tutors [post]
func (h *CommunicationHandler) SendToCoTutors(c *gin.Context) {
	tenantID, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	var req service.SendEmailToCoTutorsRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	email, err := h.communicationService.SendEmailToCoTutors(
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

	c.JSON(http.StatusCreated, email)
}

// GetEmailStatistics godoc
// @Summary Get email statistics
// @Description Get email statistics for the tenant
// @Tags Communications
// @Security BearerAuth
// @Produce json
// @Param start_date query string true "Start date (RFC3339)"
// @Param end_date query string true "End date (RFC3339)"
// @Success 200 {object} postgres.EmailStatistics
// @Router /communications/emails/statistics [get]
func (h *CommunicationHandler) GetEmailStatistics(c *gin.Context) {
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

	stats, err := h.communicationService.GetEmailStatistics(c.Request.Context(), tenantID, startDate, endDate)
	if err != nil {
		errors.InternalError(c, "failed to get statistics")
		return
	}

	c.JSON(http.StatusOK, stats)
}

// SearchEmails godoc
// @Summary Search emails
// @Description Search emails by subject or body content
// @Tags Communications
// @Security BearerAuth
// @Produce json
// @Param q query string true "Search term (min 3 characters)"
// @Param limit query int false "Number of results"
// @Success 200 {object} map[string]interface{}
// @Router /communications/emails/search [get]
func (h *CommunicationHandler) SearchEmails(c *gin.Context) {
	tenantID, _, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	searchTerm := c.Query("q")
	if searchTerm == "" {
		errors.BadRequest(c, "search term 'q' is required", nil)
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

	emails, pagination, err := h.communicationService.SearchEmails(c.Request.Context(), tenantID, searchTerm, params)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": emails, "pagination": pagination})
}

// RecipientPreviewRequest represents a request to preview recipients
type RecipientPreviewRequest struct {
	RecipientScope  domain.RecipientScope `json:"recipient_scope" validate:"required,oneof=ALL_USERS ALL_TUTORS ALL_STUDENTS CLASS COURSE SPECIFIC_USERS"`
	TargetClassID   *uuid.UUID            `json:"target_class_id,omitempty"`
	TargetCourseID  *uuid.UUID            `json:"target_course_id,omitempty"`
	SpecificUserIDs []uuid.UUID           `json:"specific_user_ids,omitempty"`
}

// PreviewRecipients godoc
// @Summary Preview email recipients
// @Description Get a preview of recipients for a given scope
// @Tags Communications
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body RecipientPreviewRequest true "Scope data"
// @Success 200 {object} map[string]interface{}
// @Router /communications/emails/preview-recipients [post]
func (h *CommunicationHandler) PreviewRecipients(c *gin.Context) {
	tenantID, _, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	var req RecipientPreviewRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	preview, total, err := h.communicationService.GetRecipientPreview(
		c.Request.Context(),
		tenantID,
		req.RecipientScope,
		req.TargetClassID,
		req.TargetCourseID,
		req.SpecificUserIDs,
	)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"preview":         preview,
		"total_recipients": total,
	})
}

// Helper method to get tenant and user IDs from context
// For SuperAdmins, tenant_id may not exist - they are platform-level users
func (h *CommunicationHandler) getTenantAndUserID(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
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
func (h *CommunicationHandler) getUserRole(c *gin.Context) domain.Role {
	role, ok := GetRoleFromContext(c)
	if !ok {
		return domain.RoleStudent
	}
	return role
}
