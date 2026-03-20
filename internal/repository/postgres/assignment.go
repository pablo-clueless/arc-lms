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

// AssignmentRepository handles database operations for assignments
type AssignmentRepository struct {
	*repository.BaseRepository
}

// NewAssignmentRepository creates a new assignment repository
func NewAssignmentRepository(db *sql.DB) *AssignmentRepository {
	return &AssignmentRepository{
		BaseRepository: repository.NewBaseRepository(db),
	}
}

// Create creates a new assignment
func (r *AssignmentRepository) Create(ctx context.Context, assignment *domain.Assignment) error {
	attachmentsJSON, _ := json.Marshal(assignment.AttachmentURLs)
	formatsJSON, _ := json.Marshal(assignment.AllowedFileFormats)
	questionsJSON, _ := json.Marshal(assignment.Questions)

	query := `
		INSERT INTO assignments (
			id, tenant_id, course_id, created_by_tutor_id, title, description,
			attachment_urls, max_marks, submission_deadline, allow_late_submission,
			hard_cutoff_date, allowed_file_formats, max_file_size, status,
			questions, created_at, updated_at, published_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`

	_, err := r.GetDB().ExecContext(ctx, query,
		assignment.ID,
		assignment.TenantID,
		assignment.CourseID,
		assignment.CreatedByTutorID,
		assignment.Title,
		assignment.Description,
		attachmentsJSON,
		assignment.MaxMarks,
		assignment.SubmissionDeadline,
		assignment.AllowLateSubmission,
		assignment.HardCutoffDate,
		formatsJSON,
		assignment.MaxFileSize,
		assignment.Status,
		questionsJSON,
		assignment.CreatedAt,
		assignment.UpdatedAt,
		assignment.PublishedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// GetByID retrieves an assignment by ID
func (r *AssignmentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Assignment, error) {
	query := `
		SELECT
			id, tenant_id, course_id, created_by_tutor_id, title, description,
			attachment_urls, max_marks, submission_deadline, allow_late_submission,
			hard_cutoff_date, allowed_file_formats, max_file_size, status,
			questions, created_at, updated_at, published_at
		FROM assignments
		WHERE id = $1
	`

	return r.scanAssignment(r.GetDB().QueryRowContext(ctx, query, id))
}

// Update updates an assignment
func (r *AssignmentRepository) Update(ctx context.Context, assignment *domain.Assignment) error {
	attachmentsJSON, _ := json.Marshal(assignment.AttachmentURLs)
	formatsJSON, _ := json.Marshal(assignment.AllowedFileFormats)
	questionsJSON, _ := json.Marshal(assignment.Questions)

	query := `
		UPDATE assignments SET
			title = $2, description = $3, attachment_urls = $4, max_marks = $5,
			submission_deadline = $6, allow_late_submission = $7, hard_cutoff_date = $8,
			allowed_file_formats = $9, max_file_size = $10, status = $11,
			questions = $12, updated_at = $13, published_at = $14
		WHERE id = $1
	`

	result, err := r.GetDB().ExecContext(ctx, query,
		assignment.ID,
		assignment.Title,
		assignment.Description,
		attachmentsJSON,
		assignment.MaxMarks,
		assignment.SubmissionDeadline,
		assignment.AllowLateSubmission,
		assignment.HardCutoffDate,
		formatsJSON,
		assignment.MaxFileSize,
		assignment.Status,
		questionsJSON,
		assignment.UpdatedAt,
		assignment.PublishedAt,
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

// Delete deletes an assignment
func (r *AssignmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM assignments WHERE id = $1`

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

// ListByCourse retrieves assignments for a course
func (r *AssignmentRepository) ListByCourse(ctx context.Context, courseID uuid.UUID, status *domain.AssessmentStatus, params repository.PaginationParams) ([]*domain.Assignment, *repository.PaginatedResult, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, nil, err
	}

	query := `
		SELECT
			id, tenant_id, course_id, created_by_tutor_id, title, description,
			attachment_urls, max_marks, submission_deadline, allow_late_submission,
			hard_cutoff_date, allowed_file_formats, max_file_size, status,
			questions, created_at, updated_at, published_at
		FROM assignments
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

	query += fmt.Sprintf(" ORDER BY submission_deadline DESC, id DESC LIMIT $%d", argIndex)
	args = append(args, params.Limit+1)

	rows, err := r.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, repository.ParseError(err)
	}
	defer rows.Close()

	assignments := make([]*domain.Assignment, 0)
	for rows.Next() {
		assignment, err := r.scanAssignmentFromRows(rows)
		if err != nil {
			return nil, nil, err
		}
		assignments = append(assignments, assignment)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, repository.ParseError(err)
	}

	hasMore := len(assignments) > params.Limit
	var nextCursor *uuid.UUID
	if hasMore {
		assignments = assignments[:params.Limit]
		nextCursor = &assignments[len(assignments)-1].ID
	}

	pagination := &repository.PaginatedResult{
		HasMore:    hasMore,
		NextCursor: nextCursor,
		Count:      len(assignments),
	}

	return assignments, pagination, nil
}

// CreateSubmission creates an assignment submission
func (r *AssignmentRepository) CreateSubmission(ctx context.Context, submission *domain.AssignmentSubmission) error {
	fileURLsJSON, _ := json.Marshal(submission.FileURLs)

	query := `
		INSERT INTO assignment_submissions (
			id, tenant_id, assignment_id, student_id, status, submitted_at,
			is_late, file_urls, answer_text, score, feedback, ip_address,
			created_at, updated_at, graded_at, graded_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	_, err := r.GetDB().ExecContext(ctx, query,
		submission.ID,
		submission.TenantID,
		submission.AssignmentID,
		submission.StudentID,
		submission.Status,
		submission.SubmittedAt,
		submission.IsLate,
		fileURLsJSON,
		repository.ToNullString(submission.AnswerText),
		submission.Score,
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

// GetSubmissionByID retrieves an assignment submission by ID
func (r *AssignmentRepository) GetSubmissionByID(ctx context.Context, id uuid.UUID) (*domain.AssignmentSubmission, error) {
	query := `
		SELECT
			id, tenant_id, assignment_id, student_id, status, submitted_at,
			is_late, file_urls, answer_text, score, feedback, ip_address,
			created_at, updated_at, graded_at, graded_by
		FROM assignment_submissions
		WHERE id = $1
	`

	return r.scanSubmission(r.GetDB().QueryRowContext(ctx, query, id))
}

// GetSubmissionByAssignmentAndStudent retrieves a student's assignment submission
func (r *AssignmentRepository) GetSubmissionByAssignmentAndStudent(ctx context.Context, assignmentID, studentID uuid.UUID) (*domain.AssignmentSubmission, error) {
	query := `
		SELECT
			id, tenant_id, assignment_id, student_id, status, submitted_at,
			is_late, file_urls, answer_text, score, feedback, ip_address,
			created_at, updated_at, graded_at, graded_by
		FROM assignment_submissions
		WHERE assignment_id = $1 AND student_id = $2
	`

	return r.scanSubmission(r.GetDB().QueryRowContext(ctx, query, assignmentID, studentID))
}

// UpdateSubmission updates an assignment submission
func (r *AssignmentRepository) UpdateSubmission(ctx context.Context, submission *domain.AssignmentSubmission) error {
	fileURLsJSON, _ := json.Marshal(submission.FileURLs)

	query := `
		UPDATE assignment_submissions SET
			status = $2, submitted_at = $3, is_late = $4, file_urls = $5,
			answer_text = $6, score = $7, feedback = $8,
			updated_at = $9, graded_at = $10, graded_by = $11
		WHERE id = $1
	`

	result, err := r.GetDB().ExecContext(ctx, query,
		submission.ID,
		submission.Status,
		submission.SubmittedAt,
		submission.IsLate,
		fileURLsJSON,
		repository.ToNullString(submission.AnswerText),
		submission.Score,
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

// ListSubmissionsByAssignment retrieves all submissions for an assignment
func (r *AssignmentRepository) ListSubmissionsByAssignment(ctx context.Context, assignmentID uuid.UUID, params repository.PaginationParams) ([]*domain.AssignmentSubmission, *repository.PaginatedResult, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, nil, err
	}

	query := `
		SELECT
			id, tenant_id, assignment_id, student_id, status, submitted_at,
			is_late, file_urls, answer_text, score, feedback, ip_address,
			created_at, updated_at, graded_at, graded_by
		FROM assignment_submissions
		WHERE assignment_id = $1
	`

	args := []interface{}{assignmentID}
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

	submissions := make([]*domain.AssignmentSubmission, 0)
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

func (r *AssignmentRepository) scanAssignment(row *sql.Row) (*domain.Assignment, error) {
	var a domain.Assignment
	var attachmentsJSON, formatsJSON, questionsJSON []byte
	var hardCutoffDate, publishedAt sql.NullTime

	err := row.Scan(
		&a.ID,
		&a.TenantID,
		&a.CourseID,
		&a.CreatedByTutorID,
		&a.Title,
		&a.Description,
		&attachmentsJSON,
		&a.MaxMarks,
		&a.SubmissionDeadline,
		&a.AllowLateSubmission,
		&hardCutoffDate,
		&formatsJSON,
		&a.MaxFileSize,
		&a.Status,
		&questionsJSON,
		&a.CreatedAt,
		&a.UpdatedAt,
		&publishedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	json.Unmarshal(attachmentsJSON, &a.AttachmentURLs)
	json.Unmarshal(formatsJSON, &a.AllowedFileFormats)
	json.Unmarshal(questionsJSON, &a.Questions)

	if hardCutoffDate.Valid {
		a.HardCutoffDate = &hardCutoffDate.Time
	}
	if publishedAt.Valid {
		a.PublishedAt = &publishedAt.Time
	}

	return &a, nil
}

func (r *AssignmentRepository) scanAssignmentFromRows(rows *sql.Rows) (*domain.Assignment, error) {
	var a domain.Assignment
	var attachmentsJSON, formatsJSON, questionsJSON []byte
	var hardCutoffDate, publishedAt sql.NullTime

	err := rows.Scan(
		&a.ID,
		&a.TenantID,
		&a.CourseID,
		&a.CreatedByTutorID,
		&a.Title,
		&a.Description,
		&attachmentsJSON,
		&a.MaxMarks,
		&a.SubmissionDeadline,
		&a.AllowLateSubmission,
		&hardCutoffDate,
		&formatsJSON,
		&a.MaxFileSize,
		&a.Status,
		&questionsJSON,
		&a.CreatedAt,
		&a.UpdatedAt,
		&publishedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	json.Unmarshal(attachmentsJSON, &a.AttachmentURLs)
	json.Unmarshal(formatsJSON, &a.AllowedFileFormats)
	json.Unmarshal(questionsJSON, &a.Questions)

	if hardCutoffDate.Valid {
		a.HardCutoffDate = &hardCutoffDate.Time
	}
	if publishedAt.Valid {
		a.PublishedAt = &publishedAt.Time
	}

	return &a, nil
}

func (r *AssignmentRepository) scanSubmission(row *sql.Row) (*domain.AssignmentSubmission, error) {
	var s domain.AssignmentSubmission
	var fileURLsJSON []byte
	var submittedAt, gradedAt sql.NullTime
	var score sql.NullInt64
	var answerText, feedback, ipAddress, gradedBy sql.NullString

	err := row.Scan(
		&s.ID,
		&s.TenantID,
		&s.AssignmentID,
		&s.StudentID,
		&s.Status,
		&submittedAt,
		&s.IsLate,
		&fileURLsJSON,
		&answerText,
		&score,
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

	json.Unmarshal(fileURLsJSON, &s.FileURLs)

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
	s.AnswerText = repository.FromNullString(answerText)
	s.Feedback = repository.FromNullString(feedback)
	s.IPAddress = repository.FromNullString(ipAddress)
	s.GradedBy = repository.FromNullUUID(gradedBy)

	return &s, nil
}

func (r *AssignmentRepository) scanSubmissionFromRows(rows *sql.Rows) (*domain.AssignmentSubmission, error) {
	var s domain.AssignmentSubmission
	var fileURLsJSON []byte
	var submittedAt, gradedAt sql.NullTime
	var score sql.NullInt64
	var answerText, feedback, ipAddress, gradedBy sql.NullString

	err := rows.Scan(
		&s.ID,
		&s.TenantID,
		&s.AssignmentID,
		&s.StudentID,
		&s.Status,
		&submittedAt,
		&s.IsLate,
		&fileURLsJSON,
		&answerText,
		&score,
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

	json.Unmarshal(fileURLsJSON, &s.FileURLs)

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
	s.AnswerText = repository.FromNullString(answerText)
	s.Feedback = repository.FromNullString(feedback)
	s.IPAddress = repository.FromNullString(ipAddress)
	s.GradedBy = repository.FromNullUUID(gradedBy)

	return &s, nil
}

// ListByDeadlineRange retrieves published assignments with deadlines in the specified time range
func (r *AssignmentRepository) ListByDeadlineRange(ctx context.Context, start, end time.Time) ([]*domain.Assignment, error) {
	query := `
		SELECT
			id, tenant_id, course_id, created_by_tutor_id, title, description,
			attachment_urls, max_marks, submission_deadline, allow_late_submission,
			hard_cutoff_date, allowed_file_formats, max_file_size, status,
			questions, created_at, updated_at, published_at
		FROM assignments
		WHERE status = $1 AND submission_deadline >= $2 AND submission_deadline < $3
		ORDER BY submission_deadline ASC
	`

	rows, err := r.GetDB().QueryContext(ctx, query, domain.AssessmentStatusPublished, start, end)
	if err != nil {
		return nil, repository.ParseError(err)
	}
	defer rows.Close()

	assignments := make([]*domain.Assignment, 0)
	for rows.Next() {
		assignment, err := r.scanAssignmentFromRows(rows)
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, assignment)
	}

	if err := rows.Err(); err != nil {
		return nil, repository.ParseError(err)
	}

	return assignments, nil
}
