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

// TermRepository handles database operations for terms
type TermRepository struct {
	*repository.BaseRepository
}

// NewTermRepository creates a new term repository
func NewTermRepository(db *sql.DB) *TermRepository {
	return &TermRepository{
		BaseRepository: repository.NewBaseRepository(db),
	}
}

// Create creates a new term with date overlap validation (BR-002)
// Must be called within a transaction
func (r *TermRepository) Create(ctx context.Context, term *domain.Term, tx *sql.Tx) error {
	if tx == nil {
		return fmt.Errorf("create term must be called within a transaction")
	}

	// First, insert the term
	holidaysJSON, err := json.Marshal(term.Holidays)
	if err != nil {
		return fmt.Errorf("failed to marshal holidays: %w", err)
	}

	nonInstructionalJSON, err := json.Marshal(term.NonInstructionalDays)
	if err != nil {
		return fmt.Errorf("failed to marshal non-instructional days: %w", err)
	}

	query := `
		INSERT INTO terms (
			id, tenant_id, session_id, ordinal, start_date, end_date,
			status, holidays, non_instructional_days, description,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err = tx.ExecContext(ctx, query,
		term.ID,
		term.TenantID,
		term.SessionID,
		term.Ordinal,
		term.StartDate,
		term.EndDate,
		term.Status,
		holidaysJSON,
		nonInstructionalJSON,
		repository.ToNullString(term.Description),
		term.CreatedAt,
		term.UpdatedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	// Insert into term_date_ranges to enforce non-overlapping dates (BR-002)
	dateRangeQuery := `
		INSERT INTO term_date_ranges (id, term_id, tenant_id, date_range)
		VALUES ($1, $2, $3, tstzrange($4, $5, '[)'))
	`

	_, err = tx.ExecContext(ctx, dateRangeQuery,
		uuid.New(),
		term.ID,
		term.TenantID,
		term.StartDate,
		term.EndDate,
	)

	if err != nil {
		// This will fail with exclusion violation if dates overlap
		return repository.ParseError(err)
	}

	return nil
}

// Get retrieves a term by ID
func (r *TermRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Term, error) {
	query := `
		SELECT
			id, tenant_id, session_id, ordinal, start_date, end_date,
			status, holidays, non_instructional_days, description,
			created_at, updated_at, activated_at, completed_at
		FROM terms
		WHERE id = $1
	`

	return r.scanTerm(r.GetDB().QueryRowContext(ctx, query, id))
}

// scanTerm scans a term from a database row
func (r *TermRepository) scanTerm(row *sql.Row) (*domain.Term, error) {
	var term domain.Term
	var holidaysJSON, nonInstructionalJSON []byte
	var description sql.NullString
	var activatedAt, completedAt sql.NullTime

	err := row.Scan(
		&term.ID,
		&term.TenantID,
		&term.SessionID,
		&term.Ordinal,
		&term.StartDate,
		&term.EndDate,
		&term.Status,
		&holidaysJSON,
		&nonInstructionalJSON,
		&description,
		&term.CreatedAt,
		&term.UpdatedAt,
		&activatedAt,
		&completedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	term.Description = repository.FromNullString(description)
	if activatedAt.Valid {
		term.ActivatedAt = &activatedAt.Time
	}
	if completedAt.Valid {
		term.CompletedAt = &completedAt.Time
	}

	if err := json.Unmarshal(holidaysJSON, &term.Holidays); err != nil {
		return nil, fmt.Errorf("failed to unmarshal holidays: %w", err)
	}

	if err := json.Unmarshal(nonInstructionalJSON, &term.NonInstructionalDays); err != nil {
		return nil, fmt.Errorf("failed to unmarshal non-instructional days: %w", err)
	}

	return &term, nil
}

// Update updates an existing term
func (r *TermRepository) Update(ctx context.Context, term *domain.Term, tx *sql.Tx) error {
	holidaysJSON, err := json.Marshal(term.Holidays)
	if err != nil {
		return fmt.Errorf("failed to marshal holidays: %w", err)
	}

	nonInstructionalJSON, err := json.Marshal(term.NonInstructionalDays)
	if err != nil {
		return fmt.Errorf("failed to marshal non-instructional days: %w", err)
	}

	query := `
		UPDATE terms
		SET
			ordinal = $2,
			start_date = $3,
			end_date = $4,
			holidays = $5,
			non_instructional_days = $6,
			description = $7,
			updated_at = $8
		WHERE id = $1
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	result, err := execer.ExecContext(ctx, query,
		term.ID,
		term.Ordinal,
		term.StartDate,
		term.EndDate,
		holidaysJSON,
		nonInstructionalJSON,
		repository.ToNullString(term.Description),
		term.UpdatedAt,
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

	// Update the date range if dates changed
	if tx != nil {
		updateRangeQuery := `
			UPDATE term_date_ranges
			SET date_range = tstzrange($2, $3, '[)')
			WHERE term_id = $1
		`
		_, err = tx.ExecContext(ctx, updateRangeQuery,
			term.ID,
			term.StartDate,
			term.EndDate,
		)
		if err != nil {
			return repository.ParseError(err)
		}
	}

	return nil
}

// Delete deletes a term
func (r *TermRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM terms WHERE id = $1`

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

	// CASCADE will automatically delete from term_date_ranges

	return nil
}

// ListBySession retrieves terms for a session with pagination
func (r *TermRepository) ListBySession(ctx context.Context, sessionID uuid.UUID, params repository.PaginationParams) ([]*domain.Term, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	whereClause := "WHERE session_id = $1"
	args := []interface{}{sessionID}
	argIndex := 2

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM terms %s", whereClause)
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT
			id, tenant_id, session_id, ordinal, start_date, end_date,
			status, holidays, non_instructional_days, description,
			created_at, updated_at, activated_at, completed_at
		FROM terms
		%s ORDER BY id %s LIMIT $%d OFFSET $%d`,
		whereClause, params.SortOrder, argIndex, argIndex+1)
	args = append(args, params.Limit, params.Offset())

	rows, err := r.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, repository.ParseError(err)
	}
	defer rows.Close()

	terms := make([]*domain.Term, 0)
	for rows.Next() {
		term, err := r.scanTermFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		terms = append(terms, term)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	return terms, total, nil
}

// scanTermFromRows scans a term from a Rows object
func (r *TermRepository) scanTermFromRows(rows *sql.Rows) (*domain.Term, error) {
	var term domain.Term
	var holidaysJSON, nonInstructionalJSON []byte
	var description sql.NullString
	var activatedAt, completedAt sql.NullTime

	err := rows.Scan(
		&term.ID,
		&term.TenantID,
		&term.SessionID,
		&term.Ordinal,
		&term.StartDate,
		&term.EndDate,
		&term.Status,
		&holidaysJSON,
		&nonInstructionalJSON,
		&description,
		&term.CreatedAt,
		&term.UpdatedAt,
		&activatedAt,
		&completedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	term.Description = repository.FromNullString(description)
	if activatedAt.Valid {
		term.ActivatedAt = &activatedAt.Time
	}
	if completedAt.Valid {
		term.CompletedAt = &completedAt.Time
	}

	if err := json.Unmarshal(holidaysJSON, &term.Holidays); err != nil {
		return nil, fmt.Errorf("failed to unmarshal holidays: %w", err)
	}

	if err := json.Unmarshal(nonInstructionalJSON, &term.NonInstructionalDays); err != nil {
		return nil, fmt.Errorf("failed to unmarshal non-instructional days: %w", err)
	}

	return &term, nil
}

// Activate activates a term (triggers billing)
func (r *TermRepository) Activate(ctx context.Context, id uuid.UUID, tx *sql.Tx) error {
	query := `
		UPDATE terms
		SET
			status = $2,
			activated_at = $3,
			updated_at = $4
		WHERE id = $1 AND status = $5
	`

	now := time.Now()
	execer := repository.GetExecer(r.GetDB(), tx)
	result, err := execer.ExecContext(ctx, query,
		id,
		domain.TermStatusActive,
		now,
		now,
		domain.TermStatusDraft,
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

// Complete completes a term
func (r *TermRepository) Complete(ctx context.Context, id uuid.UUID, tx *sql.Tx) error {
	query := `
		UPDATE terms
		SET
			status = $2,
			completed_at = $3,
			updated_at = $4
		WHERE id = $1 AND status = $5
	`

	now := time.Now()
	execer := repository.GetExecer(r.GetDB(), tx)
	result, err := execer.ExecContext(ctx, query,
		id,
		domain.TermStatusCompleted,
		now,
		now,
		domain.TermStatusActive,
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

// GetActiveTerm retrieves the active term for a session
func (r *TermRepository) GetActiveTerm(ctx context.Context, sessionID uuid.UUID) (*domain.Term, error) {
	query := `
		SELECT
			id, tenant_id, session_id, ordinal, start_date, end_date,
			status, holidays, non_instructional_days, description,
			created_at, updated_at, activated_at, completed_at
		FROM terms
		WHERE session_id = $1 AND status = $2
	`

	return r.scanTerm(r.GetDB().QueryRowContext(ctx, query, sessionID, domain.TermStatusActive))
}

// GetCurrentActiveTerm retrieves the currently active term for a tenant based on date
func (r *TermRepository) GetCurrentActiveTerm(ctx context.Context, tenantID uuid.UUID) (*domain.Term, error) {
	query := `
		SELECT
			id, tenant_id, session_id, ordinal, start_date, end_date,
			status, holidays, non_instructional_days, description,
			created_at, updated_at, activated_at, completed_at
		FROM terms
		WHERE tenant_id = $1
		  AND status = $2
		  AND start_date <= $3
		  AND end_date >= $3
	`

	now := time.Now()
	return r.scanTerm(r.GetDB().QueryRowContext(ctx, query, tenantID, domain.TermStatusActive, now))
}

// ValidateTenantAccess validates that a term belongs to a tenant
func (r *TermRepository) ValidateTenantAccess(ctx context.Context, tenantID uuid.UUID, termID uuid.UUID) error {
	query := `SELECT 1 FROM terms WHERE id = $1 AND tenant_id = $2`

	var exists int
	err := r.GetDB().QueryRowContext(ctx, query, termID, tenantID).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return repository.ErrNotFound
		}
		return repository.ParseError(err)
	}

	return nil
}
