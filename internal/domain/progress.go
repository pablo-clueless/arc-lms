package domain

import (
	"time"

	"github.com/google/uuid"
)

// ProgressStatus represents the status of a student's progress record
type ProgressStatus string

const (
	ProgressStatusInProgress ProgressStatus = "IN_PROGRESS"
	ProgressStatusCompleted  ProgressStatus = "COMPLETED"
	ProgressStatusFlagged    ProgressStatus = "FLAGGED" // Academic standing below threshold
)

// Grade represents the computed grade for a course
type Grade struct {
	ContinuousAssessment int     `json:"continuous_assessment"` // Total CA score
	Examination          int     `json:"examination"`           // Examination score
	Total                int     `json:"total"`                 // Weighted total
	Percentage           float64 `json:"percentage"`
	LetterGrade          string  `json:"letter_grade"` // A, B, C, D, E, F
	Remark               string  `json:"remark"`       // Excellent, Good, Pass, Fail, etc.
}

// AttendanceRecord represents a student's attendance for a course
type AttendanceRecord struct {
	TotalPeriods    int     `json:"total_periods"`
	PeriodsAttended int     `json:"periods_attended"`
	PeriodsAbsent   int     `json:"periods_absent"`
	Percentage      float64 `json:"percentage"`
}

// Progress represents a student's academic standing per course per term
type Progress struct {
	ID                uuid.UUID        `json:"id" validate:"required,uuid"`
	TenantID          uuid.UUID        `json:"tenant_id" validate:"required,uuid"`
	StudentID         uuid.UUID        `json:"student_id" validate:"required,uuid"`
	CourseID          uuid.UUID        `json:"course_id" validate:"required,uuid"`
	TermID            uuid.UUID        `json:"term_id" validate:"required,uuid"`
	ClassID           uuid.UUID        `json:"class_id" validate:"required,uuid"`
	Status            ProgressStatus   `json:"status" validate:"required,oneof=IN_PROGRESS COMPLETED FLAGGED"`
	QuizScores        []int            `json:"quiz_scores"`         // Array of quiz scores
	AssignmentScores  []int            `json:"assignment_scores"`   // Array of assignment scores
	ExaminationScore  *int             `json:"examination_score,omitempty"`
	Grade             *Grade           `json:"grade,omitempty"`     // Computed grade
	Attendance        AttendanceRecord `json:"attendance"`
	TutorRemarks      *string          `json:"tutor_remarks,omitempty" validate:"omitempty,max=1000"`
	PrincipalRemarks  *string          `json:"principal_remarks,omitempty" validate:"omitempty,max=1000"`
	ClassPosition     *int             `json:"class_position,omitempty"` // Ranking within class
	IsFlagged         bool             `json:"is_flagged"`          // True if below performance threshold
	FlagReason        *string          `json:"flag_reason,omitempty" validate:"omitempty,max=500"`
	CreatedAt         time.Time        `json:"created_at" validate:"required"`
	UpdatedAt         time.Time        `json:"updated_at" validate:"required"`
	CompletedAt       *time.Time       `json:"completed_at,omitempty"`
}

// ComputeContinuousAssessment calculates the total CA score
func (p *Progress) ComputeContinuousAssessment() int {
	total := 0
	for _, score := range p.QuizScores {
		total += score
	}
	for _, score := range p.AssignmentScores {
		total += score
	}
	return total
}

// ComputeGrade calculates the overall grade based on weighting
func (p *Progress) ComputeGrade(caWeight, examWeight int) *Grade {
	ca := p.ComputeContinuousAssessment()
	exam := 0
	if p.ExaminationScore != nil {
		exam = *p.ExaminationScore
	}

	// Calculate weighted total
	total := ((ca * caWeight) + (exam * examWeight)) / 100
	percentage := float64(total)

	// Determine letter grade and remark
	letterGrade, remark := getLetterGradeAndRemark(percentage)

	grade := &Grade{
		ContinuousAssessment: ca,
		Examination:          exam,
		Total:                total,
		Percentage:           percentage,
		LetterGrade:          letterGrade,
		Remark:               remark,
	}

	p.Grade = grade
	return grade
}

// getLetterGradeAndRemark returns the letter grade and remark based on percentage
func getLetterGradeAndRemark(percentage float64) (string, string) {
	switch {
	case percentage >= 80:
		return "A", "Excellent"
	case percentage >= 70:
		return "B", "Very Good"
	case percentage >= 60:
		return "C", "Good"
	case percentage >= 50:
		return "D", "Pass"
	case percentage >= 40:
		return "E", "Weak Pass"
	default:
		return "F", "Fail"
	}
}

// ComputeAttendancePercentage calculates attendance percentage
func (p *Progress) ComputeAttendancePercentage() float64 {
	if p.Attendance.TotalPeriods == 0 {
		return 0.0
	}
	return (float64(p.Attendance.PeriodsAttended) / float64(p.Attendance.TotalPeriods)) * 100
}

// IsCompleted returns true if the progress record is completed
func (p *Progress) IsCompleted() bool {
	return p.Status == ProgressStatusCompleted
}

// IsFlaggedForLowPerformance returns true if the student is flagged
func (p *Progress) IsFlaggedForLowPerformance() bool {
	return p.Status == ProgressStatusFlagged || p.IsFlagged
}

// FlagForLowPerformance flags the progress record with a reason
func (p *Progress) FlagForLowPerformance(reason string) {
	p.Status = ProgressStatusFlagged
	p.IsFlagged = true
	p.FlagReason = &reason
	p.UpdatedAt = time.Now()
}

// Complete marks the progress record as completed
func (p *Progress) Complete() {
	p.Status = ProgressStatusCompleted
	now := time.Now()
	p.CompletedAt = &now
	p.UpdatedAt = now
}

// ReportCard represents a student's complete academic report for a term
type ReportCard struct {
	ID                uuid.UUID   `json:"id" validate:"required,uuid"`
	TenantID          uuid.UUID   `json:"tenant_id" validate:"required,uuid"`
	StudentID         uuid.UUID   `json:"student_id" validate:"required,uuid"`
	TermID            uuid.UUID   `json:"term_id" validate:"required,uuid"`
	ClassID           uuid.UUID   `json:"class_id" validate:"required,uuid"`
	CourseProgresses  []Progress  `json:"course_progresses" validate:"required,min=1"` // All course progress records
	OverallPercentage float64     `json:"overall_percentage"`
	OverallGrade      string      `json:"overall_grade"`
	ClassPosition     int         `json:"class_position" validate:"required,min=1"`
	TotalStudents     int         `json:"total_students" validate:"required,min=1"`
	PrincipalRemarks  *string     `json:"principal_remarks,omitempty" validate:"omitempty,max=1000"`
	NextTermBegins    *time.Time  `json:"next_term_begins,omitempty"`
	GeneratedAt       time.Time   `json:"generated_at" validate:"required"`
	GeneratedBy       uuid.UUID   `json:"generated_by" validate:"required,uuid"`
	PDFUrl            *string     `json:"pdf_url,omitempty" validate:"omitempty,url"`
	CreatedAt         time.Time   `json:"created_at" validate:"required"`
	UpdatedAt         time.Time   `json:"updated_at" validate:"required"`
}

// ComputeOverallGrade calculates the overall grade across all courses
func (r *ReportCard) ComputeOverallGrade() {
	if len(r.CourseProgresses) == 0 {
		return
	}

	totalPercentage := 0.0
	for _, progress := range r.CourseProgresses {
		if progress.Grade != nil {
			totalPercentage += progress.Grade.Percentage
		}
	}

	r.OverallPercentage = totalPercentage / float64(len(r.CourseProgresses))
	r.OverallGrade, _ = getLetterGradeAndRemark(r.OverallPercentage)
}
