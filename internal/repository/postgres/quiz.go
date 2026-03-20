package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"arc-lms/internal/domain"
	"arc-lms/internal/repository"
)

// QuizRepository handles database operations for quizzes
type QuizRepository struct {
	*repository.BaseRepository
}

// NewQuizRepository creates a new quiz repository
func NewQuizRepository(db *sql.DB) *QuizRepository {
	return &QuizRepository{
		BaseRepository: repository.NewBaseRepository(db),
	}
}

// Create creates a new quiz
func (r *QuizRepository) Create(ctx context.Context, quiz *domain.Quiz) error {
	questionsJSON, err := json.Marshal(quiz.Questions)
	if err != nil {
		return fmt.Errorf("failed to marshal questions: %w", err)
	}

	query := `
		INSERT INTO quizzes (
			id, tenant_id, course_id, created_by_tutor_id, title, instructions,
			questions, total_marks, time_limit, availability_start, availability_end,
			status, show_before_window, allow_retake, passing_percentage,
			created_at, updated_at, published_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`

	_, err = r.GetDB().ExecContext(ctx, query,
		quiz.ID,
		quiz.TenantID,
		quiz.CourseID,
		quiz.CreatedByTutorID,
		quiz.Title,
		quiz.Instructions,
		questionsJSON,
		quiz.TotalMarks,
		quiz.TimeLimit,
		quiz.AvailabilityStart,
		quiz.AvailabilityEnd,
		quiz.Status,
		quiz.ShowBeforeWindow,
		quiz.AllowRetake,
		quiz.PassingPercentage,
		quiz.CreatedAt,
		quiz.UpdatedAt,
		quiz.PublishedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// GetByID retrieves a quiz by ID
func (r *QuizRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Quiz, error) {
	query := `
		SELECT
			id, tenant_id, course_id, created_by_tutor_id, title, instructions,
			questions, total_marks, time_limit, availability_start, availability_end,
			status, show_before_window, allow_retake, passing_percentage,
			created_at, updated_at, published_at
		FROM quizzes
		WHERE id = $1
	`

	return r.scanQuiz(r.GetDB().QueryRowContext(ctx, query, id))
}

// Update updates a quiz
func (r *QuizRepository) Update(ctx context.Context, quiz *domain.Quiz) error {
	questionsJSON, err := json.Marshal(quiz.Questions)
	if err != nil {
		return fmt.Errorf("failed to marshal questions: %w", err)
	}

	query := `
		UPDATE quizzes SET
			title = $2, instructions = $3, questions = $4, total_marks = $5,
			time_limit = $6, availability_start = $7, availability_end = $8,
			status = $9, show_before_window = $10, allow_retake = $11,
			passing_percentage = $12, updated_at = $13, published_at = $14
		WHERE id = $1
	`

	result, err := r.GetDB().ExecContext(ctx, query,
		quiz.ID,
		quiz.Title,
		quiz.Instructions,
		questionsJSON,
		quiz.TotalMarks,
		quiz.TimeLimit,
		quiz.AvailabilityStart,
		quiz.AvailabilityEnd,
		quiz.Status,
		quiz.ShowBeforeWindow,
		quiz.AllowRetake,
		quiz.PassingPercentage,
		quiz.UpdatedAt,
		quiz.PublishedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// Delete deletes a quiz
func (r *QuizRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM quizzes WHERE id = $1`

	result, err := r.GetDB().ExecContext(ctx, query, id)
	if err != nil {
		return repository.ParseError(err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// ListByCourse retrieves quizzes for a course
func (r *QuizRepository) ListByCourse(ctx context.Context, courseID uuid.UUID, status *domain.AssessmentStatus, params repository.PaginationParams) ([]*domain.Quiz, *repository.PaginatedResult, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, nil, err
	}

	query := `
		SELECT
			id, tenant_id, course_id, created_by_tutor_id, title, instructions,
			questions, total_marks, time_limit, availability_start, availability_end,
			status, show_before_window, allow_retake, passing_percentage,
			created_at, updated_at, published_at
		FROM quizzes
		WHERE course_id = $1
	`

	args := []interface{}{courseID}
	argIndex := 2

	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *status)
		argIndex++
	}

	if params.Cursor != nil {
		query += fmt.Sprintf(" AND id < $%d", argIndex)
		args = append(args, *params.Cursor)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d", argIndex)
	args = append(args, params.Limit+1)

	rows, err := r.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, repository.ParseError(err)
	}
	defer rows.Close()

	quizzes := make([]*domain.Quiz, 0)
	for rows.Next() {
		quiz, err := r.scanQuizFromRows(rows)
		if err != nil {
			return nil, nil, err
		}
		quizzes = append(quizzes, quiz)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, repository.ParseError(err)
	}

	hasMore := len(quizzes) > params.Limit
	var nextCursor *uuid.UUID
	if hasMore {
		quizzes = quizzes[:params.Limit]
		nextCursor = &quizzes[len(quizzes)-1].ID
	}

	pagination := &repository.PaginatedResult{
		HasMore:    hasMore,
		NextCursor: nextCursor,
		Count:      len(quizzes),
	}

	return quizzes, pagination, nil
}

// CreateSubmission creates a quiz submission
func (r *QuizRepository) CreateSubmission(ctx context.Context, submission *domain.QuizSubmission) error {
	answersJSON, err := json.Marshal(submission.Answers)
	if err != nil {
		return fmt.Errorf("failed to marshal answers: %w", err)
	}

	query := `
		INSERT INTO quiz_submissions (
			id, tenant_id, quiz_id, student_id, status, started_at, submitted_at,
			answers, score, percentage, is_auto_graded, feedback, ip_address,
			created_at, updated_at, graded_at, graded_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	_, err = r.GetDB().ExecContext(ctx, query,
		submission.ID,
		submission.TenantID,
		submission.QuizID,
		submission.StudentID,
		submission.Status,
		submission.StartedAt,
		submission.SubmittedAt,
		answersJSON,
		submission.Score,
		submission.Percentage,
		submission.IsAutoGraded,
		repository.ToNullString(submission.Feedback),
		repository.ToNullString(submission.IPAddress),
		submission.CreatedAt,
		submission.UpdatedAt,
		submission.GradedAt,
		repository.ToNullUUID(submission.GradedBy),
	)

	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// GetSubmissionByID retrieves a quiz submission by ID
func (r *QuizRepository) GetSubmissionByID(ctx context.Context, id uuid.UUID) (*domain.QuizSubmission, error) {
	query := `
		SELECT
			id, tenant_id, quiz_id, student_id, status, started_at, submitted_at,
			answers, score, percentage, is_auto_graded, feedback, ip_address,
			created_at, updated_at, graded_at, graded_by
		FROM quiz_submissions
		WHERE id = $1
	`

	return r.scanSubmission(r.GetDB().QueryRowContext(ctx, query, id))
}

// GetSubmissionByQuizAndStudent retrieves a student's quiz submission
func (r *QuizRepository) GetSubmissionByQuizAndStudent(ctx context.Context, quizID, studentID uuid.UUID) (*domain.QuizSubmission, error) {
	query := `
		SELECT
			id, tenant_id, quiz_id, student_id, status, started_at, submitted_at,
			answers, score, percentage, is_auto_graded, feedback, ip_address,
			created_at, updated_at, graded_at, graded_by
		FROM quiz_submissions
		WHERE quiz_id = $1 AND student_id = $2
	`

	return r.scanSubmission(r.GetDB().QueryRowContext(ctx, query, quizID, studentID))
}

// UpdateSubmission updates a quiz submission
func (r *QuizRepository) UpdateSubmission(ctx context.Context, submission *domain.QuizSubmission) error {
	answersJSON, err := json.Marshal(submission.Answers)
	if err != nil {
		return fmt.Errorf("failed to marshal answers: %w", err)
	}

	query := `
		UPDATE quiz_submissions SET
			status = $2, submitted_at = $3, answers = $4, score = $5,
			percentage = $6, is_auto_graded = $7, feedback = $8,
			updated_at = $9, graded_at = $10, graded_by = $11
		WHERE id = $1
	`

	result, err := r.GetDB().ExecContext(ctx, query,
		submission.ID,
		submission.Status,
		submission.SubmittedAt,
		answersJSON,
		submission.Score,
		submission.Percentage,
		submission.IsAutoGraded,
		repository.ToNullString(submission.Feedback),
		submission.UpdatedAt,
		submission.GradedAt,
		repository.ToNullUUID(submission.GradedBy),
	)

	if err != nil {
		return repository.ParseError(err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// ListSubmissionsByQuiz retrieves all submissions for a quiz
func (r *QuizRepository) ListSubmissionsByQuiz(ctx context.Context, quizID uuid.UUID, params repository.PaginationParams) ([]*domain.QuizSubmission, *repository.PaginatedResult, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, nil, err
	}

	query := `
		SELECT
			id, tenant_id, quiz_id, student_id, status, started_at, submitted_at,
			answers, score, percentage, is_auto_graded, feedback, ip_address,
			created_at, updated_at, graded_at, graded_by
		FROM quiz_submissions
		WHERE quiz_id = $1
	`

	args := []interface{}{quizID}
	argIndex := 2

	if params.Cursor != nil {
		query += fmt.Sprintf(" AND id < $%d", argIndex)
		args = append(args, *params.Cursor)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY submitted_at DESC NULLS LAST, id DESC LIMIT $%d", argIndex)
	args = append(args, params.Limit+1)

	rows, err := r.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, repository.ParseError(err)
	}
	defer rows.Close()

	submissions := make([]*domain.QuizSubmission, 0)
	for rows.Next() {
		submission, err := r.scanSubmissionFromRows(rows)
		if err != nil {
			return nil, nil, err
		}
		submissions = append(submissions, submission)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, repository.ParseError(err)
	}

	hasMore := len(submissions) > params.Limit
	var nextCursor *uuid.UUID
	if hasMore {
		submissions = submissions[:params.Limit]
		nextCursor = &submissions[len(submissions)-1].ID
	}

	pagination := &repository.PaginatedResult{
		HasMore:    hasMore,
		NextCursor: nextCursor,
		Count:      len(submissions),
	}

	return submissions, pagination, nil
}

func (r *QuizRepository) scanQuiz(row *sql.Row) (*domain.Quiz, error) {
	var q domain.Quiz
	var questionsJSON []byte
	var passingPercentage sql.NullInt64
	var publishedAt sql.NullTime

	err := row.Scan(
		&q.ID,
		&q.TenantID,
		&q.CourseID,
		&q.CreatedByTutorID,
		&q.Title,
		&q.Instructions,
		&questionsJSON,
		&q.TotalMarks,
		&q.TimeLimit,
		&q.AvailabilityStart,
		&q.AvailabilityEnd,
		&q.Status,
		&q.ShowBeforeWindow,
		&q.AllowRetake,
		&passingPercentage,
		&q.CreatedAt,
		&q.UpdatedAt,
		&publishedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	if err := json.Unmarshal(questionsJSON, &q.Questions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal questions: %w", err)
	}

	if passingPercentage.Valid {
		pp := int(passingPercentage.Int64)
		q.PassingPercentage = &pp
	}
	if publishedAt.Valid {
		q.PublishedAt = &publishedAt.Time
	}

	return &q, nil
}

func (r *QuizRepository) scanQuizFromRows(rows *sql.Rows) (*domain.Quiz, error) {
	var q domain.Quiz
	var questionsJSON []byte
	var passingPercentage sql.NullInt64
	var publishedAt sql.NullTime

	err := rows.Scan(
		&q.ID,
		&q.TenantID,
		&q.CourseID,
		&q.CreatedByTutorID,
		&q.Title,
		&q.Instructions,
		&questionsJSON,
		&q.TotalMarks,
		&q.TimeLimit,
		&q.AvailabilityStart,
		&q.AvailabilityEnd,
		&q.Status,
		&q.ShowBeforeWindow,
		&q.AllowRetake,
		&passingPercentage,
		&q.CreatedAt,
		&q.UpdatedAt,
		&publishedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	if err := json.Unmarshal(questionsJSON, &q.Questions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal questions: %w", err)
	}

	if passingPercentage.Valid {
		pp := int(passingPercentage.Int64)
		q.PassingPercentage = &pp
	}
	if publishedAt.Valid {
		q.PublishedAt = &publishedAt.Time
	}

	return &q, nil
}

func (r *QuizRepository) scanSubmission(row *sql.Row) (*domain.QuizSubmission, error) {
	var s domain.QuizSubmission
	var answersJSON []byte
	var startedAt, submittedAt, gradedAt sql.NullTime
	var score sql.NullInt64
	var percentage sql.NullFloat64
	var feedback, ipAddress, gradedBy sql.NullString

	err := row.Scan(
		&s.ID,
		&s.TenantID,
		&s.QuizID,
		&s.StudentID,
		&s.Status,
		&startedAt,
		&submittedAt,
		&answersJSON,
		&score,
		&percentage,
		&s.IsAutoGraded,
		&feedback,
		&ipAddress,
		&s.CreatedAt,
		&s.UpdatedAt,
		&gradedAt,
		&gradedBy,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	if err := json.Unmarshal(answersJSON, &s.Answers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal answers: %w", err)
	}

	if startedAt.Valid {
		s.StartedAt = &startedAt.Time
	}
	if submittedAt.Valid {
		s.SubmittedAt = &submittedAt.Time
	}
	if gradedAt.Valid {
		s.GradedAt = &gradedAt.Time
	}
	if score.Valid {
		sc := int(score.Int64)
		s.Score = &sc
	}
	if percentage.Valid {
		s.Percentage = &percentage.Float64
	}
	s.Feedback = repository.FromNullString(feedback)
	s.IPAddress = repository.FromNullString(ipAddress)
	s.GradedBy = repository.FromNullUUID(gradedBy)

	return &s, nil
}

func (r *QuizRepository) scanSubmissionFromRows(rows *sql.Rows) (*domain.QuizSubmission, error) {
	var s domain.QuizSubmission
	var answersJSON []byte
	var startedAt, submittedAt, gradedAt sql.NullTime
	var score sql.NullInt64
	var percentage sql.NullFloat64
	var feedback, ipAddress, gradedBy sql.NullString

	err := rows.Scan(
		&s.ID,
		&s.TenantID,
		&s.QuizID,
		&s.StudentID,
		&s.Status,
		&startedAt,
		&submittedAt,
		&answersJSON,
		&score,
		&percentage,
		&s.IsAutoGraded,
		&feedback,
		&ipAddress,
		&s.CreatedAt,
		&s.UpdatedAt,
		&gradedAt,
		&gradedBy,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	if err := json.Unmarshal(answersJSON, &s.Answers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal answers: %w", err)
	}

	if startedAt.Valid {
		s.StartedAt = &startedAt.Time
	}
	if submittedAt.Valid {
		s.SubmittedAt = &submittedAt.Time
	}
	if gradedAt.Valid {
		s.GradedAt = &gradedAt.Time
	}
	if score.Valid {
		sc := int(score.Int64)
		s.Score = &sc
	}
	if percentage.Valid {
		s.Percentage = &percentage.Float64
	}
	s.Feedback = repository.FromNullString(feedback)
	s.IPAddress = repository.FromNullString(ipAddress)
	s.GradedBy = repository.FromNullUUID(gradedBy)

	return &s, nil
}

// ListByAvailabilityEndRange retrieves published quizzes with availability end times in the specified range
func (r *QuizRepository) ListByAvailabilityEndRange(ctx context.Context, start, end time.Time) ([]*domain.Quiz, error) {
	query := `
		SELECT
			id, tenant_id, course_id, created_by_tutor_id, title, instructions,
			questions, total_marks, time_limit, availability_start, availability_end,
			status, show_before_window, allow_retake, passing_percentage,
			created_at, updated_at, published_at
		FROM quizzes
		WHERE status = $1 AND availability_end >= $2 AND availability_end < $3
		ORDER BY availability_end ASC
	`

	rows, err := r.GetDB().QueryContext(ctx, query, domain.AssessmentStatusPublished, start, end)
	if err != nil {
		return nil, repository.ParseError(err)
	}
	defer rows.Close()

	quizzes := make([]*domain.Quiz, 0)
	for rows.Next() {
		quiz, err := r.scanQuizFromRows(rows)
		if err != nil {
			return nil, err
		}
		quizzes = append(quizzes, quiz)
	}

	if err := rows.Err(); err != nil {
		return nil, repository.ParseError(err)
	}

	return quizzes, nil
}
