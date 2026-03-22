package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"arc-lms/internal/domain"
	"arc-lms/internal/repository"
)

// SwapRequestRepository handles database operations for period swap requests
type SwapRequestRepository struct {
	*repository.BaseRepository
}

// NewSwapRequestRepository creates a new swap request repository
func NewSwapRequestRepository(db *sql.DB) *SwapRequestRepository {
	return &SwapRequestRepository{
		BaseRepository: repository.NewBaseRepository(db),
	}
}

// Create creates a new swap request
func (r *SwapRequestRepository) Create(ctx context.Context, request *domain.SwapRequest, tx *sql.Tx) error {
	query := `
		INSERT INTO period_swap_requests (
			id, tenant_id, requesting_period_id, target_period_id,
			requesting_tutor_id, target_tutor_id, status, reason,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	_, err := execer.ExecContext(ctx, query,
		request.ID,
		request.TenantID,
		request.RequestingPeriodID,
		request.TargetPeriodID,
		request.RequestingTutorID,
		request.TargetTutorID,
		request.Status,
		repository.ToNullString(request.Reason),
		request.CreatedAt,
		request.UpdatedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// Get retrieves a swap request by ID
func (r *SwapRequestRepository) Get(ctx context.Context, id uuid.UUID) (*domain.SwapRequest, error) {
	query := `
		SELECT
			id, tenant_id, requesting_period_id, target_period_id,
			requesting_tutor_id, target_tutor_id, status, reason,
			rejection_reason, escalation_reason, admin_override_reason,
			admin_override_by, created_at, updated_at, responded_at, escalated_at
		FROM period_swap_requests
		WHERE id = $1
	`

	return r.scanSwapRequest(r.GetDB().QueryRowContext(ctx, query, id))
}

// scanSwapRequest scans a swap request from a database row
func (r *SwapRequestRepository) scanSwapRequest(row *sql.Row) (*domain.SwapRequest, error) {
	var request domain.SwapRequest
	var reason, rejectionReason, escalationReason, adminOverrideReason sql.NullString
	var adminOverrideBy sql.NullString
	var respondedAt, escalatedAt sql.NullTime

	err := row.Scan(
		&request.ID,
		&request.TenantID,
		&request.RequestingPeriodID,
		&request.TargetPeriodID,
		&request.RequestingTutorID,
		&request.TargetTutorID,
		&request.Status,
		&reason,
		&rejectionReason,
		&escalationReason,
		&adminOverrideReason,
		&adminOverrideBy,
		&request.CreatedAt,
		&request.UpdatedAt,
		&respondedAt,
		&escalatedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	request.Reason = repository.FromNullString(reason)
	request.RejectionReason = repository.FromNullString(rejectionReason)
	request.EscalationReason = repository.FromNullString(escalationReason)
	request.AdminOverrideReason = repository.FromNullString(adminOverrideReason)

	if adminOverrideBy.Valid {
		id, _ := uuid.Parse(adminOverrideBy.String)
		request.AdminOverrideBy = &id
	}
	if respondedAt.Valid {
		request.RespondedAt = &respondedAt.Time
	}
	if escalatedAt.Valid {
		request.EscalatedAt = &escalatedAt.Time
	}

	return &request, nil
}

// Update updates an existing swap request
func (r *SwapRequestRepository) Update(ctx context.Context, request *domain.SwapRequest, tx *sql.Tx) error {
	query := `
		UPDATE period_swap_requests
		SET
			status = $2,
			rejection_reason = $3,
			escalation_reason = $4,
			admin_override_reason = $5,
			admin_override_by = $6,
			responded_at = $7,
			escalated_at = $8,
			updated_at = $9
		WHERE id = $1
	`

	var adminOverrideBy sql.NullString
	if request.AdminOverrideBy != nil {
		adminOverrideBy = sql.NullString{String: request.AdminOverrideBy.String(), Valid: true}
	}

	var respondedAt, escalatedAt sql.NullTime
	if request.RespondedAt != nil {
		respondedAt = sql.NullTime{Time: *request.RespondedAt, Valid: true}
	}
	if request.EscalatedAt != nil {
		escalatedAt = sql.NullTime{Time: *request.EscalatedAt, Valid: true}
	}

	execer := repository.GetExecer(r.GetDB(), tx)
	result, err := execer.ExecContext(ctx, query,
		request.ID,
		request.Status,
		repository.ToNullString(request.RejectionReason),
		repository.ToNullString(request.EscalationReason),
		repository.ToNullString(request.AdminOverrideReason),
		adminOverrideBy,
		respondedAt,
		escalatedAt,
		request.UpdatedAt,
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

// Delete deletes a swap request
func (r *SwapRequestRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM period_swap_requests WHERE id = $1`

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

// ListByTutor retrieves swap requests involving a tutor (as requester or target)
func (r *SwapRequestRepository) ListByTutor(ctx context.Context, tutorID uuid.UUID, status *domain.SwapRequestStatus, params repository.PaginationParams) ([]*domain.SwapRequest, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	whereClause := "WHERE (requesting_tutor_id = $1 OR target_tutor_id = $1)"
	args := []interface{}{tutorID}
	argIndex := 2

	if status != nil {
		whereClause += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *status)
		argIndex++
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM period_swap_requests %s", whereClause)
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT
			id, tenant_id, requesting_period_id, target_period_id,
			requesting_tutor_id, target_tutor_id, status, reason,
			rejection_reason, escalation_reason, admin_override_reason,
			admin_override_by, created_at, updated_at, responded_at, escalated_at
		FROM period_swap_requests
		%s ORDER BY created_at %s LIMIT $%d OFFSET $%d`,
		whereClause, params.SortOrder, argIndex, argIndex+1)
	args = append(args, params.Limit, params.Offset())

	requests, err := r.querySwapRequests(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}

	return requests, total, nil
}

// ListPendingForTutor retrieves pending swap requests where the tutor is the target
func (r *SwapRequestRepository) ListPendingForTutor(ctx context.Context, tutorID uuid.UUID, params repository.PaginationParams) ([]*domain.SwapRequest, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	whereClause := "WHERE target_tutor_id = $1 AND status = $2"
	args := []interface{}{tutorID, domain.SwapRequestStatusPending}
	argIndex := 3

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM period_swap_requests %s", whereClause)
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT
			id, tenant_id, requesting_period_id, target_period_id,
			requesting_tutor_id, target_tutor_id, status, reason,
			rejection_reason, escalation_reason, admin_override_reason,
			admin_override_by, created_at, updated_at, responded_at, escalated_at
		FROM period_swap_requests
		%s ORDER BY created_at %s LIMIT $%d OFFSET $%d`,
		whereClause, params.SortOrder, argIndex, argIndex+1)
	args = append(args, params.Limit, params.Offset())

	requests, err := r.querySwapRequests(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}

	return requests, total, nil
}

// ListEscalated retrieves escalated swap requests for admin review
func (r *SwapRequestRepository) ListEscalated(ctx context.Context, tenantID uuid.UUID, params repository.PaginationParams) ([]*domain.SwapRequest, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	whereClause := "WHERE tenant_id = $1 AND status = $2"
	args := []interface{}{tenantID, domain.SwapRequestStatusEscalated}
	argIndex := 3

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM period_swap_requests %s", whereClause)
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT
			id, tenant_id, requesting_period_id, target_period_id,
			requesting_tutor_id, target_tutor_id, status, reason,
			rejection_reason, escalation_reason, admin_override_reason,
			admin_override_by, created_at, updated_at, responded_at, escalated_at
		FROM period_swap_requests
		%s ORDER BY escalated_at %s LIMIT $%d OFFSET $%d`,
		whereClause, params.SortOrder, argIndex, argIndex+1)
	args = append(args, params.Limit, params.Offset())

	requests, err := r.querySwapRequests(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}

	return requests, total, nil
}

// ListByTenant retrieves all swap requests for a tenant
func (r *SwapRequestRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID, status *domain.SwapRequestStatus, params repository.PaginationParams) ([]*domain.SwapRequest, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	whereClause := "WHERE tenant_id = $1"
	args := []interface{}{tenantID}
	argIndex := 2

	if status != nil {
		whereClause += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *status)
		argIndex++
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM period_swap_requests %s", whereClause)
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT
			id, tenant_id, requesting_period_id, target_period_id,
			requesting_tutor_id, target_tutor_id, status, reason,
			rejection_reason, escalation_reason, admin_override_reason,
			admin_override_by, created_at, updated_at, responded_at, escalated_at
		FROM period_swap_requests
		%s ORDER BY created_at %s LIMIT $%d OFFSET $%d`,
		whereClause, params.SortOrder, argIndex, argIndex+1)
	args = append(args, params.Limit, params.Offset())

	requests, err := r.querySwapRequests(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}

	return requests, total, nil
}

// CancelPendingByTimetable cancels all pending swap requests for periods in a timetable
func (r *SwapRequestRepository) CancelPendingByTimetable(ctx context.Context, timetableID uuid.UUID, tx *sql.Tx) error {
	query := `
		UPDATE period_swap_requests
		SET status = $1, updated_at = NOW()
		WHERE status = $2
			AND (
				requesting_period_id IN (SELECT id FROM periods WHERE timetable_id = $3)
				OR target_period_id IN (SELECT id FROM periods WHERE timetable_id = $3)
			)
	`

	execer := repository.GetExecer(r.GetDB(), tx)
	_, err := execer.ExecContext(ctx, query, domain.SwapRequestStatusCancelled, domain.SwapRequestStatusPending, timetableID)
	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// querySwapRequests executes a query and returns a list of swap requests
func (r *SwapRequestRepository) querySwapRequests(ctx context.Context, query string, args ...interface{}) ([]*domain.SwapRequest, error) {
	rows, err := r.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, repository.ParseError(err)
	}
	defer rows.Close()

	requests := make([]*domain.SwapRequest, 0)
	for rows.Next() {
		var request domain.SwapRequest
		var reason, rejectionReason, escalationReason, adminOverrideReason sql.NullString
		var adminOverrideBy sql.NullString
		var respondedAt, escalatedAt sql.NullTime

		err := rows.Scan(
			&request.ID,
			&request.TenantID,
			&request.RequestingPeriodID,
			&request.TargetPeriodID,
			&request.RequestingTutorID,
			&request.TargetTutorID,
			&request.Status,
			&reason,
			&rejectionReason,
			&escalationReason,
			&adminOverrideReason,
			&adminOverrideBy,
			&request.CreatedAt,
			&request.UpdatedAt,
			&respondedAt,
			&escalatedAt,
		)

		if err != nil {
			return nil, repository.ParseError(err)
		}

		request.Reason = repository.FromNullString(reason)
		request.RejectionReason = repository.FromNullString(rejectionReason)
		request.EscalationReason = repository.FromNullString(escalationReason)
		request.AdminOverrideReason = repository.FromNullString(adminOverrideReason)

		if adminOverrideBy.Valid {
			id, _ := uuid.Parse(adminOverrideBy.String)
			request.AdminOverrideBy = &id
		}
		if respondedAt.Valid {
			request.RespondedAt = &respondedAt.Time
		}
		if escalatedAt.Valid {
			request.EscalatedAt = &escalatedAt.Time
		}

		requests = append(requests, &request)
	}

	if err := rows.Err(); err != nil {
		return nil, repository.ParseError(err)
	}

	return requests, nil
}

// ValidateTenantAccess validates that a swap request belongs to a tenant
func (r *SwapRequestRepository) ValidateTenantAccess(ctx context.Context, tenantID uuid.UUID, requestID uuid.UUID) error {
	query := `SELECT 1 FROM period_swap_requests WHERE id = $1 AND tenant_id = $2`

	var exists int
	err := r.GetDB().QueryRowContext(ctx, query, requestID, tenantID).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return repository.ErrNotFound
		}
		return repository.ParseError(err)
	}

	return nil
}

// HasPendingRequest checks if there's already a pending swap request between two periods
func (r *SwapRequestRepository) HasPendingRequest(ctx context.Context, period1ID, period2ID uuid.UUID) (bool, error) {
	query := `
		SELECT 1 FROM period_swap_requests
		WHERE status = $1
			AND (
				(requesting_period_id = $2 AND target_period_id = $3)
				OR (requesting_period_id = $3 AND target_period_id = $2)
			)
		LIMIT 1
	`

	var exists int
	err := r.GetDB().QueryRowContext(ctx, query, domain.SwapRequestStatusPending, period1ID, period2ID).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, repository.ParseError(err)
	}

	return true, nil
}
