package domain

import (
	"time"

	"github.com/google/uuid"
)

// GuardianRelationship represents the type of relationship between guardian and student
type GuardianRelationship string

const (
	GuardianRelationshipFather    GuardianRelationship = "FATHER"
	GuardianRelationshipMother    GuardianRelationship = "MOTHER"
	GuardianRelationshipGuardian  GuardianRelationship = "GUARDIAN"
	GuardianRelationshipOther     GuardianRelationship = "OTHER"
)

// GuardianStatus represents the status of a guardian-ward relationship
type GuardianStatus string

const (
	GuardianStatusActive   GuardianStatus = "ACTIVE"
	GuardianStatusInactive GuardianStatus = "INACTIVE"
)

// Guardian represents a parent/guardian to student relationship
type Guardian struct {
	ID           uuid.UUID            `json:"id"`
	TenantID     uuid.UUID            `json:"tenant_id"`
	GuardianID   uuid.UUID            `json:"guardian_id"`   // User ID of the parent/guardian
	StudentID    uuid.UUID            `json:"student_id"`    // User ID of the student (ward)
	Relationship GuardianRelationship `json:"relationship"`
	IsPrimary    bool                 `json:"is_primary"`    // Primary contact for the student
	Status       GuardianStatus       `json:"status"`
	Notes        *string              `json:"notes,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
}

// GuardianWithDetails includes the guardian relationship with user details
type GuardianWithDetails struct {
	*Guardian
	GuardianUser *User `json:"guardian_user"`
	StudentUser  *User `json:"student_user"`
}

// WardSummary represents a summary of a ward (student) for parent dashboard
type WardSummary struct {
	Student    *User        `json:"student"`
	Enrollment *Enrollment  `json:"enrollment,omitempty"`
	Class      *Class       `json:"class,omitempty"`
	Session    *Session     `json:"session,omitempty"`
}

// IsActive checks if the guardian relationship is active
func (g *Guardian) IsActive() bool {
	return g.Status == GuardianStatusActive
}

// Deactivate deactivates the guardian relationship
func (g *Guardian) Deactivate() {
	g.Status = GuardianStatusInactive
	g.UpdatedAt = time.Now()
}

// Activate activates the guardian relationship
func (g *Guardian) Activate() {
	g.Status = GuardianStatusActive
	g.UpdatedAt = time.Now()
}
