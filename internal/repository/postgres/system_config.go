package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"arc-lms/internal/domain"
	"arc-lms/internal/repository"
)

// SystemConfigRepository handles database operations for system configs
type SystemConfigRepository struct {
	*repository.BaseRepository
}

// NewSystemConfigRepository creates a new system config repository
func NewSystemConfigRepository(db *sql.DB) *SystemConfigRepository {
	return &SystemConfigRepository{
		BaseRepository: repository.NewBaseRepository(db),
	}
}

// Create creates a new system config
func (r *SystemConfigRepository) Create(ctx context.Context, config *domain.SystemConfig) error {
	query := `
		INSERT INTO system_configs (
			id, key, value, description, category, is_sensitive,
			created_by, updated_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.GetDB().ExecContext(ctx, query,
		config.ID,
		config.Key,
		config.Value,
		repository.ToNullString(config.Description),
		config.Category,
		config.IsSensitive,
		repository.ToNullUUID(config.CreatedBy),
		repository.ToNullUUID(config.UpdatedBy),
		config.CreatedAt,
		config.UpdatedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// GetByID retrieves a system config by ID
func (r *SystemConfigRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.SystemConfig, error) {
	query := `
		SELECT
			id, key, value, description, category, is_sensitive,
			created_by, updated_by, created_at, updated_at
		FROM system_configs
		WHERE id = $1
	`

	var config domain.SystemConfig
	var description, createdBy, updatedBy sql.NullString

	err := r.GetDB().QueryRowContext(ctx, query, id).Scan(
		&config.ID,
		&config.Key,
		&config.Value,
		&description,
		&config.Category,
		&config.IsSensitive,
		&createdBy,
		&updatedBy,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	config.Description = repository.FromNullString(description)
	config.CreatedBy = repository.FromNullUUID(createdBy)
	config.UpdatedBy = repository.FromNullUUID(updatedBy)

	return &config, nil
}

// GetByKey retrieves a system config by key
func (r *SystemConfigRepository) GetByKey(ctx context.Context, key string) (*domain.SystemConfig, error) {
	query := `
		SELECT
			id, key, value, description, category, is_sensitive,
			created_by, updated_by, created_at, updated_at
		FROM system_configs
		WHERE key = $1
	`

	var config domain.SystemConfig
	var description, createdBy, updatedBy sql.NullString

	err := r.GetDB().QueryRowContext(ctx, query, key).Scan(
		&config.ID,
		&config.Key,
		&config.Value,
		&description,
		&config.Category,
		&config.IsSensitive,
		&createdBy,
		&updatedBy,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	config.Description = repository.FromNullString(description)
	config.CreatedBy = repository.FromNullUUID(createdBy)
	config.UpdatedBy = repository.FromNullUUID(updatedBy)

	return &config, nil
}

// Update updates an existing system config
func (r *SystemConfigRepository) Update(ctx context.Context, config *domain.SystemConfig) error {
	query := `
		UPDATE system_configs
		SET
			value = $2,
			description = $3,
			category = $4,
			is_sensitive = $5,
			updated_by = $6,
			updated_at = $7
		WHERE id = $1
	`

	result, err := r.GetDB().ExecContext(ctx, query,
		config.ID,
		config.Value,
		repository.ToNullString(config.Description),
		config.Category,
		config.IsSensitive,
		repository.ToNullUUID(config.UpdatedBy),
		config.UpdatedAt,
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

// Delete deletes a system config
func (r *SystemConfigRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM system_configs WHERE id = $1`

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

// List retrieves all system configs with optional category filter
func (r *SystemConfigRepository) List(ctx context.Context, category *domain.SystemConfigCategory, params repository.PaginationParams) ([]*domain.SystemConfig, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, err
	}

	query := `
		SELECT
			id, key, value, description, category, is_sensitive,
			created_by, updated_by, created_at, updated_at
		FROM system_configs
		WHERE 1=1
	`

	args := []interface{}{}
	argIndex := 1

	if category != nil {
		query += fmt.Sprintf(" AND category = $%d", argIndex)
		args = append(args, *category)
		argIndex++
	}

	if params.Cursor != nil {
		if params.SortOrder == "DESC" {
			query += fmt.Sprintf(" AND key < $%d", argIndex)
		} else {
			query += fmt.Sprintf(" AND key > $%d", argIndex)
		}
		args = append(args, *params.Cursor)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY key %s LIMIT $%d", params.SortOrder, argIndex)
	args = append(args, params.Limit+1)

	rows, err := r.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, repository.ParseError(err)
	}
	defer rows.Close()

	configs := make([]*domain.SystemConfig, 0)
	for rows.Next() {
		var config domain.SystemConfig
		var description, createdBy, updatedBy sql.NullString

		err := rows.Scan(
			&config.ID,
			&config.Key,
			&config.Value,
			&description,
			&config.Category,
			&config.IsSensitive,
			&createdBy,
			&updatedBy,
			&config.CreatedAt,
			&config.UpdatedAt,
		)

		if err != nil {
			return nil, repository.ParseError(err)
		}

		config.Description = repository.FromNullString(description)
		config.CreatedBy = repository.FromNullUUID(createdBy)
		config.UpdatedBy = repository.FromNullUUID(updatedBy)

		configs = append(configs, &config)
	}

	if err := rows.Err(); err != nil {
		return nil, repository.ParseError(err)
	}

	return configs, nil
}

// ListByCategory retrieves all system configs for a specific category
func (r *SystemConfigRepository) ListByCategory(ctx context.Context, category domain.SystemConfigCategory) ([]*domain.SystemConfig, error) {
	query := `
		SELECT
			id, key, value, description, category, is_sensitive,
			created_by, updated_by, created_at, updated_at
		FROM system_configs
		WHERE category = $1
		ORDER BY key ASC
	`

	rows, err := r.GetDB().QueryContext(ctx, query, category)
	if err != nil {
		return nil, repository.ParseError(err)
	}
	defer rows.Close()

	configs := make([]*domain.SystemConfig, 0)
	for rows.Next() {
		var config domain.SystemConfig
		var description, createdBy, updatedBy sql.NullString

		err := rows.Scan(
			&config.ID,
			&config.Key,
			&config.Value,
			&description,
			&config.Category,
			&config.IsSensitive,
			&createdBy,
			&updatedBy,
			&config.CreatedAt,
			&config.UpdatedAt,
		)

		if err != nil {
			return nil, repository.ParseError(err)
		}

		config.Description = repository.FromNullString(description)
		config.CreatedBy = repository.FromNullUUID(createdBy)
		config.UpdatedBy = repository.FromNullUUID(updatedBy)

		configs = append(configs, &config)
	}

	if err := rows.Err(); err != nil {
		return nil, repository.ParseError(err)
	}

	return configs, nil
}

// BulkUpdate updates multiple configs at once
func (r *SystemConfigRepository) BulkUpdate(ctx context.Context, configs []*domain.SystemConfig) error {
	tx, err := r.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		UPDATE system_configs
		SET
			value = $2,
			updated_by = $3,
			updated_at = $4
		WHERE key = $1
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, config := range configs {
		_, err := stmt.ExecContext(ctx,
			config.Key,
			config.Value,
			repository.ToNullUUID(config.UpdatedBy),
			config.UpdatedAt,
		)
		if err != nil {
			return repository.ParseError(err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetCategories returns all distinct categories
func (r *SystemConfigRepository) GetCategories(ctx context.Context) ([]domain.SystemConfigCategory, error) {
	query := `SELECT DISTINCT category FROM system_configs ORDER BY category`

	rows, err := r.GetDB().QueryContext(ctx, query)
	if err != nil {
		return nil, repository.ParseError(err)
	}
	defer rows.Close()

	categories := make([]domain.SystemConfigCategory, 0)
	for rows.Next() {
		var category domain.SystemConfigCategory
		if err := rows.Scan(&category); err != nil {
			return nil, repository.ParseError(err)
		}
		categories = append(categories, category)
	}

	if err := rows.Err(); err != nil {
		return nil, repository.ParseError(err)
	}

	return categories, nil
}
