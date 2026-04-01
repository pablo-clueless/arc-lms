package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"arc-lms/internal/domain"
	"arc-lms/internal/repository"
)

// GuardianRepository handles database operations for guardian relationships
type GuardianRepository struct {
	*repository.BaseRepository
}

// NewGuardianRepository creates a new guardian repository
func NewGuardianRepository(db *sql.DB) *GuardianRepository {
	return &GuardianRepository{
		BaseRepository: repository.NewBaseRepository(db),
	}
}

// Create creates a new guardian-student relationship
func (r *GuardianRepository) Create(ctx context.Context, guardian *domain.Guardian) error {
	query := `
		INSERT INTO guardians (
			id, tenant_id, guardian_id, student_id, relationship,
			is_primary, status, notes, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.GetDB().ExecContext(ctx, query,
		guardian.ID,
		guardian.TenantID,
		guardian.GuardianID,
		guardian.StudentID,
		guardian.Relationship,
		guardian.IsPrimary,
		guardian.Status,
		repository.ToNullString(guardian.Notes),
		guardian.CreatedAt,
		guardian.UpdatedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// Get retrieves a guardian relationship by ID
func (r *GuardianRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Guardian, error) {
	query := `
		SELECT
			id, tenant_id, guardian_id, student_id, relationship,
			is_primary, status, notes, created_at, updated_at
		FROM guardians
		WHERE id = $1
	`

	return r.scanGuardian(r.GetDB().QueryRowContext(ctx, query, id))
}

// GetByGuardianAndStudent retrieves a specific guardian-student relationship
func (r *GuardianRepository) GetByGuardianAndStudent(ctx context.Context, guardianID, studentID uuid.UUID) (*domain.Guardian, error) {
	query := `
		SELECT
			id, tenant_id, guardian_id, student_id, relationship,
			is_primary, status, notes, created_at, updated_at
		FROM guardians
		WHERE guardian_id = $1 AND student_id = $2
	`

	return r.scanGuardian(r.GetDB().QueryRowContext(ctx, query, guardianID, studentID))
}

// scanGuardian scans a guardian from a database row
func (r *GuardianRepository) scanGuardian(row *sql.Row) (*domain.Guardian, error) {
	var guardian domain.Guardian
	var notes sql.NullString

	err := row.Scan(
		&guardian.ID,
		&guardian.TenantID,
		&guardian.GuardianID,
		&guardian.StudentID,
		&guardian.Relationship,
		&guardian.IsPrimary,
		&guardian.Status,
		&notes,
		&guardian.CreatedAt,
		&guardian.UpdatedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	guardian.Notes = repository.FromNullString(notes)

	return &guardian, nil
}

// Update updates an existing guardian relationship
func (r *GuardianRepository) Update(ctx context.Context, guardian *domain.Guardian) error {
	query := `
		UPDATE guardians
		SET
			relationship = $2,
			is_primary = $3,
			status = $4,
			notes = $5,
			updated_at = $6
		WHERE id = $1
	`

	result, err := r.GetDB().ExecContext(ctx, query,
		guardian.ID,
		guardian.Relationship,
		guardian.IsPrimary,
		guardian.Status,
		repository.ToNullString(guardian.Notes),
		guardian.UpdatedAt,
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

// Delete deletes a guardian relationship
func (r *GuardianRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM guardians WHERE id = $1`

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

// ListByGuardian retrieves all wards for a guardian (parent)
func (r *GuardianRepository) ListByGuardian(ctx context.Context, guardianID uuid.UUID, params repository.PaginationParams) ([]*domain.Guardian, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM guardians WHERE guardian_id = $1 AND status = $2"
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, guardianID, domain.GuardianStatusActive).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := `
		SELECT
			id, tenant_id, guardian_id, student_id, relationship,
			is_primary, status, notes, created_at, updated_at
		FROM guardians
		WHERE guardian_id = $1 AND status = $2
		ORDER BY is_primary DESC, created_at DESC
		LIMIT $3 OFFSET $4
	`

	guardians, err := r.queryGuardians(ctx, query, guardianID, domain.GuardianStatusActive, params.Limit, params.Offset())
	if err != nil {
		return nil, 0, err
	}

	return guardians, total, nil
}

// ListByStudent retrieves all guardians for a student
func (r *GuardianRepository) ListByStudent(ctx context.Context, studentID uuid.UUID, params repository.PaginationParams) ([]*domain.Guardian, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM guardians WHERE student_id = $1 AND status = $2"
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, studentID, domain.GuardianStatusActive).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := `
		SELECT
			id, tenant_id, guardian_id, student_id, relationship,
			is_primary, status, notes, created_at, updated_at
		FROM guardians
		WHERE student_id = $1 AND status = $2
		ORDER BY is_primary DESC, created_at DESC
		LIMIT $3 OFFSET $4
	`

	guardians, err := r.queryGuardians(ctx, query, studentID, domain.GuardianStatusActive, params.Limit, params.Offset())
	if err != nil {
		return nil, 0, err
	}

	return guardians, total, nil
}

// ListByTenant retrieves all guardian relationships for a tenant
func (r *GuardianRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, params repository.PaginationParams) ([]*domain.Guardian, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM guardians WHERE tenant_id = $1"
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := `
		SELECT
			id, tenant_id, guardian_id, student_id, relationship,
			is_primary, status, notes, created_at, updated_at
		FROM guardians
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	guardians, err := r.queryGuardians(ctx, query, tenantID, params.Limit, params.Offset())
	if err != nil {
		return nil, 0, err
	}

	return guardians, total, nil
}

// queryGuardians executes a query and returns a list of guardians
func (r *GuardianRepository) queryGuardians(ctx context.Context, query string, args ...interface{}) ([]*domain.Guardian, error) {
	rows, err := r.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, repository.ParseError(err)
	}
	defer rows.Close()

	guardians := make([]*domain.Guardian, 0)
	for rows.Next() {
		var guardian domain.Guardian
		var notes sql.NullString

		err := rows.Scan(
			&guardian.ID,
			&guardian.TenantID,
			&guardian.GuardianID,
			&guardian.StudentID,
			&guardian.Relationship,
			&guardian.IsPrimary,
			&guardian.Status,
			&notes,
			&guardian.CreatedAt,
			&guardian.UpdatedAt,
		)

		if err != nil {
			return nil, repository.ParseError(err)
		}

		guardian.Notes = repository.FromNullString(notes)
		guardians = append(guardians, &guardian)
	}

	if err := rows.Err(); err != nil {
		return nil, repository.ParseError(err)
	}

	return guardians, nil
}

// ValidateTenantAccess validates that a guardian relationship belongs to a tenant
func (r *GuardianRepository) ValidateTenantAccess(ctx context.Context, tenantID, guardianRelationshipID uuid.UUID) error {
	query := `SELECT 1 FROM guardians WHERE id = $1 AND tenant_id = $2`

	var exists int
	err := r.GetDB().QueryRowContext(ctx, query, guardianRelationshipID, tenantID).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return repository.ErrNotFound
		}
		return repository.ParseError(err)
	}

	return nil
}

// IsGuardianOfStudent checks if a user is a guardian of a specific student
func (r *GuardianRepository) IsGuardianOfStudent(ctx context.Context, guardianID, studentID uuid.UUID) (bool, error) {
	query := `SELECT 1 FROM guardians WHERE guardian_id = $1 AND student_id = $2 AND status = $3`

	var exists int
	err := r.GetDB().QueryRowContext(ctx, query, guardianID, studentID, domain.GuardianStatusActive).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, repository.ParseError(err)
	}

	return true, nil
}

// GetWardIDs returns all student IDs that a guardian has access to
func (r *GuardianRepository) GetWardIDs(ctx context.Context, guardianID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT student_id
		FROM guardians
		WHERE guardian_id = $1 AND status = $2
	`

	rows, err := r.GetDB().QueryContext(ctx, query, guardianID, domain.GuardianStatusActive)
	if err != nil {
		return nil, repository.ParseError(err)
	}
	defer rows.Close()

	var studentIDs []uuid.UUID
	for rows.Next() {
		var studentID uuid.UUID
		if err := rows.Scan(&studentID); err != nil {
			return nil, repository.ParseError(err)
		}
		studentIDs = append(studentIDs, studentID)
	}

	if err := rows.Err(); err != nil {
		return nil, repository.ParseError(err)
	}

	return studentIDs, nil
}
