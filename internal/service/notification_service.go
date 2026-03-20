package service

import (
	"context"
	"fmt"
	"time"

	"arc-lms/internal/domain"
	ws "arc-lms/internal/pkg/websocket"
	"arc-lms/internal/repository"
	"arc-lms/internal/repository/postgres"

	"github.com/google/uuid"
)

// NotificationService handles notification operations
type NotificationService struct {
	notificationRepo *postgres.NotificationRepository
	userRepo         *postgres.UserRepository
	wsHub            *ws.Hub
}

// NewNotificationService creates a new notification service
func NewNotificationService(
	notificationRepo *postgres.NotificationRepository,
	userRepo *postgres.UserRepository,
	wsHub *ws.Hub,
) *NotificationService {
	return &NotificationService{
		notificationRepo: notificationRepo,
		userRepo:         userRepo,
		wsHub:            wsHub,
	}
}

// CreateNotificationRequest represents a request to create a notification
type CreateNotificationRequest struct {
	UserID       uuid.UUID                      `json:"user_id" validate:"required"`
	EventType    domain.NotificationEventType   `json:"event_type" validate:"required"`
	Title        string                         `json:"title" validate:"required,min=3,max=200"`
	Body         string                         `json:"body" validate:"required,min=3,max=1000"`
	Channels     []domain.NotificationChannel   `json:"channels" validate:"required,min=1"`
	Priority     domain.NotificationPriority    `json:"priority" validate:"required"`
	ActionURL    *string                        `json:"action_url,omitempty"`
	ResourceType *string                        `json:"resource_type,omitempty"`
	ResourceID   *uuid.UUID                     `json:"resource_id,omitempty"`
}

// CreateNotification creates a new notification
func (s *NotificationService) CreateNotification(
	ctx context.Context,
	tenantID uuid.UUID,
	req *CreateNotificationRequest,
) (*domain.Notification, error) {
	now := time.Now()

	notification := &domain.Notification{
		ID:           uuid.New(),
		TenantID:     tenantID,
		UserID:       req.UserID,
		EventType:    req.EventType,
		Title:        req.Title,
		Body:         req.Body,
		Channels:     req.Channels,
		Priority:     req.Priority,
		ActionURL:    req.ActionURL,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		Read:         false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.notificationRepo.Create(ctx, notification); err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	// Broadcast via WebSocket for instant delivery
	s.broadcastNotification(notification)

	return notification, nil
}

// broadcastNotification sends a notification via WebSocket to the user
func (s *NotificationService) broadcastNotification(notification *domain.Notification) {
	if s.wsHub == nil {
		return
	}

	s.wsHub.SendToUser(notification.UserID, &ws.Message{
		Type:    ws.MessageTypeNotification,
		Payload: notification,
	})
}

// broadcastNotifications sends multiple notifications via WebSocket
func (s *NotificationService) broadcastNotifications(notifications []*domain.Notification) {
	if s.wsHub == nil {
		return
	}

	for _, notification := range notifications {
		s.broadcastNotification(notification)
	}
}

// SendNotificationToUsers sends a notification to multiple users
func (s *NotificationService) SendNotificationToUsers(
	ctx context.Context,
	tenantID uuid.UUID,
	userIDs []uuid.UUID,
	eventType domain.NotificationEventType,
	title string,
	body string,
	channels []domain.NotificationChannel,
	priority domain.NotificationPriority,
	actionURL *string,
	resourceType *string,
	resourceID *uuid.UUID,
) error {
	now := time.Now()
	notifications := make([]*domain.Notification, len(userIDs))

	for i, userID := range userIDs {
		notifications[i] = &domain.Notification{
			ID:           uuid.New(),
			TenantID:     tenantID,
			UserID:       userID,
			EventType:    eventType,
			Title:        title,
			Body:         body,
			Channels:     channels,
			Priority:     priority,
			ActionURL:    actionURL,
			ResourceType: resourceType,
			ResourceID:   resourceID,
			Read:         false,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
	}

	if err := s.notificationRepo.CreateBatch(ctx, notifications); err != nil {
		return fmt.Errorf("failed to send notifications: %w", err)
	}

	// Broadcast via WebSocket for instant delivery
	s.broadcastNotifications(notifications)

	return nil
}

// GetNotification retrieves a notification by ID
func (s *NotificationService) GetNotification(ctx context.Context, id uuid.UUID) (*domain.Notification, error) {
	notification, err := s.notificationRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}
	return notification, nil
}

// ListUserNotifications retrieves notifications for a user
func (s *NotificationService) ListUserNotifications(
	ctx context.Context,
	userID uuid.UUID,
	unreadOnly bool,
	params repository.PaginationParams,
) ([]*domain.Notification, *repository.PaginatedResult, error) {
	notifications, pagination, err := s.notificationRepo.ListByUser(ctx, userID, unreadOnly, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list notifications: %w", err)
	}
	return notifications, pagination, nil
}

// MarkAsRead marks a notification as read
func (s *NotificationService) MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	// First verify the notification belongs to the user
	notification, err := s.notificationRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get notification: %w", err)
	}

	if notification.UserID != userID {
		return fmt.Errorf("notification does not belong to user")
	}

	if err := s.notificationRepo.MarkAsRead(ctx, id); err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	return nil
}

// MarkAllAsRead marks all notifications as read for a user
func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	count, err := s.notificationRepo.MarkAllAsRead(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to mark all notifications as read: %w", err)
	}
	return count, nil
}

// GetUnreadCount returns the count of unread notifications for a user
func (s *NotificationService) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	count, err := s.notificationRepo.GetUnreadCount(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}
	return count, nil
}

// DeleteNotification deletes a notification
func (s *NotificationService) DeleteNotification(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	// First verify the notification belongs to the user
	notification, err := s.notificationRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get notification: %w", err)
	}

	if notification.UserID != userID {
		return fmt.Errorf("notification does not belong to user")
	}

	if err := s.notificationRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete notification: %w", err)
	}

	return nil
}

// NotifyQuizPublished sends notifications when a quiz is published
func (s *NotificationService) NotifyQuizPublished(
	ctx context.Context,
	tenantID uuid.UUID,
	studentIDs []uuid.UUID,
	quizID uuid.UUID,
	quizTitle string,
	courseName string,
) error {
	title := "New Quiz Available"
	body := fmt.Sprintf("A new quiz '%s' has been published in %s", quizTitle, courseName)
	actionURL := fmt.Sprintf("/quizzes/%s", quizID.String())
	resourceType := "QUIZ"

	return s.SendNotificationToUsers(
		ctx,
		tenantID,
		studentIDs,
		domain.NotificationEventQuizPublished,
		title,
		body,
		[]domain.NotificationChannel{domain.NotificationChannelInApp},
		domain.NotificationPriorityNormal,
		&actionURL,
		&resourceType,
		&quizID,
	)
}

// NotifyAssignmentPublished sends notifications when an assignment is published
func (s *NotificationService) NotifyAssignmentPublished(
	ctx context.Context,
	tenantID uuid.UUID,
	studentIDs []uuid.UUID,
	assignmentID uuid.UUID,
	assignmentTitle string,
	courseName string,
	dueDate time.Time,
) error {
	title := "New Assignment"
	body := fmt.Sprintf("Assignment '%s' has been posted in %s. Due: %s",
		assignmentTitle, courseName, dueDate.Format("Jan 2, 2006"))
	actionURL := fmt.Sprintf("/assignments/%s", assignmentID.String())
	resourceType := "ASSIGNMENT"

	return s.SendNotificationToUsers(
		ctx,
		tenantID,
		studentIDs,
		domain.NotificationEventAssignmentPublished,
		title,
		body,
		[]domain.NotificationChannel{domain.NotificationChannelInApp},
		domain.NotificationPriorityNormal,
		&actionURL,
		&resourceType,
		&assignmentID,
	)
}

// NotifyGradePublished sends notification when grades are published
func (s *NotificationService) NotifyGradePublished(
	ctx context.Context,
	tenantID uuid.UUID,
	studentID uuid.UUID,
	assessmentType string,
	assessmentTitle string,
	courseName string,
) error {
	title := "Grade Published"
	body := fmt.Sprintf("Your grade for %s '%s' in %s has been published",
		assessmentType, assessmentTitle, courseName)

	return s.SendNotificationToUsers(
		ctx,
		tenantID,
		[]uuid.UUID{studentID},
		domain.NotificationEventGradePublished,
		title,
		body,
		[]domain.NotificationChannel{domain.NotificationChannelInApp},
		domain.NotificationPriorityNormal,
		nil,
		nil,
		nil,
	)
}

// NotifyMeetingScheduled sends notifications when a meeting is scheduled
func (s *NotificationService) NotifyMeetingScheduled(
	ctx context.Context,
	tenantID uuid.UUID,
	participantIDs []uuid.UUID,
	meetingID uuid.UUID,
	meetingTitle string,
	scheduledTime time.Time,
) error {
	title := "Meeting Scheduled"
	body := fmt.Sprintf("'%s' has been scheduled for %s",
		meetingTitle, scheduledTime.Format("Jan 2, 2006 at 3:04 PM"))
	actionURL := fmt.Sprintf("/meetings/%s", meetingID.String())
	resourceType := "MEETING"

	return s.SendNotificationToUsers(
		ctx,
		tenantID,
		participantIDs,
		domain.NotificationEventMeetingScheduled,
		title,
		body,
		[]domain.NotificationChannel{domain.NotificationChannelInApp},
		domain.NotificationPriorityHigh,
		&actionURL,
		&resourceType,
		&meetingID,
	)
}
