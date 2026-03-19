package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AuditAction represents the type of action being audited
type AuditAction string

const (
	// Tenant actions
	AuditActionTenantCreated    AuditAction = "TENANT_CREATED"
	AuditActionTenantUpdated    AuditAction = "TENANT_UPDATED"
	AuditActionTenantSuspended  AuditAction = "TENANT_SUSPENDED"
	AuditActionTenantReactivated AuditAction = "TENANT_REACTIVATED"

	// User actions
	AuditActionUserCreated     AuditAction = "USER_CREATED"
	AuditActionUserUpdated     AuditAction = "USER_UPDATED"
	AuditActionUserDeactivated AuditAction = "USER_DEACTIVATED"
	AuditActionUserReactivated AuditAction = "USER_REACTIVATED"
	AuditActionUserLogin       AuditAction = "USER_LOGIN"
	AuditActionUserLogout      AuditAction = "USER_LOGOUT"
	AuditActionRoleChanged     AuditAction = "ROLE_CHANGED"

	// Session/Term actions
	AuditActionSessionCreated  AuditAction = "SESSION_CREATED"
	AuditActionSessionUpdated  AuditAction = "SESSION_UPDATED"
	AuditActionSessionArchived AuditAction = "SESSION_ARCHIVED"
	AuditActionTermCreated     AuditAction = "TERM_CREATED"
	AuditActionTermUpdated     AuditAction = "TERM_UPDATED"
	AuditActionTermActivated   AuditAction = "TERM_ACTIVATED"
	AuditActionTermCompleted   AuditAction = "TERM_COMPLETED"

	// Class/Course actions
	AuditActionClassCreated       AuditAction = "CLASS_CREATED"
	AuditActionClassUpdated       AuditAction = "CLASS_UPDATED"
	AuditActionCourseCreated      AuditAction = "COURSE_CREATED"
	AuditActionCourseUpdated      AuditAction = "COURSE_UPDATED"
	AuditActionTutorReassigned    AuditAction = "TUTOR_REASSIGNED"

	// Enrollment actions
	AuditActionStudentEnrolled    AuditAction = "STUDENT_ENROLLED"
	AuditActionStudentTransferred AuditAction = "STUDENT_TRANSFERRED"
	AuditActionStudentWithdrawn   AuditAction = "STUDENT_WITHDRAWN"

	// Timetable actions
	AuditActionTimetableGenerated AuditAction = "TIMETABLE_GENERATED"
	AuditActionTimetablePublished AuditAction = "TIMETABLE_PUBLISHED"
	AuditActionPeriodSwapRequested AuditAction = "PERIOD_SWAP_REQUESTED"
	AuditActionPeriodSwapApproved  AuditAction = "PERIOD_SWAP_APPROVED"
	AuditActionPeriodSwapRejected  AuditAction = "PERIOD_SWAP_REJECTED"

	// Assessment actions
	AuditActionQuizCreated       AuditAction = "QUIZ_CREATED"
	AuditActionQuizPublished     AuditAction = "QUIZ_PUBLISHED"
	AuditActionQuizSubmitted     AuditAction = "QUIZ_SUBMITTED"
	AuditActionQuizGraded        AuditAction = "QUIZ_GRADED"
	AuditActionAssignmentCreated AuditAction = "ASSIGNMENT_CREATED"
	AuditActionAssignmentPublished AuditAction = "ASSIGNMENT_PUBLISHED"
	AuditActionAssignmentSubmitted AuditAction = "ASSIGNMENT_SUBMITTED"
	AuditActionAssignmentGraded    AuditAction = "ASSIGNMENT_GRADED"

	// Examination actions
	AuditActionExaminationCreated  AuditAction = "EXAMINATION_CREATED"
	AuditActionExaminationScheduled AuditAction = "EXAMINATION_SCHEDULED"
	AuditActionExaminationSubmitted AuditAction = "EXAMINATION_SUBMITTED"
	AuditActionExaminationGraded    AuditAction = "EXAMINATION_GRADED"
	AuditActionResultsPublished     AuditAction = "RESULTS_PUBLISHED"

	// Meeting actions
	AuditActionMeetingScheduled AuditAction = "MEETING_SCHEDULED"
	AuditActionMeetingStarted   AuditAction = "MEETING_STARTED"
	AuditActionMeetingEnded     AuditAction = "MEETING_ENDED"
	AuditActionMeetingCancelled AuditAction = "MEETING_CANCELLED"

	// Billing actions
	AuditActionInvoiceGenerated       AuditAction = "INVOICE_GENERATED"
	AuditActionInvoicePaid            AuditAction = "INVOICE_PAID"
	AuditActionBillingAdjustmentApplied AuditAction = "BILLING_ADJUSTMENT_APPLIED"
	AuditActionSubscriptionCancelled  AuditAction = "SUBSCRIPTION_CANCELLED"

	// Communication actions
	AuditActionEmailSent         AuditAction = "EMAIL_SENT"
	AuditActionNotificationSent  AuditAction = "NOTIFICATION_SENT"
)

// AuditResourceType represents the type of resource being audited
type AuditResourceType string

const (
	AuditResourceTenant       AuditResourceType = "TENANT"
	AuditResourceUser         AuditResourceType = "USER"
	AuditResourceSession      AuditResourceType = "SESSION"
	AuditResourceTerm         AuditResourceType = "TERM"
	AuditResourceClass        AuditResourceType = "CLASS"
	AuditResourceCourse       AuditResourceType = "COURSE"
	AuditResourceEnrollment   AuditResourceType = "ENROLLMENT"
	AuditResourceTimetable    AuditResourceType = "TIMETABLE"
	AuditResourcePeriod       AuditResourceType = "PERIOD"
	AuditResourceQuiz         AuditResourceType = "QUIZ"
	AuditResourceAssignment   AuditResourceType = "ASSIGNMENT"
	AuditResourceExamination  AuditResourceType = "EXAMINATION"
	AuditResourceProgress     AuditResourceType = "PROGRESS"
	AuditResourceMeeting      AuditResourceType = "MEETING"
	AuditResourceInvoice      AuditResourceType = "INVOICE"
	AuditResourceSubscription AuditResourceType = "SUBSCRIPTION"
	AuditResourceEmail        AuditResourceType = "EMAIL"
	AuditResourceNotification AuditResourceType = "NOTIFICATION"
)

// AuditLog represents an immutable audit trail entry
type AuditLog struct {
	ID               uuid.UUID         `json:"id" validate:"required,uuid"`
	TenantID         *uuid.UUID        `json:"tenant_id,omitempty" validate:"omitempty,uuid"` // NULL for platform-level actions
	ActorUserID      uuid.UUID         `json:"actor_user_id" validate:"required,uuid"`
	ActorRole        Role              `json:"actor_role" validate:"required,oneof=SUPER_ADMIN ADMIN TUTOR STUDENT"`
	Action           AuditAction       `json:"action" validate:"required"`
	ResourceType     AuditResourceType `json:"resource_type" validate:"required"`
	ResourceID       uuid.UUID         `json:"resource_id" validate:"required,uuid"`
	ResourceName     *string           `json:"resource_name,omitempty" validate:"omitempty,max=200"` // Human-readable resource name
	BeforeState      *json.RawMessage  `json:"before_state,omitempty"` // JSON snapshot of resource before change
	AfterState       *json.RawMessage  `json:"after_state,omitempty"`  // JSON snapshot of resource after change
	Changes          map[string]interface{} `json:"changes,omitempty"` // Field-level changes for updates
	IPAddress        string            `json:"ip_address" validate:"required"`
	UserAgent        *string           `json:"user_agent,omitempty" validate:"omitempty,max=500"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"` // Additional context
	IsSensitive      bool              `json:"is_sensitive"` // True for sensitive actions (billing, suspension, etc.)
	Timestamp        time.Time         `json:"timestamp" validate:"required"`
	CreatedAt        time.Time         `json:"created_at" validate:"required"`
}

// IsSensitiveAction returns true if the action is considered sensitive
func (a *AuditLog) IsSensitiveAction() bool {
	return a.IsSensitive
}

// GetChangeSummary returns a human-readable summary of changes
func (a *AuditLog) GetChangeSummary() string {
	if len(a.Changes) == 0 {
		return "No changes"
	}

	summary := ""
	for field, value := range a.Changes {
		summary += field + " changed to " + toString(value) + "; "
	}
	return summary
}

// toString converts an interface to string representation
func toString(v interface{}) string {
	if v == nil {
		return "null"
	}
	bytes, err := json.Marshal(v)
	if err != nil {
		return "unknown"
	}
	return string(bytes)
}

// NewAuditLog creates a new audit log entry
func NewAuditLog(
	tenantID *uuid.UUID,
	actorUserID uuid.UUID,
	actorRole Role,
	action AuditAction,
	resourceType AuditResourceType,
	resourceID uuid.UUID,
	ipAddress string,
) *AuditLog {
	now := time.Now()
	return &AuditLog{
		ID:          uuid.New(),
		TenantID:    tenantID,
		ActorUserID: actorUserID,
		ActorRole:   actorRole,
		Action:      action,
		ResourceType: resourceType,
		ResourceID:  resourceID,
		IPAddress:   ipAddress,
		IsSensitive: isSensitiveAction(action),
		Timestamp:   now,
		CreatedAt:   now,
	}
}

// isSensitiveAction determines if an action should be flagged as sensitive
func isSensitiveAction(action AuditAction) bool {
	sensitiveActions := map[AuditAction]bool{
		AuditActionTenantSuspended:         true,
		AuditActionTenantReactivated:       true,
		AuditActionUserDeactivated:         true,
		AuditActionRoleChanged:             true,
		AuditActionBillingAdjustmentApplied: true,
		AuditActionSubscriptionCancelled:   true,
		AuditActionStudentEnrolled:         true,
		AuditActionStudentWithdrawn:        true,
	}
	return sensitiveActions[action]
}

// WithBeforeState sets the before state snapshot
func (a *AuditLog) WithBeforeState(state interface{}) *AuditLog {
	bytes, _ := json.Marshal(state)
	rawMsg := json.RawMessage(bytes)
	a.BeforeState = &rawMsg
	return a
}

// WithAfterState sets the after state snapshot
func (a *AuditLog) WithAfterState(state interface{}) *AuditLog {
	bytes, _ := json.Marshal(state)
	rawMsg := json.RawMessage(bytes)
	a.AfterState = &rawMsg
	return a
}

// WithChanges sets the field-level changes
func (a *AuditLog) WithChanges(changes map[string]interface{}) *AuditLog {
	a.Changes = changes
	return a
}

// WithMetadata sets additional metadata
func (a *AuditLog) WithMetadata(metadata map[string]interface{}) *AuditLog {
	a.Metadata = metadata
	return a
}

// WithResourceName sets the human-readable resource name
func (a *AuditLog) WithResourceName(name string) *AuditLog {
	a.ResourceName = &name
	return a
}

// WithUserAgent sets the user agent
func (a *AuditLog) WithUserAgent(userAgent string) *AuditLog {
	a.UserAgent = &userAgent
	return a
}
