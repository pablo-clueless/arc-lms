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

// AuditRepository handles database operations for audit logs
type AuditRepository struct {
	*repository.BaseRepository
}

// NewAuditRepository creates a new audit repository
func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{
		BaseRepository: repository.NewBaseRepository(db),
	}
}

// Create creates a new audit log entry
func (r *AuditRepository) Create(ctx context.Context, auditLog *domain.AuditLog) error {
	query := `
		INSERT INTO audit_logs (
			id, tenant_id, actor_user_id, actor_role, action,
			resource_type, resource_id, resource_name, before_state,
			after_state, changes, ip_address, user_agent, metadata,
			is_sensitive, timestamp, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	var beforeStateJSON, afterStateJSON, changesJSON, metadataJSON []byte
	var err error

	if auditLog.BeforeState != nil {
		beforeStateJSON = *auditLog.BeforeState
	}
	if auditLog.AfterState != nil {
		afterStateJSON = *auditLog.AfterState
	}
	if auditLog.Changes != nil {
		changesJSON, err = json.Marshal(auditLog.Changes)
		if err != nil {
			return fmt.Errorf("failed to marshal changes: %w", err)
		}
	}
	if auditLog.Metadata != nil {
		metadataJSON, err = json.Marshal(auditLog.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	_, err = r.GetDB().ExecContext(ctx, query,
		auditLog.ID,
		repository.ToNullUUID(auditLog.TenantID),
		auditLog.ActorUserID,
		auditLog.ActorRole,
		auditLog.Action,
		auditLog.ResourceType,
		auditLog.ResourceID,
		repository.ToNullString(auditLog.ResourceName),
		beforeStateJSON,
		afterStateJSON,
		changesJSON,
		auditLog.IPAddress,
		repository.ToNullString(auditLog.UserAgent),
		metadataJSON,
		auditLog.IsSensitive,
		auditLog.Timestamp,
		auditLog.CreatedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// GetByID retrieves an audit log by ID
func (r *AuditRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.AuditLog, error) {
	query := `
		SELECT
			id, tenant_id, actor_user_id, actor_role, action,
			resource_type, resource_id, resource_name, before_state,
			after_state, changes, ip_address, user_agent, metadata,
			is_sensitive, timestamp, created_at
		FROM audit_logs
		WHERE id = $1
	`

	return r.scanAuditLog(r.GetDB().QueryRowContext(ctx, query, id))
}

// List retrieves audit logs with filters and pagination
func (r *AuditRepository) List(ctx context.Context, filters interface{}, params repository.PaginationParams) ([]*domain.AuditLog, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	// Type assert filters
	auditFilters, _ := filters.(*domain.AuditFilters)

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if auditFilters != nil {
		if auditFilters.TenantID != nil {
			whereClause += fmt.Sprintf(" AND tenant_id = $%d", argIndex)
			args = append(args, *auditFilters.TenantID)
			argIndex++
		}

		if auditFilters.ActorUserID != nil {
			whereClause += fmt.Sprintf(" AND actor_user_id = $%d", argIndex)
			args = append(args, *auditFilters.ActorUserID)
			argIndex++
		}

		if auditFilters.ActorRole != nil {
			whereClause += fmt.Sprintf(" AND actor_role = $%d", argIndex)
			args = append(args, *auditFilters.ActorRole)
			argIndex++
		}

		if auditFilters.Action != nil {
			whereClause += fmt.Sprintf(" AND action = $%d", argIndex)
			args = append(args, *auditFilters.Action)
			argIndex++
		}

		if auditFilters.ResourceType != nil {
			whereClause += fmt.Sprintf(" AND resource_type = $%d", argIndex)
			args = append(args, *auditFilters.ResourceType)
			argIndex++
		}

		if auditFilters.ResourceID != nil {
			whereClause += fmt.Sprintf(" AND resource_id = $%d", argIndex)
			args = append(args, *auditFilters.ResourceID)
			argIndex++
		}

		if auditFilters.IsSensitive != nil {
			whereClause += fmt.Sprintf(" AND is_sensitive = $%d", argIndex)
			args = append(args, *auditFilters.IsSensitive)
			argIndex++
		}

		if auditFilters.StartDate != nil {
			whereClause += fmt.Sprintf(" AND timestamp >= $%d", argIndex)
			args = append(args, *auditFilters.StartDate)
			argIndex++
		}

		if auditFilters.EndDate != nil {
			whereClause += fmt.Sprintf(" AND timestamp <= $%d", argIndex)
			args = append(args, *auditFilters.EndDate)
			argIndex++
		}
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs %s", whereClause)
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT
			id, tenant_id, actor_user_id, actor_role, action,
			resource_type, resource_id, resource_name, before_state,
			after_state, changes, ip_address, user_agent, metadata,
			is_sensitive, timestamp, created_at
		FROM audit_logs
		%s
		ORDER BY timestamp %s, id DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, params.SortOrder, argIndex, argIndex+1)

	args = append(args, params.Limit, params.Offset())

	rows, err := r.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, repository.ParseError(err)
	}
	defer rows.Close()

	logs := make([]*domain.AuditLog, 0)
	for rows.Next() {
		log, err := r.scanAuditLogFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	return logs, total, nil
}

// GetByResource retrieves audit logs for a specific resource
func (r *AuditRepository) GetByResource(ctx context.Context, resourceType domain.AuditResourceType, resourceID uuid.UUID, params repository.PaginationParams) ([]*domain.AuditLog, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM audit_logs WHERE resource_type = $1 AND resource_id = $2"
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, resourceType, resourceID).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT
			id, tenant_id, actor_user_id, actor_role, action,
			resource_type, resource_id, resource_name, before_state,
			after_state, changes, ip_address, user_agent, metadata,
			is_sensitive, timestamp, created_at
		FROM audit_logs
		WHERE resource_type = $1 AND resource_id = $2
		ORDER BY timestamp %s, id DESC
		LIMIT $3 OFFSET $4
	`, params.SortOrder)

	rows, err := r.GetDB().QueryContext(ctx, query, resourceType, resourceID, params.Limit, params.Offset())
	if err != nil {
		return nil, 0, repository.ParseError(err)
	}
	defer rows.Close()

	logs := make([]*domain.AuditLog, 0)
	for rows.Next() {
		log, err := r.scanAuditLogFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	return logs, total, nil
}

// scanAuditLog scans an audit log from a single row
func (r *AuditRepository) scanAuditLog(row *sql.Row) (*domain.AuditLog, error) {
	var log domain.AuditLog
	var tenantID sql.NullString
	var resourceName, userAgent sql.NullString
	var beforeState, afterState, changes, metadata []byte

	err := row.Scan(
		&log.ID,
		&tenantID,
		&log.ActorUserID,
		&log.ActorRole,
		&log.Action,
		&log.ResourceType,
		&log.ResourceID,
		&resourceName,
		&beforeState,
		&afterState,
		&changes,
		&log.IPAddress,
		&userAgent,
		&metadata,
		&log.IsSensitive,
		&log.Timestamp,
		&log.CreatedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	log.TenantID = repository.FromNullUUID(tenantID)
	log.ResourceName = repository.FromNullString(resourceName)
	log.UserAgent = repository.FromNullString(userAgent)

	if len(beforeState) > 0 {
		rawMsg := json.RawMessage(beforeState)
		log.BeforeState = &rawMsg
	}
	if len(afterState) > 0 {
		rawMsg := json.RawMessage(afterState)
		log.AfterState = &rawMsg
	}
	if len(changes) > 0 {
		if err := json.Unmarshal(changes, &log.Changes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal changes: %w", err)
		}
	}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &log.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &log, nil
}

// scanAuditLogFromRows scans an audit log from a Rows object
func (r *AuditRepository) scanAuditLogFromRows(rows *sql.Rows) (*domain.AuditLog, error) {
	var log domain.AuditLog
	var tenantID sql.NullString
	var resourceName, userAgent sql.NullString
	var beforeState, afterState, changes, metadata []byte

	err := rows.Scan(
		&log.ID,
		&tenantID,
		&log.ActorUserID,
		&log.ActorRole,
		&log.Action,
		&log.ResourceType,
		&log.ResourceID,
		&resourceName,
		&beforeState,
		&afterState,
		&changes,
		&log.IPAddress,
		&userAgent,
		&metadata,
		&log.IsSensitive,
		&log.Timestamp,
		&log.CreatedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	log.TenantID = repository.FromNullUUID(tenantID)
	log.ResourceName = repository.FromNullString(resourceName)
	log.UserAgent = repository.FromNullString(userAgent)

	if len(beforeState) > 0 {
		rawMsg := json.RawMessage(beforeState)
		log.BeforeState = &rawMsg
	}
	if len(afterState) > 0 {
		rawMsg := json.RawMessage(afterState)
		log.AfterState = &rawMsg
	}
	if len(changes) > 0 {
		if err := json.Unmarshal(changes, &log.Changes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal changes: %w", err)
		}
	}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &log.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &log, nil
}
