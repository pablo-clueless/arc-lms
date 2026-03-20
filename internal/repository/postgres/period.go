package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"arc-lms/internal/domain"
	"arc-lms/internal/repository"
)

// PeriodRepository handles database operations for timetable periods
type PeriodRepository struct {
	*repository.BaseRepository
}

// NewPeriodRepository creates a new period repository
func NewPeriodRepository(db *sql.DB) *PeriodRepository {
	return &PeriodRepository{
		BaseRepository: repository.NewBaseRepository(db),
	}
}

// Create creates a new period
func (r *PeriodRepository) Create(ctx context.Context, period *domain.Period, tx *sql.Tx) error {
	query := `
		INSERT INTO periods (
			id, tenant_id, timetable_id, course_id, tutor_id, class_id,
			day_of_week, start_time, end_time, period_number, notes,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	_, err := execer.ExecContext(ctx, query,
		period.ID,
		period.TenantID,
		period.TimetableID,
		period.CourseID,
		period.TutorID,
		period.ClassID,
		period.DayOfWeek,
		period.StartTime,
		period.EndTime,
		period.PeriodNumber,
		repository.ToNullString(period.Notes),
		period.CreatedAt,
		period.UpdatedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// CreateBatch creates multiple periods in a single transaction
func (r *PeriodRepository) CreateBatch(ctx context.Context, periods []*domain.Period, tx *sql.Tx) error {
	for _, period := range periods {
		if err := r.Create(ctx, period, tx); err != nil {
			return err
		}
	}
	return nil
}

// Get retrieves a period by ID
func (r *PeriodRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Period, error) {
	query := `
		SELECT
			id, tenant_id, timetable_id, course_id, tutor_id, class_id,
			day_of_week, start_time, end_time, period_number, notes,
			created_at, updated_at
		FROM periods
		WHERE id = $1
	`

	return r.scanPeriod(r.GetDB().QueryRowContext(ctx, query, id))
}

// scanPeriod scans a period from a database row
func (r *PeriodRepository) scanPeriod(row *sql.Row) (*domain.Period, error) {
	var period domain.Period
	var notes sql.NullString

	err := row.Scan(
		&period.ID,
		&period.TenantID,
		&period.TimetableID,
		&period.CourseID,
		&period.TutorID,
		&period.ClassID,
		&period.DayOfWeek,
		&period.StartTime,
		&period.EndTime,
		&period.PeriodNumber,
		&notes,
		&period.CreatedAt,
		&period.UpdatedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	period.Notes = repository.FromNullString(notes)

	return &period, nil
}

// Update updates an existing period
func (r *PeriodRepository) Update(ctx context.Context, period *domain.Period, tx *sql.Tx) error {
	query := `
		UPDATE periods
		SET
			course_id = $2,
			tutor_id = $3,
			day_of_week = $4,
			start_time = $5,
			end_time = $6,
			period_number = $7,
			notes = $8,
			updated_at = $9
		WHERE id = $1
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	result, err := execer.ExecContext(ctx, query,
		period.ID,
		period.CourseID,
		period.TutorID,
		period.DayOfWeek,
		period.StartTime,
		period.EndTime,
		period.PeriodNumber,
		repository.ToNullString(period.Notes),
		period.UpdatedAt,
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

// Delete deletes a period
func (r *PeriodRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM periods WHERE id = $1`

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

// DeleteByTimetable deletes all periods for a timetable
func (r *PeriodRepository) DeleteByTimetable(ctx context.Context, timetableID uuid.UUID, tx *sql.Tx) error {
	query := `DELETE FROM periods WHERE timetable_id = $1`

	execer := repository.GetExecer(r.GetDB(), tx)
	_, err := execer.ExecContext(ctx, query, timetableID)
	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// ListByTimetable retrieves all periods for a timetable
func (r *PeriodRepository) ListByTimetable(ctx context.Context, timetableID uuid.UUID) ([]*domain.Period, error) {
	query := `
		SELECT
			id, tenant_id, timetable_id, course_id, tutor_id, class_id,
			day_of_week, start_time, end_time, period_number, notes,
			created_at, updated_at
		FROM periods
		WHERE timetable_id = $1
		ORDER BY day_of_week, period_number
	`

	return r.queryPeriods(ctx, query, timetableID)
}

// ListByTutor retrieves all periods for a tutor within a term
func (r *PeriodRepository) ListByTutor(ctx context.Context, tutorID uuid.UUID, termID uuid.UUID) ([]*domain.Period, error) {
	query := `
		SELECT
			p.id, p.tenant_id, p.timetable_id, p.course_id, p.tutor_id, p.class_id,
			p.day_of_week, p.start_time, p.end_time, p.period_number, p.notes,
			p.created_at, p.updated_at
		FROM periods p
		JOIN timetables t ON p.timetable_id = t.id
		WHERE p.tutor_id = $1 AND t.term_id = $2 AND t.status = $3
		ORDER BY p.day_of_week, p.period_number
	`

	return r.queryPeriods(ctx, query, tutorID, termID, domain.TimetableStatusPublished)
}

// ListByClass retrieves all periods for a class within a timetable
func (r *PeriodRepository) ListByClass(ctx context.Context, classID uuid.UUID, timetableID uuid.UUID) ([]*domain.Period, error) {
	query := `
		SELECT
			id, tenant_id, timetable_id, course_id, tutor_id, class_id,
			day_of_week, start_time, end_time, period_number, notes,
			created_at, updated_at
		FROM periods
		WHERE class_id = $1 AND timetable_id = $2
		ORDER BY day_of_week, period_number
	`

	return r.queryPeriods(ctx, query, classID, timetableID)
}

// GetTutorConflicts finds periods where a tutor has conflicts on a given day/time
func (r *PeriodRepository) GetTutorConflicts(ctx context.Context, tutorID uuid.UUID, termID uuid.UUID, dayOfWeek domain.DayOfWeek, startTime, endTime time.Time) ([]*domain.Period, error) {
	query := `
		SELECT
			p.id, p.tenant_id, p.timetable_id, p.course_id, p.tutor_id, p.class_id,
			p.day_of_week, p.start_time, p.end_time, p.period_number, p.notes,
			p.created_at, p.updated_at
		FROM periods p
		JOIN timetables t ON p.timetable_id = t.id
		WHERE p.tutor_id = $1
			AND t.term_id = $2
			AND p.day_of_week = $3
			AND t.status IN ($4, $5)
			AND (
				(p.start_time < $7 AND p.end_time > $6)
			)
	`

	return r.queryPeriods(ctx, query, tutorID, termID, dayOfWeek, domain.TimetableStatusDraft, domain.TimetableStatusPublished, startTime, endTime)
}

// GetClassConflicts finds periods where a class has conflicts on a given day/time
func (r *PeriodRepository) GetClassConflicts(ctx context.Context, classID uuid.UUID, timetableID uuid.UUID, dayOfWeek domain.DayOfWeek, startTime, endTime time.Time, excludePeriodID *uuid.UUID) ([]*domain.Period, error) {
	query := `
		SELECT
			id, tenant_id, timetable_id, course_id, tutor_id, class_id,
			day_of_week, start_time, end_time, period_number, notes,
			created_at, updated_at
		FROM periods
		WHERE class_id = $1
			AND timetable_id = $2
			AND day_of_week = $3
			AND (start_time < $5 AND end_time > $4)
	`

	args := []interface{}{classID, timetableID, dayOfWeek, startTime, endTime}

	if excludePeriodID != nil {
		query += " AND id != $6"
		args = append(args, *excludePeriodID)
	}

	return r.queryPeriods(ctx, query, args...)
}

// queryPeriods executes a query and returns a list of periods
func (r *PeriodRepository) queryPeriods(ctx context.Context, query string, args ...interface{}) ([]*domain.Period, error) {
	rows, err := r.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, repository.ParseError(err)
	}
	defer rows.Close()

	periods := make([]*domain.Period, 0)
	for rows.Next() {
		var period domain.Period
		var notes sql.NullString

		err := rows.Scan(
			&period.ID,
			&period.TenantID,
			&period.TimetableID,
			&period.CourseID,
			&period.TutorID,
			&period.ClassID,
			&period.DayOfWeek,
			&period.StartTime,
			&period.EndTime,
			&period.PeriodNumber,
			&notes,
			&period.CreatedAt,
			&period.UpdatedAt,
		)

		if err != nil {
			return nil, repository.ParseError(err)
		}

		period.Notes = repository.FromNullString(notes)
		periods = append(periods, &period)
	}

	if err := rows.Err(); err != nil {
		return nil, repository.ParseError(err)
	}

	return periods, nil
}

// SwapPeriods swaps tutor assignments between two periods
func (r *PeriodRepository) SwapPeriods(ctx context.Context, period1ID, period2ID uuid.UUID, tx *sql.Tx) error {
	// Get both periods
	period1, err := r.Get(ctx, period1ID)
	if err != nil {
		return fmt.Errorf("failed to get period 1: %w", err)
	}

	period2, err := r.Get(ctx, period2ID)
	if err != nil {
		return fmt.Errorf("failed to get period 2: %w", err)
	}

	// Swap tutor and course assignments
	now := time.Now()

	query1 := `
		UPDATE periods
		SET tutor_id = $2, course_id = $3, updated_at = $4
		WHERE id = $1
	`

	query2 := `
		UPDATE periods
		SET tutor_id = $2, course_id = $3, updated_at = $4
		WHERE id = $1
	`

	execer := repository.GetExecer(r.GetDB(), tx)

	_, err = execer.ExecContext(ctx, query1, period1ID, period2.TutorID, period2.CourseID, now)
	if err != nil {
		return repository.ParseError(err)
	}

	_, err = execer.ExecContext(ctx, query2, period2ID, period1.TutorID, period1.CourseID, now)
	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// ValidateTenantAccess validates that a period belongs to a tenant
func (r *PeriodRepository) ValidateTenantAccess(ctx context.Context, tenantID uuid.UUID, periodID uuid.UUID) error {
	query := `SELECT 1 FROM periods WHERE id = $1 AND tenant_id = $2`

	var exists int
	err := r.GetDB().QueryRowContext(ctx, query, periodID, tenantID).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return repository.ErrNotFound
		}
		return repository.ParseError(err)
	}

	return nil
}
