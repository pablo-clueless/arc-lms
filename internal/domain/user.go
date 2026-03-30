package domain

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleSuperAdmin Role = "SUPER_ADMIN"
	RoleAdmin      Role = "ADMIN"
	RoleTutor      Role = "TUTOR"
	RoleStudent    Role = "STUDENT"
)

type UserStatus string

const (
	UserStatusActive      UserStatus = "ACTIVE"
	UserStatusInactive    UserStatus = "INACTIVE"
	UserStatusDeactivated UserStatus = "DEACTIVATED"
	UserStatusPending     UserStatus = "PENDING"
)

type NotificationPreference struct {
	EventType    string `json:"event_type" validate:"required"`
	InAppEnabled bool   `json:"in_app_enabled"`
	PushEnabled  bool   `json:"push_enabled"`
	EmailEnabled bool   `json:"email_enabled"`
}

func DefaultNotificationPreferences() []NotificationPreference {
	eventTypes := []string{
		"QUIZ_PUBLISHED",
		"ASSIGNMENT_PUBLISHED",
		"ASSIGNMENT_DEADLINE_APPROACHING",
		"EXAMINATION_SCHEDULED",
		"EXAMINATION_WINDOW_OPEN",
		"EXAMINATION_WINDOW_CLOSE",
		"GRADE_PUBLISHED",
		"TIMETABLE_PUBLISHED",
		"TIMETABLE_UPDATED",
		"MEETING_SCHEDULED",
		"MEETING_CANCELLED",
		"MEETING_STARTING",
		"INVOICE_GENERATED",
		"PAYMENT_OVERDUE",
		"CUSTOM",
	}

	preferences := make([]NotificationPreference, len(eventTypes))
	for i, eventType := range eventTypes {
		preferences[i] = NotificationPreference{
			EventType:    eventType,
			InAppEnabled: true,
			PushEnabled:  true,
			EmailEnabled: true,
		}
	}
	return preferences
}

type User struct {
	ID                      uuid.UUID                `json:"id" validate:"required,uuid"`
	TenantID                *uuid.UUID               `json:"tenant_id,omitempty" validate:"omitempty,uuid"`
	Role                    Role                     `json:"role" validate:"required,oneof=SUPER_ADMIN ADMIN TUTOR STUDENT"`
	Email                   string                   `json:"email" validate:"required,email"`
	PasswordHash            string                   `json:"-"`
	FirstName               string                   `json:"first_name" validate:"required,min=1,max=100"`
	LastName                string                   `json:"last_name" validate:"required,min=1,max=100"`
	MiddleName              *string                  `json:"middle_name,omitempty" validate:"omitempty,max=100"`
	ProfilePhoto            *string                  `json:"profile_photo,omitempty" validate:"omitempty,url"`
	Phone                   *string                  `json:"phone,omitempty" validate:"omitempty,min=10,max=20"`
	Status                  UserStatus               `json:"status" validate:"required,oneof=ACTIVE INACTIVE DEACTIVATED PENDING"`
	Permissions             []string                 `json:"permissions,omitempty"`
	NotificationPreferences []NotificationPreference `json:"notification_preferences"`
	LastLoginAt             *time.Time               `json:"last_login_at,omitempty"`
	PasswordResetToken      *string                  `json:"-"`
	PasswordResetExpiry     *time.Time               `json:"-"`
	InvitationToken         *string                  `json:"-"`
	InvitationExpiry        *time.Time               `json:"-"`
	DeactivatedAt           *time.Time               `json:"deactivated_at,omitempty"`
	DeactivationReason      *string                  `json:"deactivation_reason,omitempty"`
	CreatedAt               time.Time                `json:"created_at" validate:"required"`
	UpdatedAt               time.Time                `json:"updated_at" validate:"required"`
}

func (u *User) FullName() string {
	if u.MiddleName != nil && *u.MiddleName != "" {
		return u.FirstName + " " + *u.MiddleName + " " + u.LastName
	}
	return u.FirstName + " " + u.LastName
}

func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

func (u *User) IsDeactivated() bool {
	return u.Status == UserStatusDeactivated
}

func (u *User) IsSuperAdmin() bool {
	return u.Role == RoleSuperAdmin
}

func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

func (u *User) IsTutor() bool {
	return u.Role == RoleTutor
}

func (u *User) IsStudent() bool {
	return u.Role == RoleStudent
}

func (u *User) HasPermission(permission string) bool {
	for _, perm := range u.Permissions {
		if perm == permission {
			return true
		}
	}
	return false
}

func (u *User) Deactivate(reason string) {
	u.Status = UserStatusDeactivated
	u.DeactivationReason = &reason
	now := time.Now()
	u.DeactivatedAt = &now
	u.UpdatedAt = now
}

func (u *User) Reactivate() {
	u.Status = UserStatusActive
	u.DeactivationReason = nil
	u.DeactivatedAt = nil
	u.UpdatedAt = time.Now()
}

func (u *User) RecordLogin() {
	now := time.Now()
	u.LastLoginAt = &now
	u.UpdatedAt = now
}
