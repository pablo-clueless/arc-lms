package domain

import (
	"time"

	"github.com/google/uuid"
)

// DayOfWeek represents the day of the week for a period
type DayOfWeek string

const (
	DayMonday    DayOfWeek = "MONDAY"
	DayTuesday   DayOfWeek = "TUESDAY"
	DayWednesday DayOfWeek = "WEDNESDAY"
	DayThursday  DayOfWeek = "THURSDAY"
	DayFriday    DayOfWeek = "FRIDAY"
	DaySaturday  DayOfWeek = "SATURDAY" // Some schools have Saturday classes
)

// SwapRequestStatus represents the status of a period swap request
type SwapRequestStatus string

const (
	SwapRequestStatusPending   SwapRequestStatus = "PENDING"
	SwapRequestStatusApproved  SwapRequestStatus = "APPROVED"
	SwapRequestStatusRejected  SwapRequestStatus = "REJECTED"
	SwapRequestStatusEscalated SwapRequestStatus = "ESCALATED"
	SwapRequestStatusCancelled SwapRequestStatus = "CANCELLED"
)

// Period represents a single scheduled block in the timetable
type Period struct {
	ID          uuid.UUID `json:"id" validate:"required,uuid"`
	TenantID    uuid.UUID `json:"tenant_id" validate:"required,uuid"`
	TimetableID uuid.UUID `json:"timetable_id" validate:"required,uuid"`
	CourseID    uuid.UUID `json:"course_id" validate:"required,uuid"`
	TutorID     uuid.UUID `json:"tutor_id" validate:"required,uuid"`
	ClassID     uuid.UUID `json:"class_id" validate:"required,uuid"`
	DayOfWeek   DayOfWeek `json:"day_of_week" validate:"required,oneof=MONDAY TUESDAY WEDNESDAY THURSDAY FRIDAY SATURDAY"`
	StartTime   time.Time `json:"start_time" validate:"required"` // Time of day (will be repeated weekly)
	EndTime     time.Time `json:"end_time" validate:"required,gtfield=StartTime"`
	PeriodNumber int      `json:"period_number" validate:"required,min=1,max=15"` // Position in daily schedule
	Notes       *string   `json:"notes,omitempty" validate:"omitempty,max=500"`
	CreatedAt   time.Time `json:"created_at" validate:"required"`
	UpdatedAt   time.Time `json:"updated_at" validate:"required"`
}

// Duration returns the duration of the period
func (p *Period) Duration() time.Duration {
	return p.EndTime.Sub(p.StartTime)
}

// OverlapsWith checks if this period overlaps with another period
func (p *Period) OverlapsWith(other *Period) bool {
	// Must be same day
	if p.DayOfWeek != other.DayOfWeek {
		return false
	}

	// Check time overlap
	return (p.StartTime.Before(other.EndTime) && p.EndTime.After(other.StartTime)) ||
		(other.StartTime.Before(p.EndTime) && other.EndTime.After(p.StartTime))
}

// SwapRequest represents a request to swap periods between tutors
type SwapRequest struct {
	ID                  uuid.UUID         `json:"id" validate:"required,uuid"`
	TenantID            uuid.UUID         `json:"tenant_id" validate:"required,uuid"`
	RequestingPeriodID  uuid.UUID         `json:"requesting_period_id" validate:"required,uuid"`
	TargetPeriodID      uuid.UUID         `json:"target_period_id" validate:"required,uuid"`
	RequestingTutorID   uuid.UUID         `json:"requesting_tutor_id" validate:"required,uuid"`
	TargetTutorID       uuid.UUID         `json:"target_tutor_id" validate:"required,uuid"`
	Status              SwapRequestStatus `json:"status" validate:"required,oneof=PENDING APPROVED REJECTED ESCALATED CANCELLED"`
	Reason              *string           `json:"reason,omitempty" validate:"omitempty,max=500"`
	RejectionReason     *string           `json:"rejection_reason,omitempty" validate:"omitempty,max=500"`
	EscalationReason    *string           `json:"escalation_reason,omitempty" validate:"omitempty,max=500"`
	AdminOverrideReason *string           `json:"admin_override_reason,omitempty" validate:"omitempty,max=500"`
	AdminOverrideBy     *uuid.UUID        `json:"admin_override_by,omitempty" validate:"omitempty,uuid"`
	CreatedAt           time.Time         `json:"created_at" validate:"required"`
	UpdatedAt           time.Time         `json:"updated_at" validate:"required"`
	RespondedAt         *time.Time        `json:"responded_at,omitempty"`
	EscalatedAt         *time.Time        `json:"escalated_at,omitempty"`
}

// IsPending returns true if the swap request is pending
func (s *SwapRequest) IsPending() bool {
	return s.Status == SwapRequestStatusPending
}

// IsApproved returns true if the swap request is approved
func (s *SwapRequest) IsApproved() bool {
	return s.Status == SwapRequestStatusApproved
}

// IsRejected returns true if the swap request is rejected
func (s *SwapRequest) IsRejected() bool {
	return s.Status == SwapRequestStatusRejected
}

// IsEscalated returns true if the swap request is escalated
func (s *SwapRequest) IsEscalated() bool {
	return s.Status == SwapRequestStatusEscalated
}

// Approve marks the swap request as approved
func (s *SwapRequest) Approve() {
	s.Status = SwapRequestStatusApproved
	now := time.Now()
	s.RespondedAt = &now
	s.UpdatedAt = now
}

// Reject marks the swap request as rejected with a reason
func (s *SwapRequest) Reject(reason string) {
	s.Status = SwapRequestStatusRejected
	s.RejectionReason = &reason
	now := time.Now()
	s.RespondedAt = &now
	s.UpdatedAt = now
}

// Escalate marks the swap request as escalated with a reason
func (s *SwapRequest) Escalate(reason string) {
	s.Status = SwapRequestStatusEscalated
	s.EscalationReason = &reason
	now := time.Now()
	s.EscalatedAt = &now
	s.UpdatedAt = now
}

// AdminOverride allows an admin to force approve the swap with a reason
func (s *SwapRequest) AdminOverride(adminID uuid.UUID, reason string) {
	s.Status = SwapRequestStatusApproved
	s.AdminOverrideReason = &reason
	s.AdminOverrideBy = &adminID
	now := time.Now()
	s.RespondedAt = &now
	s.UpdatedAt = now
}

// Cancel marks the swap request as cancelled
func (s *SwapRequest) Cancel() {
	s.Status = SwapRequestStatusCancelled
	s.UpdatedAt = time.Now()
}
