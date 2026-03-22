package service

import (
	"context"
	"fmt"
	"time"

	"arc-lms/internal/domain"
	"arc-lms/internal/repository"
	"arc-lms/internal/repository/postgres"

	"github.com/google/uuid"
)

// ExaminationService handles examination management operations
type ExaminationService struct {
	examRepo     *postgres.ExaminationRepository
	courseRepo   *postgres.CourseRepository
	auditService *AuditService
}

// NewExaminationService creates a new examination service
func NewExaminationService(
	examRepo *postgres.ExaminationRepository,
	courseRepo *postgres.CourseRepository,
	auditService *AuditService,
) *ExaminationService {
	return &ExaminationService{
		examRepo:     examRepo,
		courseRepo:   courseRepo,
		auditService: auditService,
	}
}

// CreateExaminationRequest represents a request to create an examination
type CreateExaminationRequest struct {
	CourseID     uuid.UUID                    `json:"course_id" validate:"required,uuid"`
	TermID       uuid.UUID                    `json:"term_id" validate:"required,uuid"`
	Title        string                       `json:"title" validate:"required,min=3,max=200"`
	Instructions string                       `json:"instructions" validate:"required,min=10,max=3000"`
	Questions    []domain.ExaminationQuestion `json:"questions" validate:"required,min=1,dive"`
	TotalMarks   int                          `json:"total_marks" validate:"required,min=1"`
	Duration     int                          `json:"duration" validate:"required,min=30,max=300"`
	WindowStart  time.Time                    `json:"window_start" validate:"required"`
	WindowEnd    time.Time                    `json:"window_end" validate:"required,gtfield=WindowStart"`
}

// UpdateExaminationRequest represents a request to update an examination
type UpdateExaminationRequest struct {
	Title        *string                       `json:"title,omitempty" validate:"omitempty,min=3,max=200"`
	Instructions *string                       `json:"instructions,omitempty" validate:"omitempty,min=10,max=3000"`
	Questions    *[]domain.ExaminationQuestion `json:"questions,omitempty" validate:"omitempty,min=1,dive"`
	TotalMarks   *int                          `json:"total_marks,omitempty" validate:"omitempty,min=1"`
	Duration     *int                          `json:"duration,omitempty" validate:"omitempty,min=30,max=300"`
	WindowStart  *time.Time                    `json:"window_start,omitempty"`
	WindowEnd    *time.Time                    `json:"window_end,omitempty"`
}

// CreateExamination creates a new examination
func (s *ExaminationService) CreateExamination(
	ctx context.Context,
	tenantID uuid.UUID,
	req *CreateExaminationRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Examination, error) {
	// Verify course belongs to tenant
	course, err := s.courseRepo.Get(ctx, req.CourseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get course: %w", err)
	}
	if course.TenantID != tenantID {
		return nil, fmt.Errorf("course does not belong to tenant")
	}

	// Verify actor can create examination (ADMIN or assigned tutor)
	if actorRole == domain.RoleTutor && course.AssignedTutorID != actorID {
		return nil, fmt.Errorf("you are not assigned to this course")
	}

	// Validate window dates
	if req.WindowEnd.Before(req.WindowStart) {
		return nil, fmt.Errorf("window end must be after window start")
	}

	// Calculate total marks from questions
	calculatedTotal := 0
	for i := range req.Questions {
		req.Questions[i].ID = uuid.New()
		req.Questions[i].OrderIndex = i
		calculatedTotal += req.Questions[i].Marks
	}

	if calculatedTotal != req.TotalMarks {
		return nil, fmt.Errorf("total marks (%d) does not match sum of question marks (%d)", req.TotalMarks, calculatedTotal)
	}

	now := time.Now()
	exam := &domain.Examination{
		ID:               uuid.New(),
		TenantID:         tenantID,
		CourseID:         req.CourseID,
		TermID:           req.TermID,
		CreatedByID:      actorID,
		Title:            req.Title,
		Instructions:     req.Instructions,
		Questions:        req.Questions,
		TotalMarks:       req.TotalMarks,
		Duration:         req.Duration,
		WindowStart:      req.WindowStart,
		WindowEnd:        req.WindowEnd,
		Status:           domain.ExaminationStatusDraft,
		ResultsPublished: false,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.examRepo.Create(ctx, exam, nil); err != nil {
		return nil, fmt.Errorf("failed to create examination: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionExaminationCreated,
		actorID,
		actorRole,
		&tenantID,
		domain.AuditResourceExamination,
		exam.ID,
		nil,
		exam,
		ipAddress,
	)

	return exam, nil
}

// GetExamination retrieves an examination by ID
func (s *ExaminationService) GetExamination(ctx context.Context, id uuid.UUID) (*domain.Examination, error) {
	return s.examRepo.Get(ctx, id)
}

// UpdateExamination updates an examination (only allowed if in DRAFT status)
func (s *ExaminationService) UpdateExamination(
	ctx context.Context,
	id uuid.UUID,
	req *UpdateExaminationRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Examination, error) {
	exam, err := s.examRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get examination: %w", err)
	}

	// Only allow updates when in DRAFT status
	if exam.Status != domain.ExaminationStatusDraft {
		return nil, fmt.Errorf("can only update examinations in DRAFT status")
	}

	// Store before state for audit
	beforeState := *exam

	// Update fields
	if req.Title != nil {
		exam.Title = *req.Title
	}
	if req.Instructions != nil {
		exam.Instructions = *req.Instructions
	}
	if req.Questions != nil {
		// Recalculate total if questions are updated
		calculatedTotal := 0
		for i := range *req.Questions {
			(*req.Questions)[i].ID = uuid.New()
			(*req.Questions)[i].OrderIndex = i
			calculatedTotal += (*req.Questions)[i].Marks
		}
		exam.Questions = *req.Questions
		exam.TotalMarks = calculatedTotal
	}
	if req.TotalMarks != nil {
		exam.TotalMarks = *req.TotalMarks
	}
	if req.Duration != nil {
		exam.Duration = *req.Duration
	}
	if req.WindowStart != nil {
		exam.WindowStart = *req.WindowStart
	}
	if req.WindowEnd != nil {
		exam.WindowEnd = *req.WindowEnd
	}

	// Validate window dates
	if exam.WindowEnd.Before(exam.WindowStart) {
		return nil, fmt.Errorf("window end must be after window start")
	}

	exam.UpdatedAt = time.Now()

	if err := s.examRepo.Update(ctx, exam); err != nil {
		return nil, fmt.Errorf("failed to update examination: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionExaminationUpdated,
		actorID,
		actorRole,
		&exam.TenantID,
		domain.AuditResourceExamination,
		exam.ID,
		&beforeState,
		exam,
		ipAddress,
	)

	return exam, nil
}

// DeleteExamination deletes an examination (only allowed if in DRAFT status)
func (s *ExaminationService) DeleteExamination(
	ctx context.Context,
	id uuid.UUID,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) error {
	exam, err := s.examRepo.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get examination: %w", err)
	}

	// Only allow deletion when in DRAFT status
	if exam.Status != domain.ExaminationStatusDraft {
		return fmt.Errorf("can only delete examinations in DRAFT status")
	}

	if err := s.examRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete examination: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionExaminationDeleted,
		actorID,
		actorRole,
		&exam.TenantID,
		domain.AuditResourceExamination,
		exam.ID,
		exam,
		nil,
		ipAddress,
	)

	return nil
}

// ScheduleExamination schedules an examination (moves from DRAFT to SCHEDULED)
func (s *ExaminationService) ScheduleExamination(
	ctx context.Context,
	id uuid.UUID,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Examination, error) {
	exam, err := s.examRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get examination: %w", err)
	}

	if exam.Status != domain.ExaminationStatusDraft {
		return nil, fmt.Errorf("can only schedule examinations in DRAFT status")
	}

	// Validate examination has questions
	if len(exam.Questions) == 0 {
		return nil, fmt.Errorf("examination must have at least one question")
	}

	// Validate window is in the future
	if exam.WindowStart.Before(time.Now()) {
		return nil, fmt.Errorf("examination window start must be in the future")
	}

	exam.Schedule()

	if err := s.examRepo.Update(ctx, exam); err != nil {
		return nil, fmt.Errorf("failed to schedule examination: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionExaminationScheduled,
		actorID,
		actorRole,
		&exam.TenantID,
		domain.AuditResourceExamination,
		exam.ID,
		nil,
		exam,
		ipAddress,
	)

	return exam, nil
}

// ListExaminations lists examinations with filters
func (s *ExaminationService) ListExaminations(
	ctx context.Context,
	tenantID uuid.UUID,
	courseID *uuid.UUID,
	termID *uuid.UUID,
	status *domain.ExaminationStatus,
	params repository.PaginationParams,
) ([]*domain.Examination, *repository.PaginatedResult, error) {
	var exams []*domain.Examination
	var total int
	var err error

	if courseID != nil {
		exams, total, err = s.examRepo.ListByCourse(ctx, *courseID, params)
	} else if termID != nil {
		exams, total, err = s.examRepo.ListByTerm(ctx, *termID, params)
	} else {
		exams, total, err = s.examRepo.ListByTenant(ctx, tenantID, status, params)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to list examinations: %w", err)
	}

	pagination := repository.BuildPaginatedResult(total, params)

	return exams, &pagination, nil
}

// StartExaminationRequest represents a request to start an examination
type StartExaminationRequest struct {
	IPAddress string `json:"ip_address" validate:"required"`
}

// StartExamination starts an examination attempt for a student
func (s *ExaminationService) StartExamination(
	ctx context.Context,
	examID uuid.UUID,
	studentID uuid.UUID,
	ipAddress string,
) (*domain.ExaminationSubmission, error) {
	exam, err := s.examRepo.Get(ctx, examID)
	if err != nil {
		return nil, fmt.Errorf("failed to get examination: %w", err)
	}

	now := time.Now()

	// Check if examination is accessible
	if !exam.CanAccess(now) {
		if now.Before(exam.WindowStart) {
			return nil, fmt.Errorf("examination has not started yet")
		}
		if now.After(exam.WindowEnd) {
			return nil, fmt.Errorf("examination window has ended")
		}
		return nil, fmt.Errorf("examination is not accessible")
	}

	// Check if student already has a submission
	existing, err := s.examRepo.GetSubmissionByStudentAndExam(ctx, studentID, examID)
	if err == nil && existing != nil {
		// Check if already submitted
		if existing.IsSubmitted() {
			return nil, fmt.Errorf("you have already submitted this examination")
		}
		// Return existing in-progress submission
		if existing.Status == domain.ExamSubmissionStatusInProgress {
			return existing, nil
		}
	}

	// Create new submission
	submission := &domain.ExaminationSubmission{
		ID:              uuid.New(),
		TenantID:        exam.TenantID,
		ExaminationID:   examID,
		StudentID:       studentID,
		Status:          domain.ExamSubmissionStatusInProgress,
		StartedAt:       &now,
		Answers:         make([]domain.ExaminationAnswer, 0),
		IntegrityEvents: make([]domain.IntegrityEvent, 0),
		IPAddress:       ipAddress,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.examRepo.CreateSubmission(ctx, submission, nil); err != nil {
		return nil, fmt.Errorf("failed to create submission: %w", err)
	}

	return submission, nil
}

// SaveAnswerRequest represents a request to save an answer
type SaveAnswerRequest struct {
	QuestionID      uuid.UUID `json:"question_id" validate:"required,uuid"`
	AnswerText      *string   `json:"answer_text,omitempty"`
	SelectedOptions []string  `json:"selected_options,omitempty"`
}

// SaveAnswer saves an individual answer (for progressive saving)
func (s *ExaminationService) SaveAnswer(
	ctx context.Context,
	submissionID uuid.UUID,
	req *SaveAnswerRequest,
	studentID uuid.UUID,
) (*domain.ExaminationSubmission, error) {
	submission, err := s.examRepo.GetSubmission(ctx, submissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get submission: %w", err)
	}

	// Verify ownership
	if submission.StudentID != studentID {
		return nil, fmt.Errorf("you do not own this submission")
	}

	// Verify submission is in progress
	if submission.Status != domain.ExamSubmissionStatusInProgress {
		return nil, fmt.Errorf("submission is not in progress")
	}

	// Check if examination window has ended
	exam, err := s.examRepo.Get(ctx, submission.ExaminationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get examination: %w", err)
	}

	now := time.Now()
	if now.After(exam.WindowEnd) {
		// Auto-submit if window has ended
		return s.autoSubmit(ctx, submission, exam)
	}

	// Check duration limit
	if submission.StartedAt != nil {
		deadline := submission.StartedAt.Add(time.Duration(exam.Duration) * time.Minute)
		if now.After(deadline) {
			return s.autoSubmit(ctx, submission, exam)
		}
	}

	// Find or create answer
	found := false
	for i, ans := range submission.Answers {
		if ans.QuestionID == req.QuestionID {
			submission.Answers[i].AnswerText = req.AnswerText
			submission.Answers[i].SelectedOptions = req.SelectedOptions
			submission.Answers[i].SubmittedAt = now
			found = true
			break
		}
	}

	if !found {
		submission.Answers = append(submission.Answers, domain.ExaminationAnswer{
			QuestionID:      req.QuestionID,
			AnswerText:      req.AnswerText,
			SelectedOptions: req.SelectedOptions,
			SubmittedAt:     now,
		})
	}

	submission.UpdatedAt = now

	if err := s.examRepo.UpdateSubmission(ctx, submission, nil); err != nil {
		return nil, fmt.Errorf("failed to save answer: %w", err)
	}

	return submission, nil
}

// RecordIntegrityEventRequest represents a request to record an integrity event
type RecordIntegrityEventRequest struct {
	EventType   string  `json:"event_type" validate:"required"`
	Description *string `json:"description,omitempty"`
}

// RecordIntegrityEvent records an integrity event during examination
func (s *ExaminationService) RecordIntegrityEvent(
	ctx context.Context,
	submissionID uuid.UUID,
	req *RecordIntegrityEventRequest,
	studentID uuid.UUID,
) error {
	submission, err := s.examRepo.GetSubmission(ctx, submissionID)
	if err != nil {
		return fmt.Errorf("failed to get submission: %w", err)
	}

	// Verify ownership
	if submission.StudentID != studentID {
		return fmt.Errorf("you do not own this submission")
	}

	// Verify submission is in progress
	if submission.Status != domain.ExamSubmissionStatusInProgress {
		return fmt.Errorf("submission is not in progress")
	}

	description := ""
	if req.Description != nil {
		description = *req.Description
	}

	submission.RecordIntegrityEvent(req.EventType, description)

	if err := s.examRepo.UpdateSubmission(ctx, submission, nil); err != nil {
		return fmt.Errorf("failed to record integrity event: %w", err)
	}

	return nil
}

// SubmitExamination submits an examination
func (s *ExaminationService) SubmitExamination(
	ctx context.Context,
	submissionID uuid.UUID,
	studentID uuid.UUID,
	ipAddress string,
) (*domain.ExaminationSubmission, error) {
	submission, err := s.examRepo.GetSubmission(ctx, submissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get submission: %w", err)
	}

	// Verify ownership
	if submission.StudentID != studentID {
		return nil, fmt.Errorf("you do not own this submission")
	}

	// Verify submission is in progress
	if submission.Status != domain.ExamSubmissionStatusInProgress {
		return nil, fmt.Errorf("submission is not in progress")
	}

	exam, err := s.examRepo.Get(ctx, submission.ExaminationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get examination: %w", err)
	}

	// Submit and grade
	return s.submitAndGrade(ctx, submission, exam, false)
}

// autoSubmit auto-submits an examination when time expires
func (s *ExaminationService) autoSubmit(
	ctx context.Context,
	submission *domain.ExaminationSubmission,
	exam *domain.Examination,
) (*domain.ExaminationSubmission, error) {
	return s.submitAndGrade(ctx, submission, exam, true)
}

// submitAndGrade submits and auto-grades an examination
func (s *ExaminationService) submitAndGrade(
	ctx context.Context,
	submission *domain.ExaminationSubmission,
	exam *domain.Examination,
	autoSubmitted bool,
) (*domain.ExaminationSubmission, error) {
	now := time.Now()
	submission.Status = domain.ExamSubmissionStatusSubmitted
	submission.SubmittedAt = &now
	submission.AutoSubmitted = autoSubmitted
	submission.UpdatedAt = now

	// Auto-grade eligible questions
	totalScore := 0
	hasManualGrading := false

	for i, answer := range submission.Answers {
		// Find the question
		var question *domain.ExaminationQuestion
		for _, q := range exam.Questions {
			if q.ID == answer.QuestionID {
				question = &q
				break
			}
		}

		if question == nil {
			continue
		}

		// Auto-grade based on question type
		switch question.Type {
		case domain.QuestionTypeMultipleChoice:
			// Single answer - check exact match
			isCorrect := len(answer.SelectedOptions) == 1 &&
				len(question.CorrectAnswers) == 1 &&
				answer.SelectedOptions[0] == question.CorrectAnswers[0]
			submission.Answers[i].IsCorrect = &isCorrect
			if isCorrect {
				marks := question.Marks
				submission.Answers[i].MarksEarned = &marks
				totalScore += marks
			} else {
				zero := 0
				submission.Answers[i].MarksEarned = &zero
			}

		case domain.QuestionTypeMultipleAnswer:
			// Multiple answers - check all correct and no wrong
			correct := true
			for _, selected := range answer.SelectedOptions {
				found := false
				for _, correctAns := range question.CorrectAnswers {
					if selected == correctAns {
						found = true
						break
					}
				}
				if !found {
					correct = false
					break
				}
			}
			// Also check all correct answers are selected
			if correct {
				for _, correctAns := range question.CorrectAnswers {
					found := false
					for _, selected := range answer.SelectedOptions {
						if selected == correctAns {
							found = true
							break
						}
					}
					if !found {
						correct = false
						break
					}
				}
			}
			submission.Answers[i].IsCorrect = &correct
			if correct {
				marks := question.Marks
				submission.Answers[i].MarksEarned = &marks
				totalScore += marks
			} else {
				zero := 0
				submission.Answers[i].MarksEarned = &zero
			}

		case domain.QuestionTypeTrueFalse:
			// True/False - check exact match
			isCorrect := len(answer.SelectedOptions) == 1 &&
				len(question.CorrectAnswers) == 1 &&
				answer.SelectedOptions[0] == question.CorrectAnswers[0]
			submission.Answers[i].IsCorrect = &isCorrect
			if isCorrect {
				marks := question.Marks
				submission.Answers[i].MarksEarned = &marks
				totalScore += marks
			} else {
				zero := 0
				submission.Answers[i].MarksEarned = &zero
			}

		case domain.QuestionTypeShortAnswer, domain.QuestionTypeEssay:
			// Requires manual grading
			hasManualGrading = true
		}
	}

	submission.Score = &totalScore
	percentage := float64(totalScore) / float64(exam.TotalMarks) * 100
	submission.Percentage = &percentage

	// If no manual grading needed, mark as graded
	if !hasManualGrading {
		submission.Status = domain.ExamSubmissionStatusGraded
		submission.IsAutoGraded = true
		submission.GradedAt = &now
	}

	if err := s.examRepo.UpdateSubmission(ctx, submission, nil); err != nil {
		return nil, fmt.Errorf("failed to submit examination: %w", err)
	}

	return submission, nil
}

// GradeAnswerRequest represents a request to grade a single answer
type GradeAnswerRequest struct {
	QuestionID  uuid.UUID `json:"question_id" validate:"required,uuid"`
	MarksEarned int       `json:"marks_earned" validate:"required,min=0"`
	Feedback    *string   `json:"feedback,omitempty"`
}

// GradeSubmission grades an examination submission (for manual grading)
func (s *ExaminationService) GradeSubmission(
	ctx context.Context,
	submissionID uuid.UUID,
	grades []GradeAnswerRequest,
	feedback *string,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.ExaminationSubmission, error) {
	submission, err := s.examRepo.GetSubmission(ctx, submissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get submission: %w", err)
	}

	// Verify submission is submitted but not yet graded
	if submission.Status != domain.ExamSubmissionStatusSubmitted {
		return nil, fmt.Errorf("submission must be in SUBMITTED status to grade")
	}

	exam, err := s.examRepo.Get(ctx, submission.ExaminationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get examination: %w", err)
	}

	// Apply grades
	totalScore := 0
	for i, answer := range submission.Answers {
		// Check if already auto-graded
		if submission.Answers[i].MarksEarned != nil {
			totalScore += *submission.Answers[i].MarksEarned
			continue
		}

		// Find manual grade
		for _, grade := range grades {
			if grade.QuestionID == answer.QuestionID {
				// Validate marks don't exceed question max
				for _, q := range exam.Questions {
					if q.ID == grade.QuestionID {
						if grade.MarksEarned > q.Marks {
							return nil, fmt.Errorf("marks for question %s exceed maximum (%d)", q.ID, q.Marks)
						}
						break
					}
				}
				submission.Answers[i].MarksEarned = &grade.MarksEarned
				submission.Answers[i].Feedback = grade.Feedback
				totalScore += grade.MarksEarned
				break
			}
		}
	}

	now := time.Now()
	submission.Score = &totalScore
	percentage := float64(totalScore) / float64(exam.TotalMarks) * 100
	submission.Percentage = &percentage
	submission.Status = domain.ExamSubmissionStatusGraded
	submission.Feedback = feedback
	submission.GradedAt = &now
	submission.GradedBy = &actorID
	submission.UpdatedAt = now

	if err := s.examRepo.UpdateSubmission(ctx, submission, nil); err != nil {
		return nil, fmt.Errorf("failed to grade submission: %w", err)
	}

	return submission, nil
}

// PublishResults publishes examination results to students
func (s *ExaminationService) PublishResults(
	ctx context.Context,
	examID uuid.UUID,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Examination, error) {
	exam, err := s.examRepo.Get(ctx, examID)
	if err != nil {
		return nil, fmt.Errorf("failed to get examination: %w", err)
	}

	if exam.ResultsPublished {
		return nil, fmt.Errorf("results are already published")
	}

	// Check that all submissions are graded
	counts, err := s.examRepo.CountSubmissionsByStatus(ctx, examID)
	if err != nil {
		return nil, fmt.Errorf("failed to count submissions: %w", err)
	}

	if counts[domain.ExamSubmissionStatusSubmitted] > 0 {
		return nil, fmt.Errorf("cannot publish results: %d submissions are still pending grading", counts[domain.ExamSubmissionStatusSubmitted])
	}

	if counts[domain.ExamSubmissionStatusInProgress] > 0 {
		return nil, fmt.Errorf("cannot publish results: %d students are still in progress", counts[domain.ExamSubmissionStatusInProgress])
	}

	// Publish results
	exam.PublishResults(actorID)
	exam.Complete()

	if err := s.examRepo.Update(ctx, exam); err != nil {
		return nil, fmt.Errorf("failed to publish results: %w", err)
	}

	// Update all graded submissions to published
	if err := s.examRepo.PublishResultsToStudents(ctx, examID, nil); err != nil {
		return nil, fmt.Errorf("failed to publish results to students: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionExaminationResultsPublished,
		actorID,
		actorRole,
		&exam.TenantID,
		domain.AuditResourceExamination,
		exam.ID,
		nil,
		exam,
		ipAddress,
	)

	return exam, nil
}

// GetSubmission retrieves an examination submission
func (s *ExaminationService) GetSubmission(ctx context.Context, id uuid.UUID) (*domain.ExaminationSubmission, error) {
	return s.examRepo.GetSubmission(ctx, id)
}

// GetStudentSubmission retrieves a student's submission for an examination
func (s *ExaminationService) GetStudentSubmission(ctx context.Context, studentID, examID uuid.UUID) (*domain.ExaminationSubmission, error) {
	return s.examRepo.GetSubmissionByStudentAndExam(ctx, studentID, examID)
}

// ListSubmissions lists submissions for an examination
func (s *ExaminationService) ListSubmissions(
	ctx context.Context,
	examID uuid.UUID,
	params repository.PaginationParams,
) ([]*domain.ExaminationSubmission, *repository.PaginatedResult, error) {
	submissions, total, err := s.examRepo.ListSubmissionsByExam(ctx, examID, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list submissions: %w", err)
	}

	pagination := repository.BuildPaginatedResult(total, params)

	return submissions, &pagination, nil
}

// GetPendingGradingSubmissions retrieves submissions pending manual grading
func (s *ExaminationService) GetPendingGradingSubmissions(ctx context.Context, examID uuid.UUID) ([]*domain.ExaminationSubmission, error) {
	return s.examRepo.GetPendingGradingSubmissions(ctx, examID)
}

// GetExaminationStats retrieves statistics for an examination
func (s *ExaminationService) GetExaminationStats(ctx context.Context, examID uuid.UUID) (*ExaminationStats, error) {
	exam, err := s.examRepo.Get(ctx, examID)
	if err != nil {
		return nil, fmt.Errorf("failed to get examination: %w", err)
	}

	counts, err := s.examRepo.CountSubmissionsByStatus(ctx, examID)
	if err != nil {
		return nil, fmt.Errorf("failed to count submissions: %w", err)
	}

	return &ExaminationStats{
		ExaminationID:    examID,
		TotalMarks:       exam.TotalMarks,
		Duration:         exam.Duration,
		NotStarted:       counts[domain.ExamSubmissionStatusNotStarted],
		InProgress:       counts[domain.ExamSubmissionStatusInProgress],
		Submitted:        counts[domain.ExamSubmissionStatusSubmitted],
		Graded:           counts[domain.ExamSubmissionStatusGraded],
		Published:        counts[domain.ExamSubmissionStatusPublished],
		ResultsPublished: exam.ResultsPublished,
	}, nil
}

// ExaminationStats represents examination statistics
type ExaminationStats struct {
	ExaminationID    uuid.UUID `json:"examination_id"`
	TotalMarks       int       `json:"total_marks"`
	Duration         int       `json:"duration"`
	NotStarted       int       `json:"not_started"`
	InProgress       int       `json:"in_progress"`
	Submitted        int       `json:"submitted"`
	Graded           int       `json:"graded"`
	Published        int       `json:"published"`
	ResultsPublished bool      `json:"results_published"`
}
