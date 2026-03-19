package domain

import (
	"time"

	"github.com/google/uuid"
)

// Role represents the user's role in the system
type Role string

const (
	RoleSuperAdmin Role = "SUPER_ADMIN"
	RoleAdmin      Role = "ADMIN"
	RoleTutor      Role = "TUTOR"
	RoleStudent    Role = "STUDENT"
)

// UserStatus represents the user's account status
type UserStatus string

const (
	UserStatusActive     UserStatus = "ACTIVE"
	UserStatusInactive   UserStatus = "INACTIVE"
	UserStatusDeactivated UserStatus = "DEACTIVATED"
	UserStatusPending    UserStatus = "PENDING" // awaiting invitation acceptance
)

// NotificationPreference holds user notification settings
type NotificationPreference struct {
	EventType     string `json:"event_type" validate:"required"`
	InAppEnabled  bool   `json:"in_app_enabled"`
	PushEnabled   bool   `json:"push_enabled"`
	EmailEnabled  bool   `json:"email_enabled"`
}

// User represents a platform user (SUPER_ADMIN, ADMIN, TUTOR, or STUDENT)
type User struct {
	ID                      uuid.UUID                 `json:"id" validate:"required,uuid"`
	TenantID                *uuid.UUID                `json:"tenant_id,omitempty" validate:"omitempty,uuid"` // NULL for SUPER_ADMIN
	Role                    Role                      `json:"role" validate:"required,oneof=SUPER_ADMIN ADMIN TUTOR STUDENT"`
	Email                   string                    `json:"email" validate:"required,email"`
	PasswordHash            string                    `json:"-"` // Never expose in JSON
	FirstName               string                    `json:"first_name" validate:"required,min=1,max=100"`
	LastName                string                    `json:"last_name" validate:"required,min=1,max=100"`
	MiddleName              *string                   `json:"middle_name,omitempty" validate:"omitempty,max=100"`
	ProfilePhoto            *string                   `json:"profile_photo,omitempty" validate:"omitempty,url"`
	Phone                   *string                   `json:"phone,omitempty" validate:"omitempty,min=10,max=20"`
	Status                  UserStatus                `json:"status" validate:"required,oneof=ACTIVE INACTIVE DEACTIVATED PENDING"`
	Permissions             []string                  `json:"permissions,omitempty"` // array of "resource:action" strings (e.g., "tenant:create")
	NotificationPreferences []NotificationPreference  `json:"notification_preferences"`
	LastLoginAt             *time.Time                `json:"last_login_at,omitempty"`
	PasswordResetToken      *string                   `json:"-"` // Never expose in JSON
	PasswordResetExpiry     *time.Time                `json:"-"` // Never expose in JSON
	InvitationToken         *string                   `json:"-"` // Never expose in JSON
	InvitationExpiry        *time.Time                `json:"-"` // Never expose in JSON
	DeactivatedAt           *time.Time                `json:"deactivated_at,omitempty"`
	DeactivationReason      *string                   `json:"deactivation_reason,omitempty"`
	CreatedAt               time.Time                 `json:"created_at" validate:"required"`
	UpdatedAt               time.Time                 `json:"updated_at" validate:"required"`
}

// FullName returns the user's full name
func (u *User) FullName() string {
	if u.MiddleName != nil && *u.MiddleName != "" {
		return u.FirstName + " " + *u.MiddleName + " " + u.LastName
	}
	return u.FirstName + " " + u.LastName
}

// IsActive returns true if the user is in active status
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

// IsDeactivated returns true if the user is deactivated
func (u *User) IsDeactivated() bool {
	return u.Status == UserStatusDeactivated
}

// IsSuperAdmin returns true if the user is a SUPER_ADMIN
func (u *User) IsSuperAdmin() bool {
	return u.Role == RoleSuperAdmin
}

// IsAdmin returns true if the user is an ADMIN
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsTutor returns true if the user is a TUTOR
func (u *User) IsTutor() bool {
	return u.Role == RoleTutor
}

// IsStudent returns true if the user is a STUDENT
func (u *User) IsStudent() bool {
	return u.Role == RoleStudent
}

// HasPermission checks if the user has a specific permission
func (u *User) HasPermission(permission string) bool {
	for _, perm := range u.Permissions {
		if perm == permission {
			return true
		}
	}
	return false
}

// Deactivate marks the user as deactivated with a reason
func (u *User) Deactivate(reason string) {
	u.Status = UserStatusDeactivated
	u.DeactivationReason = &reason
	now := time.Now()
	u.DeactivatedAt = &now
	u.UpdatedAt = now
}

// Reactivate marks the user as active and clears deactivation data
func (u *User) Reactivate() {
	u.Status = UserStatusActive
	u.DeactivationReason = nil
	u.DeactivatedAt = nil
	u.UpdatedAt = time.Now()
}

// RecordLogin updates the last login timestamp
func (u *User) RecordLogin() {
	now := time.Now()
	u.LastLoginAt = &now
	u.UpdatedAt = now
}
