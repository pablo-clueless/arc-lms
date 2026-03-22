package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"arc-lms/internal/domain"
	"arc-lms/internal/repository"
)

// ProgressRepository handles database operations for progress records
type ProgressRepository struct {
	*repository.BaseRepository
}

// NewProgressRepository creates a new progress repository
func NewProgressRepository(db *sql.DB) *ProgressRepository {
	return &ProgressRepository{
		BaseRepository: repository.NewBaseRepository(db),
	}
}

// Create creates a new progress record
func (r *ProgressRepository) Create(ctx context.Context, progress *domain.Progress, tx *sql.Tx) error {
	gradeJSON, err := json.Marshal(progress.Grade)
	if err != nil {
		return fmt.Errorf("failed to marshal grade: %w", err)
	}

	attendanceJSON, err := json.Marshal(progress.Attendance)
	if err != nil {
		return fmt.Errorf("failed to marshal attendance: %w", err)
	}

	query := `
		INSERT INTO progress (
			id, tenant_id, student_id, course_id, term_id, class_id,
			status, quiz_scores, assignment_scores, examination_score,
			grade, attendance, tutor_remarks, principal_remarks,
			class_position, is_flagged, flag_reason,
			created_at, updated_at, completed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	_, err = execer.ExecContext(ctx, query,
		progress.ID,
		progress.TenantID,
		progress.StudentID,
		progress.CourseID,
		progress.TermID,
		progress.ClassID,
		progress.Status,
		pq.Array(progress.QuizScores),
		pq.Array(progress.AssignmentScores),
		progress.ExaminationScore,
		gradeJSON,
		attendanceJSON,
		progress.TutorRemarks,
		progress.PrincipalRemarks,
		progress.ClassPosition,
		progress.IsFlagged,
		progress.FlagReason,
		progress.CreatedAt,
		progress.UpdatedAt,
		progress.CompletedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// Get retrieves a progress record by ID
func (r *ProgressRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Progress, error) {
	query := `
		SELECT
			id, tenant_id, student_id, course_id, term_id, class_id,
			status, quiz_scores, assignment_scores, examination_score,
			grade, attendance, tutor_remarks, principal_remarks,
			class_position, is_flagged, flag_reason,
			created_at, updated_at, completed_at
		FROM progress
		WHERE id = $1
	`

	return r.scanProgress(r.GetDB().QueryRowContext(ctx, query, id))
}

// GetByStudentCourseAndTerm retrieves progress for a student in a course for a term
func (r *ProgressRepository) GetByStudentCourseAndTerm(ctx context.Context, studentID, courseID, termID uuid.UUID) (*domain.Progress, error) {
	query := `
		SELECT
			id, tenant_id, student_id, course_id, term_id, class_id,
			status, quiz_scores, assignment_scores, examination_score,
			grade, attendance, tutor_remarks, principal_remarks,
			class_position, is_flagged, flag_reason,
			created_at, updated_at, completed_at
		FROM progress
		WHERE student_id = $1 AND course_id = $2 AND term_id = $3
	`

	return r.scanProgress(r.GetDB().QueryRowContext(ctx, query, studentID, courseID, termID))
}

// scanProgress scans a progress record from a database row
func (r *ProgressRepository) scanProgress(row *sql.Row) (*domain.Progress, error) {
	var progress domain.Progress
	var gradeJSON, attendanceJSON []byte
	var examScore, classPosition sql.NullInt64
	var tutorRemarks, principalRemarks, flagReason sql.NullString
	var completedAt sql.NullTime

	err := row.Scan(
		&progress.ID,
		&progress.TenantID,
		&progress.StudentID,
		&progress.CourseID,
		&progress.TermID,
		&progress.ClassID,
		&progress.Status,
		pq.Array(&progress.QuizScores),
		pq.Array(&progress.AssignmentScores),
		&examScore,
		&gradeJSON,
		&attendanceJSON,
		&tutorRemarks,
		&principalRemarks,
		&classPosition,
		&progress.IsFlagged,
		&flagReason,
		&progress.CreatedAt,
		&progress.UpdatedAt,
		&completedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	if examScore.Valid {
		score := int(examScore.Int64)
		progress.ExaminationScore = &score
	}
	if classPosition.Valid {
		pos := int(classPosition.Int64)
		progress.ClassPosition = &pos
	}
	if tutorRemarks.Valid {
		progress.TutorRemarks = &tutorRemarks.String
	}
	if principalRemarks.Valid {
		progress.PrincipalRemarks = &principalRemarks.String
	}
	if flagReason.Valid {
		progress.FlagReason = &flagReason.String
	}
	if completedAt.Valid {
		progress.CompletedAt = &completedAt.Time
	}

	if len(gradeJSON) > 0 && string(gradeJSON) != "null" {
		if err := json.Unmarshal(gradeJSON, &progress.Grade); err != nil {
			return nil, fmt.Errorf("failed to unmarshal grade: %w", err)
		}
	}

	if err := json.Unmarshal(attendanceJSON, &progress.Attendance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attendance: %w", err)
	}

	return &progress, nil
}

// Update updates a progress record
func (r *ProgressRepository) Update(ctx context.Context, progress *domain.Progress, tx *sql.Tx) error {
	gradeJSON, err := json.Marshal(progress.Grade)
	if err != nil {
		return fmt.Errorf("failed to marshal grade: %w", err)
	}

	attendanceJSON, err := json.Marshal(progress.Attendance)
	if err != nil {
		return fmt.Errorf("failed to marshal attendance: %w", err)
	}

	query := `
		UPDATE progress
		SET
			status = $2,
			quiz_scores = $3,
			assignment_scores = $4,
			examination_score = $5,
			grade = $6,
			attendance = $7,
			tutor_remarks = $8,
			principal_remarks = $9,
			class_position = $10,
			is_flagged = $11,
			flag_reason = $12,
			updated_at = $13,
			completed_at = $14
		WHERE id = $1
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	result, err := execer.ExecContext(ctx, query,
		progress.ID,
		progress.Status,
		pq.Array(progress.QuizScores),
		pq.Array(progress.AssignmentScores),
		progress.ExaminationScore,
		gradeJSON,
		attendanceJSON,
		progress.TutorRemarks,
		progress.PrincipalRemarks,
		progress.ClassPosition,
		progress.IsFlagged,
		progress.FlagReason,
		progress.UpdatedAt,
		progress.CompletedAt,
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

// ListByStudent retrieves all progress records for a student
func (r *ProgressRepository) ListByStudent(ctx context.Context, studentID uuid.UUID, params repository.PaginationParams) ([]*domain.Progress, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	whereClause := "WHERE student_id = $1"
	args := []interface{}{studentID}
	argIndex := 2

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM progress %s", whereClause)
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT
			id, tenant_id, student_id, course_id, term_id, class_id,
			status, quiz_scores, assignment_scores, examination_score,
			grade, attendance, tutor_remarks, principal_remarks,
			class_position, is_flagged, flag_reason,
			created_at, updated_at, completed_at
		FROM progress
		%s ORDER BY created_at %s LIMIT $%d OFFSET $%d`,
		whereClause, params.SortOrder, argIndex, argIndex+1)
	args = append(args, params.Limit, params.Offset())

	records, err := r.queryProgressRecords(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

// ListByStudentAndTerm retrieves progress records for a student in a specific term
func (r *ProgressRepository) ListByStudentAndTerm(ctx context.Context, studentID, termID uuid.UUID) ([]*domain.Progress, error) {
	query := `
		SELECT
			id, tenant_id, student_id, course_id, term_id, class_id,
			status, quiz_scores, assignment_scores, examination_score,
			grade, attendance, tutor_remarks, principal_remarks,
			class_position, is_flagged, flag_reason,
			created_at, updated_at, completed_at
		FROM progress
		WHERE student_id = $1 AND term_id = $2
		ORDER BY created_at ASC
	`

	return r.queryProgressRecords(ctx, query, studentID, termID)
}

// ListByCourse retrieves progress records for all students in a course
func (r *ProgressRepository) ListByCourse(ctx context.Context, courseID uuid.UUID, params repository.PaginationParams) ([]*domain.Progress, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	whereClause := "WHERE course_id = $1"
	args := []interface{}{courseID}
	argIndex := 2

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM progress %s", whereClause)
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT
			id, tenant_id, student_id, course_id, term_id, class_id,
			status, quiz_scores, assignment_scores, examination_score,
			grade, attendance, tutor_remarks, principal_remarks,
			class_position, is_flagged, flag_reason,
			created_at, updated_at, completed_at
		FROM progress
		%s ORDER BY created_at %s LIMIT $%d OFFSET $%d`,
		whereClause, params.SortOrder, argIndex, argIndex+1)
	args = append(args, params.Limit, params.Offset())

	records, err := r.queryProgressRecords(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

// ListByCourseAndTerm retrieves progress records for all students in a course for a term
func (r *ProgressRepository) ListByCourseAndTerm(ctx context.Context, courseID, termID uuid.UUID) ([]*domain.Progress, error) {
	query := `
		SELECT
			id, tenant_id, student_id, course_id, term_id, class_id,
			status, quiz_scores, assignment_scores, examination_score,
			grade, attendance, tutor_remarks, principal_remarks,
			class_position, is_flagged, flag_reason,
			created_at, updated_at, completed_at
		FROM progress
		WHERE course_id = $1 AND term_id = $2
		ORDER BY class_position ASC NULLS LAST
	`

	return r.queryProgressRecords(ctx, query, courseID, termID)
}

// ListByClass retrieves progress records for all students in a class
func (r *ProgressRepository) ListByClass(ctx context.Context, classID uuid.UUID, params repository.PaginationParams) ([]*domain.Progress, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	whereClause := "WHERE class_id = $1"
	args := []interface{}{classID}
	argIndex := 2

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM progress %s", whereClause)
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT
			id, tenant_id, student_id, course_id, term_id, class_id,
			status, quiz_scores, assignment_scores, examination_score,
			grade, attendance, tutor_remarks, principal_remarks,
			class_position, is_flagged, flag_reason,
			created_at, updated_at, completed_at
		FROM progress
		%s ORDER BY created_at %s LIMIT $%d OFFSET $%d`,
		whereClause, params.SortOrder, argIndex, argIndex+1)
	args = append(args, params.Limit, params.Offset())

	records, err := r.queryProgressRecords(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

// ListByClassAndTerm retrieves progress records for all students in a class for a term
func (r *ProgressRepository) ListByClassAndTerm(ctx context.Context, classID, termID uuid.UUID) ([]*domain.Progress, error) {
	query := `
		SELECT
			id, tenant_id, student_id, course_id, term_id, class_id,
			status, quiz_scores, assignment_scores, examination_score,
			grade, attendance, tutor_remarks, principal_remarks,
			class_position, is_flagged, flag_reason,
			created_at, updated_at, completed_at
		FROM progress
		WHERE class_id = $1 AND term_id = $2
		ORDER BY student_id, course_id
	`

	return r.queryProgressRecords(ctx, query, classID, termID)
}

// ListFlaggedByTenant retrieves all flagged progress records for a tenant
func (r *ProgressRepository) ListFlaggedByTenant(ctx context.Context, tenantID uuid.UUID, params repository.PaginationParams) ([]*domain.Progress, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	whereClause := "WHERE tenant_id = $1 AND is_flagged = true"
	args := []interface{}{tenantID}
	argIndex := 2

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM progress %s", whereClause)
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT
			id, tenant_id, student_id, course_id, term_id, class_id,
			status, quiz_scores, assignment_scores, examination_score,
			grade, attendance, tutor_remarks, principal_remarks,
			class_position, is_flagged, flag_reason,
			created_at, updated_at, completed_at
		FROM progress
		%s ORDER BY updated_at %s LIMIT $%d OFFSET $%d`,
		whereClause, params.SortOrder, argIndex, argIndex+1)
	args = append(args, params.Limit, params.Offset())

	records, err := r.queryProgressRecords(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

// ListFlaggedByCourse retrieves all flagged progress records for a course
func (r *ProgressRepository) ListFlaggedByCourse(ctx context.Context, courseID uuid.UUID) ([]*domain.Progress, error) {
	query := `
		SELECT
			id, tenant_id, student_id, course_id, term_id, class_id,
			status, quiz_scores, assignment_scores, examination_score,
			grade, attendance, tutor_remarks, principal_remarks,
			class_position, is_flagged, flag_reason,
			created_at, updated_at, completed_at
		FROM progress
		WHERE course_id = $1 AND is_flagged = true
		ORDER BY updated_at DESC
	`

	return r.queryProgressRecords(ctx, query, courseID)
}

// queryProgressRecords executes a query and returns a list of progress records
func (r *ProgressRepository) queryProgressRecords(ctx context.Context, query string, args ...interface{}) ([]*domain.Progress, error) {
	rows, err := r.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, repository.ParseError(err)
	}
	defer rows.Close()

	records := make([]*domain.Progress, 0)
	for rows.Next() {
		var progress domain.Progress
		var gradeJSON, attendanceJSON []byte
		var examScore, classPosition sql.NullInt64
		var tutorRemarks, principalRemarks, flagReason sql.NullString
		var completedAt sql.NullTime

		err := rows.Scan(
			&progress.ID,
			&progress.TenantID,
			&progress.StudentID,
			&progress.CourseID,
			&progress.TermID,
			&progress.ClassID,
			&progress.Status,
			pq.Array(&progress.QuizScores),
			pq.Array(&progress.AssignmentScores),
			&examScore,
			&gradeJSON,
			&attendanceJSON,
			&tutorRemarks,
			&principalRemarks,
			&classPosition,
			&progress.IsFlagged,
			&flagReason,
			&progress.CreatedAt,
			&progress.UpdatedAt,
			&completedAt,
		)

		if err != nil {
			return nil, repository.ParseError(err)
		}

		if examScore.Valid {
			score := int(examScore.Int64)
			progress.ExaminationScore = &score
		}
		if classPosition.Valid {
			pos := int(classPosition.Int64)
			progress.ClassPosition = &pos
		}
		if tutorRemarks.Valid {
			progress.TutorRemarks = &tutorRemarks.String
		}
		if principalRemarks.Valid {
			progress.PrincipalRemarks = &principalRemarks.String
		}
		if flagReason.Valid {
			progress.FlagReason = &flagReason.String
		}
		if completedAt.Valid {
			progress.CompletedAt = &completedAt.Time
		}

		if len(gradeJSON) > 0 && string(gradeJSON) != "null" {
			if err := json.Unmarshal(gradeJSON, &progress.Grade); err != nil {
				return nil, fmt.Errorf("failed to unmarshal grade: %w", err)
			}
		}

		if err := json.Unmarshal(attendanceJSON, &progress.Attendance); err != nil {
			return nil, fmt.Errorf("failed to unmarshal attendance: %w", err)
		}

		records = append(records, &progress)
	}

	if err := rows.Err(); err != nil {
		return nil, repository.ParseError(err)
	}

	return records, nil
}

// ===================== Report Card Operations =====================

// CreateReportCard creates a new report card
func (r *ProgressRepository) CreateReportCard(ctx context.Context, reportCard *domain.ReportCard, tx *sql.Tx) error {
	progressesJSON, err := json.Marshal(reportCard.CourseProgresses)
	if err != nil {
		return fmt.Errorf("failed to marshal course progresses: %w", err)
	}

	query := `
		INSERT INTO report_cards (
			id, tenant_id, student_id, term_id, class_id,
			course_progresses, overall_percentage, overall_grade,
			class_position, total_students, principal_remarks,
			next_term_begins, generated_at, generated_by, pdf_url,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	_, err = execer.ExecContext(ctx, query,
		reportCard.ID,
		reportCard.TenantID,
		reportCard.StudentID,
		reportCard.TermID,
		reportCard.ClassID,
		progressesJSON,
		reportCard.OverallPercentage,
		reportCard.OverallGrade,
		reportCard.ClassPosition,
		reportCard.TotalStudents,
		reportCard.PrincipalRemarks,
		reportCard.NextTermBegins,
		reportCard.GeneratedAt,
		reportCard.GeneratedBy,
		reportCard.PDFUrl,
		reportCard.CreatedAt,
		reportCard.UpdatedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// GetReportCard retrieves a report card by ID
func (r *ProgressRepository) GetReportCard(ctx context.Context, id uuid.UUID) (*domain.ReportCard, error) {
	query := `
		SELECT
			id, tenant_id, student_id, term_id, class_id,
			course_progresses, overall_percentage, overall_grade,
			class_position, total_students, principal_remarks,
			next_term_begins, generated_at, generated_by, pdf_url,
			created_at, updated_at
		FROM report_cards
		WHERE id = $1
	`

	return r.scanReportCard(r.GetDB().QueryRowContext(ctx, query, id))
}

// GetReportCardByStudentAndTerm retrieves a report card for a student in a term
func (r *ProgressRepository) GetReportCardByStudentAndTerm(ctx context.Context, studentID, termID uuid.UUID) (*domain.ReportCard, error) {
	query := `
		SELECT
			id, tenant_id, student_id, term_id, class_id,
			course_progresses, overall_percentage, overall_grade,
			class_position, total_students, principal_remarks,
			next_term_begins, generated_at, generated_by, pdf_url,
			created_at, updated_at
		FROM report_cards
		WHERE student_id = $1 AND term_id = $2
	`

	return r.scanReportCard(r.GetDB().QueryRowContext(ctx, query, studentID, termID))
}

// scanReportCard scans a report card from a database row
func (r *ProgressRepository) scanReportCard(row *sql.Row) (*domain.ReportCard, error) {
	var reportCard domain.ReportCard
	var progressesJSON []byte
	var principalRemarks, pdfUrl sql.NullString
	var nextTermBegins sql.NullTime

	err := row.Scan(
		&reportCard.ID,
		&reportCard.TenantID,
		&reportCard.StudentID,
		&reportCard.TermID,
		&reportCard.ClassID,
		&progressesJSON,
		&reportCard.OverallPercentage,
		&reportCard.OverallGrade,
		&reportCard.ClassPosition,
		&reportCard.TotalStudents,
		&principalRemarks,
		&nextTermBegins,
		&reportCard.GeneratedAt,
		&reportCard.GeneratedBy,
		&pdfUrl,
		&reportCard.CreatedAt,
		&reportCard.UpdatedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	if principalRemarks.Valid {
		reportCard.PrincipalRemarks = &principalRemarks.String
	}
	if pdfUrl.Valid {
		reportCard.PDFUrl = &pdfUrl.String
	}
	if nextTermBegins.Valid {
		reportCard.NextTermBegins = &nextTermBegins.Time
	}

	if err := json.Unmarshal(progressesJSON, &reportCard.CourseProgresses); err != nil {
		return nil, fmt.Errorf("failed to unmarshal course progresses: %w", err)
	}

	return &reportCard, nil
}

// UpdateReportCard updates a report card
func (r *ProgressRepository) UpdateReportCard(ctx context.Context, reportCard *domain.ReportCard, tx *sql.Tx) error {
	progressesJSON, err := json.Marshal(reportCard.CourseProgresses)
	if err != nil {
		return fmt.Errorf("failed to marshal course progresses: %w", err)
	}

	query := `
		UPDATE report_cards
		SET
			course_progresses = $2,
			overall_percentage = $3,
			overall_grade = $4,
			class_position = $5,
			total_students = $6,
			principal_remarks = $7,
			next_term_begins = $8,
			pdf_url = $9,
			updated_at = $10
		WHERE id = $1
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	result, err := execer.ExecContext(ctx, query,
		reportCard.ID,
		progressesJSON,
		reportCard.OverallPercentage,
		reportCard.OverallGrade,
		reportCard.ClassPosition,
		reportCard.TotalStudents,
		reportCard.PrincipalRemarks,
		reportCard.NextTermBegins,
		reportCard.PDFUrl,
		reportCard.UpdatedAt,
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

// ListReportCardsByStudent retrieves all report cards for a student
func (r *ProgressRepository) ListReportCardsByStudent(ctx context.Context, studentID uuid.UUID) ([]*domain.ReportCard, error) {
	query := `
		SELECT
			id, tenant_id, student_id, term_id, class_id,
			course_progresses, overall_percentage, overall_grade,
			class_position, total_students, principal_remarks,
			next_term_begins, generated_at, generated_by, pdf_url,
			created_at, updated_at
		FROM report_cards
		WHERE student_id = $1
		ORDER BY generated_at DESC
	`

	return r.queryReportCards(ctx, query, studentID)
}

// ListReportCardsByClassAndTerm retrieves all report cards for a class in a term
func (r *ProgressRepository) ListReportCardsByClassAndTerm(ctx context.Context, classID, termID uuid.UUID) ([]*domain.ReportCard, error) {
	query := `
		SELECT
			id, tenant_id, student_id, term_id, class_id,
			course_progresses, overall_percentage, overall_grade,
			class_position, total_students, principal_remarks,
			next_term_begins, generated_at, generated_by, pdf_url,
			created_at, updated_at
		FROM report_cards
		WHERE class_id = $1 AND term_id = $2
		ORDER BY class_position ASC
	`

	return r.queryReportCards(ctx, query, classID, termID)
}

// queryReportCards executes a query and returns a list of report cards
func (r *ProgressRepository) queryReportCards(ctx context.Context, query string, args ...interface{}) ([]*domain.ReportCard, error) {
	rows, err := r.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, repository.ParseError(err)
	}
	defer rows.Close()

	reportCards := make([]*domain.ReportCard, 0)
	for rows.Next() {
		var reportCard domain.ReportCard
		var progressesJSON []byte
		var principalRemarks, pdfUrl sql.NullString
		var nextTermBegins sql.NullTime

		err := rows.Scan(
			&reportCard.ID,
			&reportCard.TenantID,
			&reportCard.StudentID,
			&reportCard.TermID,
			&reportCard.ClassID,
			&progressesJSON,
			&reportCard.OverallPercentage,
			&reportCard.OverallGrade,
			&reportCard.ClassPosition,
			&reportCard.TotalStudents,
			&principalRemarks,
			&nextTermBegins,
			&reportCard.GeneratedAt,
			&reportCard.GeneratedBy,
			&pdfUrl,
			&reportCard.CreatedAt,
			&reportCard.UpdatedAt,
		)

		if err != nil {
			return nil, repository.ParseError(err)
		}

		if principalRemarks.Valid {
			reportCard.PrincipalRemarks = &principalRemarks.String
		}
		if pdfUrl.Valid {
			reportCard.PDFUrl = &pdfUrl.String
		}
		if nextTermBegins.Valid {
			reportCard.NextTermBegins = &nextTermBegins.Time
		}

		if err := json.Unmarshal(progressesJSON, &reportCard.CourseProgresses); err != nil {
			return nil, fmt.Errorf("failed to unmarshal course progresses: %w", err)
		}

		reportCards = append(reportCards, &reportCard)
	}

	if err := rows.Err(); err != nil {
		return nil, repository.ParseError(err)
	}

	return reportCards, nil
}

// ===================== Statistics & Aggregation =====================

// GetCourseStatistics retrieves statistics for a course in a term
func (r *ProgressRepository) GetCourseStatistics(ctx context.Context, courseID, termID uuid.UUID) (*CourseStatistics, error) {
	query := `
		SELECT
			COUNT(*) as total_students,
			COALESCE(AVG((grade->>'percentage')::float), 0) as average_percentage,
			COALESCE(MAX((grade->>'percentage')::float), 0) as highest_percentage,
			COALESCE(MIN((grade->>'percentage')::float), 0) as lowest_percentage,
			COUNT(*) FILTER (WHERE is_flagged = true) as flagged_count,
			COALESCE(AVG((attendance->>'percentage')::float), 0) as average_attendance
		FROM progress
		WHERE course_id = $1 AND term_id = $2 AND grade IS NOT NULL
	`

	var stats CourseStatistics
	stats.CourseID = courseID
	stats.TermID = termID

	err := r.GetDB().QueryRowContext(ctx, query, courseID, termID).Scan(
		&stats.TotalStudents,
		&stats.AveragePercentage,
		&stats.HighestPercentage,
		&stats.LowestPercentage,
		&stats.FlaggedCount,
		&stats.AverageAttendance,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	return &stats, nil
}

// CourseStatistics represents aggregated statistics for a course
type CourseStatistics struct {
	CourseID          uuid.UUID `json:"course_id"`
	TermID            uuid.UUID `json:"term_id"`
	TotalStudents     int       `json:"total_students"`
	AveragePercentage float64   `json:"average_percentage"`
	HighestPercentage float64   `json:"highest_percentage"`
	LowestPercentage  float64   `json:"lowest_percentage"`
	FlaggedCount      int       `json:"flagged_count"`
	AverageAttendance float64   `json:"average_attendance"`
}

// GetClassStatistics retrieves statistics for a class in a term
func (r *ProgressRepository) GetClassStatistics(ctx context.Context, classID, termID uuid.UUID) (*ClassStatistics, error) {
	query := `
		SELECT
			COUNT(DISTINCT student_id) as total_students,
			COUNT(DISTINCT course_id) as total_courses,
			COALESCE(AVG((grade->>'percentage')::float), 0) as average_percentage,
			COUNT(*) FILTER (WHERE is_flagged = true) as flagged_count,
			COALESCE(AVG((attendance->>'percentage')::float), 0) as average_attendance
		FROM progress
		WHERE class_id = $1 AND term_id = $2
	`

	var stats ClassStatistics
	stats.ClassID = classID
	stats.TermID = termID

	err := r.GetDB().QueryRowContext(ctx, query, classID, termID).Scan(
		&stats.TotalStudents,
		&stats.TotalCourses,
		&stats.AveragePercentage,
		&stats.FlaggedCount,
		&stats.AverageAttendance,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	return &stats, nil
}

// ClassStatistics represents aggregated statistics for a class
type ClassStatistics struct {
	ClassID           uuid.UUID `json:"class_id"`
	TermID            uuid.UUID `json:"term_id"`
	TotalStudents     int       `json:"total_students"`
	TotalCourses      int       `json:"total_courses"`
	AveragePercentage float64   `json:"average_percentage"`
	FlaggedCount      int       `json:"flagged_count"`
	AverageAttendance float64   `json:"average_attendance"`
}

// ===================== Attendance Operations =====================

// AttendanceEntry represents a single attendance record
type AttendanceEntry struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	ProgressID uuid.UUID `json:"progress_id"`
	PeriodID   uuid.UUID `json:"period_id"`
	Date       time.Time `json:"date"`
	IsPresent  bool      `json:"is_present"`
	MarkedBy   uuid.UUID `json:"marked_by"`
	MarkedAt   time.Time `json:"marked_at"`
}

// CreateAttendanceEntry creates a new attendance entry
func (r *ProgressRepository) CreateAttendanceEntry(ctx context.Context, entry *AttendanceEntry, tx *sql.Tx) error {
	query := `
		INSERT INTO attendance_entries (
			id, tenant_id, progress_id, period_id, date, is_present, marked_by, marked_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (progress_id, period_id, date) DO UPDATE SET
			is_present = $6,
			marked_by = $7,
			marked_at = $8
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	_, err := execer.ExecContext(ctx, query,
		entry.ID,
		entry.TenantID,
		entry.ProgressID,
		entry.PeriodID,
		entry.Date,
		entry.IsPresent,
		entry.MarkedBy,
		entry.MarkedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// GetAttendanceByProgressAndDateRange retrieves attendance entries for a progress record
func (r *ProgressRepository) GetAttendanceByProgressAndDateRange(ctx context.Context, progressID uuid.UUID, startDate, endDate time.Time) ([]*AttendanceEntry, error) {
	query := `
		SELECT id, tenant_id, progress_id, period_id, date, is_present, marked_by, marked_at
		FROM attendance_entries
		WHERE progress_id = $1 AND date >= $2 AND date <= $3
		ORDER BY date ASC
	`

	rows, err := r.GetDB().QueryContext(ctx, query, progressID, startDate, endDate)
	if err != nil {
		return nil, repository.ParseError(err)
	}
	defer rows.Close()

	entries := make([]*AttendanceEntry, 0)
	for rows.Next() {
		var entry AttendanceEntry
		err := rows.Scan(
			&entry.ID,
			&entry.TenantID,
			&entry.ProgressID,
			&entry.PeriodID,
			&entry.Date,
			&entry.IsPresent,
			&entry.MarkedBy,
			&entry.MarkedAt,
		)
		if err != nil {
			return nil, repository.ParseError(err)
		}
		entries = append(entries, &entry)
	}

	return entries, nil
}

// ComputeAttendanceSummary computes attendance summary for a progress record
func (r *ProgressRepository) ComputeAttendanceSummary(ctx context.Context, progressID uuid.UUID) (*domain.AttendanceRecord, error) {
	query := `
		SELECT
			COUNT(*) as total_periods,
			COUNT(*) FILTER (WHERE is_present = true) as periods_attended,
			COUNT(*) FILTER (WHERE is_present = false) as periods_absent
		FROM attendance_entries
		WHERE progress_id = $1
	`

	var attendance domain.AttendanceRecord
	err := r.GetDB().QueryRowContext(ctx, query, progressID).Scan(
		&attendance.TotalPeriods,
		&attendance.PeriodsAttended,
		&attendance.PeriodsAbsent,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	if attendance.TotalPeriods > 0 {
		attendance.Percentage = (float64(attendance.PeriodsAttended) / float64(attendance.TotalPeriods)) * 100
	}

	return &attendance, nil
}
