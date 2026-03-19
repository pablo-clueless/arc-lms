package domain

import (
	"time"

	"github.com/google/uuid"
)

// TermOrdinal represents the term number within a session
type TermOrdinal string

const (
	TermOrdinalFirst  TermOrdinal = "FIRST"
	TermOrdinalSecond TermOrdinal = "SECOND"
	TermOrdinalThird  TermOrdinal = "THIRD"
)

// TermStatus represents the status of a term
type TermStatus string

const (
	TermStatusDraft    TermStatus = "DRAFT"
	TermStatusActive   TermStatus = "ACTIVE"
	TermStatusCompleted TermStatus = "COMPLETED"
)

// Holiday represents a non-instructional day within a term
type Holiday struct {
	Date        time.Time `json:"date" validate:"required"`
	Name        string    `json:"name" validate:"required,min=3,max=100"`
	Description *string   `json:"description,omitempty" validate:"omitempty,max=500"`
	IsPublic    bool      `json:"is_public"` // true for national holidays, false for school-specific
}

// Term represents one of three academic periods within a session
type Term struct {
	ID                uuid.UUID   `json:"id" validate:"required,uuid"`
	TenantID          uuid.UUID   `json:"tenant_id" validate:"required,uuid"`
	SessionID         uuid.UUID   `json:"session_id" validate:"required,uuid"`
	Ordinal           TermOrdinal `json:"ordinal" validate:"required,oneof=FIRST SECOND THIRD"`
	StartDate         time.Time   `json:"start_date" validate:"required"`
	EndDate           time.Time   `json:"end_date" validate:"required,gtfield=StartDate"`
	Status            TermStatus  `json:"status" validate:"required,oneof=DRAFT ACTIVE COMPLETED"`
	Holidays          []Holiday   `json:"holidays"`
	NonInstructionalDays []time.Time `json:"non_instructional_days"` // school-specific non-instructional days
	Description       *string     `json:"description,omitempty" validate:"omitempty,max=500"`
	CreatedAt         time.Time   `json:"created_at" validate:"required"`
	UpdatedAt         time.Time   `json:"updated_at" validate:"required"`
	ActivatedAt       *time.Time  `json:"activated_at,omitempty"`
	CompletedAt       *time.Time  `json:"completed_at,omitempty"`
}

// IsActive returns true if the term is active
func (t *Term) IsActive() bool {
	return t.Status == TermStatusActive
}

// IsCompleted returns true if the term is completed
func (t *Term) IsCompleted() bool {
	return t.Status == TermStatusCompleted
}

// IsDraft returns true if the term is in draft status
func (t *Term) IsDraft() bool {
	return t.Status == TermStatusDraft
}

// Activate marks the term as active
func (t *Term) Activate() {
	t.Status = TermStatusActive
	now := time.Now()
	t.ActivatedAt = &now
	t.UpdatedAt = now
}

// Complete marks the term as completed
func (t *Term) Complete() {
	t.Status = TermStatusCompleted
	now := time.Now()
	t.CompletedAt = &now
	t.UpdatedAt = now
}

// GetOrdinalNumber returns the numeric value of the term ordinal (1, 2, or 3)
func (t *Term) GetOrdinalNumber() int {
	switch t.Ordinal {
	case TermOrdinalFirst:
		return 1
	case TermOrdinalSecond:
		return 2
	case TermOrdinalThird:
		return 3
	default:
		return 0
	}
}

// IsInstructionalDay returns true if the given date is an instructional day
func (t *Term) IsInstructionalDay(date time.Time) bool {
	// Check if date is within term range
	if date.Before(t.StartDate) || date.After(t.EndDate) {
		return false
	}

	// Check if date is a weekend (Saturday or Sunday)
	if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
		return false
	}

	// Check if date is a holiday
	for _, holiday := range t.Holidays {
		if holiday.Date.Year() == date.Year() &&
			holiday.Date.Month() == date.Month() &&
			holiday.Date.Day() == date.Day() {
			return false
		}
	}

	// Check if date is a non-instructional day
	for _, nonInstructionalDay := range t.NonInstructionalDays {
		if nonInstructionalDay.Year() == date.Year() &&
			nonInstructionalDay.Month() == date.Month() &&
			nonInstructionalDay.Day() == date.Day() {
			return false
		}
	}

	return true
}
