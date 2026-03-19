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

// CourseRepository handles database operations for courses
type CourseRepository struct {
	*repository.BaseRepository
}

// NewCourseRepository creates a new course repository
func NewCourseRepository(db *sql.DB) *CourseRepository {
	return &CourseRepository{
		BaseRepository: repository.NewBaseRepository(db),
	}
}

// Create creates a new course
func (r *CourseRepository) Create(ctx context.Context, course *domain.Course, tx *sql.Tx) error {
	var gradeWeightingJSON []byte
	var err error
	if course.CustomGradeWeighting != nil {
		gradeWeightingJSON, err = json.Marshal(course.CustomGradeWeighting)
		if err != nil {
			return fmt.Errorf("failed to marshal grade weighting: %w", err)
		}
	}

	materialsJSON, err := json.Marshal(course.Materials)
	if err != nil {
		return fmt.Errorf("failed to marshal materials: %w", err)
	}

	query := `
		INSERT INTO courses (
			id, tenant_id, session_id, class_id, term_id,
			name, subject_code, description, assigned_tutor_id,
			status, max_periods_per_week, custom_grade_weighting,
			materials, syllabus, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	var maxPeriods sql.NullInt32
	if course.MaxPeriodsPerWeek != nil {
		maxPeriods = sql.NullInt32{Int32: int32(*course.MaxPeriodsPerWeek), Valid: true}
	}

	execer := repository.GetExecer(r.GetDB(), tx)
	_, err = execer.ExecContext(ctx, query,
		course.ID,
		course.TenantID,
		course.SessionID,
		course.ClassID,
		course.TermID,
		course.Name,
		course.SubjectCode,
		repository.ToNullString(course.Description),
		course.AssignedTutorID,
		course.Status,
		maxPeriods,
		gradeWeightingJSON,
		materialsJSON,
		repository.ToNullString(course.Syllabus),
		course.CreatedAt,
		course.UpdatedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// Get retrieves a course by ID
func (r *CourseRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Course, error) {
	query := `
		SELECT
			id, tenant_id, session_id, class_id, term_id,
			name, subject_code, description, assigned_tutor_id,
			status, max_periods_per_week, custom_grade_weighting,
			materials, syllabus, created_at, updated_at
		FROM courses
		WHERE id = $1
	`

	return r.scanCourse(r.GetDB().QueryRowContext(ctx, query, id))
}

// scanCourse scans a course from a database row
func (r *CourseRepository) scanCourse(row *sql.Row) (*domain.Course, error) {
	var course domain.Course
	var description, syllabus sql.NullString
	var maxPeriods sql.NullInt32
	var gradeWeightingJSON, materialsJSON []byte

	err := row.Scan(
		&course.ID,
		&course.TenantID,
		&course.SessionID,
		&course.ClassID,
		&course.TermID,
		&course.Name,
		&course.SubjectCode,
		&description,
		&course.AssignedTutorID,
		&course.Status,
		&maxPeriods,
		&gradeWeightingJSON,
		&materialsJSON,
		&syllabus,
		&course.CreatedAt,
		&course.UpdatedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	course.Description = repository.FromNullString(description)
	course.Syllabus = repository.FromNullString(syllabus)

	if maxPeriods.Valid {
		periods := int(maxPeriods.Int32)
		course.MaxPeriodsPerWeek = &periods
	}

	if gradeWeightingJSON != nil {
		var gradeWeighting domain.GradeWeighting
		if err := json.Unmarshal(gradeWeightingJSON, &gradeWeighting); err != nil {
			return nil, fmt.Errorf("failed to unmarshal grade weighting: %w", err)
		}
		course.CustomGradeWeighting = &gradeWeighting
	}

	if err := json.Unmarshal(materialsJSON, &course.Materials); err != nil {
		return nil, fmt.Errorf("failed to unmarshal materials: %w", err)
	}

	return &course, nil
}

// Update updates an existing course
func (r *CourseRepository) Update(ctx context.Context, course *domain.Course, tx *sql.Tx) error {
	var gradeWeightingJSON []byte
	var err error
	if course.CustomGradeWeighting != nil {
		gradeWeightingJSON, err = json.Marshal(course.CustomGradeWeighting)
		if err != nil {
			return fmt.Errorf("failed to marshal grade weighting: %w", err)
		}
	}

	materialsJSON, err := json.Marshal(course.Materials)
	if err != nil {
		return fmt.Errorf("failed to marshal materials: %w", err)
	}

	query := `
		UPDATE courses
		SET
			name = $2,
			subject_code = $3,
			description = $4,
			status = $5,
			max_periods_per_week = $6,
			custom_grade_weighting = $7,
			materials = $8,
			syllabus = $9,
			updated_at = $10
		WHERE id = $1
	`

	var maxPeriods sql.NullInt32
	if course.MaxPeriodsPerWeek != nil {
		maxPeriods = sql.NullInt32{Int32: int32(*course.MaxPeriodsPerWeek), Valid: true}
	}

	execer := repository.GetExecer(r.GetDB(), tx)
	result, err := execer.ExecContext(ctx, query,
		course.ID,
		course.Name,
		course.SubjectCode,
		repository.ToNullString(course.Description),
		course.Status,
		maxPeriods,
		gradeWeightingJSON,
		materialsJSON,
		repository.ToNullString(course.Syllabus),
		course.UpdatedAt,
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

// Delete deletes a course
func (r *CourseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM courses WHERE id = $1`

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

// ListByClass retrieves courses for a class with pagination
func (r *CourseRepository) ListByClass(ctx context.Context, classID uuid.UUID, params repository.PaginationParams) ([]*domain.Course, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, err
	}

	query := `
		SELECT
			id, tenant_id, session_id, class_id, term_id,
			name, subject_code, description, assigned_tutor_id,
			status, max_periods_per_week, custom_grade_weighting,
			materials, syllabus, created_at, updated_at
		FROM courses
		WHERE class_id = $1
	`

	args := []interface{}{classID}
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

	query += fmt.Sprintf(" ORDER BY id %s LIMIT $%d", params.SortOrder, argIndex)
	args = append(args, params.Limit+1)

	return r.queryCourses(ctx, query, args...)
}

// ListByTerm retrieves courses for a term with pagination
func (r *CourseRepository) ListByTerm(ctx context.Context, termID uuid.UUID, params repository.PaginationParams) ([]*domain.Course, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, err
	}

	query := `
		SELECT
			id, tenant_id, session_id, class_id, term_id,
			name, subject_code, description, assigned_tutor_id,
			status, max_periods_per_week, custom_grade_weighting,
			materials, syllabus, created_at, updated_at
		FROM courses
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

	query += fmt.Sprintf(" ORDER BY id %s LIMIT $%d", params.SortOrder, argIndex)
	args = append(args, params.Limit+1)

	return r.queryCourses(ctx, query, args...)
}

// ListByTutor retrieves courses assigned to a tutor with pagination
func (r *CourseRepository) ListByTutor(ctx context.Context, tutorID uuid.UUID, params repository.PaginationParams) ([]*domain.Course, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, err
	}

	query := `
		SELECT
			id, tenant_id, session_id, class_id, term_id,
			name, subject_code, description, assigned_tutor_id,
			status, max_periods_per_week, custom_grade_weighting,
			materials, syllabus, created_at, updated_at
		FROM courses
		WHERE assigned_tutor_id = $1
	`

	args := []interface{}{tutorID}
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

	query += fmt.Sprintf(" ORDER BY id %s LIMIT $%d", params.SortOrder, argIndex)
	args = append(args, params.Limit+1)

	return r.queryCourses(ctx, query, args...)
}

// queryCourses executes a query and returns a list of courses
func (r *CourseRepository) queryCourses(ctx context.Context, query string, args ...interface{}) ([]*domain.Course, error) {
	rows, err := r.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, repository.ParseError(err)
	}
	defer rows.Close()

	courses := make([]*domain.Course, 0)
	for rows.Next() {
		course, err := r.scanCourseFromRows(rows)
		if err != nil {
			return nil, err
		}
		courses = append(courses, course)
	}

	if err := rows.Err(); err != nil {
		return nil, repository.ParseError(err)
	}

	return courses, nil
}

// scanCourseFromRows scans a course from a Rows object
func (r *CourseRepository) scanCourseFromRows(rows *sql.Rows) (*domain.Course, error) {
	var course domain.Course
	var description, syllabus sql.NullString
	var maxPeriods sql.NullInt32
	var gradeWeightingJSON, materialsJSON []byte

	err := rows.Scan(
		&course.ID,
		&course.TenantID,
		&course.SessionID,
		&course.ClassID,
		&course.TermID,
		&course.Name,
		&course.SubjectCode,
		&description,
		&course.AssignedTutorID,
		&course.Status,
		&maxPeriods,
		&gradeWeightingJSON,
		&materialsJSON,
		&syllabus,
		&course.CreatedAt,
		&course.UpdatedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	course.Description = repository.FromNullString(description)
	course.Syllabus = repository.FromNullString(syllabus)

	if maxPeriods.Valid {
		periods := int(maxPeriods.Int32)
		course.MaxPeriodsPerWeek = &periods
	}

	if gradeWeightingJSON != nil {
		var gradeWeighting domain.GradeWeighting
		if err := json.Unmarshal(gradeWeightingJSON, &gradeWeighting); err != nil {
			return nil, fmt.Errorf("failed to unmarshal grade weighting: %w", err)
		}
		course.CustomGradeWeighting = &gradeWeighting
	}

	if err := json.Unmarshal(materialsJSON, &course.Materials); err != nil {
		return nil, fmt.Errorf("failed to unmarshal materials: %w", err)
	}

	return &course, nil
}

// ReassignTutor changes the assigned tutor for a course
func (r *CourseRepository) ReassignTutor(ctx context.Context, courseID uuid.UUID, newTutorID uuid.UUID, tx *sql.Tx) error {
	query := `
		UPDATE courses
		SET
			assigned_tutor_id = $2,
			updated_at = $3
		WHERE id = $1
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	result, err := execer.ExecContext(ctx, query, courseID, newTutorID, time.Now())
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

// ValidateTenantAccess validates that a course belongs to a tenant
func (r *CourseRepository) ValidateTenantAccess(ctx context.Context, tenantID uuid.UUID, courseID uuid.UUID) error {
	query := `SELECT 1 FROM courses WHERE id = $1 AND tenant_id = $2`

	var exists int
	err := r.GetDB().QueryRowContext(ctx, query, courseID, tenantID).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return repository.ErrNotFound
		}
		return repository.ParseError(err)
	}

	return nil
}
