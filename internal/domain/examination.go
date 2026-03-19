package domain

import (
	"time"

	"github.com/google/uuid"
)

// ExaminationStatus represents the status of an examination
type ExaminationStatus string

const (
	ExaminationStatusDraft     ExaminationStatus = "DRAFT"
	ExaminationStatusScheduled ExaminationStatus = "SCHEDULED"
	ExaminationStatusInProgress ExaminationStatus = "IN_PROGRESS"
	ExaminationStatusCompleted ExaminationStatus = "COMPLETED"
	ExaminationStatusArchived  ExaminationStatus = "ARCHIVED"
)

// ExaminationSubmissionStatus represents the status of a student's examination attempt
type ExaminationSubmissionStatus string

const (
	ExamSubmissionStatusNotStarted ExaminationSubmissionStatus = "NOT_STARTED"
	ExamSubmissionStatusInProgress ExaminationSubmissionStatus = "IN_PROGRESS"
	ExamSubmissionStatusSubmitted  ExaminationSubmissionStatus = "SUBMITTED"
	ExamSubmissionStatusGraded     ExaminationSubmissionStatus = "GRADED"
	ExamSubmissionStatusPublished  ExaminationSubmissionStatus = "PUBLISHED"
)

// ExaminationQuestion represents a question in an examination
type ExaminationQuestion struct {
	ID              uuid.UUID    `json:"id" validate:"required,uuid"`
	Type            QuestionType `json:"type" validate:"required,oneof=MULTIPLE_CHOICE MULTIPLE_ANSWER TRUE_FALSE SHORT_ANSWER ESSAY"`
	Text            string       `json:"text" validate:"required,min=5,max=3000"`
	Options         []string     `json:"options,omitempty"`          // For multiple choice/answer questions
	CorrectAnswers  []string     `json:"correct_answers,omitempty"`  // For auto-gradable questions
	Marks           int          `json:"marks" validate:"required,min=1,max=100"`
	IsConfidential  bool         `json:"is_confidential"` // If true, not shown in student review
	Explanation     *string      `json:"explanation,omitempty" validate:"omitempty,max=1000"`
	OrderIndex      int          `json:"order_index" validate:"required,min=0"`
	AttachmentURLs  []string     `json:"attachment_urls,omitempty"`
}

// Examination represents a formal end-of-term assessment
type Examination struct {
	ID                uuid.UUID         `json:"id" validate:"required,uuid"`
	TenantID          uuid.UUID         `json:"tenant_id" validate:"required,uuid"`
	CourseID          uuid.UUID         `json:"course_id" validate:"required,uuid"`
	TermID            uuid.UUID         `json:"term_id" validate:"required,uuid"`
	CreatedByID       uuid.UUID         `json:"created_by_id" validate:"required,uuid"` // ADMIN or Tutor
	Title             string            `json:"title" validate:"required,min=3,max=200"`
	Instructions      string            `json:"instructions" validate:"required,min=10,max=3000"`
	Questions         []ExaminationQuestion `json:"questions" validate:"required,min=1,dive"`
	TotalMarks        int               `json:"total_marks" validate:"required,min=1"`
	Duration          int               `json:"duration" validate:"required,min=30,max=300"` // in minutes
	WindowStart       time.Time         `json:"window_start" validate:"required"`
	WindowEnd         time.Time         `json:"window_end" validate:"required,gtfield=WindowStart"`
	Status            ExaminationStatus `json:"status" validate:"required,oneof=DRAFT SCHEDULED IN_PROGRESS COMPLETED ARCHIVED"`
	ResultsPublished  bool              `json:"results_published"`
	ResultsPublishedAt *time.Time       `json:"results_published_at,omitempty"`
	ResultsPublishedBy *uuid.UUID       `json:"results_published_by,omitempty" validate:"omitempty,uuid"`
	CreatedAt         time.Time         `json:"created_at" validate:"required"`
	UpdatedAt         time.Time         `json:"updated_at" validate:"required"`
	ScheduledAt       *time.Time        `json:"scheduled_at,omitempty"`
}

// IsScheduled returns true if the examination is scheduled
func (e *Examination) IsScheduled() bool {
	return e.Status == ExaminationStatusScheduled
}

// IsInProgress returns true if the examination is currently in progress
func (e *Examination) IsInProgress() bool {
	return e.Status == ExaminationStatusInProgress
}

// IsCompleted returns true if the examination is completed
func (e *Examination) IsCompleted() bool {
	return e.Status == ExaminationStatusCompleted
}

// IsWithinWindow returns true if the current time is within the examination window
func (e *Examination) IsWithinWindow(now time.Time) bool {
	return now.After(e.WindowStart) && now.Before(e.WindowEnd)
}

// CanAccess returns true if a student can access the examination at the given time
func (e *Examination) CanAccess(now time.Time) bool {
	return e.IsScheduled() && e.IsWithinWindow(now)
}

// Schedule marks the examination as scheduled
func (e *Examination) Schedule() {
	e.Status = ExaminationStatusScheduled
	now := time.Now()
	e.ScheduledAt = &now
	e.UpdatedAt = now
}

// Start marks the examination as in progress
func (e *Examination) Start() {
	e.Status = ExaminationStatusInProgress
	e.UpdatedAt = time.Now()
}

// Complete marks the examination as completed
func (e *Examination) Complete() {
	e.Status = ExaminationStatusCompleted
	e.UpdatedAt = time.Now()
}

// PublishResults marks the results as published
func (e *Examination) PublishResults(publishedBy uuid.UUID) {
	e.ResultsPublished = true
	now := time.Now()
	e.ResultsPublishedAt = &now
	e.ResultsPublishedBy = &publishedBy
	e.UpdatedAt = now
}

// IntegrityEvent represents events tracked for examination integrity
type IntegrityEvent struct {
	EventType   string    `json:"event_type" validate:"required"` // "TAB_SWITCH", "FOCUS_LOSS", "COPY_ATTEMPT", etc.
	Timestamp   time.Time `json:"timestamp" validate:"required"`
	Description *string   `json:"description,omitempty" validate:"omitempty,max=500"`
}

// ExaminationSubmission represents a student's examination attempt
type ExaminationSubmission struct {
	ID                uuid.UUID                   `json:"id" validate:"required,uuid"`
	TenantID          uuid.UUID                   `json:"tenant_id" validate:"required,uuid"`
	ExaminationID     uuid.UUID                   `json:"examination_id" validate:"required,uuid"`
	StudentID         uuid.UUID                   `json:"student_id" validate:"required,uuid"`
	Status            ExaminationSubmissionStatus `json:"status" validate:"required,oneof=NOT_STARTED IN_PROGRESS SUBMITTED GRADED PUBLISHED"`
	StartedAt         *time.Time                  `json:"started_at,omitempty"`
	SubmittedAt       *time.Time                  `json:"submitted_at,omitempty"`
	AutoSubmitted     bool                        `json:"auto_submitted"` // True if auto-submitted on time expiry
	Answers           []ExaminationAnswer         `json:"answers"`
	Score             *int                        `json:"score,omitempty"`
	Percentage        *float64                    `json:"percentage,omitempty"`
	IsAutoGraded      bool                        `json:"is_auto_graded"`
	Feedback          *string                     `json:"feedback,omitempty" validate:"omitempty,max=2000"`
	IntegrityEvents   []IntegrityEvent            `json:"integrity_events"` // Tab switches, focus loss, etc.
	IPAddress         string                      `json:"ip_address" validate:"required"`
	CreatedAt         time.Time                   `json:"created_at" validate:"required"`
	UpdatedAt         time.Time                   `json:"updated_at" validate:"required"`
	GradedAt          *time.Time                  `json:"graded_at,omitempty"`
	GradedBy          *uuid.UUID                  `json:"graded_by,omitempty" validate:"omitempty,uuid"`
	ResultsPublishedToStudent bool                `json:"results_published_to_student"`
}

// ExaminationAnswer represents a student's answer to an examination question
type ExaminationAnswer struct {
	QuestionID      uuid.UUID `json:"question_id" validate:"required,uuid"`
	AnswerText      *string   `json:"answer_text,omitempty"`
	SelectedOptions []string  `json:"selected_options,omitempty"`
	IsCorrect       *bool     `json:"is_correct,omitempty"`
	MarksEarned     *int      `json:"marks_earned,omitempty"`
	Feedback        *string   `json:"feedback,omitempty" validate:"omitempty,max=1000"`
	SubmittedAt     time.Time `json:"submitted_at" validate:"required"` // Individual answer submission time
}

// IsGraded returns true if the submission has been graded
func (s *ExaminationSubmission) IsGraded() bool {
	return s.Status == ExamSubmissionStatusGraded || s.Status == ExamSubmissionStatusPublished
}

// IsSubmitted returns true if the submission has been submitted
func (s *ExaminationSubmission) IsSubmitted() bool {
	return s.Status == ExamSubmissionStatusSubmitted || s.IsGraded()
}

// HasIntegrityIssues returns true if any integrity events were recorded
func (s *ExaminationSubmission) HasIntegrityIssues() bool {
	return len(s.IntegrityEvents) > 0
}

// RecordIntegrityEvent adds an integrity event to the submission
func (s *ExaminationSubmission) RecordIntegrityEvent(eventType, description string) {
	event := IntegrityEvent{
		EventType:   eventType,
		Timestamp:   time.Now(),
		Description: &description,
	}
	s.IntegrityEvents = append(s.IntegrityEvents, event)
	s.UpdatedAt = time.Now()
}
