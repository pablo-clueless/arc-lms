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

// NotificationRepository handles database operations for notifications
type NotificationRepository struct {
	*repository.BaseRepository
}

// NewNotificationRepository creates a new notification repository
func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{
		BaseRepository: repository.NewBaseRepository(db),
	}
}

// Create creates a new notification
func (r *NotificationRepository) Create(ctx context.Context, notification *domain.Notification) error {
	channelsJSON, err := json.Marshal(notification.Channels)
	if err != nil {
		return fmt.Errorf("failed to marshal channels: %w", err)
	}

	query := `
		INSERT INTO notifications (
			id, tenant_id, user_id, event_type, title, body,
			channels, priority, action_url, resource_type, resource_id,
			is_read, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err = r.GetDB().ExecContext(ctx, query,
		notification.ID,
		notification.TenantID,
		notification.UserID,
		notification.EventType,
		notification.Title,
		notification.Body,
		channelsJSON,
		notification.Priority,
		repository.ToNullString(notification.ActionURL),
		repository.ToNullString(notification.ResourceType),
		repository.ToNullUUID(notification.ResourceID),
		notification.Read,
		notification.CreatedAt,
		notification.UpdatedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// GetByID retrieves a notification by ID
func (r *NotificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Notification, error) {
	query := `
		SELECT
			id, tenant_id, user_id, event_type, title, body,
			channels, priority, action_url, resource_type, resource_id,
			is_read, read_at, delivered_at, failed_at, failure_reason,
			created_at, updated_at
		FROM notifications
		WHERE id = $1
	`

	return r.scanNotification(r.GetDB().QueryRowContext(ctx, query, id))
}

// ListByUser retrieves notifications for a user with pagination
func (r *NotificationRepository) ListByUser(ctx context.Context, userID uuid.UUID, unreadOnly bool, params repository.PaginationParams) ([]*domain.Notification, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	whereClause := "WHERE user_id = $1"
	args := []interface{}{userID}
	argIndex := 2

	if unreadOnly {
		whereClause += " AND is_read = FALSE"
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM notifications %s", whereClause)
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT
			id, tenant_id, user_id, event_type, title, body,
			channels, priority, action_url, resource_type, resource_id,
			is_read, read_at, delivered_at, failed_at, failure_reason,
			created_at, updated_at
		FROM notifications
		%s ORDER BY created_at %s, id %s LIMIT $%d OFFSET $%d`,
		whereClause, params.SortOrder, params.SortOrder, argIndex, argIndex+1)
	args = append(args, params.Limit, params.Offset())

	rows, err := r.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, repository.ParseError(err)
	}
	defer rows.Close()

	notifications := make([]*domain.Notification, 0)
	for rows.Next() {
		notification, err := r.scanNotificationFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		notifications = append(notifications, notification)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	return notifications, total, nil
}

// MarkAsRead marks a notification as read
func (r *NotificationRepository) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE notifications
		SET is_read = TRUE, read_at = $2, updated_at = $3
		WHERE id = $1 AND is_read = FALSE
	`

	now := time.Now()
	result, err := r.GetDB().ExecContext(ctx, query, id, now, now)
	if err != nil {
		return repository.ParseError(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		// Either not found or already read - check if exists
		var exists bool
		err = r.GetDB().QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM notifications WHERE id = $1)", id).Scan(&exists)
		if err != nil {
			return repository.ParseError(err)
		}
		if !exists {
			return repository.ErrNotFound
		}
		// Already read, not an error
	}

	return nil
}

// MarkAllAsRead marks all notifications as read for a user
func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	query := `
		UPDATE notifications
		SET is_read = TRUE, read_at = $2, updated_at = $3
		WHERE user_id = $1 AND is_read = FALSE
	`

	now := time.Now()
	result, err := r.GetDB().ExecContext(ctx, query, userID, now, now)
	if err != nil {
		return 0, repository.ParseError(err)
	}

	return result.RowsAffected()
}

// GetUnreadCount returns the count of unread notifications for a user
func (r *NotificationRepository) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE`

	var count int
	err := r.GetDB().QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, repository.ParseError(err)
	}

	return count, nil
}

// Delete deletes a notification
func (r *NotificationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM notifications WHERE id = $1`

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

// CreateBatch creates multiple notifications in a single transaction
func (r *NotificationRepository) CreateBatch(ctx context.Context, notifications []*domain.Notification) error {
	tx, err := r.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO notifications (
			id, tenant_id, user_id, event_type, title, body,
			channels, priority, action_url, resource_type, resource_id,
			is_read, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, notification := range notifications {
		channelsJSON, err := json.Marshal(notification.Channels)
		if err != nil {
			return fmt.Errorf("failed to marshal channels: %w", err)
		}

		_, err = stmt.ExecContext(ctx,
			notification.ID,
			notification.TenantID,
			notification.UserID,
			notification.EventType,
			notification.Title,
			notification.Body,
			channelsJSON,
			notification.Priority,
			repository.ToNullString(notification.ActionURL),
			repository.ToNullString(notification.ResourceType),
			repository.ToNullUUID(notification.ResourceID),
			notification.Read,
			notification.CreatedAt,
			notification.UpdatedAt,
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

// scanNotification scans a notification from a single row
func (r *NotificationRepository) scanNotification(row *sql.Row) (*domain.Notification, error) {
	var n domain.Notification
	var channelsJSON []byte
	var actionURL, resourceType, failureReason sql.NullString
	var resourceID sql.NullString
	var readAt, deliveredAt, failedAt sql.NullTime

	err := row.Scan(
		&n.ID,
		&n.TenantID,
		&n.UserID,
		&n.EventType,
		&n.Title,
		&n.Body,
		&channelsJSON,
		&n.Priority,
		&actionURL,
		&resourceType,
		&resourceID,
		&n.Read,
		&readAt,
		&deliveredAt,
		&failedAt,
		&failureReason,
		&n.CreatedAt,
		&n.UpdatedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	if err := json.Unmarshal(channelsJSON, &n.Channels); err != nil {
		return nil, fmt.Errorf("failed to unmarshal channels: %w", err)
	}

	n.ActionURL = repository.FromNullString(actionURL)
	n.ResourceType = repository.FromNullString(resourceType)
	n.ResourceID = repository.FromNullUUID(resourceID)
	n.FailureReason = repository.FromNullString(failureReason)

	if readAt.Valid {
		n.ReadAt = &readAt.Time
	}
	if deliveredAt.Valid {
		n.DeliveredAt = &deliveredAt.Time
	}
	if failedAt.Valid {
		n.FailedAt = &failedAt.Time
	}

	return &n, nil
}

// scanNotificationFromRows scans a notification from a Rows object
func (r *NotificationRepository) scanNotificationFromRows(rows *sql.Rows) (*domain.Notification, error) {
	var n domain.Notification
	var channelsJSON []byte
	var actionURL, resourceType, failureReason sql.NullString
	var resourceID sql.NullString
	var readAt, deliveredAt, failedAt sql.NullTime

	err := rows.Scan(
		&n.ID,
		&n.TenantID,
		&n.UserID,
		&n.EventType,
		&n.Title,
		&n.Body,
		&channelsJSON,
		&n.Priority,
		&actionURL,
		&resourceType,
		&resourceID,
		&n.Read,
		&readAt,
		&deliveredAt,
		&failedAt,
		&failureReason,
		&n.CreatedAt,
		&n.UpdatedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	if err := json.Unmarshal(channelsJSON, &n.Channels); err != nil {
		return nil, fmt.Errorf("failed to unmarshal channels: %w", err)
	}

	n.ActionURL = repository.FromNullString(actionURL)
	n.ResourceType = repository.FromNullString(resourceType)
	n.ResourceID = repository.FromNullUUID(resourceID)
	n.FailureReason = repository.FromNullString(failureReason)

	if readAt.Valid {
		n.ReadAt = &readAt.Time
	}
	if deliveredAt.Valid {
		n.DeliveredAt = &deliveredAt.Time
	}
	if failedAt.Valid {
		n.FailedAt = &failedAt.Time
	}

	return &n, nil
}

// DeleteOldReadNotifications deletes read notifications older than the cutoff date
func (r *NotificationRepository) DeleteOldReadNotifications(ctx context.Context, cutoffDate time.Time) (int64, error) {
	query := `
		DELETE FROM notifications
		WHERE is_read = TRUE AND read_at < $1
	`

	result, err := r.GetDB().ExecContext(ctx, query, cutoffDate)
	if err != nil {
		return 0, repository.ParseError(err)
	}

	return result.RowsAffected()
}
