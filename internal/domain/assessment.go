package domain

import (
	"time"

	"github.com/google/uuid"
)

// AssessmentType represents the type of assessment
type AssessmentType string

const (
	AssessmentTypeQuiz       AssessmentType = "QUIZ"
	AssessmentTypeAssignment AssessmentType = "ASSIGNMENT"
)

// AssessmentStatus represents the status of an assessment
type AssessmentStatus string

const (
	AssessmentStatusDraft     AssessmentStatus = "DRAFT"
	AssessmentStatusPublished AssessmentStatus = "PUBLISHED"
	AssessmentStatusArchived  AssessmentStatus = "ARCHIVED"
)

// QuestionType represents the type of question
type QuestionType string

const (
	QuestionTypeMultipleChoice   QuestionType = "MULTIPLE_CHOICE"        // single answer
	QuestionTypeMultipleAnswer   QuestionType = "MULTIPLE_ANSWER"        // multiple answers
	QuestionTypeTrueFalse        QuestionType = "TRUE_FALSE"
	QuestionTypeShortAnswer      QuestionType = "SHORT_ANSWER"           // requires manual grading
	QuestionTypeEssay            QuestionType = "ESSAY"                   // requires manual grading
)

// SubmissionStatus represents the status of a student's submission
type SubmissionStatus string

const (
	SubmissionStatusNotStarted SubmissionStatus = "NOT_STARTED"
	SubmissionStatusInProgress SubmissionStatus = "IN_PROGRESS"
	SubmissionStatusSubmitted  SubmissionStatus = "SUBMITTED"
	SubmissionStatusLate       SubmissionStatus = "LATE"
	SubmissionStatusGraded     SubmissionStatus = "GRADED"
)

// Question represents a question in a quiz or assignment
type Question struct {
	ID              uuid.UUID    `json:"id" validate:"required,uuid"`
	Type            QuestionType `json:"type" validate:"required,oneof=MULTIPLE_CHOICE MULTIPLE_ANSWER TRUE_FALSE SHORT_ANSWER ESSAY"`
	Text            string       `json:"text" validate:"required,min=5,max=2000"`
	Options         []string     `json:"options,omitempty"`          // For multiple choice/answer questions
	CorrectAnswers  []string     `json:"correct_answers,omitempty"`  // For auto-gradable questions
	Marks           int          `json:"marks" validate:"required,min=1,max=100"`
	Explanation     *string      `json:"explanation,omitempty" validate:"omitempty,max=1000"` // Shown after submission
	OrderIndex      int          `json:"order_index" validate:"required,min=0"`
	AttachmentURLs  []string     `json:"attachment_urls,omitempty"`
}

// Quiz represents a formative assessment
type Quiz struct {
	ID                    uuid.UUID        `json:"id" validate:"required,uuid"`
	TenantID              uuid.UUID        `json:"tenant_id" validate:"required,uuid"`
	CourseID              uuid.UUID        `json:"course_id" validate:"required,uuid"`
	CreatedByTutorID      uuid.UUID        `json:"created_by_tutor_id" validate:"required,uuid"`
	Title                 string           `json:"title" validate:"required,min=3,max=200"`
	Instructions          string           `json:"instructions" validate:"required,min=10,max=2000"`
	Questions             []Question       `json:"questions" validate:"required,min=1,dive"`
	TotalMarks            int              `json:"total_marks" validate:"required,min=1"`
	TimeLimit             int              `json:"time_limit" validate:"required,min=1,max=300"` // in minutes
	AvailabilityStart     time.Time        `json:"availability_start" validate:"required"`
	AvailabilityEnd       time.Time        `json:"availability_end" validate:"required,gtfield=AvailabilityStart"`
	Status                AssessmentStatus `json:"status" validate:"required,oneof=DRAFT PUBLISHED ARCHIVED"`
	ShowBeforeWindow      bool             `json:"show_before_window"` // visible to students before availability window
	AllowRetake           bool             `json:"allow_retake"`
	PassingPercentage     *int             `json:"passing_percentage,omitempty" validate:"omitempty,min=0,max=100"`
	CreatedAt             time.Time        `json:"created_at" validate:"required"`
	UpdatedAt             time.Time        `json:"updated_at" validate:"required"`
	PublishedAt           *time.Time       `json:"published_at,omitempty"`
}

// IsPublished returns true if the quiz is published
func (q *Quiz) IsPublished() bool {
	return q.Status == AssessmentStatusPublished
}

// IsAvailable returns true if the quiz is currently available for submission
func (q *Quiz) IsAvailable(now time.Time) bool {
	return q.IsPublished() && now.After(q.AvailabilityStart) && now.Before(q.AvailabilityEnd)
}

// Publish marks the quiz as published
func (q *Quiz) Publish() {
	q.Status = AssessmentStatusPublished
	now := time.Now()
	q.PublishedAt = &now
	q.UpdatedAt = now
}

// Assignment represents a task-based assessment
type Assignment struct {
	ID                   uuid.UUID        `json:"id" validate:"required,uuid"`
	TenantID             uuid.UUID        `json:"tenant_id" validate:"required,uuid"`
	CourseID             uuid.UUID        `json:"course_id" validate:"required,uuid"`
	CreatedByTutorID     uuid.UUID        `json:"created_by_tutor_id" validate:"required,uuid"`
	Title                string           `json:"title" validate:"required,min=3,max=200"`
	Description          string           `json:"description" validate:"required,min=10,max=5000"`
	AttachmentURLs       []string         `json:"attachment_urls,omitempty"`
	MaxMarks             int              `json:"max_marks" validate:"required,min=1,max=100"`
	SubmissionDeadline   time.Time        `json:"submission_deadline" validate:"required"`
	AllowLateSubmission  bool             `json:"allow_late_submission"`
	HardCutoffDate       *time.Time       `json:"hard_cutoff_date,omitempty"` // If set, no submissions after this
	AllowedFileFormats   []string         `json:"allowed_file_formats,omitempty"` // e.g., ["pdf", "docx", "jpg"]
	MaxFileSize          int              `json:"max_file_size" validate:"required,min=1,max=104857600"` // in bytes (max 100MB)
	Status               AssessmentStatus `json:"status" validate:"required,oneof=DRAFT PUBLISHED ARCHIVED"`
	Questions            []Question       `json:"questions,omitempty"` // Optional structured questions
	CreatedAt            time.Time        `json:"created_at" validate:"required"`
	UpdatedAt            time.Time        `json:"updated_at" validate:"required"`
	PublishedAt          *time.Time       `json:"published_at,omitempty"`
}

// IsPublished returns true if the assignment is published
func (a *Assignment) IsPublished() bool {
	return a.Status == AssessmentStatusPublished
}

// IsOverdue returns true if the deadline has passed
func (a *Assignment) IsOverdue(now time.Time) bool {
	return now.After(a.SubmissionDeadline)
}

// CanSubmit returns true if submission is still allowed
func (a *Assignment) CanSubmit(now time.Time) bool {
	if !a.IsPublished() {
		return false
	}

	// Check hard cutoff if set
	if a.HardCutoffDate != nil && now.After(*a.HardCutoffDate) {
		return false
	}

	// If late submission is allowed and no hard cutoff, can still submit
	if a.AllowLateSubmission {
		return true
	}

	// Otherwise, can only submit before deadline
	return now.Before(a.SubmissionDeadline)
}

// Publish marks the assignment as published
func (a *Assignment) Publish() {
	a.Status = AssessmentStatusPublished
	now := time.Now()
	a.PublishedAt = &now
	a.UpdatedAt = now
}

// QuizSubmission represents a student's quiz attempt
type QuizSubmission struct {
	ID          uuid.UUID        `json:"id" validate:"required,uuid"`
	TenantID    uuid.UUID        `json:"tenant_id" validate:"required,uuid"`
	QuizID      uuid.UUID        `json:"quiz_id" validate:"required,uuid"`
	StudentID   uuid.UUID        `json:"student_id" validate:"required,uuid"`
	Status      SubmissionStatus `json:"status" validate:"required,oneof=NOT_STARTED IN_PROGRESS SUBMITTED LATE GRADED"`
	StartedAt   *time.Time       `json:"started_at,omitempty"`
	SubmittedAt *time.Time       `json:"submitted_at,omitempty"`
	Answers     []Answer         `json:"answers"`
	Score       *int             `json:"score,omitempty"` // Total score earned
	Percentage  *float64         `json:"percentage,omitempty"`
	IsAutoGraded bool            `json:"is_auto_graded"`
	Feedback    *string          `json:"feedback,omitempty" validate:"omitempty,max=2000"`
	IPAddress   *string          `json:"ip_address,omitempty"`
	CreatedAt   time.Time        `json:"created_at" validate:"required"`
	UpdatedAt   time.Time        `json:"updated_at" validate:"required"`
	GradedAt    *time.Time       `json:"graded_at,omitempty"`
	GradedBy    *uuid.UUID       `json:"graded_by,omitempty" validate:"omitempty,uuid"`
}

// Answer represents a student's answer to a question
type Answer struct {
	QuestionID      uuid.UUID `json:"question_id" validate:"required,uuid"`
	AnswerText      *string   `json:"answer_text,omitempty"`
	SelectedOptions []string  `json:"selected_options,omitempty"` // For multiple choice/answer
	IsCorrect       *bool     `json:"is_correct,omitempty"`       // For auto-gradable questions
	MarksEarned     *int      `json:"marks_earned,omitempty"`
	Feedback        *string   `json:"feedback,omitempty" validate:"omitempty,max=1000"` // Tutor feedback for manual grading
}

// AssignmentSubmission represents a student's assignment submission
type AssignmentSubmission struct {
	ID             uuid.UUID        `json:"id" validate:"required,uuid"`
	TenantID       uuid.UUID        `json:"tenant_id" validate:"required,uuid"`
	AssignmentID   uuid.UUID        `json:"assignment_id" validate:"required,uuid"`
	StudentID      uuid.UUID        `json:"student_id" validate:"required,uuid"`
	Status         SubmissionStatus `json:"status" validate:"required,oneof=NOT_STARTED IN_PROGRESS SUBMITTED LATE GRADED"`
	SubmittedAt    *time.Time       `json:"submitted_at,omitempty"`
	IsLate         bool             `json:"is_late"`
	FileURLs       []string         `json:"file_urls,omitempty"`
	AnswerText     *string          `json:"answer_text,omitempty" validate:"omitempty,max=10000"` // For text-based submissions
	Score          *int             `json:"score,omitempty"`
	Feedback       *string          `json:"feedback,omitempty" validate:"omitempty,max=2000"`
	IPAddress      *string          `json:"ip_address,omitempty"`
	CreatedAt      time.Time        `json:"created_at" validate:"required"`
	UpdatedAt      time.Time        `json:"updated_at" validate:"required"`
	GradedAt       *time.Time       `json:"graded_at,omitempty"`
	GradedBy       *uuid.UUID       `json:"graded_by,omitempty" validate:"omitempty,uuid"`
}

// IsGraded returns true if the submission has been graded
func (s *QuizSubmission) IsGraded() bool {
	return s.Status == SubmissionStatusGraded
}

// IsSubmitted returns true if the submission has been submitted
func (s *QuizSubmission) IsSubmitted() bool {
	return s.Status == SubmissionStatusSubmitted || s.Status == SubmissionStatusLate || s.Status == SubmissionStatusGraded
}

// IsGraded returns true if the assignment submission has been graded
func (s *AssignmentSubmission) IsGraded() bool {
	return s.Status == SubmissionStatusGraded
}

// IsSubmitted returns true if the assignment submission has been submitted
func (s *AssignmentSubmission) IsSubmitted() bool {
	return s.Status == SubmissionStatusSubmitted || s.Status == SubmissionStatusLate || s.Status == SubmissionStatusGraded
}
