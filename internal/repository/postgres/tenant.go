package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"arc-lms/internal/domain"
	"arc-lms/internal/repository"
)

// TenantRepository handles database operations for tenants
type TenantRepository struct {
	*repository.BaseRepository
}

// NewTenantRepository creates a new tenant repository
func NewTenantRepository(db *sql.DB) *TenantRepository {
	return &TenantRepository{
		BaseRepository: repository.NewBaseRepository(db),
	}
}

// Create creates a new tenant
func (r *TenantRepository) Create(ctx context.Context, tenant *domain.Tenant) error {
	configJSON, err := json.Marshal(tenant.Configuration)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	billingJSON, err := json.Marshal(tenant.BillingContact)
	if err != nil {
		return fmt.Errorf("failed to marshal billing contact: %w", err)
	}

	query := `
		INSERT INTO tenants (
			id, name, school_type, contact_email, address, logo,
			status, configuration, billing_contact, principal_admin_id,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err = r.GetDB().ExecContext(ctx, query,
		tenant.ID,
		tenant.Name,
		tenant.SchoolType,
		tenant.ContactEmail,
		tenant.Address,
		repository.StringToNullString(tenant.Logo),
		tenant.Status,
		configJSON,
		billingJSON,
		tenant.PrincipalAdminID,
		tenant.CreatedAt,
		tenant.UpdatedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// Get retrieves a tenant by ID
func (r *TenantRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	query := `
		SELECT
			id, name, school_type, contact_email, address, logo,
			status, configuration, billing_contact, suspension_reason,
			principal_admin_id, created_at, updated_at, suspended_at
		FROM tenants
		WHERE id = $1
	`

	var tenant domain.Tenant
	var configJSON, billingJSON []byte
	var logo, suspensionReason sql.NullString
	var suspendedAt sql.NullTime

	err := r.GetDB().QueryRowContext(ctx, query, id).Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.SchoolType,
		&tenant.ContactEmail,
		&tenant.Address,
		&logo,
		&tenant.Status,
		&configJSON,
		&billingJSON,
		&suspensionReason,
		&tenant.PrincipalAdminID,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
		&suspendedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	if logo.Valid {
		tenant.Logo = logo.String
	}
	if suspensionReason.Valid {
		tenant.SuspensionReason = &suspensionReason.String
	}
	if suspendedAt.Valid {
		tenant.SuspendedAt = &suspendedAt.Time
	}

	if err := json.Unmarshal(configJSON, &tenant.Configuration); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	if err := json.Unmarshal(billingJSON, &tenant.BillingContact); err != nil {
		return nil, fmt.Errorf("failed to unmarshal billing contact: %w", err)
	}

	return &tenant, nil
}

// Update updates an existing tenant
func (r *TenantRepository) Update(ctx context.Context, tenant *domain.Tenant) error {
	configJSON, err := json.Marshal(tenant.Configuration)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	billingJSON, err := json.Marshal(tenant.BillingContact)
	if err != nil {
		return fmt.Errorf("failed to marshal billing contact: %w", err)
	}

	query := `
		UPDATE tenants
		SET
			name = $2,
			school_type = $3,
			contact_email = $4,
			address = $5,
			logo = $6,
			configuration = $7,
			billing_contact = $8,
			principal_admin_id = $9,
			updated_at = $10
		WHERE id = $1
	`

	result, err := r.GetDB().ExecContext(ctx, query,
		tenant.ID,
		tenant.Name,
		tenant.SchoolType,
		tenant.ContactEmail,
		tenant.Address,
		repository.StringToNullString(tenant.Logo),
		configJSON,
		billingJSON,
		tenant.PrincipalAdminID,
		tenant.UpdatedAt,
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

// Delete deletes a tenant (CASCADE will delete all related data)
func (r *TenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM tenants WHERE id = $1`

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

// List retrieves tenants with optional status filter and pagination
func (r *TenantRepository) List(ctx context.Context, status *domain.TenantStatus, params repository.PaginationParams) ([]*domain.Tenant, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, err
	}

	query := `
		SELECT
			id, name, school_type, contact_email, address, logo,
			status, configuration, billing_contact, suspension_reason,
			principal_admin_id, created_at, updated_at, suspended_at
		FROM tenants
		WHERE 1=1
	`

	args := []interface{}{}
	argIndex := 1

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
	args = append(args, params.Limit+1) // Fetch one extra to check if there's more

	rows, err := r.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, repository.ParseError(err)
	}
	defer rows.Close()

	tenants := make([]*domain.Tenant, 0)
	for rows.Next() {
		var tenant domain.Tenant
		var configJSON, billingJSON []byte
		var logo, suspensionReason sql.NullString
		var suspendedAt sql.NullTime

		err := rows.Scan(
			&tenant.ID,
			&tenant.Name,
			&tenant.SchoolType,
			&tenant.ContactEmail,
			&tenant.Address,
			&logo,
			&tenant.Status,
			&configJSON,
			&billingJSON,
			&suspensionReason,
			&tenant.PrincipalAdminID,
			&tenant.CreatedAt,
			&tenant.UpdatedAt,
			&suspendedAt,
		)

		if err != nil {
			return nil, repository.ParseError(err)
		}

		if logo.Valid {
			tenant.Logo = logo.String
		}
		if suspensionReason.Valid {
			tenant.SuspensionReason = &suspensionReason.String
		}
		if suspendedAt.Valid {
			tenant.SuspendedAt = &suspendedAt.Time
		}

		if err := json.Unmarshal(configJSON, &tenant.Configuration); err != nil {
			return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
		}

		if err := json.Unmarshal(billingJSON, &tenant.BillingContact); err != nil {
			return nil, fmt.Errorf("failed to unmarshal billing contact: %w", err)
		}

		tenants = append(tenants, &tenant)
	}

	if err := rows.Err(); err != nil {
		return nil, repository.ParseError(err)
	}

	return tenants, nil
}

// Suspend suspends a tenant with a reason
func (r *TenantRepository) Suspend(ctx context.Context, id uuid.UUID, reason string, tx *sql.Tx) error {
	query := `
		UPDATE tenants
		SET
			status = $2,
			suspension_reason = $3,
			suspended_at = $4,
			updated_at = $5
		WHERE id = $1 AND status = $6
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	result, err := execer.ExecContext(ctx, query,
		id,
		domain.TenantStatusSuspended,
		reason,
		sql.NullTime{Time: domain.Tenant{}.UpdatedAt, Valid: true},
		domain.Tenant{}.UpdatedAt,
		domain.TenantStatusActive,
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

// Reactivate reactivates a suspended tenant
func (r *TenantRepository) Reactivate(ctx context.Context, id uuid.UUID, tx *sql.Tx) error {
	query := `
		UPDATE tenants
		SET
			status = $2,
			suspension_reason = NULL,
			suspended_at = NULL,
			updated_at = $3
		WHERE id = $1 AND status = $4
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	result, err := execer.ExecContext(ctx, query,
		id,
		domain.TenantStatusActive,
		domain.Tenant{}.UpdatedAt,
		domain.TenantStatusSuspended,
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
