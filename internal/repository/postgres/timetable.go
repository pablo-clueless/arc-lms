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

// TimetableRepository handles database operations for timetables
type TimetableRepository struct {
	*repository.BaseRepository
}

// NewTimetableRepository creates a new timetable repository
func NewTimetableRepository(db *sql.DB) *TimetableRepository {
	return &TimetableRepository{
		BaseRepository: repository.NewBaseRepository(db),
	}
}

// Create creates a new timetable
func (r *TimetableRepository) Create(ctx context.Context, timetable *domain.Timetable, tx *sql.Tx) error {
	query := `
		INSERT INTO timetables (
			id, tenant_id, class_id, term_id, status,
			generated_at, generated_by, generation_version, notes,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	_, err := execer.ExecContext(ctx, query,
		timetable.ID,
		timetable.TenantID,
		timetable.ClassID,
		timetable.TermID,
		timetable.Status,
		timetable.GeneratedAt,
		timetable.GeneratedBy,
		timetable.GenerationVersion,
		repository.ToNullString(timetable.Notes),
		timetable.CreatedAt,
		timetable.UpdatedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// Get retrieves a timetable by ID
func (r *TimetableRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Timetable, error) {
	query := `
		SELECT
			id, tenant_id, class_id, term_id, status,
			generated_at, generated_by, published_at, published_by,
			generation_version, notes, created_at, updated_at, archived_at
		FROM timetables
		WHERE id = $1
	`

	var timetable domain.Timetable
	var publishedAt, archivedAt sql.NullTime
	var publishedBy sql.NullString
	var notes sql.NullString

	err := r.GetDB().QueryRowContext(ctx, query, id).Scan(
		&timetable.ID,
		&timetable.TenantID,
		&timetable.ClassID,
		&timetable.TermID,
		&timetable.Status,
		&timetable.GeneratedAt,
		&timetable.GeneratedBy,
		&publishedAt,
		&publishedBy,
		&timetable.GenerationVersion,
		&notes,
		&timetable.CreatedAt,
		&timetable.UpdatedAt,
		&archivedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	if publishedAt.Valid {
		timetable.PublishedAt = &publishedAt.Time
	}
	if publishedBy.Valid {
		pubBy, _ := uuid.Parse(publishedBy.String)
		timetable.PublishedBy = &pubBy
	}
	if archivedAt.Valid {
		timetable.ArchivedAt = &archivedAt.Time
	}
	timetable.Notes = repository.FromNullString(notes)

	return &timetable, nil
}

// Update updates an existing timetable
func (r *TimetableRepository) Update(ctx context.Context, timetable *domain.Timetable, tx *sql.Tx) error {
	query := `
		UPDATE timetables
		SET
			notes = $2,
			updated_at = $3
		WHERE id = $1
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	result, err := execer.ExecContext(ctx, query,
		timetable.ID,
		repository.ToNullString(timetable.Notes),
		timetable.UpdatedAt,
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

// Delete deletes a timetable
func (r *TimetableRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM timetables WHERE id = $1`

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

// ListByClass retrieves timetables for a class with pagination
func (r *TimetableRepository) ListByClass(ctx context.Context, classID uuid.UUID, params repository.PaginationParams) ([]*domain.Timetable, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	whereClause := "WHERE class_id = $1"
	args := []interface{}{classID}
	argIndex := 2

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM timetables %s", whereClause)
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT
			id, tenant_id, class_id, term_id, status,
			generated_at, generated_by, published_at, published_by,
			generation_version, notes, created_at, updated_at, archived_at
		FROM timetables
		%s ORDER BY id %s LIMIT $%d OFFSET $%d`,
		whereClause, params.SortOrder, argIndex, argIndex+1)
	args = append(args, params.Limit, params.Offset())

	timetables, err := r.queryTimetables(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}

	return timetables, total, nil
}

// ListByTerm retrieves timetables for a term with pagination
func (r *TimetableRepository) ListByTerm(ctx context.Context, termID uuid.UUID, params repository.PaginationParams) ([]*domain.Timetable, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	whereClause := "WHERE term_id = $1"
	args := []interface{}{termID}
	argIndex := 2

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM timetables %s", whereClause)
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT
			id, tenant_id, class_id, term_id, status,
			generated_at, generated_by, published_at, published_by,
			generation_version, notes, created_at, updated_at, archived_at
		FROM timetables
		%s ORDER BY id %s LIMIT $%d OFFSET $%d`,
		whereClause, params.SortOrder, argIndex, argIndex+1)
	args = append(args, params.Limit, params.Offset())

	timetables, err := r.queryTimetables(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}

	return timetables, total, nil
}

// queryTimetables executes a query and returns a list of timetables
func (r *TimetableRepository) queryTimetables(ctx context.Context, query string, args ...interface{}) ([]*domain.Timetable, error) {
	rows, err := r.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, repository.ParseError(err)
	}
	defer rows.Close()

	timetables := make([]*domain.Timetable, 0)
	for rows.Next() {
		var timetable domain.Timetable
		var publishedAt, archivedAt sql.NullTime
		var publishedBy sql.NullString
		var notes sql.NullString

		err := rows.Scan(
			&timetable.ID,
			&timetable.TenantID,
			&timetable.ClassID,
			&timetable.TermID,
			&timetable.Status,
			&timetable.GeneratedAt,
			&timetable.GeneratedBy,
			&publishedAt,
			&publishedBy,
			&timetable.GenerationVersion,
			&notes,
			&timetable.CreatedAt,
			&timetable.UpdatedAt,
			&archivedAt,
		)

		if err != nil {
			return nil, repository.ParseError(err)
		}

		if publishedAt.Valid {
			timetable.PublishedAt = &publishedAt.Time
		}
		if publishedBy.Valid {
			pubBy, _ := uuid.Parse(publishedBy.String)
			timetable.PublishedBy = &pubBy
		}
		if archivedAt.Valid {
			timetable.ArchivedAt = &archivedAt.Time
		}
		timetable.Notes = repository.FromNullString(notes)

		timetables = append(timetables, &timetable)
	}

	if err := rows.Err(); err != nil {
		return nil, repository.ParseError(err)
	}

	return timetables, nil
}

// Publish publishes a timetable
func (r *TimetableRepository) Publish(ctx context.Context, id uuid.UUID, publishedBy uuid.UUID, tx *sql.Tx) error {
	query := `
		UPDATE timetables
		SET
			status = $2,
			published_at = $3,
			published_by = $4,
			updated_at = $5
		WHERE id = $1 AND status = $6
	`

	now := time.Now()
	execer := repository.GetExecer(r.GetDB(), tx)
	result, err := execer.ExecContext(ctx, query,
		id,
		domain.TimetableStatusPublished,
		now,
		publishedBy,
		now,
		domain.TimetableStatusDraft,
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

// GetPublishedTimetable retrieves the published timetable for a class and term
func (r *TimetableRepository) GetPublishedTimetable(ctx context.Context, classID uuid.UUID, termID uuid.UUID) (*domain.Timetable, error) {
	query := `
		SELECT
			id, tenant_id, class_id, term_id, status,
			generated_at, generated_by, published_at, published_by,
			generation_version, notes, created_at, updated_at, archived_at
		FROM timetables
		WHERE class_id = $1 AND term_id = $2 AND status = $3
		ORDER BY generation_version DESC
		LIMIT 1
	`

	var timetable domain.Timetable
	var publishedAt, archivedAt sql.NullTime
	var publishedBy sql.NullString
	var notes sql.NullString

	err := r.GetDB().QueryRowContext(ctx, query, classID, termID, domain.TimetableStatusPublished).Scan(
		&timetable.ID,
		&timetable.TenantID,
		&timetable.ClassID,
		&timetable.TermID,
		&timetable.Status,
		&timetable.GeneratedAt,
		&timetable.GeneratedBy,
		&publishedAt,
		&publishedBy,
		&timetable.GenerationVersion,
		&notes,
		&timetable.CreatedAt,
		&timetable.UpdatedAt,
		&archivedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	if publishedAt.Valid {
		timetable.PublishedAt = &publishedAt.Time
	}
	if publishedBy.Valid {
		pubBy, _ := uuid.Parse(publishedBy.String)
		timetable.PublishedBy = &pubBy
	}
	if archivedAt.Valid {
		timetable.ArchivedAt = &archivedAt.Time
	}
	timetable.Notes = repository.FromNullString(notes)

	return &timetable, nil
}

// ValidateTenantAccess validates that a timetable belongs to a tenant
func (r *TimetableRepository) ValidateTenantAccess(ctx context.Context, tenantID uuid.UUID, timetableID uuid.UUID) error {
	query := `SELECT 1 FROM timetables WHERE id = $1 AND tenant_id = $2`

	var exists int
	err := r.GetDB().QueryRowContext(ctx, query, timetableID, tenantID).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return repository.ErrNotFound
		}
		return repository.ParseError(err)
	}

	return nil
}

// GetByClassAndTerm retrieves a non-archived timetable for a class and term
func (r *TimetableRepository) GetByClassAndTerm(ctx context.Context, classID uuid.UUID, termID uuid.UUID) (*domain.Timetable, error) {
	query := `
		SELECT
			id, tenant_id, class_id, term_id, status,
			generated_at, generated_by, published_at, published_by,
			generation_version, notes, created_at, updated_at, archived_at
		FROM timetables
		WHERE class_id = $1 AND term_id = $2 AND status != $3
		ORDER BY generation_version DESC
		LIMIT 1
	`

	var timetable domain.Timetable
	var publishedAt, archivedAt sql.NullTime
	var publishedBy sql.NullString
	var notes sql.NullString

	err := r.GetDB().QueryRowContext(ctx, query, classID, termID, domain.TimetableStatusArchived).Scan(
		&timetable.ID,
		&timetable.TenantID,
		&timetable.ClassID,
		&timetable.TermID,
		&timetable.Status,
		&timetable.GeneratedAt,
		&timetable.GeneratedBy,
		&publishedAt,
		&publishedBy,
		&timetable.GenerationVersion,
		&notes,
		&timetable.CreatedAt,
		&timetable.UpdatedAt,
		&archivedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	if publishedAt.Valid {
		timetable.PublishedAt = &publishedAt.Time
	}
	if publishedBy.Valid {
		pubBy, _ := uuid.Parse(publishedBy.String)
		timetable.PublishedBy = &pubBy
	}
	if archivedAt.Valid {
		timetable.ArchivedAt = &archivedAt.Time
	}
	timetable.Notes = repository.FromNullString(notes)

	return &timetable, nil
}

// Archive archives a timetable
func (r *TimetableRepository) Archive(ctx context.Context, id uuid.UUID, tx *sql.Tx) error {
	query := `
		UPDATE timetables
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
		domain.TimetableStatusArchived,
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

// ListByTenant retrieves timetables for a tenant with pagination
func (r *TimetableRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, params repository.PaginationParams) ([]*domain.Timetable, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	whereClause := "WHERE tenant_id = $1 AND status != $2"
	args := []interface{}{tenantID, domain.TimetableStatusArchived}
	argIndex := 3

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM timetables %s", whereClause)
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT
			id, tenant_id, class_id, term_id, status,
			generated_at, generated_by, published_at, published_by,
			generation_version, notes, created_at, updated_at, archived_at
		FROM timetables
		%s ORDER BY id %s LIMIT $%d OFFSET $%d`,
		whereClause, params.SortOrder, argIndex, argIndex+1)
	args = append(args, params.Limit, params.Offset())

	timetables, err := r.queryTimetables(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}

	return timetables, total, nil
}
