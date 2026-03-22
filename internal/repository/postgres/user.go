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

// UserRepository handles database operations for users
type UserRepository struct {
	*repository.BaseRepository
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{
		BaseRepository: repository.NewBaseRepository(db),
	}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	permissionsJSON, err := json.Marshal(user.Permissions)
	if err != nil {
		return fmt.Errorf("failed to marshal permissions: %w", err)
	}

	preferencesJSON, err := json.Marshal(user.NotificationPreferences)
	if err != nil {
		return fmt.Errorf("failed to marshal notification preferences: %w", err)
	}

	query := `
		INSERT INTO users (
			id, tenant_id, role, email, password_hash,
			first_name, last_name, middle_name, profile_photo, phone,
			status, permissions, notification_preferences,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	_, err = r.GetDB().ExecContext(ctx, query,
		user.ID,
		repository.ToNullUUID(user.TenantID),
		user.Role,
		user.Email,
		user.PasswordHash,
		user.FirstName,
		user.LastName,
		repository.ToNullString(user.MiddleName),
		repository.ToNullString(user.ProfilePhoto),
		repository.ToNullString(user.Phone),
		user.Status,
		permissionsJSON,
		preferencesJSON,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		return repository.ParseError(err)
	}

	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT
			id, tenant_id, role, email, password_hash,
			first_name, last_name, middle_name, profile_photo, phone,
			status, permissions, notification_preferences,
			last_login_at, password_reset_token, password_reset_expiry,
			invitation_token, invitation_expiry,
			deactivated_at, deactivation_reason,
			created_at, updated_at
		FROM users
		WHERE id = $1
	`

	return r.scanUser(r.GetDB().QueryRowContext(ctx, query, id))
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT
			id, tenant_id, role, email, password_hash,
			first_name, last_name, middle_name, profile_photo, phone,
			status, permissions, notification_preferences,
			last_login_at, password_reset_token, password_reset_expiry,
			invitation_token, invitation_expiry,
			deactivated_at, deactivation_reason,
			created_at, updated_at
		FROM users
		WHERE email = $1
	`

	return r.scanUser(r.GetDB().QueryRowContext(ctx, query, email))
}

// scanUser scans a user from a database row
func (r *UserRepository) scanUser(row *sql.Row) (*domain.User, error) {
	var user domain.User
	var tenantID, middleName, profilePhoto, phone sql.NullString
	var permissionsJSON, preferencesJSON []byte
	var lastLoginAt, passwordResetExpiry, invitationExpiry, deactivatedAt sql.NullTime
	var passwordResetToken, invitationToken, deactivationReason sql.NullString

	err := row.Scan(
		&user.ID,
		&tenantID,
		&user.Role,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&middleName,
		&profilePhoto,
		&phone,
		&user.Status,
		&permissionsJSON,
		&preferencesJSON,
		&lastLoginAt,
		&passwordResetToken,
		&passwordResetExpiry,
		&invitationToken,
		&invitationExpiry,
		&deactivatedAt,
		&deactivationReason,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	user.TenantID = repository.FromNullUUID(tenantID)
	user.MiddleName = repository.FromNullString(middleName)
	user.ProfilePhoto = repository.FromNullString(profilePhoto)
	user.Phone = repository.FromNullString(phone)

	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}
	if passwordResetToken.Valid {
		user.PasswordResetToken = &passwordResetToken.String
	}
	if passwordResetExpiry.Valid {
		user.PasswordResetExpiry = &passwordResetExpiry.Time
	}
	if invitationToken.Valid {
		user.InvitationToken = &invitationToken.String
	}
	if invitationExpiry.Valid {
		user.InvitationExpiry = &invitationExpiry.Time
	}
	if deactivatedAt.Valid {
		user.DeactivatedAt = &deactivatedAt.Time
	}
	if deactivationReason.Valid {
		user.DeactivationReason = &deactivationReason.String
	}

	if err := json.Unmarshal(permissionsJSON, &user.Permissions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal permissions: %w", err)
	}

	if err := json.Unmarshal(preferencesJSON, &user.NotificationPreferences); err != nil {
		return nil, fmt.Errorf("failed to unmarshal notification preferences: %w", err)
	}

	return &user, nil
}

// Update updates an existing user
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	permissionsJSON, err := json.Marshal(user.Permissions)
	if err != nil {
		return fmt.Errorf("failed to marshal permissions: %w", err)
	}

	preferencesJSON, err := json.Marshal(user.NotificationPreferences)
	if err != nil {
		return fmt.Errorf("failed to marshal notification preferences: %w", err)
	}

	query := `
		UPDATE users
		SET
			first_name = $2,
			last_name = $3,
			middle_name = $4,
			profile_photo = $5,
			phone = $6,
			permissions = $7,
			notification_preferences = $8,
			updated_at = $9
		WHERE id = $1
	`

	result, err := r.GetDB().ExecContext(ctx, query,
		user.ID,
		user.FirstName,
		user.LastName,
		repository.ToNullString(user.MiddleName),
		repository.ToNullString(user.ProfilePhoto),
		repository.ToNullString(user.Phone),
		permissionsJSON,
		preferencesJSON,
		user.UpdatedAt,
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

// Delete deletes a user
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`

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

// List retrieves users with filters and pagination
func (r *UserRepository) List(ctx context.Context, tenantID *uuid.UUID, role *domain.Role, status *domain.UserStatus, params repository.PaginationParams) ([]*domain.User, int, error) {
	if err := repository.ValidatePaginationParams(&params); err != nil {
		return nil, 0, err
	}

	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if tenantID != nil {
		whereClause += fmt.Sprintf(" AND tenant_id = $%d", argIndex)
		args = append(args, *tenantID)
		argIndex++
	}

	if role != nil {
		whereClause += fmt.Sprintf(" AND role = $%d", argIndex)
		args = append(args, *role)
		argIndex++
	}

	if status != nil {
		whereClause += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *status)
		argIndex++
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users %s", whereClause)
	var total int
	if err := r.GetDB().QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT
			id, tenant_id, role, email, password_hash,
			first_name, last_name, middle_name, profile_photo, phone,
			status, permissions, notification_preferences,
			last_login_at, password_reset_token, password_reset_expiry,
			invitation_token, invitation_expiry,
			deactivated_at, deactivation_reason,
			created_at, updated_at
		FROM users
		%s
		ORDER BY created_at %s
		LIMIT $%d OFFSET $%d
	`, whereClause, params.SortOrder, argIndex, argIndex+1)

	args = append(args, params.Limit, params.Offset())

	rows, err := r.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, repository.ParseError(err)
	}
	defer rows.Close()

	users := make([]*domain.User, 0)
	for rows.Next() {
		user, err := r.scanUserFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, repository.ParseError(err)
	}

	return users, total, nil
}

// scanUserFromRows scans a user from a Rows object
func (r *UserRepository) scanUserFromRows(rows *sql.Rows) (*domain.User, error) {
	var user domain.User
	var tenantID, middleName, profilePhoto, phone sql.NullString
	var permissionsJSON, preferencesJSON []byte
	var lastLoginAt, passwordResetExpiry, invitationExpiry, deactivatedAt sql.NullTime
	var passwordResetToken, invitationToken, deactivationReason sql.NullString

	err := rows.Scan(
		&user.ID,
		&tenantID,
		&user.Role,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&middleName,
		&profilePhoto,
		&phone,
		&user.Status,
		&permissionsJSON,
		&preferencesJSON,
		&lastLoginAt,
		&passwordResetToken,
		&passwordResetExpiry,
		&invitationToken,
		&invitationExpiry,
		&deactivatedAt,
		&deactivationReason,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, repository.ParseError(err)
	}

	user.TenantID = repository.FromNullUUID(tenantID)
	user.MiddleName = repository.FromNullString(middleName)
	user.ProfilePhoto = repository.FromNullString(profilePhoto)
	user.Phone = repository.FromNullString(phone)

	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}
	if passwordResetToken.Valid {
		user.PasswordResetToken = &passwordResetToken.String
	}
	if passwordResetExpiry.Valid {
		user.PasswordResetExpiry = &passwordResetExpiry.Time
	}
	if invitationToken.Valid {
		user.InvitationToken = &invitationToken.String
	}
	if invitationExpiry.Valid {
		user.InvitationExpiry = &invitationExpiry.Time
	}
	if deactivatedAt.Valid {
		user.DeactivatedAt = &deactivatedAt.Time
	}
	if deactivationReason.Valid {
		user.DeactivationReason = &deactivationReason.String
	}

	if err := json.Unmarshal(permissionsJSON, &user.Permissions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal permissions: %w", err)
	}

	if err := json.Unmarshal(preferencesJSON, &user.NotificationPreferences); err != nil {
		return nil, fmt.Errorf("failed to unmarshal notification preferences: %w", err)
	}

	return &user, nil
}

// Deactivate deactivates a user
func (r *UserRepository) Deactivate(ctx context.Context, id uuid.UUID, reason string) error {
	query := `
		UPDATE users
		SET
			status = $2,
			deactivation_reason = $3,
			deactivated_at = $4,
			updated_at = $5
		WHERE id = $1
	`

	now := time.Now()
	result, err := r.GetDB().ExecContext(ctx, query,
		id,
		domain.UserStatusDeactivated,
		reason,
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

// Reactivate reactivates a deactivated user
func (r *UserRepository) Reactivate(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE users
		SET
			status = $2,
			deactivation_reason = NULL,
			deactivated_at = NULL,
			updated_at = $3
		WHERE id = $1
	`

	result, err := r.GetDB().ExecContext(ctx, query,
		id,
		domain.UserStatusActive,
		time.Now(),
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

// SetPasswordResetToken sets a password reset token for a user
func (r *UserRepository) SetPasswordResetToken(ctx context.Context, id uuid.UUID, token string, expiry time.Time) error {
	query := `
		UPDATE users
		SET
			password_reset_token = $2,
			password_reset_expiry = $3,
			updated_at = $4
		WHERE id = $1
	`

	result, err := r.GetDB().ExecContext(ctx, query, id, token, expiry, time.Now())
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

// ClearPasswordResetToken clears the password reset token
func (r *UserRepository) ClearPasswordResetToken(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE users
		SET
			password_reset_token = NULL,
			password_reset_expiry = NULL,
			updated_at = $2
		WHERE id = $1
	`

	result, err := r.GetDB().ExecContext(ctx, query, id, time.Now())
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

// SetInvitationToken sets an invitation token for a user
func (r *UserRepository) SetInvitationToken(ctx context.Context, id uuid.UUID, token string, expiry time.Time) error {
	query := `
		UPDATE users
		SET
			invitation_token = $2,
			invitation_expiry = $3,
			status = $4,
			updated_at = $5
		WHERE id = $1
	`

	result, err := r.GetDB().ExecContext(ctx, query,
		id,
		token,
		expiry,
		domain.UserStatusPending,
		time.Now(),
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

// ClearInvitationToken clears the invitation token and activates the user
func (r *UserRepository) ClearInvitationToken(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE users
		SET
			invitation_token = NULL,
			invitation_expiry = NULL,
			status = $2,
			updated_at = $3
		WHERE id = $1
	`

	result, err := r.GetDB().ExecContext(ctx, query,
		id,
		domain.UserStatusActive,
		time.Now(),
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

// UpdatePassword updates a user's password hash
func (r *UserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	query := `
		UPDATE users
		SET
			password_hash = $2,
			updated_at = $3
		WHERE id = $1
	`

	result, err := r.GetDB().ExecContext(ctx, query, id, passwordHash, time.Now())
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

// RecordLogin updates the last login timestamp
func (r *UserRepository) RecordLogin(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE users
		SET
			last_login_at = $2,
			updated_at = $3
		WHERE id = $1
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
		return repository.ErrNotFound
	}

	return nil
}

// GetByPasswordResetToken retrieves a user by password reset token
func (r *UserRepository) GetByPasswordResetToken(ctx context.Context, token string) (*domain.User, error) {
	query := `
		SELECT
			id, tenant_id, role, email, password_hash,
			first_name, last_name, middle_name, profile_photo, phone,
			status, permissions, notification_preferences,
			last_login_at, password_reset_token, password_reset_expiry,
			invitation_token, invitation_expiry,
			deactivated_at, deactivation_reason,
			created_at, updated_at
		FROM users
		WHERE password_reset_token = $1
	`

	return r.scanUser(r.GetDB().QueryRowContext(ctx, query, token))
}

// GetByInvitationToken retrieves a user by invitation token
func (r *UserRepository) GetByInvitationToken(ctx context.Context, token string) (*domain.User, error) {
	query := `
		SELECT
			id, tenant_id, role, email, password_hash,
			first_name, last_name, middle_name, profile_photo, phone,
			status, permissions, notification_preferences,
			last_login_at, password_reset_token, password_reset_expiry,
			invitation_token, invitation_expiry,
			deactivated_at, deactivation_reason,
			created_at, updated_at
		FROM users
		WHERE invitation_token = $1
	`

	return r.scanUser(r.GetDB().QueryRowContext(ctx, query, token))
}

// UserGrowthPoint represents user count at a specific date
type UserGrowthPoint struct {
	Date  time.Time `json:"date"`
	Count int       `json:"count"`
}

// GetUserGrowth returns user registration counts grouped by date for the last N days
func (r *UserRepository) GetUserGrowth(ctx context.Context, days int) ([]UserGrowthPoint, error) {
	query := `
		SELECT
			DATE(created_at) as date,
			COUNT(*) as count
		FROM users
		WHERE created_at >= NOW() - INTERVAL '1 day' * $1
		GROUP BY DATE(created_at)
		ORDER BY date ASC
	`

	rows, err := r.GetDB().QueryContext(ctx, query, days)
	if err != nil {
		return nil, repository.ParseError(err)
	}
	defer rows.Close()

	var growth []UserGrowthPoint
	for rows.Next() {
		var point UserGrowthPoint
		if err := rows.Scan(&point.Date, &point.Count); err != nil {
			return nil, repository.ParseError(err)
		}
		growth = append(growth, point)
	}

	if err := rows.Err(); err != nil {
		return nil, repository.ParseError(err)
	}

	return growth, nil
}

// GetTotalUserCount returns the total number of users in the system
func (r *UserRepository) GetTotalUserCount(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM users`

	var count int
	err := r.GetDB().QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, repository.ParseError(err)
	}

	return count, nil
}
