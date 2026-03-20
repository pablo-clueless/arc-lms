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

// AssessmentService handles quiz and assignment operations
type AssessmentService struct {
	quizRepo       *postgres.QuizRepository
	assignmentRepo *postgres.AssignmentRepository
	courseRepo     *postgres.CourseRepository
	auditService   *AuditService
}

// NewAssessmentService creates a new assessment service
func NewAssessmentService(
	quizRepo *postgres.QuizRepository,
	assignmentRepo *postgres.AssignmentRepository,
	courseRepo *postgres.CourseRepository,
	auditService *AuditService,
) *AssessmentService {
	return &AssessmentService{
		quizRepo:       quizRepo,
		assignmentRepo: assignmentRepo,
		courseRepo:     courseRepo,
		auditService:   auditService,
	}
}

// CreateQuizRequest represents a request to create a quiz
type CreateQuizRequest struct {
	CourseID          uuid.UUID         `json:"course_id" validate:"required"`
	Title             string            `json:"title" validate:"required,min=3,max=200"`
	Instructions      string            `json:"instructions" validate:"required,min=10,max=2000"`
	Questions         []domain.Question `json:"questions" validate:"required,min=1"`
	TimeLimit         int               `json:"time_limit" validate:"required,min=1,max=300"`
	AvailabilityStart time.Time         `json:"availability_start" validate:"required"`
	AvailabilityEnd   time.Time         `json:"availability_end" validate:"required"`
	ShowBeforeWindow  bool              `json:"show_before_window"`
	AllowRetake       bool              `json:"allow_retake"`
	PassingPercentage *int              `json:"passing_percentage,omitempty"`
}

// CreateQuiz creates a new quiz
func (s *AssessmentService) CreateQuiz(ctx context.Context, tenantID, tutorID uuid.UUID, req *CreateQuizRequest) (*domain.Quiz, error) {
	// Verify course exists and belongs to tenant
	course, err := s.courseRepo.Get(ctx, req.CourseID)
	if err != nil {
		return nil, fmt.Errorf("course not found: %w", err)
	}
	if course.TenantID != tenantID {
		return nil, fmt.Errorf("course does not belong to tenant")
	}

	// Calculate total marks
	totalMarks := 0
	for i := range req.Questions {
		req.Questions[i].ID = uuid.New()
		req.Questions[i].OrderIndex = i
		totalMarks += req.Questions[i].Marks
	}

	now := time.Now()
	quiz := &domain.Quiz{
		ID:                uuid.New(),
		TenantID:          tenantID,
		CourseID:          req.CourseID,
		CreatedByTutorID:  tutorID,
		Title:             req.Title,
		Instructions:      req.Instructions,
		Questions:         req.Questions,
		TotalMarks:        totalMarks,
		TimeLimit:         req.TimeLimit,
		AvailabilityStart: req.AvailabilityStart,
		AvailabilityEnd:   req.AvailabilityEnd,
		Status:            domain.AssessmentStatusDraft,
		ShowBeforeWindow:  req.ShowBeforeWindow,
		AllowRetake:       req.AllowRetake,
		PassingPercentage: req.PassingPercentage,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.quizRepo.Create(ctx, quiz); err != nil {
		return nil, fmt.Errorf("failed to create quiz: %w", err)
	}

	return quiz, nil
}

// GetQuiz retrieves a quiz by ID
func (s *AssessmentService) GetQuiz(ctx context.Context, id uuid.UUID) (*domain.Quiz, error) {
	return s.quizRepo.GetByID(ctx, id)
}

// UpdateQuiz updates a quiz
func (s *AssessmentService) UpdateQuiz(ctx context.Context, id uuid.UUID, req *CreateQuizRequest) (*domain.Quiz, error) {
	quiz, err := s.quizRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("quiz not found: %w", err)
	}

	// Cannot update published quiz
	if quiz.Status == domain.AssessmentStatusPublished {
		return nil, fmt.Errorf("cannot update published quiz")
	}

	// Recalculate total marks
	totalMarks := 0
	for i := range req.Questions {
		if req.Questions[i].ID == uuid.Nil {
			req.Questions[i].ID = uuid.New()
		}
		req.Questions[i].OrderIndex = i
		totalMarks += req.Questions[i].Marks
	}

	quiz.Title = req.Title
	quiz.Instructions = req.Instructions
	quiz.Questions = req.Questions
	quiz.TotalMarks = totalMarks
	quiz.TimeLimit = req.TimeLimit
	quiz.AvailabilityStart = req.AvailabilityStart
	quiz.AvailabilityEnd = req.AvailabilityEnd
	quiz.ShowBeforeWindow = req.ShowBeforeWindow
	quiz.AllowRetake = req.AllowRetake
	quiz.PassingPercentage = req.PassingPercentage
	quiz.UpdatedAt = time.Now()

	if err := s.quizRepo.Update(ctx, quiz); err != nil {
		return nil, fmt.Errorf("failed to update quiz: %w", err)
	}

	return quiz, nil
}

// PublishQuiz publishes a quiz
func (s *AssessmentService) PublishQuiz(ctx context.Context, id uuid.UUID) (*domain.Quiz, error) {
	quiz, err := s.quizRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("quiz not found: %w", err)
	}

	if quiz.Status == domain.AssessmentStatusPublished {
		return nil, fmt.Errorf("quiz is already published")
	}

	quiz.Publish()
	if err := s.quizRepo.Update(ctx, quiz); err != nil {
		return nil, fmt.Errorf("failed to publish quiz: %w", err)
	}

	return quiz, nil
}

// DeleteQuiz deletes a quiz
func (s *AssessmentService) DeleteQuiz(ctx context.Context, id uuid.UUID) error {
	quiz, err := s.quizRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("quiz not found: %w", err)
	}

	if quiz.Status == domain.AssessmentStatusPublished {
		return fmt.Errorf("cannot delete published quiz")
	}

	return s.quizRepo.Delete(ctx, id)
}

// ListQuizzesByCourse lists quizzes for a course
func (s *AssessmentService) ListQuizzesByCourse(ctx context.Context, courseID uuid.UUID, status *domain.AssessmentStatus, params repository.PaginationParams) ([]*domain.Quiz, *repository.PaginatedResult, error) {
	return s.quizRepo.ListByCourse(ctx, courseID, status, params)
}

// SubmitQuizRequest represents a quiz submission
type SubmitQuizRequest struct {
	Answers []domain.Answer `json:"answers" validate:"required"`
}

// StartQuiz starts a quiz attempt for a student
func (s *AssessmentService) StartQuiz(ctx context.Context, quizID, studentID uuid.UUID, ipAddress string) (*domain.QuizSubmission, error) {
	quiz, err := s.quizRepo.GetByID(ctx, quizID)
	if err != nil {
		return nil, fmt.Errorf("quiz not found: %w", err)
	}

	now := time.Now()
	if !quiz.IsAvailable(now) {
		return nil, fmt.Errorf("quiz is not available")
	}

	// Check for existing submission
	existing, err := s.quizRepo.GetSubmissionByQuizAndStudent(ctx, quizID, studentID)
	if err == nil && existing != nil {
		if existing.Status != domain.SubmissionStatusNotStarted {
			if !quiz.AllowRetake {
				return nil, fmt.Errorf("quiz does not allow retakes")
			}
		}
		// Return existing submission
		return existing, nil
	}

	submission := &domain.QuizSubmission{
		ID:        uuid.New(),
		TenantID:  quiz.TenantID,
		QuizID:    quizID,
		StudentID: studentID,
		Status:    domain.SubmissionStatusInProgress,
		StartedAt: &now,
		Answers:   []domain.Answer{},
		IPAddress: &ipAddress,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.quizRepo.CreateSubmission(ctx, submission); err != nil {
		return nil, fmt.Errorf("failed to start quiz: %w", err)
	}

	return submission, nil
}

// SubmitQuiz submits a student's quiz answers
func (s *AssessmentService) SubmitQuiz(ctx context.Context, submissionID uuid.UUID, req *SubmitQuizRequest) (*domain.QuizSubmission, error) {
	submission, err := s.quizRepo.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return nil, fmt.Errorf("submission not found: %w", err)
	}

	if submission.IsSubmitted() {
		return nil, fmt.Errorf("quiz already submitted")
	}

	quiz, err := s.quizRepo.GetByID(ctx, submission.QuizID)
	if err != nil {
		return nil, fmt.Errorf("quiz not found: %w", err)
	}

	now := time.Now()

	// Check if submission is late
	status := domain.SubmissionStatusSubmitted
	if now.After(quiz.AvailabilityEnd) {
		status = domain.SubmissionStatusLate
	}

	// Auto-grade auto-gradable questions
	score := 0
	isAutoGraded := true
	gradedAnswers := make([]domain.Answer, len(req.Answers))

	questionMap := make(map[uuid.UUID]domain.Question)
	for _, q := range quiz.Questions {
		questionMap[q.ID] = q
	}

	for i, answer := range req.Answers {
		gradedAnswers[i] = answer
		question, exists := questionMap[answer.QuestionID]
		if !exists {
			continue
		}

		switch question.Type {
		case domain.QuestionTypeMultipleChoice, domain.QuestionTypeTrueFalse:
			// Auto-grade
			if len(answer.SelectedOptions) > 0 && len(question.CorrectAnswers) > 0 {
				isCorrect := answer.SelectedOptions[0] == question.CorrectAnswers[0]
				gradedAnswers[i].IsCorrect = &isCorrect
				if isCorrect {
					marks := question.Marks
					gradedAnswers[i].MarksEarned = &marks
					score += marks
				} else {
					zero := 0
					gradedAnswers[i].MarksEarned = &zero
				}
			}
		case domain.QuestionTypeMultipleAnswer:
			// Auto-grade - all correct answers must be selected
			if len(answer.SelectedOptions) == len(question.CorrectAnswers) {
				correct := true
				correctSet := make(map[string]bool)
				for _, c := range question.CorrectAnswers {
					correctSet[c] = true
				}
				for _, s := range answer.SelectedOptions {
					if !correctSet[s] {
						correct = false
						break
					}
				}
				gradedAnswers[i].IsCorrect = &correct
				if correct {
					marks := question.Marks
					gradedAnswers[i].MarksEarned = &marks
					score += marks
				} else {
					zero := 0
					gradedAnswers[i].MarksEarned = &zero
				}
			}
		case domain.QuestionTypeShortAnswer, domain.QuestionTypeEssay:
			// Requires manual grading
			isAutoGraded = false
		}
	}

	submission.Answers = gradedAnswers
	submission.Status = status
	submission.SubmittedAt = &now
	submission.UpdatedAt = now
	submission.IsAutoGraded = isAutoGraded

	if isAutoGraded {
		submission.Score = &score
		percentage := float64(score) / float64(quiz.TotalMarks) * 100
		submission.Percentage = &percentage
		submission.Status = domain.SubmissionStatusGraded
		submission.GradedAt = &now
	}

	if err := s.quizRepo.UpdateSubmission(ctx, submission); err != nil {
		return nil, fmt.Errorf("failed to submit quiz: %w", err)
	}

	return submission, nil
}

// GradeQuizRequest represents a grading request
type GradeQuizRequest struct {
	Answers  []domain.Answer `json:"answers" validate:"required"`
	Feedback *string         `json:"feedback,omitempty"`
}

// GradeQuiz grades a quiz submission (for manual grading)
func (s *AssessmentService) GradeQuiz(ctx context.Context, submissionID, tutorID uuid.UUID, req *GradeQuizRequest) (*domain.QuizSubmission, error) {
	submission, err := s.quizRepo.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return nil, fmt.Errorf("submission not found: %w", err)
	}

	if !submission.IsSubmitted() {
		return nil, fmt.Errorf("quiz not yet submitted")
	}

	quiz, err := s.quizRepo.GetByID(ctx, submission.QuizID)
	if err != nil {
		return nil, fmt.Errorf("quiz not found: %w", err)
	}

	// Calculate total score
	score := 0
	for _, answer := range req.Answers {
		if answer.MarksEarned != nil {
			score += *answer.MarksEarned
		}
	}

	now := time.Now()
	percentage := float64(score) / float64(quiz.TotalMarks) * 100

	submission.Answers = req.Answers
	submission.Score = &score
	submission.Percentage = &percentage
	submission.Feedback = req.Feedback
	submission.Status = domain.SubmissionStatusGraded
	submission.GradedAt = &now
	submission.GradedBy = &tutorID
	submission.UpdatedAt = now

	if err := s.quizRepo.UpdateSubmission(ctx, submission); err != nil {
		return nil, fmt.Errorf("failed to grade quiz: %w", err)
	}

	return submission, nil
}

// GetQuizSubmission retrieves a quiz submission
func (s *AssessmentService) GetQuizSubmission(ctx context.Context, id uuid.UUID) (*domain.QuizSubmission, error) {
	return s.quizRepo.GetSubmissionByID(ctx, id)
}

// ListQuizSubmissions lists submissions for a quiz
func (s *AssessmentService) ListQuizSubmissions(ctx context.Context, quizID uuid.UUID, params repository.PaginationParams) ([]*domain.QuizSubmission, *repository.PaginatedResult, error) {
	return s.quizRepo.ListSubmissionsByQuiz(ctx, quizID, params)
}

// === Assignment Methods ===

// CreateAssignmentRequest represents a request to create an assignment
type CreateAssignmentRequest struct {
	CourseID            uuid.UUID         `json:"course_id" validate:"required"`
	Title               string            `json:"title" validate:"required,min=3,max=200"`
	Description         string            `json:"description" validate:"required,min=10,max=5000"`
	AttachmentURLs      []string          `json:"attachment_urls,omitempty"`
	MaxMarks            int               `json:"max_marks" validate:"required,min=1,max=100"`
	SubmissionDeadline  time.Time         `json:"submission_deadline" validate:"required"`
	AllowLateSubmission bool              `json:"allow_late_submission"`
	HardCutoffDate      *time.Time        `json:"hard_cutoff_date,omitempty"`
	AllowedFileFormats  []string          `json:"allowed_file_formats,omitempty"`
	MaxFileSize         int               `json:"max_file_size" validate:"required,min=1,max=104857600"`
	Questions           []domain.Question `json:"questions,omitempty"`
}

// CreateAssignment creates a new assignment
func (s *AssessmentService) CreateAssignment(ctx context.Context, tenantID, tutorID uuid.UUID, req *CreateAssignmentRequest) (*domain.Assignment, error) {
	// Verify course exists
	course, err := s.courseRepo.Get(ctx, req.CourseID)
	if err != nil {
		return nil, fmt.Errorf("course not found: %w", err)
	}
	if course.TenantID != tenantID {
		return nil, fmt.Errorf("course does not belong to tenant")
	}

	// Generate IDs for questions
	for i := range req.Questions {
		req.Questions[i].ID = uuid.New()
		req.Questions[i].OrderIndex = i
	}

	now := time.Now()
	assignment := &domain.Assignment{
		ID:                  uuid.New(),
		TenantID:            tenantID,
		CourseID:            req.CourseID,
		CreatedByTutorID:    tutorID,
		Title:               req.Title,
		Description:         req.Description,
		AttachmentURLs:      req.AttachmentURLs,
		MaxMarks:            req.MaxMarks,
		SubmissionDeadline:  req.SubmissionDeadline,
		AllowLateSubmission: req.AllowLateSubmission,
		HardCutoffDate:      req.HardCutoffDate,
		AllowedFileFormats:  req.AllowedFileFormats,
		MaxFileSize:         req.MaxFileSize,
		Status:              domain.AssessmentStatusDraft,
		Questions:           req.Questions,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := s.assignmentRepo.Create(ctx, assignment); err != nil {
		return nil, fmt.Errorf("failed to create assignment: %w", err)
	}

	return assignment, nil
}

// GetAssignment retrieves an assignment by ID
func (s *AssessmentService) GetAssignment(ctx context.Context, id uuid.UUID) (*domain.Assignment, error) {
	return s.assignmentRepo.GetByID(ctx, id)
}

// UpdateAssignment updates an assignment
func (s *AssessmentService) UpdateAssignment(ctx context.Context, id uuid.UUID, req *CreateAssignmentRequest) (*domain.Assignment, error) {
	assignment, err := s.assignmentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("assignment not found: %w", err)
	}

	if assignment.Status == domain.AssessmentStatusPublished {
		return nil, fmt.Errorf("cannot update published assignment")
	}

	for i := range req.Questions {
		if req.Questions[i].ID == uuid.Nil {
			req.Questions[i].ID = uuid.New()
		}
		req.Questions[i].OrderIndex = i
	}

	assignment.Title = req.Title
	assignment.Description = req.Description
	assignment.AttachmentURLs = req.AttachmentURLs
	assignment.MaxMarks = req.MaxMarks
	assignment.SubmissionDeadline = req.SubmissionDeadline
	assignment.AllowLateSubmission = req.AllowLateSubmission
	assignment.HardCutoffDate = req.HardCutoffDate
	assignment.AllowedFileFormats = req.AllowedFileFormats
	assignment.MaxFileSize = req.MaxFileSize
	assignment.Questions = req.Questions
	assignment.UpdatedAt = time.Now()

	if err := s.assignmentRepo.Update(ctx, assignment); err != nil {
		return nil, fmt.Errorf("failed to update assignment: %w", err)
	}

	return assignment, nil
}

// PublishAssignment publishes an assignment
func (s *AssessmentService) PublishAssignment(ctx context.Context, id uuid.UUID) (*domain.Assignment, error) {
	assignment, err := s.assignmentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("assignment not found: %w", err)
	}

	if assignment.Status == domain.AssessmentStatusPublished {
		return nil, fmt.Errorf("assignment is already published")
	}

	assignment.Publish()
	if err := s.assignmentRepo.Update(ctx, assignment); err != nil {
		return nil, fmt.Errorf("failed to publish assignment: %w", err)
	}

	return assignment, nil
}

// DeleteAssignment deletes an assignment
func (s *AssessmentService) DeleteAssignment(ctx context.Context, id uuid.UUID) error {
	assignment, err := s.assignmentRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("assignment not found: %w", err)
	}

	if assignment.Status == domain.AssessmentStatusPublished {
		return fmt.Errorf("cannot delete published assignment")
	}

	return s.assignmentRepo.Delete(ctx, id)
}

// ListAssignmentsByCourse lists assignments for a course
func (s *AssessmentService) ListAssignmentsByCourse(ctx context.Context, courseID uuid.UUID, status *domain.AssessmentStatus, params repository.PaginationParams) ([]*domain.Assignment, *repository.PaginatedResult, error) {
	return s.assignmentRepo.ListByCourse(ctx, courseID, status, params)
}

// SubmitAssignmentRequest represents an assignment submission
type SubmitAssignmentRequest struct {
	FileURLs   []string `json:"file_urls,omitempty"`
	AnswerText *string  `json:"answer_text,omitempty"`
}

// SubmitAssignment submits an assignment
func (s *AssessmentService) SubmitAssignment(ctx context.Context, assignmentID, studentID uuid.UUID, req *SubmitAssignmentRequest, ipAddress string) (*domain.AssignmentSubmission, error) {
	assignment, err := s.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, fmt.Errorf("assignment not found: %w", err)
	}

	now := time.Now()
	if !assignment.CanSubmit(now) {
		return nil, fmt.Errorf("assignment submission is closed")
	}

	// Check for existing submission
	existing, err := s.assignmentRepo.GetSubmissionByAssignmentAndStudent(ctx, assignmentID, studentID)
	if err == nil && existing != nil && existing.IsSubmitted() {
		return nil, fmt.Errorf("assignment already submitted")
	}

	isLate := assignment.IsOverdue(now)
	status := domain.SubmissionStatusSubmitted
	if isLate {
		status = domain.SubmissionStatusLate
	}

	submission := &domain.AssignmentSubmission{
		ID:           uuid.New(),
		TenantID:     assignment.TenantID,
		AssignmentID: assignmentID,
		StudentID:    studentID,
		Status:       status,
		SubmittedAt:  &now,
		IsLate:       isLate,
		FileURLs:     req.FileURLs,
		AnswerText:   req.AnswerText,
		IPAddress:    &ipAddress,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.assignmentRepo.CreateSubmission(ctx, submission); err != nil {
		return nil, fmt.Errorf("failed to submit assignment: %w", err)
	}

	return submission, nil
}

// GradeAssignmentRequest represents a grading request
type GradeAssignmentRequest struct {
	Score    int     `json:"score" validate:"required,min=0"`
	Feedback *string `json:"feedback,omitempty"`
}

// GradeAssignment grades an assignment submission
func (s *AssessmentService) GradeAssignment(ctx context.Context, submissionID, tutorID uuid.UUID, req *GradeAssignmentRequest) (*domain.AssignmentSubmission, error) {
	submission, err := s.assignmentRepo.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return nil, fmt.Errorf("submission not found: %w", err)
	}

	if !submission.IsSubmitted() {
		return nil, fmt.Errorf("assignment not yet submitted")
	}

	now := time.Now()
	submission.Score = &req.Score
	submission.Feedback = req.Feedback
	submission.Status = domain.SubmissionStatusGraded
	submission.GradedAt = &now
	submission.GradedBy = &tutorID
	submission.UpdatedAt = now

	if err := s.assignmentRepo.UpdateSubmission(ctx, submission); err != nil {
		return nil, fmt.Errorf("failed to grade assignment: %w", err)
	}

	return submission, nil
}

// GetAssignmentSubmission retrieves an assignment submission
func (s *AssessmentService) GetAssignmentSubmission(ctx context.Context, id uuid.UUID) (*domain.AssignmentSubmission, error) {
	return s.assignmentRepo.GetSubmissionByID(ctx, id)
}

// ListAssignmentSubmissions lists submissions for an assignment
func (s *AssessmentService) ListAssignmentSubmissions(ctx context.Context, assignmentID uuid.UUID, params repository.PaginationParams) ([]*domain.AssignmentSubmission, *repository.PaginatedResult, error) {
	return s.assignmentRepo.ListSubmissionsByAssignment(ctx, assignmentID, params)
}
