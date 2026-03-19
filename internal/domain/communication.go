package domain

import (
	"time"

	"github.com/google/uuid"
)

// CommunicationType represents the type of communication
type CommunicationType string

const (
	CommunicationTypeEmail        CommunicationType = "EMAIL"
	CommunicationTypeNotification CommunicationType = "NOTIFICATION"
	CommunicationTypeSMS          CommunicationType = "SMS"
)

// CommunicationStatus represents the delivery status
type CommunicationStatus string

const (
	CommunicationStatusDraft     CommunicationStatus = "DRAFT"
	CommunicationStatusScheduled CommunicationStatus = "SCHEDULED"
	CommunicationStatusSending   CommunicationStatus = "SENDING"
	CommunicationStatusSent      CommunicationStatus = "SENT"
	CommunicationStatusFailed    CommunicationStatus = "FAILED"
	CommunicationStatusCancelled CommunicationStatus = "CANCELLED"
)

// RecipientScope represents the recipient targeting scope
type RecipientScope string

const (
	RecipientScopeAllUsers      RecipientScope = "ALL_USERS"       // All users in tenant
	RecipientScopeAllTutors     RecipientScope = "ALL_TUTORS"      // All tutors in tenant
	RecipientScopeAllStudents   RecipientScope = "ALL_STUDENTS"    // All students in tenant
	RecipientScopeClass         RecipientScope = "CLASS"           // All users in specific class
	RecipientScopeCourse        RecipientScope = "COURSE"          // All students in specific course
	RecipientScopeSpecific      RecipientScope = "SPECIFIC_USERS"  // Specific user IDs
)

// DeliveryRecipient represents a single recipient's delivery status
type DeliveryRecipient struct {
	UserID       uuid.UUID               `json:"user_id" validate:"required,uuid"`
	Email        string                  `json:"email" validate:"required,email"`
	Status       CommunicationStatus     `json:"status" validate:"required,oneof=SCHEDULED SENDING SENT FAILED"`
	SentAt       *time.Time              `json:"sent_at,omitempty"`
	FailedAt     *time.Time              `json:"failed_at,omitempty"`
	FailureReason *string                `json:"failure_reason,omitempty" validate:"omitempty,max=500"`
	OpenedAt     *time.Time              `json:"opened_at,omitempty"`
	ClickedAt    *time.Time              `json:"clicked_at,omitempty"`
}

// Email represents an email communication
type Email struct {
	ID               uuid.UUID             `json:"id" validate:"required,uuid"`
	TenantID         uuid.UUID             `json:"tenant_id" validate:"required,uuid"`
	SenderID         uuid.UUID             `json:"sender_id" validate:"required,uuid"`
	Subject          string                `json:"subject" validate:"required,min=3,max=200"`
	Body             string                `json:"body" validate:"required,min=10"`
	HTMLBody         *string               `json:"html_body,omitempty"`
	RecipientScope   RecipientScope        `json:"recipient_scope" validate:"required,oneof=ALL_USERS ALL_TUTORS ALL_STUDENTS CLASS COURSE SPECIFIC_USERS"`
	TargetClassID    *uuid.UUID            `json:"target_class_id,omitempty" validate:"omitempty,uuid"`
	TargetCourseID   *uuid.UUID            `json:"target_course_id,omitempty" validate:"omitempty,uuid"`
	SpecificUserIDs  []uuid.UUID           `json:"specific_user_ids,omitempty"`
	Recipients       []DeliveryRecipient   `json:"recipients"`
	Status           CommunicationStatus   `json:"status" validate:"required,oneof=DRAFT SCHEDULED SENDING SENT FAILED CANCELLED"`
	ScheduledFor     *time.Time            `json:"scheduled_for,omitempty"`
	SentAt           *time.Time            `json:"sent_at,omitempty"`
	AttachmentURLs   []string              `json:"attachment_urls,omitempty"`
	TotalRecipients  int                   `json:"total_recipients"`
	SuccessCount     int                   `json:"success_count"`
	FailureCount     int                   `json:"failure_count"`
	CreatedAt        time.Time             `json:"created_at" validate:"required"`
	UpdatedAt        time.Time             `json:"updated_at" validate:"required"`
	CancelledAt      *time.Time            `json:"cancelled_at,omitempty"`
}

// IsDraft returns true if the email is in draft status
func (e *Email) IsDraft() bool {
	return e.Status == CommunicationStatusDraft
}

// IsScheduled returns true if the email is scheduled
func (e *Email) IsScheduled() bool {
	return e.Status == CommunicationStatusScheduled
}

// IsSent returns true if the email has been sent
func (e *Email) IsSent() bool {
	return e.Status == CommunicationStatusSent
}

// CanCancel returns true if the email can be cancelled
func (e *Email) CanCancel() bool {
	return e.Status == CommunicationStatusDraft || e.Status == CommunicationStatusScheduled
}

// Cancel marks the email as cancelled
func (e *Email) Cancel() {
	if e.CanCancel() {
		e.Status = CommunicationStatusCancelled
		now := time.Now()
		e.CancelledAt = &now
		e.UpdatedAt = now
	}
}

// MarkAsSent marks the email as sent
func (e *Email) MarkAsSent() {
	e.Status = CommunicationStatusSent
	now := time.Now()
	e.SentAt = &now
	e.UpdatedAt = now
}

// NotificationChannel represents the delivery channel for a notification
type NotificationChannel string

const (
	NotificationChannelInApp NotificationChannel = "IN_APP"
	NotificationChannelPush  NotificationChannel = "PUSH"
	NotificationChannelEmail NotificationChannel = "EMAIL"
	NotificationChannelSMS   NotificationChannel = "SMS"
)

// NotificationEventType represents the type of event triggering the notification
type NotificationEventType string

const (
	NotificationEventQuizPublished          NotificationEventType = "QUIZ_PUBLISHED"
	NotificationEventAssignmentPublished    NotificationEventType = "ASSIGNMENT_PUBLISHED"
	NotificationEventAssignmentDeadline     NotificationEventType = "ASSIGNMENT_DEADLINE_APPROACHING"
	NotificationEventExaminationScheduled   NotificationEventType = "EXAMINATION_SCHEDULED"
	NotificationEventExaminationWindowOpen  NotificationEventType = "EXAMINATION_WINDOW_OPEN"
	NotificationEventExaminationWindowClose NotificationEventType = "EXAMINATION_WINDOW_CLOSE"
	NotificationEventGradePublished         NotificationEventType = "GRADE_PUBLISHED"
	NotificationEventTimetablePublished     NotificationEventType = "TIMETABLE_PUBLISHED"
	NotificationEventTimetableUpdated       NotificationEventType = "TIMETABLE_UPDATED"
	NotificationEventMeetingScheduled       NotificationEventType = "MEETING_SCHEDULED"
	NotificationEventMeetingCancelled       NotificationEventType = "MEETING_CANCELLED"
	NotificationEventMeetingStarting        NotificationEventType = "MEETING_STARTING"
	NotificationEventInvoiceGenerated       NotificationEventType = "INVOICE_GENERATED"
	NotificationEventPaymentOverdue         NotificationEventType = "PAYMENT_OVERDUE"
	NotificationEventCustom                 NotificationEventType = "CUSTOM"
)

// NotificationPriority represents the priority level of a notification
type NotificationPriority string

const (
	NotificationPriorityLow      NotificationPriority = "LOW"
	NotificationPriorityNormal   NotificationPriority = "NORMAL"
	NotificationPriorityHigh     NotificationPriority = "HIGH"
	NotificationPriorityUrgent   NotificationPriority = "URGENT"
)

// Notification represents an in-app or push notification
type Notification struct {
	ID            uuid.UUID             `json:"id" validate:"required,uuid"`
	TenantID      uuid.UUID             `json:"tenant_id" validate:"required,uuid"`
	UserID        uuid.UUID             `json:"user_id" validate:"required,uuid"`
	EventType     NotificationEventType `json:"event_type" validate:"required"`
	Title         string                `json:"title" validate:"required,min=3,max=200"`
	Body          string                `json:"body" validate:"required,min=3,max=1000"`
	Channels      []NotificationChannel `json:"channels" validate:"required,min=1"`
	Priority      NotificationPriority  `json:"priority" validate:"required,oneof=LOW NORMAL HIGH URGENT"`
	ActionURL     *string               `json:"action_url,omitempty" validate:"omitempty,url"` // Deep link or URL
	ResourceType  *string               `json:"resource_type,omitempty"` // e.g., "quiz", "assignment", "meeting"
	ResourceID    *uuid.UUID            `json:"resource_id,omitempty" validate:"omitempty,uuid"`
	Read          bool                  `json:"is_read"`
	ReadAt        *time.Time            `json:"read_at,omitempty"`
	DeliveredAt   *time.Time            `json:"delivered_at,omitempty"`
	FailedAt      *time.Time            `json:"failed_at,omitempty"`
	FailureReason *string               `json:"failure_reason,omitempty" validate:"omitempty,max=500"`
	CreatedAt     time.Time             `json:"created_at" validate:"required"`
	UpdatedAt     time.Time             `json:"updated_at" validate:"required"`
}

// IsRead returns true if the notification has been read
func (n *Notification) IsRead() bool {
	return n.Read
}

// MarkAsRead marks the notification as read
func (n *Notification) MarkAsRead() {
	if !n.Read {
		n.Read = true
		now := time.Now()
		n.ReadAt = &now
		n.UpdatedAt = now
	}
}

// MarkAsDelivered marks the notification as delivered
func (n *Notification) MarkAsDelivered() {
	now := time.Now()
	n.DeliveredAt = &now
	n.UpdatedAt = now
}

// MarkAsFailed marks the notification as failed with a reason
func (n *Notification) MarkAsFailed(reason string) {
	n.FailureReason = &reason
	now := time.Now()
	n.FailedAt = &now
	n.UpdatedAt = now
}
