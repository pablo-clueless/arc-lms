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

// SessionRepository handles database operations for sessions
type SessionRepository struct {
	*repository.BaseRepository
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{
		BaseRepository: repository.NewBaseRepository(db),
	}
}

// Create creates a new session
func (r *SessionRepository) Create(ctx context.Context, session *domain.Session, tx *sql.Tx) error {
	query := `
		INSERT INTO sessions (
			id, tenant_id, label, start_year, end_year,
			status, description, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	_, err := execer.ExecContext(ctx, query,
		session.ID,
		session.TenantID,
		session.Label,
		session.StartYear,
		session.EndYear,
		session.Status,
		repository.ToNullString(session.Description),
		session.CreatedAt,
		session.UpdatedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// Get retrieves a session by ID
func (r *SessionRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Session, error) {
	query := `
		SELECT
			id, tenant_id, label, start_year, end_year,
			status, description, created_at, updated_at, archived_at
		FROM sessions
		WHERE id = $1
	`

	var session domain.Session
	var description sql.NullString
	var archivedAt sql.NullTime

	err := r.GetDB().QueryRowContext(ctx, query, id).Scan(
		&session.ID,
		&session.TenantID,
		&session.Label,
		&session.StartYear,
		&session.EndYear,
		&session.Status,
		&description,
		&session.CreatedAt,
		&session.UpdatedAt,
		&archivedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	session.Description = repository.FromNullString(description)
	if archivedAt.Valid {
		session.ArchivedAt = &archivedAt.Time
	}

	return &session, nil
}

// Update updates an existing session
func (r *SessionRepository) Update(ctx context.Context, session *domain.Session, tx *sql.Tx) error {
	query := `
		UPDATE sessions
		SET
			label = $2,
			start_year = $3,
			end_year = $4,
			description = $5,
			updated_at = $6
		WHERE id = $1
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	result, err := execer.ExecContext(ctx, query,
		session.ID,
		session.Label,
		session.StartYear,
		session.EndYear,
		repository.ToNullString(session.Description),
		session.UpdatedAt,
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

// Delete deletes a session
func (r *SessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM sessions WHERE id = $1`

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

// ListByTenant retrieves sessions for a tenant with pagination
func (r *SessionRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, status *domain.SessionStatus, params repository.PaginationParams) ([]*domain.Session, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, err
	}

	query := `
		SELECT
			id, tenant_id, label, start_year, end_year,
			status, description, created_at, updated_at, archived_at
		FROM sessions
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

	query += fmt.Sprintf(" ORDER BY id %s LIMIT $%d", params.SortOrder, argIndex)
	args = append(args, params.Limit+1)

	rows, err := r.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, repository.ParseError(err)
	}
	defer rows.Close()

	sessions := make([]*domain.Session, 0)
	for rows.Next() {
		var session domain.Session
		var description sql.NullString
		var archivedAt sql.NullTime

		err := rows.Scan(
			&session.ID,
			&session.TenantID,
			&session.Label,
			&session.StartYear,
			&session.EndYear,
			&session.Status,
			&description,
			&session.CreatedAt,
			&session.UpdatedAt,
			&archivedAt,
		)

		if err != nil {
			return nil, repository.ParseError(err)
		}

		session.Description = repository.FromNullString(description)
		if archivedAt.Valid {
			session.ArchivedAt = &archivedAt.Time
		}

		sessions = append(sessions, &session)
	}

	if err := rows.Err(); err != nil {
		return nil, repository.ParseError(err)
	}

	return sessions, nil
}

// Activate activates a session (enforces BR-007: only one active session per tenant)
// This operation must be done within a transaction to handle the constraint check
func (r *SessionRepository) Activate(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, tx *sql.Tx) error {
	if tx == nil {
		return fmt.Errorf("activate must be called within a transaction")
	}

	// Check if there's already an active session for this tenant
	checkQuery := `
		SELECT id FROM sessions
		WHERE tenant_id = $1 AND status = $2 AND id != $3
		FOR UPDATE
	`

	var existingID uuid.UUID
	err := tx.QueryRowContext(ctx, checkQuery, tenantID, domain.SessionStatusActive, id).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		return repository.ParseError(err)
	}

	if err == nil {
		// There's already an active session
		return fmt.Errorf("tenant already has an active session (BR-007): %w", repository.ErrDuplicateKey)
	}

	// Activate the session
	query := `
		UPDATE sessions
		SET
			status = $2,
			updated_at = $3
		WHERE id = $1 AND tenant_id = $4
	`

	result, err := tx.ExecContext(ctx, query,
		id,
		domain.SessionStatusActive,
		time.Now(),
		tenantID,
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

// Archive archives a session
func (r *SessionRepository) Archive(ctx context.Context, id uuid.UUID, tx *sql.Tx) error {
	query := `
		UPDATE sessions
		SET
			status = $2,
			archived_at = $3,
			updated_at = $4
		WHERE id = $1
	`

	now := time.Now()
	execer := repository.GetExecer(r.GetDB(), tx)
	result, err := execer.ExecContext(ctx, query,
		id,
		domain.SessionStatusArchived,
		now,
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

// GetActiveSession retrieves the active session for a tenant
func (r *SessionRepository) GetActiveSession(ctx context.Context, tenantID uuid.UUID) (*domain.Session, error) {
	query := `
		SELECT
			id, tenant_id, label, start_year, end_year,
			status, description, created_at, updated_at, archived_at
		FROM sessions
		WHERE tenant_id = $1 AND status = $2
	`

	var session domain.Session
	var description sql.NullString
	var archivedAt sql.NullTime

	err := r.GetDB().QueryRowContext(ctx, query, tenantID, domain.SessionStatusActive).Scan(
		&session.ID,
		&session.TenantID,
		&session.Label,
		&session.StartYear,
		&session.EndYear,
		&session.Status,
		&description,
		&session.CreatedAt,
		&session.UpdatedAt,
		&archivedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	session.Description = repository.FromNullString(description)
	if archivedAt.Valid {
		session.ArchivedAt = &archivedAt.Time
	}

	return &session, nil
}

// ValidateTenantAccess validates that a session belongs to a tenant
func (r *SessionRepository) ValidateTenantAccess(ctx context.Context, tenantID uuid.UUID, sessionID uuid.UUID) error {
	query := `SELECT 1 FROM sessions WHERE id = $1 AND tenant_id = $2`

	var exists int
	err := r.GetDB().QueryRowContext(ctx, query, sessionID, tenantID).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return repository.ErrNotFound
		}
		return repository.ParseError(err)
	}

	return nil
}
