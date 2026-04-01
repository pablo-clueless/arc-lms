package domain

import (
	"time"

	"github.com/google/uuid"
)

type CourseStatus string

const (
	CourseStatusActive   CourseStatus = "ACTIVE"
	CourseStatusInactive CourseStatus = "INACTIVE"
	CourseStatusDraft    CourseStatus = "DRAFT"
)

type GradeWeighting struct {
	ContinuousAssessment int `json:"continuous_assessment" validate:"required,min=0,max=100"`
	Examination          int `json:"examination" validate:"required,min=0,max=100"`
}

type ContentType string

const (
	ContentTypeText     ContentType = "TEXT"
	ContentTypeVideo    ContentType = "VIDEO"
	ContentTypeImage    ContentType = "IMAGE"
	ContentTypeAudio    ContentType = "AUDIO"
	ContentTypeDocument ContentType = "DOCUMENT"
	ContentTypeLink     ContentType = "LINK"
)

type CourseContent struct {
	ID          uuid.UUID   `json:"id"`
	CourseID    uuid.UUID   `json:"course_id"`
	Title       string      `json:"title" validate:"required,min=2,max=200"`
	ContentType ContentType `json:"content_type" validate:"required,oneof=TEXT VIDEO IMAGE AUDIO DOCUMENT LINK"`
	Content     string      `json:"content" validate:"required"`
	Description *string     `json:"description,omitempty" validate:"omitempty,max=1000"`
	OrderIndex  int         `json:"order_index"`
	Duration    *int        `json:"duration,omitempty"`
	FileSize    *int64      `json:"file_size,omitempty"`
	MimeType    *string     `json:"mime_type,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type Course struct {
	ID                   uuid.UUID       `json:"id" validate:"required,uuid"`
	TenantID             uuid.UUID       `json:"tenant_id" validate:"required,uuid"`
	SessionID            uuid.UUID       `json:"session_id" validate:"required,uuid"`
	ClassID              uuid.UUID       `json:"class_id" validate:"required,uuid"`
	TermID               uuid.UUID       `json:"term_id" validate:"required,uuid"`
	Name                 string          `json:"name" validate:"required,min=2,max=100"`
	SubjectCode          string          `json:"subject_code" validate:"required,min=2,max=20"`
	Description          *string         `json:"description,omitempty" validate:"omitempty,max=1000"`
	AssignedTutorID      uuid.UUID       `json:"assigned_tutor_id" validate:"required,uuid"`
	Status               CourseStatus    `json:"status" validate:"required,oneof=ACTIVE INACTIVE DRAFT"`
	MaxPeriodsPerWeek    *int            `json:"max_periods_per_week,omitempty" validate:"omitempty,min=1,max=20"`
	CustomGradeWeighting *GradeWeighting `json:"custom_grade_weighting,omitempty"`
	Materials            []string        `json:"materials"`
	Syllabus             *string         `json:"syllabus,omitempty" validate:"omitempty,url"`
	CreatedAt            time.Time       `json:"created_at" validate:"required"`
	UpdatedAt            time.Time       `json:"updated_at" validate:"required"`
	CourseContents       []CourseContent `json:"course_contents"`
}

func (c *Course) IsActive() bool {
	return c.Status == CourseStatusActive
}

func (c *Course) IsInactive() bool {
	return c.Status == CourseStatusInactive
}

func (c *Course) IsDraft() bool {
	return c.Status == CourseStatusDraft
}

func (c *Course) Activate() {
	c.Status = CourseStatusActive
	c.UpdatedAt = time.Now()
}

func (c *Course) Deactivate() {
	c.Status = CourseStatusInactive
	c.UpdatedAt = time.Now()
}

func (c *Course) ReassignTutor(newTutorID uuid.UUID) {
	c.AssignedTutorID = newTutorID
	c.UpdatedAt = time.Now()
}
