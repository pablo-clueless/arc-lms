package handler

import (
	"net/http"

	"arc-lms/internal/pkg/errors"
	"arc-lms/internal/repository"
	"arc-lms/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// NotificationHandler handles notification HTTP requests
type NotificationHandler struct {
	notificationService *service.NotificationService
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(notificationService *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
	}
}

// ListNotificationsRequest represents query parameters for listing notifications
type ListNotificationsRequest struct {
	UnreadOnly bool   `form:"unread_only"`
	Cursor     string `form:"cursor"`
	Limit      int    `form:"limit,default=50"`
}

// ListNotifications godoc
// @Summary List notifications
// @Description Get paginated list of notifications for the authenticated user
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Param unread_only query bool false "Only return unread notifications"
// @Param cursor query string false "Pagination cursor"
// @Param limit query int false "Number of results (default 50, max 100)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Router /notifications [get]
func (h *NotificationHandler) ListNotifications(c *gin.Context) {
	// Get user ID from context
	userIDValue, exists := c.Get("user_id")
	if !exists {
		errors.Unauthorized(c, "user not authenticated")
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		errors.BadRequest(c, "invalid user ID format", nil)
		return
	}

	// Parse query parameters
	var req ListNotificationsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		errors.BadRequest(c, "invalid query parameters", nil)
		return
	}

	// Build pagination params
	params := repository.PaginationParams{
		Limit:     req.Limit,
		SortOrder: "DESC",
	}

	if req.Cursor != "" {
		cursor, err := uuid.Parse(req.Cursor)
		if err != nil {
			errors.BadRequest(c, "invalid cursor format", nil)
			return
		}
		params.Cursor = &cursor
	}

	// Get notifications
	notifications, pagination, err := h.notificationService.ListUserNotifications(
		c.Request.Context(),
		userID,
		req.UnreadOnly,
		params,
	)
	if err != nil {
		errors.InternalError(c, "failed to retrieve notifications")
		return
	}

	// Get unread count
	unreadCount, _ := h.notificationService.GetUnreadCount(c.Request.Context(), userID)

	c.JSON(http.StatusOK, gin.H{
		"data":         notifications,
		"pagination":   pagination,
		"unread_count": unreadCount,
	})
}

// GetNotification godoc
// @Summary Get notification by ID
// @Description Get a specific notification
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Param id path string true "Notification ID"
// @Success 200 {object} domain.Notification
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /notifications/{id} [get]
func (h *NotificationHandler) GetNotification(c *gin.Context) {
	// Get user ID from context
	userIDValue, exists := c.Get("user_id")
	if !exists {
		errors.Unauthorized(c, "user not authenticated")
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		errors.BadRequest(c, "invalid user ID format", nil)
		return
	}

	// Parse notification ID
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		errors.BadRequest(c, "invalid notification ID format", nil)
		return
	}

	// Get notification
	notification, err := h.notificationService.GetNotification(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			errors.NotFound(c, "notification not found")
			return
		}
		errors.InternalError(c, "failed to retrieve notification")
		return
	}

	// Verify ownership
	if notification.UserID != userID {
		errors.Forbidden(c, "cannot access notifications belonging to other users")
		return
	}

	c.JSON(http.StatusOK, notification)
}

// MarkAsRead godoc
// @Summary Mark notification as read
// @Description Mark a specific notification as read
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Param id path string true "Notification ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /notifications/{id}/read [post]
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	// Get user ID from context
	userIDValue, exists := c.Get("user_id")
	if !exists {
		errors.Unauthorized(c, "user not authenticated")
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		errors.BadRequest(c, "invalid user ID format", nil)
		return
	}

	// Parse notification ID
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		errors.BadRequest(c, "invalid notification ID format", nil)
		return
	}

	// Mark as read
	if err := h.notificationService.MarkAsRead(c.Request.Context(), id, userID); err != nil {
		if err == repository.ErrNotFound {
			errors.NotFound(c, "notification not found")
			return
		}
		if err.Error() == "notification does not belong to user" {
			errors.Forbidden(c, "cannot modify notifications belonging to other users")
			return
		}
		errors.InternalError(c, "failed to mark notification as read")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "notification marked as read",
	})
}

// MarkAllAsRead godoc
// @Summary Mark all notifications as read
// @Description Mark all notifications as read for the authenticated user
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} errors.ErrorResponse
// @Router /notifications/mark-all-read [post]
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	// Get user ID from context
	userIDValue, exists := c.Get("user_id")
	if !exists {
		errors.Unauthorized(c, "user not authenticated")
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		errors.BadRequest(c, "invalid user ID format", nil)
		return
	}

	// Mark all as read
	count, err := h.notificationService.MarkAllAsRead(c.Request.Context(), userID)
	if err != nil {
		errors.InternalError(c, "failed to mark notifications as read")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "all notifications marked as read",
		"updated_count": count,
	})
}

// GetUnreadCount godoc
// @Summary Get unread notification count
// @Description Get the count of unread notifications for the authenticated user
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} errors.ErrorResponse
// @Router /notifications/unread-count [get]
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	// Get user ID from context
	userIDValue, exists := c.Get("user_id")
	if !exists {
		errors.Unauthorized(c, "user not authenticated")
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		errors.BadRequest(c, "invalid user ID format", nil)
		return
	}

	// Get count
	count, err := h.notificationService.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		errors.InternalError(c, "failed to get unread count")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"unread_count": count,
	})
}

// DeleteNotification godoc
// @Summary Delete notification
// @Description Delete a specific notification
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Param id path string true "Notification ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /notifications/{id} [delete]
func (h *NotificationHandler) DeleteNotification(c *gin.Context) {
	// Get user ID from context
	userIDValue, exists := c.Get("user_id")
	if !exists {
		errors.Unauthorized(c, "user not authenticated")
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		errors.BadRequest(c, "invalid user ID format", nil)
		return
	}

	// Parse notification ID
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		errors.BadRequest(c, "invalid notification ID format", nil)
		return
	}

	// Delete notification
	if err := h.notificationService.DeleteNotification(c.Request.Context(), id, userID); err != nil {
		if err == repository.ErrNotFound {
			errors.NotFound(c, "notification not found")
			return
		}
		if err.Error() == "notification does not belong to user" {
			errors.Forbidden(c, "cannot delete notifications belonging to other users")
			return
		}
		errors.InternalError(c, "failed to delete notification")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "notification deleted",
	})
}
