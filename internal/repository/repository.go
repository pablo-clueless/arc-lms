package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Common repository errors
var (
	ErrNotFound           = errors.New("resource not found")
	ErrDuplicateKey       = errors.New("duplicate key violation")
	ErrForeignKeyViolation = errors.New("foreign key violation")
	ErrCheckViolation     = errors.New("check constraint violation")
	ErrExclusionViolation = errors.New("exclusion constraint violation")
	ErrDeadlock           = errors.New("deadlock detected")
	ErrSerializationFailure = errors.New("serialization failure")
	ErrInvalidInput       = errors.New("invalid input")
	ErrConcurrentUpdate   = errors.New("concurrent update detected")
)

// TxFunc represents a function that executes within a transaction
type TxFunc func(tx *sql.Tx) error

// Repository defines the base repository interface with transaction support
type Repository interface {
	// Transaction management
	BeginTx(ctx context.Context) (*sql.Tx, error)
	CommitTx(tx *sql.Tx) error
	RollbackTx(tx *sql.Tx) error
	WithTransaction(ctx context.Context, fn TxFunc) error
}

// BaseRepository provides common repository functionality
type BaseRepository struct {
	db *sql.DB
}

// NewBaseRepository creates a new base repository
func NewBaseRepository(db *sql.DB) *BaseRepository {
	return &BaseRepository{db: db}
}

// BeginTx starts a new transaction
func (r *BaseRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
}

// CommitTx commits a transaction
func (r *BaseRepository) CommitTx(tx *sql.Tx) error {
	return tx.Commit()
}

// RollbackTx rolls back a transaction
func (r *BaseRepository) RollbackTx(tx *sql.Tx) error {
	return tx.Rollback()
}

// WithTransaction executes a function within a transaction
// Automatically commits on success or rolls back on error
func (r *BaseRepository) WithTransaction(ctx context.Context, fn TxFunc) error {
	tx, err := r.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = r.RollbackTx(tx)
			panic(p) // re-throw panic after rollback
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := r.RollbackTx(tx); rbErr != nil {
			return fmt.Errorf("transaction error: %w, rollback error: %v", err, rbErr)
		}
		return err
	}

	if err := r.CommitTx(tx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ParseError converts a database error to a repository error
func ParseError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}

	// PostgreSQL errors
	if pqErr, ok := err.(*pq.Error); ok {
		switch pqErr.Code {
		case "23505": // unique_violation
			return ErrDuplicateKey
		case "23503": // foreign_key_violation
			return ErrForeignKeyViolation
		case "23514": // check_violation
			return ErrCheckViolation
		case "23P01": // exclusion_violation
			return ErrExclusionViolation
		case "40P01": // deadlock_detected
			return ErrDeadlock
		case "40001": // serialization_failure
			return ErrSerializationFailure
		case "22P02", "22003", "23502": // invalid_text_representation, numeric_value_out_of_range, not_null_violation
			return ErrInvalidInput
		}
	}

	return err
}

// GetDB returns the underlying database connection
func (r *BaseRepository) GetDB() *sql.DB {
	return r.db
}

// DBStats represents database connection pool statistics
type DBStats struct {
	MaxOpenConnections int   `json:"max_open_connections"`
	OpenConnections    int   `json:"open_connections"`
	InUse              int   `json:"in_use"`
	Idle               int   `json:"idle"`
	WaitCount          int64 `json:"wait_count"`
	WaitDurationMs     int64 `json:"wait_duration_ms"`
}

// GetDBStats returns database connection pool statistics
func (r *BaseRepository) GetDBStats() *DBStats {
	stats := r.db.Stats()
	return &DBStats{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDurationMs:     stats.WaitDuration.Milliseconds(),
	}
}

// Execer defines the interface for database execution (works with both *sql.DB and *sql.Tx)
type Execer interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// GetExecer returns the appropriate execer (transaction or database)
func GetExecer(db *sql.DB, tx *sql.Tx) Execer {
	if tx != nil {
		return tx
	}
	return db
}

// PaginationParams defines parameters for cursor-based pagination
type PaginationParams struct {
	Cursor    *uuid.UUID // Last seen ID for cursor-based pagination
	Limit     int        // Number of records to fetch
	SortOrder string     // "ASC" or "DESC"
}

// DefaultPaginationParams returns sensible defaults for pagination
func DefaultPaginationParams() PaginationParams {
	return PaginationParams{
		Cursor:    nil,
		Limit:     50,
		SortOrder: "DESC",
	}
}

// ValidatePaginationParams validates and normalizes pagination parameters
func ValidatePaginationParams(params *PaginationParams) error {
	if params.Limit <= 0 {
		params.Limit = 50
	}
	if params.Limit > 100 {
		params.Limit = 100
	}
	if params.SortOrder != "ASC" && params.SortOrder != "DESC" {
		params.SortOrder = "DESC"
	}
	return nil
}

// PaginatedResult holds paginated query results
type PaginatedResult struct {
	HasMore    bool       `json:"has_more"`
	NextCursor *uuid.UUID `json:"next_cursor,omitempty"`
	Count      int        `json:"count"`
}

// BuildPaginatedResult builds pagination metadata from results
func BuildPaginatedResult(ids []uuid.UUID, limit int) PaginatedResult {
	result := PaginatedResult{
		HasMore: len(ids) > limit,
		Count:   len(ids),
	}

	if result.HasMore {
		// Set next cursor to the last ID (before the extra record)
		result.NextCursor = &ids[limit-1]
		result.Count = limit
	}

	return result
}

// TenantScoped defines repositories that support tenant isolation
type TenantScoped interface {
	ValidateTenantAccess(ctx context.Context, tenantID uuid.UUID, resourceID uuid.UUID) error
}

// NullUUID handles nullable UUID fields
type NullUUID struct {
	UUID  uuid.UUID
	Valid bool
}

// NullString handles nullable string fields
type NullString struct {
	String string
	Valid  bool
}

// ToNullUUID converts a *uuid.UUID to sql.NullString for database operations
func ToNullUUID(u *uuid.UUID) sql.NullString {
	if u == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: u.String(), Valid: true}
}

// FromNullUUID converts sql.NullString to *uuid.UUID
func FromNullUUID(ns sql.NullString) *uuid.UUID {
	if !ns.Valid {
		return nil
	}
	u, err := uuid.Parse(ns.String)
	if err != nil {
		return nil
	}
	return &u
}

// ToNullString converts a *string to sql.NullString
func ToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

// StringToNullString converts a string to sql.NullString, treating empty strings as NULL
func StringToNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

// FromNullString converts sql.NullString to *string
func FromNullString(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

// ToNullTime converts a *time.Time to sql.NullTime
func ToNullTime(t *interface{}) sql.NullTime {
	return sql.NullTime{Valid: false}
}

// Transaction represents a database transaction context
type Transaction *sql.Tx

// Import statement needed at top of file
// Note: These interfaces define contracts for repository implementations in postgres/ package

// TenantRepository defines methods for tenant data access
type TenantRepository interface {
	Create(ctx context.Context, tenant interface{}) error
	Get(ctx context.Context, id uuid.UUID) (interface{}, error)
	Update(ctx context.Context, tenant interface{}) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, status interface{}, params PaginationParams) (interface{}, error)
	Suspend(ctx context.Context, id uuid.UUID, reason string, tx *sql.Tx) error
	Reactivate(ctx context.Context, id uuid.UUID, tx *sql.Tx) error
}

// UserRepository defines methods for user data access
type UserRepository interface {
	Create(ctx context.Context, user interface{}) error
	GetByID(ctx context.Context, id uuid.UUID) (interface{}, error)
	GetByEmail(ctx context.Context, email string) (interface{}, error)
	Update(ctx context.Context, user interface{}) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, tenantID *uuid.UUID, role interface{}, status interface{}, params PaginationParams) (interface{}, error)
	Deactivate(ctx context.Context, id uuid.UUID, reason string) error
	Reactivate(ctx context.Context, id uuid.UUID) error
	SetPasswordResetToken(ctx context.Context, id uuid.UUID, token string, expiry interface{}) error
	ClearPasswordResetToken(ctx context.Context, id uuid.UUID) error
	SetInvitationToken(ctx context.Context, id uuid.UUID, token string, expiry interface{}) error
	ClearInvitationToken(ctx context.Context, id uuid.UUID) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	RecordLogin(ctx context.Context, id uuid.UUID) error
}

// SessionRepository defines methods for session data access
type SessionRepository interface {
	Create(ctx context.Context, session interface{}) error
	GetByID(ctx context.Context, id uuid.UUID) (interface{}, error)
	Update(ctx context.Context, session interface{}) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, tenantID uuid.UUID, status interface{}, params PaginationParams) (interface{}, error)
	GetActiveSession(ctx context.Context, tenantID uuid.UUID) (interface{}, error)
	Activate(ctx context.Context, id uuid.UUID, tx *sql.Tx) error
	Archive(ctx context.Context, id uuid.UUID) error
}

// TermRepository defines methods for term data access
type TermRepository interface {
	Create(ctx context.Context, term interface{}) error
	GetByID(ctx context.Context, id uuid.UUID) (interface{}, error)
	Update(ctx context.Context, term interface{}) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, sessionID uuid.UUID, status interface{}, params PaginationParams) (interface{}, error)
	GetActiveTerm(ctx context.Context, sessionID uuid.UUID) (interface{}, error)
	Activate(ctx context.Context, id uuid.UUID, studentCount int, tx *sql.Tx) error
	Complete(ctx context.Context, id uuid.UUID) error
}

// ClassRepository defines methods for class data access
type ClassRepository interface {
	Create(ctx context.Context, class interface{}) error
	GetByID(ctx context.Context, id uuid.UUID) (interface{}, error)
	Update(ctx context.Context, class interface{}) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, tenantID uuid.UUID, sessionID *uuid.UUID, params PaginationParams) (interface{}, error)
}

// CourseRepository defines methods for course data access
type CourseRepository interface {
	Create(ctx context.Context, course interface{}) error
	GetByID(ctx context.Context, id uuid.UUID) (interface{}, error)
	Update(ctx context.Context, course interface{}) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, tenantID uuid.UUID, classID *uuid.UUID, termID *uuid.UUID, tutorID *uuid.UUID, params PaginationParams) (interface{}, error)
	ReassignTutor(ctx context.Context, courseID uuid.UUID, newTutorID uuid.UUID, tx *sql.Tx) error
}

// EnrollmentRepository defines methods for enrollment data access
type EnrollmentRepository interface {
	Create(ctx context.Context, enrollment interface{}) error
	GetByID(ctx context.Context, id uuid.UUID) (interface{}, error)
	Update(ctx context.Context, enrollment interface{}) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, tenantID uuid.UUID, classID *uuid.UUID, sessionID *uuid.UUID, status interface{}, params PaginationParams) (interface{}, error)
	Transfer(ctx context.Context, id uuid.UUID, newClassID uuid.UUID, reason string, tx *sql.Tx) error
	Withdraw(ctx context.Context, id uuid.UUID, reason string, tx *sql.Tx) error
	Suspend(ctx context.Context, id uuid.UUID, reason string, tx *sql.Tx) error
	Reactivate(ctx context.Context, id uuid.UUID, tx *sql.Tx) error
	CountActiveStudents(ctx context.Context, sessionID uuid.UUID) (int, error)
}

// TimetableRepository defines methods for timetable data access
type TimetableRepository interface {
	Create(ctx context.Context, timetable interface{}) error
	GetByID(ctx context.Context, id uuid.UUID) (interface{}, error)
	Update(ctx context.Context, timetable interface{}) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, tenantID uuid.UUID, classID *uuid.UUID, termID *uuid.UUID, status interface{}, params PaginationParams) (interface{}, error)
	Publish(ctx context.Context, id uuid.UUID) error
}

// AuditRepository defines methods for audit log data access
type AuditRepository interface {
	Create(ctx context.Context, auditLog interface{}) error
	GetByID(ctx context.Context, id uuid.UUID) (interface{}, error)
	List(ctx context.Context, filters interface{}, params PaginationParams) (interface{}, error)
	GetByResource(ctx context.Context, resourceType interface{}, resourceID uuid.UUID, params PaginationParams) (interface{}, error)
}

// InvoiceRepository defines methods for invoice data access
type InvoiceRepository interface {
	Create(ctx context.Context, invoice interface{}) error
	GetByID(ctx context.Context, id uuid.UUID) (interface{}, error)
	Update(ctx context.Context, invoice interface{}) error
	List(ctx context.Context, tenantID uuid.UUID, status interface{}, params PaginationParams) (interface{}, error)
	MarkAsPaid(ctx context.Context, id uuid.UUID, tx *sql.Tx) error
}

// SubscriptionRepository defines methods for subscription data access
type SubscriptionRepository interface {
	Create(ctx context.Context, subscription interface{}) error
	GetByID(ctx context.Context, id uuid.UUID) (interface{}, error)
	GetByTenant(ctx context.Context, tenantID uuid.UUID) (interface{}, error)
	Update(ctx context.Context, subscription interface{}) error
	List(ctx context.Context, status interface{}, params PaginationParams) (interface{}, error)
}
