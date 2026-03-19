package domain

import (
	"time"

	"github.com/google/uuid"
)

// EnrollmentStatus represents the status of a student enrollment
type EnrollmentStatus string

const (
	EnrollmentStatusActive     EnrollmentStatus = "ACTIVE"
	EnrollmentStatusTransferred EnrollmentStatus = "TRANSFERRED"
	EnrollmentStatusWithdrawn   EnrollmentStatus = "WITHDRAWN"
	EnrollmentStatusSuspended   EnrollmentStatus = "SUSPENDED"
)

// Enrollment represents a student's enrollment in a class for a session
type Enrollment struct {
	ID               uuid.UUID         `json:"id" validate:"required,uuid"`
	TenantID         uuid.UUID         `json:"tenant_id" validate:"required,uuid"`
	StudentID        uuid.UUID         `json:"student_id" validate:"required,uuid"`
	ClassID          uuid.UUID         `json:"class_id" validate:"required,uuid"`
	SessionID        uuid.UUID         `json:"session_id" validate:"required,uuid"`
	Status           EnrollmentStatus  `json:"status" validate:"required,oneof=ACTIVE TRANSFERRED WITHDRAWN SUSPENDED"`
	EnrollmentDate   time.Time         `json:"enrollment_date" validate:"required"`
	WithdrawalDate   *time.Time        `json:"withdrawal_date,omitempty"`
	WithdrawalReason *string           `json:"withdrawal_reason,omitempty" validate:"omitempty,max=500"`
	TransferredToClassID *uuid.UUID    `json:"transferred_to_class_id,omitempty" validate:"omitempty,uuid"`
	TransferDate     *time.Time        `json:"transfer_date,omitempty"`
	TransferReason   *string           `json:"transfer_reason,omitempty" validate:"omitempty,max=500"`
	SuspensionDate   *time.Time        `json:"suspension_date,omitempty"`
	SuspensionReason *string           `json:"suspension_reason,omitempty" validate:"omitempty,max=500"`
	Notes            *string           `json:"notes,omitempty" validate:"omitempty,max=1000"`
	CreatedAt        time.Time         `json:"created_at" validate:"required"`
	UpdatedAt        time.Time         `json:"updated_at" validate:"required"`
}

// IsActive returns true if the enrollment is active
func (e *Enrollment) IsActive() bool {
	return e.Status == EnrollmentStatusActive
}

// IsTransferred returns true if the enrollment was transferred
func (e *Enrollment) IsTransferred() bool {
	return e.Status == EnrollmentStatusTransferred
}

// IsWithdrawn returns true if the enrollment was withdrawn
func (e *Enrollment) IsWithdrawn() bool {
	return e.Status == EnrollmentStatusWithdrawn
}

// IsSuspended returns true if the enrollment is suspended
func (e *Enrollment) IsSuspended() bool {
	return e.Status == EnrollmentStatusSuspended
}

// Withdraw marks the enrollment as withdrawn with a reason
func (e *Enrollment) Withdraw(reason string) {
	e.Status = EnrollmentStatusWithdrawn
	e.WithdrawalReason = &reason
	now := time.Now()
	e.WithdrawalDate = &now
	e.UpdatedAt = now
}

// Transfer moves the enrollment to a different class with a reason
func (e *Enrollment) Transfer(newClassID uuid.UUID, reason string) {
	e.Status = EnrollmentStatusTransferred
	e.TransferredToClassID = &newClassID
	e.TransferReason = &reason
	now := time.Now()
	e.TransferDate = &now
	e.UpdatedAt = now
}

// Suspend temporarily suspends the enrollment with a reason
func (e *Enrollment) Suspend(reason string) {
	e.Status = EnrollmentStatusSuspended
	e.SuspensionReason = &reason
	now := time.Now()
	e.SuspensionDate = &now
	e.UpdatedAt = now
}

// Reactivate marks the enrollment as active
func (e *Enrollment) Reactivate() {
	e.Status = EnrollmentStatusActive
	e.SuspensionDate = nil
	e.SuspensionReason = nil
	e.UpdatedAt = time.Now()
}
