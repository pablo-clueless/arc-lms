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

// EnrollmentRepository handles database operations for enrollments
type EnrollmentRepository struct {
	*repository.BaseRepository
}

// NewEnrollmentRepository creates a new enrollment repository
func NewEnrollmentRepository(db *sql.DB) *EnrollmentRepository {
	return &EnrollmentRepository{
		BaseRepository: repository.NewBaseRepository(db),
	}
}

// Enroll creates a new enrollment (enforces BR-003: one class per session per student)
// Must be called within a transaction with FOR UPDATE lock
func (r *EnrollmentRepository) Enroll(ctx context.Context, enrollment *domain.Enrollment, tx *sql.Tx) error {
	if tx == nil {
		return fmt.Errorf("enroll must be called within a transaction")
	}

	// Check for existing enrollment in the same session (BR-003)
	// Use FOR UPDATE to prevent race conditions
	checkQuery := `
		SELECT id FROM enrollments
		WHERE tenant_id = $1 AND student_id = $2 AND session_id = $3
		FOR UPDATE
	`

	var existingID uuid.UUID
	err := tx.QueryRowContext(ctx, checkQuery, enrollment.TenantID, enrollment.StudentID, enrollment.SessionID).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		return repository.ParseError(err)
	}

	if err == nil {
		// Student already enrolled in a class for this session
		return fmt.Errorf("student already enrolled in a class for this session (BR-003): %w", repository.ErrDuplicateKey)
	}

	// Create the enrollment
	query := `
		INSERT INTO enrollments (
			id, tenant_id, student_id, class_id, session_id,
			status, enrollment_date, notes, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = tx.ExecContext(ctx, query,
		enrollment.ID,
		enrollment.TenantID,
		enrollment.StudentID,
		enrollment.ClassID,
		enrollment.SessionID,
		enrollment.Status,
		enrollment.EnrollmentDate,
		repository.ToNullString(enrollment.Notes),
		enrollment.CreatedAt,
		enrollment.UpdatedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// Get retrieves an enrollment by ID
func (r *EnrollmentRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Enrollment, error) {
	query := `
		SELECT
			id, tenant_id, student_id, class_id, session_id,
			status, enrollment_date, withdrawal_date, withdrawal_reason,
			transferred_to_class_id, transfer_date, transfer_reason,
			suspension_date, suspension_reason, notes,
			created_at, updated_at
		FROM enrollments
		WHERE id = $1
	`

	return r.scanEnrollment(r.GetDB().QueryRowContext(ctx, query, id))
}

// scanEnrollment scans an enrollment from a database row
func (r *EnrollmentRepository) scanEnrollment(row *sql.Row) (*domain.Enrollment, error) {
	var enrollment domain.Enrollment
	var withdrawalDate, transferDate, suspensionDate sql.NullTime
	var withdrawalReason, transferReason, suspensionReason, notes sql.NullString
	var transferredToClassID sql.NullString

	err := row.Scan(
		&enrollment.ID,
		&enrollment.TenantID,
		&enrollment.StudentID,
		&enrollment.ClassID,
		&enrollment.SessionID,
		&enrollment.Status,
		&enrollment.EnrollmentDate,
		&withdrawalDate,
		&withdrawalReason,
		&transferredToClassID,
		&transferDate,
		&transferReason,
		&suspensionDate,
		&suspensionReason,
		&notes,
		&enrollment.CreatedAt,
		&enrollment.UpdatedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	if withdrawalDate.Valid {
		enrollment.WithdrawalDate = &withdrawalDate.Time
	}
	if withdrawalReason.Valid {
		enrollment.WithdrawalReason = &withdrawalReason.String
	}
	if transferredToClassID.Valid {
		classID, _ := uuid.Parse(transferredToClassID.String)
		enrollment.TransferredToClassID = &classID
	}
	if transferDate.Valid {
		enrollment.TransferDate = &transferDate.Time
	}
	if transferReason.Valid {
		enrollment.TransferReason = &transferReason.String
	}
	if suspensionDate.Valid {
		enrollment.SuspensionDate = &suspensionDate.Time
	}
	if suspensionReason.Valid {
		enrollment.SuspensionReason = &suspensionReason.String
	}
	if notes.Valid {
		enrollment.Notes = &notes.String
	}

	return &enrollment, nil
}

// Update updates an existing enrollment
func (r *EnrollmentRepository) Update(ctx context.Context, enrollment *domain.Enrollment, tx *sql.Tx) error {
	query := `
		UPDATE enrollments
		SET
			status = $2,
			notes = $3,
			updated_at = $4
		WHERE id = $1
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	result, err := execer.ExecContext(ctx, query,
		enrollment.ID,
		enrollment.Status,
		repository.ToNullString(enrollment.Notes),
		enrollment.UpdatedAt,
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

// Delete deletes an enrollment
func (r *EnrollmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM enrollments WHERE id = $1`

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

// ListByStudent retrieves enrollments for a student with pagination
func (r *EnrollmentRepository) ListByStudent(ctx context.Context, studentID uuid.UUID, params repository.PaginationParams) ([]*domain.Enrollment, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, err
	}

	query := `
		SELECT
			id, tenant_id, student_id, class_id, session_id,
			status, enrollment_date, withdrawal_date, withdrawal_reason,
			transferred_to_class_id, transfer_date, transfer_reason,
			suspension_date, suspension_reason, notes,
			created_at, updated_at
		FROM enrollments
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

	query += fmt.Sprintf(" ORDER BY id %s LIMIT $%d", params.SortOrder, argIndex)
	args = append(args, params.Limit+1)

	return r.queryEnrollments(ctx, query, args...)
}

// ListByClass retrieves enrollments for a class with pagination
func (r *EnrollmentRepository) ListByClass(ctx context.Context, classID uuid.UUID, params repository.PaginationParams) ([]*domain.Enrollment, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, err
	}

	query := `
		SELECT
			id, tenant_id, student_id, class_id, session_id,
			status, enrollment_date, withdrawal_date, withdrawal_reason,
			transferred_to_class_id, transfer_date, transfer_reason,
			suspension_date, suspension_reason, notes,
			created_at, updated_at
		FROM enrollments
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

	return r.queryEnrollments(ctx, query, args...)
}

// ListBySession retrieves enrollments for a session with pagination
func (r *EnrollmentRepository) ListBySession(ctx context.Context, sessionID uuid.UUID, params repository.PaginationParams) ([]*domain.Enrollment, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, err
	}

	query := `
		SELECT
			id, tenant_id, student_id, class_id, session_id,
			status, enrollment_date, withdrawal_date, withdrawal_reason,
			transferred_to_class_id, transfer_date, transfer_reason,
			suspension_date, suspension_reason, notes,
			created_at, updated_at
		FROM enrollments
		WHERE session_id = $1
	`

	args := []interface{}{sessionID}
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

	return r.queryEnrollments(ctx, query, args...)
}

// queryEnrollments executes a query and returns a list of enrollments
func (r *EnrollmentRepository) queryEnrollments(ctx context.Context, query string, args ...interface{}) ([]*domain.Enrollment, error) {
	rows, err := r.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, repository.ParseError(err)
	}
	defer rows.Close()

	enrollments := make([]*domain.Enrollment, 0)
	for rows.Next() {
		enrollment, err := r.scanEnrollmentFromRows(rows)
		if err != nil {
			return nil, err
		}
		enrollments = append(enrollments, enrollment)
	}

	if err := rows.Err(); err != nil {
		return nil, repository.ParseError(err)
	}

	return enrollments, nil
}

// scanEnrollmentFromRows scans an enrollment from a Rows object
func (r *EnrollmentRepository) scanEnrollmentFromRows(rows *sql.Rows) (*domain.Enrollment, error) {
	var enrollment domain.Enrollment
	var withdrawalDate, transferDate, suspensionDate sql.NullTime
	var withdrawalReason, transferReason, suspensionReason, notes sql.NullString
	var transferredToClassID sql.NullString

	err := rows.Scan(
		&enrollment.ID,
		&enrollment.TenantID,
		&enrollment.StudentID,
		&enrollment.ClassID,
		&enrollment.SessionID,
		&enrollment.Status,
		&enrollment.EnrollmentDate,
		&withdrawalDate,
		&withdrawalReason,
		&transferredToClassID,
		&transferDate,
		&transferReason,
		&suspensionDate,
		&suspensionReason,
		&notes,
		&enrollment.CreatedAt,
		&enrollment.UpdatedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	if withdrawalDate.Valid {
		enrollment.WithdrawalDate = &withdrawalDate.Time
	}
	if withdrawalReason.Valid {
		enrollment.WithdrawalReason = &withdrawalReason.String
	}
	if transferredToClassID.Valid {
		classID, _ := uuid.Parse(transferredToClassID.String)
		enrollment.TransferredToClassID = &classID
	}
	if transferDate.Valid {
		enrollment.TransferDate = &transferDate.Time
	}
	if transferReason.Valid {
		enrollment.TransferReason = &transferReason.String
	}
	if suspensionDate.Valid {
		enrollment.SuspensionDate = &suspensionDate.Time
	}
	if suspensionReason.Valid {
		enrollment.SuspensionReason = &suspensionReason.String
	}
	if notes.Valid {
		enrollment.Notes = &notes.String
	}

	return &enrollment, nil
}

// Transfer transfers a student to a different class (within same session)
// Must be called within a transaction
func (r *EnrollmentRepository) Transfer(ctx context.Context, enrollmentID uuid.UUID, newClassID uuid.UUID, reason string, tx *sql.Tx) error {
	if tx == nil {
		return fmt.Errorf("transfer must be called within a transaction")
	}

	query := `
		UPDATE enrollments
		SET
			status = $2,
			transferred_to_class_id = $3,
			transfer_date = $4,
			transfer_reason = $5,
			updated_at = $6
		WHERE id = $1
	`

	now := time.Now()
	result, err := tx.ExecContext(ctx, query,
		enrollmentID,
		domain.EnrollmentStatusTransferred,
		newClassID,
		now,
		reason,
		now,
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

// Withdraw withdraws a student from a class
func (r *EnrollmentRepository) Withdraw(ctx context.Context, enrollmentID uuid.UUID, reason string, tx *sql.Tx) error {
	query := `
		UPDATE enrollments
		SET
			status = $2,
			withdrawal_date = $3,
			withdrawal_reason = $4,
			updated_at = $5
		WHERE id = $1
	`

	now := time.Now()
	execer := repository.GetExecer(r.GetDB(), tx)
	result, err := execer.ExecContext(ctx, query,
		enrollmentID,
		domain.EnrollmentStatusWithdrawn,
		now,
		reason,
		now,
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

// Suspend suspends a student enrollment
func (r *EnrollmentRepository) Suspend(ctx context.Context, enrollmentID uuid.UUID, reason string, tx *sql.Tx) error {
	query := `
		UPDATE enrollments
		SET
			status = $2,
			suspension_date = $3,
			suspension_reason = $4,
			updated_at = $5
		WHERE id = $1
	`

	now := time.Now()
	execer := repository.GetExecer(r.GetDB(), tx)
	result, err := execer.ExecContext(ctx, query,
		enrollmentID,
		domain.EnrollmentStatusSuspended,
		now,
		reason,
		now,
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

// Reactivate reactivates a suspended enrollment
func (r *EnrollmentRepository) Reactivate(ctx context.Context, enrollmentID uuid.UUID, tx *sql.Tx) error {
	query := `
		UPDATE enrollments
		SET
			status = $2,
			suspension_date = NULL,
			suspension_reason = NULL,
			updated_at = $3
		WHERE id = $1
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	result, err := execer.ExecContext(ctx, query,
		enrollmentID,
		domain.EnrollmentStatusActive,
		time.Now(),
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

// ValidateTenantAccess validates that an enrollment belongs to a tenant
func (r *EnrollmentRepository) ValidateTenantAccess(ctx context.Context, tenantID uuid.UUID, enrollmentID uuid.UUID) error {
	query := `SELECT 1 FROM enrollments WHERE id = $1 AND tenant_id = $2`

	var exists int
	err := r.GetDB().QueryRowContext(ctx, query, enrollmentID, tenantID).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return repository.ErrNotFound
		}
		return repository.ParseError(err)
	}

	return nil
}
