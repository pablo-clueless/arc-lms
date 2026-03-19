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
