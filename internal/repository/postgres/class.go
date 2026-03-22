package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"arc-lms/internal/domain"
	"arc-lms/internal/repository"
)

// ClassRepository handles database operations for classes
type ClassRepository struct {
	*repository.BaseRepository
}

// NewClassRepository creates a new class repository
func NewClassRepository(db *sql.DB) *ClassRepository {
	return &ClassRepository{
		BaseRepository: repository.NewBaseRepository(db),
	}
}

// Create creates a new class
func (r *ClassRepository) Create(ctx context.Context, class *domain.Class, tx *sql.Tx) error {
	query := `
		INSERT INTO classes (
			id, tenant_id, session_id, name, arm, level,
			capacity, status, description, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	var capacity sql.NullInt32
	if class.Capacity != nil {
		capacity = sql.NullInt32{Int32: int32(*class.Capacity), Valid: true}
	}

	execer := repository.GetExecer(r.GetDB(), tx)
	_, err := execer.ExecContext(ctx, query,
		class.ID,
		class.TenantID,
		class.SessionID,
		class.Name,
		class.Arm,
		class.Level,
		capacity,
		class.Status,
		repository.ToNullString(class.Description),
		class.CreatedAt,
		class.UpdatedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// Get retrieves a class by ID
func (r *ClassRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Class, error) {
	query := `
		SELECT
			id, tenant_id, session_id, name, arm, level,
			capacity, status, description, created_at, updated_at
		FROM classes
		WHERE id = $1
	`

	var class domain.Class
	var capacity sql.NullInt32
	var description sql.NullString

	err := r.GetDB().QueryRowContext(ctx, query, id).Scan(
		&class.ID,
		&class.TenantID,
		&class.SessionID,
		&class.Name,
		&class.Arm,
		&class.Level,
		&capacity,
		&class.Status,
		&description,
		&class.CreatedAt,
		&class.UpdatedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	if capacity.Valid {
		cap := int(capacity.Int32)
		class.Capacity = &cap
	}
	class.Description = repository.FromNullString(description)

	return &class, nil
}

// Update updates an existing class
func (r *ClassRepository) Update(ctx context.Context, class *domain.Class, tx *sql.Tx) error {
	query := `
		UPDATE classes
		SET
			name = $2,
			arm = $3,
			level = $4,
			capacity = $5,
			status = $6,
			description = $7,
			updated_at = $8
		WHERE id = $1
	`

	var capacity sql.NullInt32
	if class.Capacity != nil {
		capacity = sql.NullInt32{Int32: int32(*class.Capacity), Valid: true}
	}

	execer := repository.GetExecer(r.GetDB(), tx)
	result, err := execer.ExecContext(ctx, query,
		class.ID,
		class.Name,
		class.Arm,
		class.Level,
		capacity,
		class.Status,
		repository.ToNullString(class.Description),
		class.UpdatedAt,
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

// Delete deletes a class
func (r *ClassRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM classes WHERE id = $1`

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

// ListBySession retrieves classes for a session with pagination
func (r *ClassRepository) ListBySession(ctx context.Context, sessionID uuid.UUID, params repository.PaginationParams) ([]*domain.Class, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM classes WHERE session_id = $1"
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, sessionID).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT
			id, tenant_id, session_id, name, arm, level,
			capacity, status, description, created_at, updated_at
		FROM classes
		WHERE session_id = $1
		ORDER BY id %s
		LIMIT $2 OFFSET $3
	`, params.SortOrder)

	rows, err := r.GetDB().QueryContext(ctx, query, sessionID, params.Limit, params.Offset())
	if err != nil {
		return nil, 0, repository.ParseError(err)
	}
	defer rows.Close()

	classes := make([]*domain.Class, 0)
	for rows.Next() {
		var class domain.Class
		var capacity sql.NullInt32
		var description sql.NullString

		err := rows.Scan(
			&class.ID,
			&class.TenantID,
			&class.SessionID,
			&class.Name,
			&class.Arm,
			&class.Level,
			&capacity,
			&class.Status,
			&description,
			&class.CreatedAt,
			&class.UpdatedAt,
		)

		if err != nil {
			return nil, 0, repository.ParseError(err)
		}

		if capacity.Valid {
			cap := int(capacity.Int32)
			class.Capacity = &cap
		}
		class.Description = repository.FromNullString(description)

		classes = append(classes, &class)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	return classes, total, nil
}

// ListByTenant retrieves classes for a tenant with pagination
func (r *ClassRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, params repository.PaginationParams) ([]*domain.Class, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM classes WHERE tenant_id = $1"
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT
			id, tenant_id, session_id, name, arm, level,
			capacity, status, description, created_at, updated_at
		FROM classes
		WHERE tenant_id = $1
		ORDER BY id %s
		LIMIT $2 OFFSET $3
	`, params.SortOrder)

	rows, err := r.GetDB().QueryContext(ctx, query, tenantID, params.Limit, params.Offset())
	if err != nil {
		return nil, 0, repository.ParseError(err)
	}
	defer rows.Close()

	classes := make([]*domain.Class, 0)
	for rows.Next() {
		var class domain.Class
		var capacity sql.NullInt32
		var description sql.NullString

		err := rows.Scan(
			&class.ID,
			&class.TenantID,
			&class.SessionID,
			&class.Name,
			&class.Arm,
			&class.Level,
			&capacity,
			&class.Status,
			&description,
			&class.CreatedAt,
			&class.UpdatedAt,
		)

		if err != nil {
			return nil, 0, repository.ParseError(err)
		}

		if capacity.Valid {
			cap := int(capacity.Int32)
			class.Capacity = &cap
		}
		class.Description = repository.FromNullString(description)

		classes = append(classes, &class)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	return classes, total, nil
}

// ValidateTenantAccess validates that a class belongs to a tenant
func (r *ClassRepository) ValidateTenantAccess(ctx context.Context, tenantID uuid.UUID, classID uuid.UUID) error {
	query := `SELECT 1 FROM classes WHERE id = $1 AND tenant_id = $2`

	var exists int
	err := r.GetDB().QueryRowContext(ctx, query, classID, tenantID).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return repository.ErrNotFound
		}
		return repository.ParseError(err)
	}

	return nil
}
