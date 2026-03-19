package domain

import (
	"time"

	"github.com/google/uuid"
)

// TimetableStatus represents the status of a timetable
type TimetableStatus string

const (
	TimetableStatusDraft     TimetableStatus = "DRAFT"
	TimetableStatusPublished TimetableStatus = "PUBLISHED"
	TimetableStatusArchived  TimetableStatus = "ARCHIVED"
)

// Timetable represents an auto-generated schedule for a class within a term
type Timetable struct {
	ID               uuid.UUID       `json:"id" validate:"required,uuid"`
	TenantID         uuid.UUID       `json:"tenant_id" validate:"required,uuid"`
	ClassID          uuid.UUID       `json:"class_id" validate:"required,uuid"`
	TermID           uuid.UUID       `json:"term_id" validate:"required,uuid"`
	Status           TimetableStatus `json:"status" validate:"required,oneof=DRAFT PUBLISHED ARCHIVED"`
	GeneratedAt      time.Time       `json:"generated_at" validate:"required"`
	GeneratedBy      uuid.UUID       `json:"generated_by" validate:"required,uuid"` // User ID who triggered generation
	PublishedAt      *time.Time      `json:"published_at,omitempty"`
	PublishedBy      *uuid.UUID      `json:"published_by,omitempty" validate:"omitempty,uuid"`
	GenerationVersion int            `json:"generation_version" validate:"required,min=1"` // Incremented on regeneration
	Notes            *string         `json:"notes,omitempty" validate:"omitempty,max=1000"`
	CreatedAt        time.Time       `json:"created_at" validate:"required"`
	UpdatedAt        time.Time       `json:"updated_at" validate:"required"`
	ArchivedAt       *time.Time      `json:"archived_at,omitempty"`
}

// IsDraft returns true if the timetable is in draft status
func (t *Timetable) IsDraft() bool {
	return t.Status == TimetableStatusDraft
}

// IsPublished returns true if the timetable is published
func (t *Timetable) IsPublished() bool {
	return t.Status == TimetableStatusPublished
}

// IsArchived returns true if the timetable is archived
func (t *Timetable) IsArchived() bool {
	return t.Status == TimetableStatusArchived
}

// Publish marks the timetable as published
func (t *Timetable) Publish(publishedBy uuid.UUID) {
	t.Status = TimetableStatusPublished
	now := time.Now()
	t.PublishedAt = &now
	t.PublishedBy = &publishedBy
	t.UpdatedAt = now
}

// Archive marks the timetable as archived
func (t *Timetable) Archive() {
	t.Status = TimetableStatusArchived
	now := time.Now()
	t.ArchivedAt = &now
	t.UpdatedAt = now
}
