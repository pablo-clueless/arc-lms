package domain

import (
	"time"

	"github.com/google/uuid"
)

// SessionStatus represents the status of an academic session
type SessionStatus string

const (
	SessionStatusActive   SessionStatus = "ACTIVE"
	SessionStatusArchived SessionStatus = "ARCHIVED"
	SessionStatusDraft    SessionStatus = "DRAFT"
)

// Session represents an academic year (e.g., 2025/2026)
type Session struct {
	ID          uuid.UUID     `json:"id" validate:"required,uuid"`
	TenantID    uuid.UUID     `json:"tenant_id" validate:"required,uuid"`
	Label       string        `json:"label" validate:"required,min=7,max=20"` // e.g., "2025/2026"
	StartYear   int           `json:"start_year" validate:"required,min=2000,max=2100"`
	EndYear     int           `json:"end_year" validate:"required,min=2000,max=2100"`
	Status      SessionStatus `json:"status" validate:"required,oneof=ACTIVE ARCHIVED DRAFT"`
	Description *string       `json:"description,omitempty" validate:"omitempty,max=500"`
	CreatedAt   time.Time     `json:"created_at" validate:"required"`
	UpdatedAt   time.Time     `json:"updated_at" validate:"required"`
	ArchivedAt  *time.Time    `json:"archived_at,omitempty"`
}

// IsActive returns true if the session is active
func (s *Session) IsActive() bool {
	return s.Status == SessionStatusActive
}

// IsArchived returns true if the session is archived
func (s *Session) IsArchived() bool {
	return s.Status == SessionStatusArchived
}

// IsDraft returns true if the session is in draft status
func (s *Session) IsDraft() bool {
	return s.Status == SessionStatusDraft
}

// Activate marks the session as active
func (s *Session) Activate() {
	s.Status = SessionStatusActive
	s.UpdatedAt = time.Now()
}

// Archive marks the session as archived
func (s *Session) Archive() {
	s.Status = SessionStatusArchived
	now := time.Now()
	s.ArchivedAt = &now
	s.UpdatedAt = now
}
