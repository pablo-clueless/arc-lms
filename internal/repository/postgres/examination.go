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

// ExaminationRepository handles database operations for examinations
type ExaminationRepository struct {
	*repository.BaseRepository
}

// NewExaminationRepository creates a new examination repository
func NewExaminationRepository(db *sql.DB) *ExaminationRepository {
	return &ExaminationRepository{
		BaseRepository: repository.NewBaseRepository(db),
	}
}

// Create creates a new examination
func (r *ExaminationRepository) Create(ctx context.Context, exam *domain.Examination, tx *sql.Tx) error {
	questionsJSON, err := json.Marshal(exam.Questions)
	if err != nil {
		return fmt.Errorf("failed to marshal questions: %w", err)
	}

	query := `
		INSERT INTO examinations (
			id, tenant_id, course_id, term_id, created_by_id,
			title, instructions, questions, total_marks, duration,
			window_start, window_end, status, results_published,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	_, err = execer.ExecContext(ctx, query,
		exam.ID,
		exam.TenantID,
		exam.CourseID,
		exam.TermID,
		exam.CreatedByID,
		exam.Title,
		exam.Instructions,
		questionsJSON,
		exam.TotalMarks,
		exam.Duration,
		exam.WindowStart,
		exam.WindowEnd,
		exam.Status,
		exam.ResultsPublished,
		exam.CreatedAt,
		exam.UpdatedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// Get retrieves an examination by ID
func (r *ExaminationRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Examination, error) {
	query := `
		SELECT
			id, tenant_id, course_id, term_id, created_by_id,
			title, instructions, questions, total_marks, duration,
			window_start, window_end, status, results_published,
			results_published_at, results_published_by,
			created_at, updated_at, scheduled_at
		FROM examinations
		WHERE id = $1
	`

	return r.scanExamination(r.GetDB().QueryRowContext(ctx, query, id))
}

// scanExamination scans an examination from a database row
func (r *ExaminationRepository) scanExamination(row *sql.Row) (*domain.Examination, error) {
	var exam domain.Examination
	var questionsJSON []byte
	var resultsPublishedAt, scheduledAt sql.NullTime
	var resultsPublishedBy sql.NullString

	err := row.Scan(
		&exam.ID,
		&exam.TenantID,
		&exam.CourseID,
		&exam.TermID,
		&exam.CreatedByID,
		&exam.Title,
		&exam.Instructions,
		&questionsJSON,
		&exam.TotalMarks,
		&exam.Duration,
		&exam.WindowStart,
		&exam.WindowEnd,
		&exam.Status,
		&exam.ResultsPublished,
		&resultsPublishedAt,
		&resultsPublishedBy,
		&exam.CreatedAt,
		&exam.UpdatedAt,
		&scheduledAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	if resultsPublishedAt.Valid {
		exam.ResultsPublishedAt = &resultsPublishedAt.Time
	}
	if resultsPublishedBy.Valid {
		pubBy, _ := uuid.Parse(resultsPublishedBy.String)
		exam.ResultsPublishedBy = &pubBy
	}
	if scheduledAt.Valid {
		exam.ScheduledAt = &scheduledAt.Time
	}

	if err := json.Unmarshal(questionsJSON, &exam.Questions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal questions: %w", err)
	}

	return &exam, nil
}

// UpdateWithTx updates an examination with an optional transaction
func (r *ExaminationRepository) UpdateWithTx(ctx context.Context, exam *domain.Examination, tx *sql.Tx) error {
	questionsJSON, err := json.Marshal(exam.Questions)
	if err != nil {
		return fmt.Errorf("failed to marshal questions: %w", err)
	}

	var resultsPublishedAt sql.NullTime
	var resultsPublishedBy sql.NullString
	var scheduledAt sql.NullTime

	if exam.ResultsPublishedAt != nil {
		resultsPublishedAt = sql.NullTime{Time: *exam.ResultsPublishedAt, Valid: true}
	}
	if exam.ResultsPublishedBy != nil {
		resultsPublishedBy = sql.NullString{String: exam.ResultsPublishedBy.String(), Valid: true}
	}
	if exam.ScheduledAt != nil {
		scheduledAt = sql.NullTime{Time: *exam.ScheduledAt, Valid: true}
	}

	query := `
		UPDATE examinations
		SET
			title = $2,
			instructions = $3,
			questions = $4,
			total_marks = $5,
			duration = $6,
			window_start = $7,
			window_end = $8,
			status = $9,
			results_published = $10,
			results_published_at = $11,
			results_published_by = $12,
			scheduled_at = $13,
			updated_at = $14
		WHERE id = $1
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	result, err := execer.ExecContext(ctx, query,
		exam.ID,
		exam.Title,
		exam.Instructions,
		questionsJSON,
		exam.TotalMarks,
		exam.Duration,
		exam.WindowStart,
		exam.WindowEnd,
		exam.Status,
		exam.ResultsPublished,
		resultsPublishedAt,
		resultsPublishedBy,
		scheduledAt,
		exam.UpdatedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// Delete deletes an examination
func (r *ExaminationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM examinations WHERE id = $1`

	result, err := r.GetDB().ExecContext(ctx, query, id)
	if err != nil {
		return repository.ParseError(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// ListByCourse retrieves examinations for a course
func (r *ExaminationRepository) ListByCourse(ctx context.Context, courseID uuid.UUID, params repository.PaginationParams) ([]*domain.Examination, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, err
	}

	query := `
		SELECT
			id, tenant_id, course_id, term_id, created_by_id,
			title, instructions, questions, total_marks, duration,
			window_start, window_end, status, results_published,
			results_published_at, results_published_by,
			created_at, updated_at, scheduled_at
		FROM examinations
		WHERE course_id = $1
	`

	args := []interface{}{courseID}
	argIndex := 2

	if params.Cursor != nil {
		if params.SortOrder == "DESC" {
			query += fmt.Sprintf(" AND id < $%d", argIndex)
		} else {
			query += fmt.Sprintf(" AND id > $%d", argIndex)
		}
		args = append(args, *params.Cursor)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY created_at %s LIMIT $%d", params.SortOrder, argIndex)
	args = append(args, params.Limit+1)

	return r.queryExaminations(ctx, query, args...)
}

// ListByTerm retrieves examinations for a term
func (r *ExaminationRepository) ListByTerm(ctx context.Context, termID uuid.UUID, params repository.PaginationParams) ([]*domain.Examination, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, err
	}

	query := `
		SELECT
			id, tenant_id, course_id, term_id, created_by_id,
			title, instructions, questions, total_marks, duration,
			window_start, window_end, status, results_published,
			results_published_at, results_published_by,
			created_at, updated_at, scheduled_at
		FROM examinations
		WHERE term_id = $1
	`

	args := []interface{}{termID}
	argIndex := 2

	if params.Cursor != nil {
		if params.SortOrder == "DESC" {
			query += fmt.Sprintf(" AND id < $%d", argIndex)
		} else {
			query += fmt.Sprintf(" AND id > $%d", argIndex)
		}
		args = append(args, *params.Cursor)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY window_start %s LIMIT $%d", params.SortOrder, argIndex)
	args = append(args, params.Limit+1)

	return r.queryExaminations(ctx, query, args...)
}

// ListByTenant retrieves examinations for a tenant
func (r *ExaminationRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, status *domain.ExaminationStatus, params repository.PaginationParams) ([]*domain.Examination, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, err
	}

	query := `
		SELECT
			id, tenant_id, course_id, term_id, created_by_id,
			title, instructions, questions, total_marks, duration,
			window_start, window_end, status, results_published,
			results_published_at, results_published_by,
			created_at, updated_at, scheduled_at
		FROM examinations
		WHERE tenant_id = $1
	`

	args := []interface{}{tenantID}
	argIndex := 2

	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *status)
		argIndex++
	}

	if params.Cursor != nil {
		if params.SortOrder == "DESC" {
			query += fmt.Sprintf(" AND id < $%d", argIndex)
		} else {
			query += fmt.Sprintf(" AND id > $%d", argIndex)
		}
		args = append(args, *params.Cursor)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY created_at %s LIMIT $%d", params.SortOrder, argIndex)
	args = append(args, params.Limit+1)

	return r.queryExaminations(ctx, query, args...)
}

// GetActiveExaminations retrieves examinations that are currently in their window
func (r *ExaminationRepository) GetActiveExaminations(ctx context.Context, tenantID uuid.UUID) ([]*domain.Examination, error) {
	now := time.Now()
	query := `
		SELECT
			id, tenant_id, course_id, term_id, created_by_id,
			title, instructions, questions, total_marks, duration,
			window_start, window_end, status, results_published,
			results_published_at, results_published_by,
			created_at, updated_at, scheduled_at
		FROM examinations
		WHERE tenant_id = $1
			AND status IN ('SCHEDULED', 'IN_PROGRESS')
			AND window_start <= $2
			AND window_end >= $2
		ORDER BY window_start ASC
	`

	return r.queryExaminations(ctx, query, tenantID, now)
}

// queryExaminations executes a query and returns a list of examinations
func (r *ExaminationRepository) queryExaminations(ctx context.Context, query string, args ...interface{}) ([]*domain.Examination, error) {
	rows, err := r.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, repository.ParseError(err)
	}
	defer rows.Close()

	exams := make([]*domain.Examination, 0)
	for rows.Next() {
		var exam domain.Examination
		var questionsJSON []byte
		var resultsPublishedAt, scheduledAt sql.NullTime
		var resultsPublishedBy sql.NullString

		err := rows.Scan(
			&exam.ID,
			&exam.TenantID,
			&exam.CourseID,
			&exam.TermID,
			&exam.CreatedByID,
			&exam.Title,
			&exam.Instructions,
			&questionsJSON,
			&exam.TotalMarks,
			&exam.Duration,
			&exam.WindowStart,
			&exam.WindowEnd,
			&exam.Status,
			&exam.ResultsPublished,
			&resultsPublishedAt,
			&resultsPublishedBy,
			&exam.CreatedAt,
			&exam.UpdatedAt,
			&scheduledAt,
		)

		if err != nil {
			return nil, repository.ParseError(err)
		}

		if resultsPublishedAt.Valid {
			exam.ResultsPublishedAt = &resultsPublishedAt.Time
		}
		if resultsPublishedBy.Valid {
			pubBy, _ := uuid.Parse(resultsPublishedBy.String)
			exam.ResultsPublishedBy = &pubBy
		}
		if scheduledAt.Valid {
			exam.ScheduledAt = &scheduledAt.Time
		}

		if err := json.Unmarshal(questionsJSON, &exam.Questions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal questions: %w", err)
		}

		exams = append(exams, &exam)
	}

	if err := rows.Err(); err != nil {
		return nil, repository.ParseError(err)
	}

	return exams, nil
}

// ValidateTenantAccess validates that an examination belongs to a tenant
func (r *ExaminationRepository) ValidateTenantAccess(ctx context.Context, tenantID uuid.UUID, examID uuid.UUID) error {
	query := `SELECT 1 FROM examinations WHERE id = $1 AND tenant_id = $2`

	var exists int
	err := r.GetDB().QueryRowContext(ctx, query, examID, tenantID).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return repository.ErrNotFound
		}
		return repository.ParseError(err)
	}

	return nil
}

// ===================== Examination Submissions =====================

// CreateSubmission creates a new examination submission
func (r *ExaminationRepository) CreateSubmission(ctx context.Context, sub *domain.ExaminationSubmission, tx *sql.Tx) error {
	answersJSON, err := json.Marshal(sub.Answers)
	if err != nil {
		return fmt.Errorf("failed to marshal answers: %w", err)
	}

	integrityEventsJSON, err := json.Marshal(sub.IntegrityEvents)
	if err != nil {
		return fmt.Errorf("failed to marshal integrity events: %w", err)
	}

	query := `
		INSERT INTO examination_submissions (
			id, tenant_id, examination_id, student_id, status,
			started_at, answers, integrity_events, ip_address,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	var startedAt sql.NullTime
	if sub.StartedAt != nil {
		startedAt = sql.NullTime{Time: *sub.StartedAt, Valid: true}
	}

	execer := repository.GetExecer(r.GetDB(), tx)
	_, err = execer.ExecContext(ctx, query,
		sub.ID,
		sub.TenantID,
		sub.ExaminationID,
		sub.StudentID,
		sub.Status,
		startedAt,
		answersJSON,
		integrityEventsJSON,
		sub.IPAddress,
		sub.CreatedAt,
		sub.UpdatedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// GetSubmission retrieves an examination submission by ID
func (r *ExaminationRepository) GetSubmission(ctx context.Context, id uuid.UUID) (*domain.ExaminationSubmission, error) {
	query := `
		SELECT
			id, tenant_id, examination_id, student_id, status,
			started_at, submitted_at, auto_submitted, answers,
			score, percentage, is_auto_graded, feedback,
			integrity_events, ip_address, created_at, updated_at,
			graded_at, graded_by, results_published_to_student
		FROM examination_submissions
		WHERE id = $1
	`

	return r.scanSubmission(r.GetDB().QueryRowContext(ctx, query, id))
}

// GetSubmissionByStudentAndExam retrieves a student's submission for an examination
func (r *ExaminationRepository) GetSubmissionByStudentAndExam(ctx context.Context, studentID, examID uuid.UUID) (*domain.ExaminationSubmission, error) {
	query := `
		SELECT
			id, tenant_id, examination_id, student_id, status,
			started_at, submitted_at, auto_submitted, answers,
			score, percentage, is_auto_graded, feedback,
			integrity_events, ip_address, created_at, updated_at,
			graded_at, graded_by, results_published_to_student
		FROM examination_submissions
		WHERE student_id = $1 AND examination_id = $2
	`

	return r.scanSubmission(r.GetDB().QueryRowContext(ctx, query, studentID, examID))
}

// scanSubmission scans a submission from a database row
func (r *ExaminationRepository) scanSubmission(row *sql.Row) (*domain.ExaminationSubmission, error) {
	var sub domain.ExaminationSubmission
	var startedAt, submittedAt, gradedAt sql.NullTime
	var gradedBy sql.NullString
	var answersJSON, integrityEventsJSON []byte
	var score sql.NullInt64
	var percentage sql.NullFloat64
	var feedback sql.NullString

	err := row.Scan(
		&sub.ID,
		&sub.TenantID,
		&sub.ExaminationID,
		&sub.StudentID,
		&sub.Status,
		&startedAt,
		&submittedAt,
		&sub.AutoSubmitted,
		&answersJSON,
		&score,
		&percentage,
		&sub.IsAutoGraded,
		&feedback,
		&integrityEventsJSON,
		&sub.IPAddress,
		&sub.CreatedAt,
		&sub.UpdatedAt,
		&gradedAt,
		&gradedBy,
		&sub.ResultsPublishedToStudent,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	if startedAt.Valid {
		sub.StartedAt = &startedAt.Time
	}
	if submittedAt.Valid {
		sub.SubmittedAt = &submittedAt.Time
	}
	if gradedAt.Valid {
		sub.GradedAt = &gradedAt.Time
	}
	if gradedBy.Valid {
		gBy, _ := uuid.Parse(gradedBy.String)
		sub.GradedBy = &gBy
	}
	if score.Valid {
		s := int(score.Int64)
		sub.Score = &s
	}
	if percentage.Valid {
		sub.Percentage = &percentage.Float64
	}
	if feedback.Valid {
		sub.Feedback = &feedback.String
	}

	if err := json.Unmarshal(answersJSON, &sub.Answers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal answers: %w", err)
	}

	if err := json.Unmarshal(integrityEventsJSON, &sub.IntegrityEvents); err != nil {
		return nil, fmt.Errorf("failed to unmarshal integrity events: %w", err)
	}

	return &sub, nil
}

// UpdateSubmission updates an examination submission
func (r *ExaminationRepository) UpdateSubmission(ctx context.Context, sub *domain.ExaminationSubmission, tx *sql.Tx) error {
	answersJSON, err := json.Marshal(sub.Answers)
	if err != nil {
		return fmt.Errorf("failed to marshal answers: %w", err)
	}

	integrityEventsJSON, err := json.Marshal(sub.IntegrityEvents)
	if err != nil {
		return fmt.Errorf("failed to marshal integrity events: %w", err)
	}

	var submittedAt, gradedAt sql.NullTime
	var gradedBy, feedback sql.NullString
	var score sql.NullInt64
	var percentage sql.NullFloat64

	if sub.SubmittedAt != nil {
		submittedAt = sql.NullTime{Time: *sub.SubmittedAt, Valid: true}
	}
	if sub.GradedAt != nil {
		gradedAt = sql.NullTime{Time: *sub.GradedAt, Valid: true}
	}
	if sub.GradedBy != nil {
		gradedBy = sql.NullString{String: sub.GradedBy.String(), Valid: true}
	}
	if sub.Score != nil {
		score = sql.NullInt64{Int64: int64(*sub.Score), Valid: true}
	}
	if sub.Percentage != nil {
		percentage = sql.NullFloat64{Float64: *sub.Percentage, Valid: true}
	}
	if sub.Feedback != nil {
		feedback = sql.NullString{String: *sub.Feedback, Valid: true}
	}

	query := `
		UPDATE examination_submissions
		SET
			status = $2,
			submitted_at = $3,
			auto_submitted = $4,
			answers = $5,
			score = $6,
			percentage = $7,
			is_auto_graded = $8,
			feedback = $9,
			integrity_events = $10,
			graded_at = $11,
			graded_by = $12,
			results_published_to_student = $13,
			updated_at = $14
		WHERE id = $1
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	result, err := execer.ExecContext(ctx, query,
		sub.ID,
		sub.Status,
		submittedAt,
		sub.AutoSubmitted,
		answersJSON,
		score,
		percentage,
		sub.IsAutoGraded,
		feedback,
		integrityEventsJSON,
		gradedAt,
		gradedBy,
		sub.ResultsPublishedToStudent,
		sub.UpdatedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// ListSubmissionsByExam retrieves all submissions for an examination
func (r *ExaminationRepository) ListSubmissionsByExam(ctx context.Context, examID uuid.UUID, params repository.PaginationParams) ([]*domain.ExaminationSubmission, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, err
	}

	query := `
		SELECT
			id, tenant_id, examination_id, student_id, status,
			started_at, submitted_at, auto_submitted, answers,
			score, percentage, is_auto_graded, feedback,
			integrity_events, ip_address, created_at, updated_at,
			graded_at, graded_by, results_published_to_student
		FROM examination_submissions
		WHERE examination_id = $1
	`

	args := []interface{}{examID}
	argIndex := 2

	if params.Cursor != nil {
		if params.SortOrder == "DESC" {
			query += fmt.Sprintf(" AND id < $%d", argIndex)
		} else {
			query += fmt.Sprintf(" AND id > $%d", argIndex)
		}
		args = append(args, *params.Cursor)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY submitted_at %s NULLS LAST LIMIT $%d", params.SortOrder, argIndex)
	args = append(args, params.Limit+1)

	return r.querySubmissions(ctx, query, args...)
}

// ListSubmissionsByStudent retrieves all submissions for a student
func (r *ExaminationRepository) ListSubmissionsByStudent(ctx context.Context, studentID uuid.UUID, params repository.PaginationParams) ([]*domain.ExaminationSubmission, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, err
	}

	query := `
		SELECT
			id, tenant_id, examination_id, student_id, status,
			started_at, submitted_at, auto_submitted, answers,
			score, percentage, is_auto_graded, feedback,
			integrity_events, ip_address, created_at, updated_at,
			graded_at, graded_by, results_published_to_student
		FROM examination_submissions
		WHERE student_id = $1
	`

	args := []interface{}{studentID}
	argIndex := 2

	if params.Cursor != nil {
		if params.SortOrder == "DESC" {
			query += fmt.Sprintf(" AND id < $%d", argIndex)
		} else {
			query += fmt.Sprintf(" AND id > $%d", argIndex)
		}
		args = append(args, *params.Cursor)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY created_at %s LIMIT $%d", params.SortOrder, argIndex)
	args = append(args, params.Limit+1)

	return r.querySubmissions(ctx, query, args...)
}

// GetPendingGradingSubmissions retrieves submissions that need manual grading
func (r *ExaminationRepository) GetPendingGradingSubmissions(ctx context.Context, examID uuid.UUID) ([]*domain.ExaminationSubmission, error) {
	query := `
		SELECT
			id, tenant_id, examination_id, student_id, status,
			started_at, submitted_at, auto_submitted, answers,
			score, percentage, is_auto_graded, feedback,
			integrity_events, ip_address, created_at, updated_at,
			graded_at, graded_by, results_published_to_student
		FROM examination_submissions
		WHERE examination_id = $1 AND status = $2
		ORDER BY submitted_at ASC
	`

	return r.querySubmissions(ctx, query, examID, domain.ExamSubmissionStatusSubmitted)
}

// querySubmissions executes a query and returns a list of submissions
func (r *ExaminationRepository) querySubmissions(ctx context.Context, query string, args ...interface{}) ([]*domain.ExaminationSubmission, error) {
	rows, err := r.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, repository.ParseError(err)
	}
	defer rows.Close()

	submissions := make([]*domain.ExaminationSubmission, 0)
	for rows.Next() {
		var sub domain.ExaminationSubmission
		var startedAt, submittedAt, gradedAt sql.NullTime
		var gradedBy sql.NullString
		var answersJSON, integrityEventsJSON []byte
		var score sql.NullInt64
		var percentage sql.NullFloat64
		var feedback sql.NullString

		err := rows.Scan(
			&sub.ID,
			&sub.TenantID,
			&sub.ExaminationID,
			&sub.StudentID,
			&sub.Status,
			&startedAt,
			&submittedAt,
			&sub.AutoSubmitted,
			&answersJSON,
			&score,
			&percentage,
			&sub.IsAutoGraded,
			&feedback,
			&integrityEventsJSON,
			&sub.IPAddress,
			&sub.CreatedAt,
			&sub.UpdatedAt,
			&gradedAt,
			&gradedBy,
			&sub.ResultsPublishedToStudent,
		)

		if err != nil {
			return nil, repository.ParseError(err)
		}

		if startedAt.Valid {
			sub.StartedAt = &startedAt.Time
		}
		if submittedAt.Valid {
			sub.SubmittedAt = &submittedAt.Time
		}
		if gradedAt.Valid {
			sub.GradedAt = &gradedAt.Time
		}
		if gradedBy.Valid {
			gBy, _ := uuid.Parse(gradedBy.String)
			sub.GradedBy = &gBy
		}
		if score.Valid {
			s := int(score.Int64)
			sub.Score = &s
		}
		if percentage.Valid {
			sub.Percentage = &percentage.Float64
		}
		if feedback.Valid {
			sub.Feedback = &feedback.String
		}

		if err := json.Unmarshal(answersJSON, &sub.Answers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal answers: %w", err)
		}

		if err := json.Unmarshal(integrityEventsJSON, &sub.IntegrityEvents); err != nil {
			return nil, fmt.Errorf("failed to unmarshal integrity events: %w", err)
		}

		submissions = append(submissions, &sub)
	}

	if err := rows.Err(); err != nil {
		return nil, repository.ParseError(err)
	}

	return submissions, nil
}

// CountSubmissionsByStatus counts submissions by status for an examination
func (r *ExaminationRepository) CountSubmissionsByStatus(ctx context.Context, examID uuid.UUID) (map[domain.ExaminationSubmissionStatus]int, error) {
	query := `
		SELECT status, COUNT(*)
		FROM examination_submissions
		WHERE examination_id = $1
		GROUP BY status
	`

	rows, err := r.GetDB().QueryContext(ctx, query, examID)
	if err != nil {
		return nil, repository.ParseError(err)
	}
	defer rows.Close()

	counts := make(map[domain.ExaminationSubmissionStatus]int)
	for rows.Next() {
		var status domain.ExaminationSubmissionStatus
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, repository.ParseError(err)
		}
		counts[status] = count
	}

	return counts, nil
}

// PublishResultsToStudents marks all graded submissions as published to students
func (r *ExaminationRepository) PublishResultsToStudents(ctx context.Context, examID uuid.UUID, tx *sql.Tx) error {
	query := `
		UPDATE examination_submissions
		SET
			status = $2,
			results_published_to_student = true,
			updated_at = $3
		WHERE examination_id = $1 AND status = $4
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	_, err := execer.ExecContext(ctx, query, examID, domain.ExamSubmissionStatusPublished, time.Now(), domain.ExamSubmissionStatusGraded)
	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// HasSubmissionIntegrityIssues checks if any submission has integrity issues
func (r *ExaminationRepository) HasSubmissionIntegrityIssues(ctx context.Context, submissionID uuid.UUID) (bool, int, error) {
	query := `
		SELECT jsonb_array_length(integrity_events)
		FROM examination_submissions
		WHERE id = $1
	`

	var count int
	err := r.GetDB().QueryRowContext(ctx, query, submissionID).Scan(&count)
	if err != nil {
		return false, 0, repository.ParseError(err)
	}

	return count > 0, count, nil
}

// ListByWindowStart retrieves scheduled examinations whose window started within the given time range
func (r *ExaminationRepository) ListByWindowStart(ctx context.Context, start, end time.Time) ([]*domain.Examination, error) {
	query := `
		SELECT
			id, tenant_id, course_id, term_id, created_by_id,
			title, instructions, questions, total_marks, duration,
			window_start, window_end, status, results_published,
			results_published_at, results_published_by,
			created_at, updated_at, scheduled_at
		FROM examinations
		WHERE status = $1 AND window_start >= $2 AND window_start < $3
		ORDER BY window_start ASC
	`

	return r.queryExaminations(ctx, query, domain.ExaminationStatusScheduled, start, end)
}

// ListByWindowEnd retrieves in-progress examinations whose window ended within the given time range
func (r *ExaminationRepository) ListByWindowEnd(ctx context.Context, start, end time.Time) ([]*domain.Examination, error) {
	query := `
		SELECT
			id, tenant_id, course_id, term_id, created_by_id,
			title, instructions, questions, total_marks, duration,
			window_start, window_end, status, results_published,
			results_published_at, results_published_by,
			created_at, updated_at, scheduled_at
		FROM examinations
		WHERE status = $1 AND window_end >= $2 AND window_end < $3
		ORDER BY window_end ASC
	`

	return r.queryExaminations(ctx, query, domain.ExaminationStatusInProgress, start, end)
}

// AutoSubmitInProgress auto-submits all in-progress submissions for an examination
func (r *ExaminationRepository) AutoSubmitInProgress(ctx context.Context, examID uuid.UUID) (int, error) {
	now := time.Now()
	query := `
		UPDATE examination_submissions
		SET
			status = $2,
			submitted_at = $3,
			auto_submitted = true,
			updated_at = $3
		WHERE examination_id = $1 AND status = $4
	`

	result, err := r.GetDB().ExecContext(ctx, query, examID, domain.ExamSubmissionStatusSubmitted, now, domain.ExamSubmissionStatusInProgress)
	if err != nil {
		return 0, repository.ParseError(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rowsAffected), nil
}

// Update updates an examination (convenience method without transaction)
func (r *ExaminationRepository) Update(ctx context.Context, exam *domain.Examination) error {
	return r.UpdateWithTx(ctx, exam, nil)
}
