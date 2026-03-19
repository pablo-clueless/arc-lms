package domain

import (
	"time"

	"github.com/google/uuid"
)

// SchoolLevel represents the educational level
type SchoolLevel string

const (
	SchoolLevelPrimary   SchoolLevel = "PRIMARY"
	SchoolLevelSecondary SchoolLevel = "SECONDARY"
)

// ClassStatus represents the status of a class
type ClassStatus string

const (
	ClassStatusActive   ClassStatus = "ACTIVE"
	ClassStatusInactive ClassStatus = "INACTIVE"
)

// Class represents a student cohort group (e.g., JSS1A, Primary 4B)
type Class struct {
	ID          uuid.UUID   `json:"id" validate:"required,uuid"`
	TenantID    uuid.UUID   `json:"tenant_id" validate:"required,uuid"`
	SessionID   uuid.UUID   `json:"session_id" validate:"required,uuid"`
	Name        string      `json:"name" validate:"required,min=2,max=50"` // e.g., "JSS1", "Primary 5"
	Arm         string      `json:"arm" validate:"required,min=1,max=20"`  // e.g., "A", "B", "Gold", "Diamond"
	Level       SchoolLevel `json:"level" validate:"required,oneof=PRIMARY SECONDARY"`
	Capacity    *int        `json:"capacity,omitempty" validate:"omitempty,min=1,max=200"`
	Status      ClassStatus `json:"status" validate:"required,oneof=ACTIVE INACTIVE"`
	Description *string     `json:"description,omitempty" validate:"omitempty,max=500"`
	CreatedAt   time.Time   `json:"created_at" validate:"required"`
	UpdatedAt   time.Time   `json:"updated_at" validate:"required"`
}

// FullName returns the complete class name with arm (e.g., "JSS1A")
func (c *Class) FullName() string {
	return c.Name + c.Arm
}

// IsActive returns true if the class is active
func (c *Class) IsActive() bool {
	return c.Status == ClassStatusActive
}

// IsInactive returns true if the class is inactive
func (c *Class) IsInactive() bool {
	return c.Status == ClassStatusInactive
}

// Deactivate marks the class as inactive
func (c *Class) Deactivate() {
	c.Status = ClassStatusInactive
	c.UpdatedAt = time.Now()
}

// Activate marks the class as active
func (c *Class) Activate() {
	c.Status = ClassStatusActive
	c.UpdatedAt = time.Now()
}
