package domain

import (
	"time"

	"github.com/google/uuid"
)

// TenantStatus represents the operational status of a tenant
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "ACTIVE"
	TenantStatusSuspended TenantStatus = "SUSPENDED"
)

// SchoolType represents the type of educational institution
type SchoolType string

const (
	SchoolTypePrimary   SchoolType = "PRIMARY"
	SchoolTypeSecondary SchoolType = "SECONDARY"
	SchoolTypeCombined  SchoolType = "COMBINED"
)

// TenantConfiguration holds tenant-specific settings
type TenantConfiguration struct {
	Timezone              string            `json:"timezone" validate:"required"`
	SchoolLevel           SchoolType        `json:"school_level" validate:"required,oneof=PRIMARY SECONDARY COMBINED"`
	PeriodDuration        int               `json:"period_duration" validate:"required,min=20,max=90"`        // in minutes
	DailyPeriodLimit      int               `json:"daily_period_limit" validate:"required,min=4,max=12"`      // max periods per day
	MaxPeriodsPerWeek     map[string]int    `json:"max_periods_per_week"`                                     // course_name -> max weekly periods
	GradeWeighting        map[string]int    `json:"grade_weighting" validate:"required"`                      // e.g., {"continuous_assessment": 40, "examination": 60}
	AttendanceThreshold   int               `json:"attendance_threshold" validate:"required,min=0,max=100"`   // minimum percentage
	InvoiceGracePeriod    int               `json:"invoice_grace_period" validate:"required,min=1,max=60"`    // in days
	SuspensionThreshold   int               `json:"suspension_threshold" validate:"required,min=1,max=90"`    // in days
	BrandingAssets        map[string]string `json:"branding_assets"`                                          // URLs to logos, colors, etc.
	CommunicationPrefs    map[string]bool   `json:"communication_prefs"`                                      // email_enabled, sms_enabled, etc.
	SupportedClasses      []string          `json:"supported_classes"`                                        // e.g., ["JSS1", "JSS2", "SS1"]
	NotificationSettings  map[string]bool   `json:"notification_settings"`                                    // event_type -> enabled
	MeetingRecordingRetention int           `json:"meeting_recording_retention" validate:"min=1,max=365"` // in days
}

// BillingContact holds billing-related contact information
type BillingContact struct {
	Name  string `json:"name" validate:"required,min=2,max=100"`
	Email string `json:"email" validate:"required,email"`
	Phone string `json:"phone" validate:"required,min=10,max=20"`
}

// Tenant represents a school or school group in the multi-tenant system
type Tenant struct {
	ID              uuid.UUID            `json:"id" validate:"required,uuid"`
	Name            string               `json:"name" validate:"required,min=3,max=200"`
	SchoolType      SchoolType           `json:"school_type" validate:"required,oneof=PRIMARY SECONDARY COMBINED"`
	ContactEmail    string               `json:"contact_email" validate:"required,email"`
	Address         string               `json:"address" validate:"required,min=10,max=500"`
	Logo            string               `json:"logo" validate:"omitempty,url"`
	Status          TenantStatus         `json:"status" validate:"required,oneof=ACTIVE SUSPENDED"`
	Configuration   TenantConfiguration  `json:"configuration" validate:"required"`
	BillingContact  BillingContact       `json:"billing_contact" validate:"required"`
	SuspensionReason *string             `json:"suspension_reason,omitempty"`
	PrincipalAdminID uuid.UUID           `json:"principal_admin_id" validate:"required,uuid"` // ADMIN with full permissions
	CreatedAt       time.Time            `json:"created_at" validate:"required"`
	UpdatedAt       time.Time            `json:"updated_at" validate:"required"`
	SuspendedAt     *time.Time           `json:"suspended_at,omitempty"`
}

// IsActive returns true if the tenant is in active status
func (t *Tenant) IsActive() bool {
	return t.Status == TenantStatusActive
}

// IsSuspended returns true if the tenant is suspended
func (t *Tenant) IsSuspended() bool {
	return t.Status == TenantStatusSuspended
}

// Suspend marks the tenant as suspended with a reason
func (t *Tenant) Suspend(reason string) {
	t.Status = TenantStatusSuspended
	t.SuspensionReason = &reason
	now := time.Now()
	t.SuspendedAt = &now
	t.UpdatedAt = now
}

// Reactivate marks the tenant as active and clears suspension data
func (t *Tenant) Reactivate() {
	t.Status = TenantStatusActive
	t.SuspensionReason = nil
	t.SuspendedAt = nil
	t.UpdatedAt = time.Now()
}
