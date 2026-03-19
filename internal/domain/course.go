package domain

import (
	"time"

	"github.com/google/uuid"
)

// CourseStatus represents the status of a course
type CourseStatus string

const (
	CourseStatusActive   CourseStatus = "ACTIVE"
	CourseStatusInactive CourseStatus = "INACTIVE"
	CourseStatusDraft    CourseStatus = "DRAFT"
)

// GradeWeighting represents custom grade weighting for a course
type GradeWeighting struct {
	ContinuousAssessment int `json:"continuous_assessment" validate:"required,min=0,max=100"`
	Examination          int `json:"examination" validate:"required,min=0,max=100"`
}

// Course represents a subject taught within a class
type Course struct {
	ID                  uuid.UUID       `json:"id" validate:"required,uuid"`
	TenantID            uuid.UUID       `json:"tenant_id" validate:"required,uuid"`
	SessionID           uuid.UUID       `json:"session_id" validate:"required,uuid"`
	ClassID             uuid.UUID       `json:"class_id" validate:"required,uuid"`
	TermID              uuid.UUID       `json:"term_id" validate:"required,uuid"`
	Name                string          `json:"name" validate:"required,min=2,max=100"` // e.g., "Mathematics", "English Language"
	SubjectCode         string          `json:"subject_code" validate:"required,min=2,max=20"` // e.g., "MATH", "ENG"
	Description         *string         `json:"description,omitempty" validate:"omitempty,max=1000"`
	AssignedTutorID     uuid.UUID       `json:"assigned_tutor_id" validate:"required,uuid"` // One tutor per course per term
	Status              CourseStatus    `json:"status" validate:"required,oneof=ACTIVE INACTIVE DRAFT"`
	MaxPeriodsPerWeek   *int            `json:"max_periods_per_week,omitempty" validate:"omitempty,min=1,max=20"`
	CustomGradeWeighting *GradeWeighting `json:"custom_grade_weighting,omitempty"` // Overrides tenant default
	Materials           []string        `json:"materials"` // URLs to course materials
	Syllabus            *string         `json:"syllabus,omitempty" validate:"omitempty,url"` // URL to syllabus document
	CreatedAt           time.Time       `json:"created_at" validate:"required"`
	UpdatedAt           time.Time       `json:"updated_at" validate:"required"`
}

// IsActive returns true if the course is active
func (c *Course) IsActive() bool {
	return c.Status == CourseStatusActive
}

// IsInactive returns true if the course is inactive
func (c *Course) IsInactive() bool {
	return c.Status == CourseStatusInactive
}

// IsDraft returns true if the course is in draft status
func (c *Course) IsDraft() bool {
	return c.Status == CourseStatusDraft
}

// Activate marks the course as active
func (c *Course) Activate() {
	c.Status = CourseStatusActive
	c.UpdatedAt = time.Now()
}

// Deactivate marks the course as inactive
func (c *Course) Deactivate() {
	c.Status = CourseStatusInactive
	c.UpdatedAt = time.Now()
}

// ReassignTutor changes the assigned tutor for the course
func (c *Course) ReassignTutor(newTutorID uuid.UUID) {
	c.AssignedTutorID = newTutorID
	c.UpdatedAt = time.Now()
}
