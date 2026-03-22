package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"arc-lms/internal/domain"
	"arc-lms/internal/repository"

	"github.com/google/uuid"
)

// SubscriptionRepository handles subscription persistence operations
type SubscriptionRepository struct {
	db *sql.DB
}

// NewSubscriptionRepository creates a new subscription repository
func NewSubscriptionRepository(db *sql.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

// Create creates a new subscription
func (r *SubscriptionRepository) Create(ctx context.Context, subscription *domain.Subscription, tx *sql.Tx) error {
	query := `
		INSERT INTO subscriptions (
			id, tenant_id, session_id, status, price_per_student_per_term,
			currency, start_date, end_date, cancelled_at, cancellation_reason,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	execer := repository.GetExecer(r.db, tx)
	_, err := execer.ExecContext(ctx, query,
		subscription.ID,
		subscription.TenantID,
		subscription.SessionID,
		subscription.Status,
		subscription.PricePerStudentPerTerm,
		subscription.Currency,
		subscription.StartDate,
		subscription.EndDate,
		subscription.CancelledAt,
		subscription.CancellationReason,
		subscription.CreatedAt,
		subscription.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	return nil
}

// Get retrieves a subscription by ID
func (r *SubscriptionRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	query := `
		SELECT
			id, tenant_id, session_id, status, price_per_student_per_term,
			currency, start_date, end_date, cancelled_at, cancellation_reason,
			created_at, updated_at
		FROM subscriptions
		WHERE id = $1
	`

	return r.scanSubscription(r.db.QueryRowContext(ctx, query, id))
}

// GetByTenant retrieves the active subscription for a tenant
func (r *SubscriptionRepository) GetByTenant(ctx context.Context, tenantID uuid.UUID) (*domain.Subscription, error) {
	query := `
		SELECT
			id, tenant_id, session_id, status, price_per_student_per_term,
			currency, start_date, end_date, cancelled_at, cancellation_reason,
			created_at, updated_at
		FROM subscriptions
		WHERE tenant_id = $1 AND status != 'CANCELLED'
		ORDER BY created_at DESC
		LIMIT 1
	`

	return r.scanSubscription(r.db.QueryRowContext(ctx, query, tenantID))
}

// GetByTenantAndSession retrieves a subscription for a specific tenant and session
func (r *SubscriptionRepository) GetByTenantAndSession(ctx context.Context, tenantID, sessionID uuid.UUID) (*domain.Subscription, error) {
	query := `
		SELECT
			id, tenant_id, session_id, status, price_per_student_per_term,
			currency, start_date, end_date, cancelled_at, cancellation_reason,
			created_at, updated_at
		FROM subscriptions
		WHERE tenant_id = $1 AND session_id = $2
		LIMIT 1
	`

	return r.scanSubscription(r.db.QueryRowContext(ctx, query, tenantID, sessionID))
}

// Update updates a subscription
func (r *SubscriptionRepository) Update(ctx context.Context, subscription *domain.Subscription, tx *sql.Tx) error {
	query := `
		UPDATE subscriptions SET
			status = $2,
			end_date = $3,
			cancelled_at = $4,
			cancellation_reason = $5,
			updated_at = $6
		WHERE id = $1
	`

	execer := repository.GetExecer(r.db, tx)
	result, err := execer.ExecContext(ctx, query,
		subscription.ID,
		subscription.Status,
		subscription.EndDate,
		subscription.CancelledAt,
		subscription.CancellationReason,
		subscription.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// ListByStatus retrieves subscriptions by status
func (r *SubscriptionRepository) ListByStatus(
	ctx context.Context,
	status *domain.SubscriptionStatus,
	params repository.PaginationParams,
) ([]*domain.Subscription, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if status != nil {
		whereClause += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *status)
		argIndex++
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM subscriptions %s", whereClause)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT
			id, tenant_id, session_id, status, price_per_student_per_term,
			currency, start_date, end_date, cancelled_at, cancellation_reason,
			created_at, updated_at
		FROM subscriptions
		%s ORDER BY created_at %s LIMIT $%d OFFSET $%d`,
		whereClause, params.SortOrder, argIndex, argIndex+1)
	args = append(args, params.Limit, params.Offset())

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list subscriptions: %w", err)
	}
	defer rows.Close()

	subscriptions, err := r.scanSubscriptions(rows)
	if err != nil {
		return nil, 0, err
	}

	return subscriptions, total, nil
}

// ListOverdue retrieves subscriptions that are overdue
func (r *SubscriptionRepository) ListOverdue(ctx context.Context) ([]*domain.Subscription, error) {
	query := `
		SELECT
			id, tenant_id, session_id, status, price_per_student_per_term,
			currency, start_date, end_date, cancelled_at, cancellation_reason,
			created_at, updated_at
		FROM subscriptions
		WHERE status = 'PAYMENT_OVERDUE'
		ORDER BY updated_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list overdue subscriptions: %w", err)
	}
	defer rows.Close()

	return r.scanSubscriptions(rows)
}

// GetSubscriptionStatistics retrieves subscription statistics
func (r *SubscriptionRepository) GetSubscriptionStatistics(ctx context.Context) (*SubscriptionStatistics, error) {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE status = 'ACTIVE') as active_count,
			COUNT(*) FILTER (WHERE status = 'PAYMENT_OVERDUE') as overdue_count,
			COUNT(*) FILTER (WHERE status = 'SUSPENDED') as suspended_count,
			COUNT(*) FILTER (WHERE status = 'CANCELLED') as cancelled_count,
			COUNT(*) as total_count
		FROM subscriptions
	`

	var stats SubscriptionStatistics
	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.ActiveCount,
		&stats.OverdueCount,
		&stats.SuspendedCount,
		&stats.CancelledCount,
		&stats.TotalCount,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get subscription statistics: %w", err)
	}

	return &stats, nil
}

// SubscriptionStatistics represents subscription statistics
type SubscriptionStatistics struct {
	ActiveCount    int `json:"active_count"`
	OverdueCount   int `json:"overdue_count"`
	SuspendedCount int `json:"suspended_count"`
	CancelledCount int `json:"cancelled_count"`
	TotalCount     int `json:"total_count"`
}

// scanSubscription scans a subscription from a row
func (r *SubscriptionRepository) scanSubscription(row *sql.Row) (*domain.Subscription, error) {
	var subscription domain.Subscription
	var endDate, cancelledAt sql.NullTime
	var cancellationReason sql.NullString

	err := row.Scan(
		&subscription.ID,
		&subscription.TenantID,
		&subscription.SessionID,
		&subscription.Status,
		&subscription.PricePerStudentPerTerm,
		&subscription.Currency,
		&subscription.StartDate,
		&endDate,
		&cancelledAt,
		&cancellationReason,
		&subscription.CreatedAt,
		&subscription.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan subscription: %w", err)
	}

	if endDate.Valid {
		subscription.EndDate = &endDate.Time
	}
	if cancelledAt.Valid {
		subscription.CancelledAt = &cancelledAt.Time
	}
	if cancellationReason.Valid {
		subscription.CancellationReason = &cancellationReason.String
	}

	return &subscription, nil
}

// scanSubscriptions scans subscriptions from rows
func (r *SubscriptionRepository) scanSubscriptions(rows *sql.Rows) ([]*domain.Subscription, error) {
	var subscriptions []*domain.Subscription

	for rows.Next() {
		var subscription domain.Subscription
		var endDate, cancelledAt sql.NullTime
		var cancellationReason sql.NullString

		err := rows.Scan(
			&subscription.ID,
			&subscription.TenantID,
			&subscription.SessionID,
			&subscription.Status,
			&subscription.PricePerStudentPerTerm,
			&subscription.Currency,
			&subscription.StartDate,
			&endDate,
			&cancelledAt,
			&cancellationReason,
			&subscription.CreatedAt,
			&subscription.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan subscription: %w", err)
		}

		if endDate.Valid {
			subscription.EndDate = &endDate.Time
		}
		if cancelledAt.Valid {
			subscription.CancelledAt = &cancelledAt.Time
		}
		if cancellationReason.Valid {
			subscription.CancellationReason = &cancellationReason.String
		}

		subscriptions = append(subscriptions, &subscription)
	}

	return subscriptions, nil
}

// MarkAsOverdue marks active subscriptions with overdue invoices
func (r *SubscriptionRepository) MarkAsOverdue(ctx context.Context, tenantID uuid.UUID) error {
	query := `
		UPDATE subscriptions
		SET status = 'PAYMENT_OVERDUE', updated_at = $1
		WHERE tenant_id = $2 AND status = 'ACTIVE'
	`

	_, err := r.db.ExecContext(ctx, query, time.Now(), tenantID)
	return err
}

// Suspend suspends a subscription
func (r *SubscriptionRepository) Suspend(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE subscriptions
		SET status = 'SUSPENDED', updated_at = $1
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

// Reactivate reactivates a subscription
func (r *SubscriptionRepository) Reactivate(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE subscriptions
		SET status = 'ACTIVE', updated_at = $1
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}
