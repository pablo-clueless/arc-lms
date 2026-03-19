# Repository Layer Implementation Summary

## Overview
This document summarizes the comprehensive repository layer implementation for the Nigerian LMS API. All repositories follow consistent patterns with proper transaction support, idempotency checks, and race condition prevention.

## Completed Repository Files

### Core Repositories (✅ Implemented)

1. **internal/repository/repository.go** - Base repository interface with:
   - Transaction management helpers (BeginTx, CommitTx, RollbackTx, WithTransaction)
   - Error handling utilities (ParseError for PostgreSQL errors)
   - Pagination support (cursor-based)
   - Helper functions for NULL handling
   - TenantScoped interface for tenant isolation

2. **internal/repository/postgres/tenant.go** - Tenant operations:
   - Create, Get, Update, Delete, List tenants
   - Suspend/Reactivate with reason tracking
   - Filter by status with pagination
   - Full JSONB support for configuration and billing contact

3. **internal/repository/postgres/user.go** - User management:
   - Create, GetByID, GetByEmail, Update, Delete
   - List with filters (tenant_id, role, status)
   - Deactivate/Reactivate
   - Password reset token management (Set/Clear)
   - Invitation token management (Set/Clear)
   - RecordLogin for last_login_at tracking
   - UpdatePassword
   - Full JSONB support for permissions and notification preferences

4. **internal/repository/postgres/session.go** - Session operations:
   - Create, Get, Update, Delete
   - ListByTenant with status filter
   - **Activate (enforces BR-007: one active session per tenant)**
     - Uses FOR UPDATE lock in transaction
     - Checks for existing active session before activating
   - Archive
   - GetActiveSession
   - ValidateTenantAccess

5. **internal/repository/postgres/term.go** - Term operations:
   - **Create (enforces BR-002: non-overlapping dates)**
     - Inserts into term_date_ranges table
     - Uses PostgreSQL exclusion constraint
     - Must be called within transaction
   - Get, Update, Delete
   - ListBySession
   - **Activate (triggers billing)**
   - Complete
   - GetActiveTerm, GetCurrentActiveTerm
   - ValidateTenantAccess
   - Full JSONB support for holidays and non_instructional_days

6. **internal/repository/postgres/class.go** - Class operations:
   - Create, Get, Update, Delete
   - ListBySession, ListByTenant
   - ValidateTenantAccess
   - Handles optional capacity field

7. **internal/repository/postgres/course.go** - Course operations:
   - Create, Get, Update, Delete
   - ListByClass, ListByTerm, ListByTutor
   - ReassignTutor
   - ValidateTenantAccess
   - Full JSONB support for custom grade weighting and materials

8. **internal/repository/postgres/enrollment.go** - Enrollment operations:
   - **Enroll (enforces BR-003: one class per session per student)**
     - Uses FOR UPDATE lock in transaction
     - Checks for existing enrollment before creating
     - Must be called within transaction
   - Get, Update, Delete
   - ListByStudent, ListByClass, ListBySession
   - Transfer (within same session)
   - Withdraw, Suspend, Reactivate
   - ValidateTenantAccess

9. **internal/repository/postgres/timetable.go** - Timetable operations:
   - Create, Get, Update, Delete
   - ListByClass, ListByTerm
   - Publish
   - GetPublishedTimetable
   - ValidateTenantAccess
   - Tracks generation_version for versioning

## Repository Pattern Structure

### Common Patterns

All repositories follow these patterns:

```go
// 1. Repository struct embeds BaseRepository
type XRepository struct {
    *repository.BaseRepository
}

// 2. Constructor
func NewXRepository(db *sql.DB) *XRepository {
    return &XRepository{
        BaseRepository: repository.NewBaseRepository(db),
    }
}

// 3. CRUD operations accept context as first parameter
func (r *XRepository) Create(ctx context.Context, entity *domain.Entity, tx *sql.Tx) error

// 4. List operations use pagination
func (r *XRepository) List(ctx context.Context, params repository.PaginationParams) ([]*domain.Entity, error)

// 5. Tenant isolation validation
func (r *XRepository) ValidateTenantAccess(ctx context.Context, tenantID uuid.UUID, resourceID uuid.UUID) error
```

### Transaction Support

Operations that require atomicity accept `*sql.Tx`:

```go
// If tx is provided, use it; otherwise use db connection
execer := repository.GetExecer(r.GetDB(), tx)
execer.ExecContext(ctx, query, args...)
```

### Race Condition Prevention

Critical operations use `FOR UPDATE` locks:

```go
// Example: Checking for active session before activating another
checkQuery := `
    SELECT id FROM sessions
    WHERE tenant_id = $1 AND status = $2 AND id != $3
    FOR UPDATE
`
```

### Idempotency Support

Constraint violations are handled gracefully:

```go
if err != nil {
    return repository.ParseError(err)  // Converts to ErrDuplicateKey, etc.
}
```

## Remaining Repository Files to Implement

The following repositories follow the same patterns established above:

### Assessment Repositories (To Implement)

10. **period.go** - Period operations with swap support:
    - Create (batch insert), Get, Update, Delete
    - ListByTimetable
    - **CheckTutorAvailability** (prevents double-booking)
    - **SwapPeriods** (uses FOR UPDATE locks for atomic swap)

11. **swap_request.go** - Swap request operations:
    - Create, Get, Update, Delete
    - ListPending
    - Approve, Reject, Escalate

12. **quiz.go** - Quiz operations:
    - Create, Get, Update, Delete
    - ListByCourse
    - Publish

13. **quiz_submission.go** - Quiz submission operations:
    - **Create (enforces BR-018: one attempt)**
    - Get, Update
    - ListByQuiz, ListByStudent
    - AutoGrade (for multiple choice)

14. **assignment.go** - Assignment operations:
    - Create, Get, Update, Delete
    - ListByCourse
    - Publish

15. **assignment_submission.go** - Assignment submission operations:
    - Create, Get, Update
    - ListByAssignment, ListByStudent
    - Grade

16. **examination.go** - Examination operations:
    - Create, Get, Update, Delete
    - ListByCourse, ListByTerm
    - PublishResults

17. **examination_submission.go** - Examination submission operations:
    - **Create (enforces BR-013: one per student)**
    - Get, Update
    - RecordIntegrityEvent
    - AutoSubmitOnTimeout

### Progress & Reporting Repositories (To Implement)

18. **progress.go** - Progress tracking:
    - CreateOrUpdate, Get
    - GetByStudentCourseTerm
    - ComputeGrades
    - FlagLowPerformance

19. **report_card.go** - Report card operations:
    - Generate, Get, Update
    - ListByStudent, ListByTerm

### Communication Repositories (To Implement)

20. **meeting.go** - Virtual meeting operations:
    - Create, Get, Update, Delete
    - ListByClass, ListByTutor
    - Start, End
    - RecordParticipantEvent

21. **communication.go** - Email operations:
    - Create, Get, Update, Delete
    - ListByTenant, ListBySender
    - Schedule, MarkSent, MarkFailed

22. **notification.go** - Notification operations:
    - Create, Get, Update, Delete
    - ListByUser (unread first)
    - MarkRead, MarkDelivered
    - BatchCreate (for broadcasts)

### Billing Repository (To Implement)

23. **billing.go** - Billing operations:
    - CreateSubscription (with snapshot count - BR-009)
    - CreateInvoice
    - Get, Update
    - ListByTenant, ListByStatus
    - MarkPaid, MarkOverdue
    - **Uses FOR UPDATE locks for billing snapshot**

### Audit Repository (To Implement)

24. **audit.go** - Audit log operations (immutable):
    - Create (no update/delete - BR-015)
    - List with filters (tenant, user, resource type, action, date range)
    - Supports compliance reporting

## Business Rules Enforcement

### BR-001: SUPER_ADMIN Tenant Isolation
- Enforced at database level with CHECK constraint
- User repository respects NULL tenant_id for SUPER_ADMIN

### BR-002: Non-Overlapping Term Dates
- **Implemented in term.go**
- Uses term_date_ranges table with EXCLUDE USING GIST constraint
- Creates date range on term creation within transaction

### BR-003: One Enrollment Per Student Per Session
- **Implemented in enrollment.go**
- Enroll() uses FOR UPDATE lock to check existing enrollment
- Must be called within transaction

### BR-007: One Active Session Per Tenant
- **Implemented in session.go**
- Activate() uses FOR UPDATE lock to check existing active session
- Must be called within transaction

### BR-009: Billing Snapshot on Term Activation
- To be implemented in billing.go
- Uses FOR UPDATE lock when creating subscription snapshot

### BR-013: One Examination Submission Per Student
- To be implemented in examination_submission.go
- Similar pattern to enrollment (FOR UPDATE check)

### BR-015: Immutable Audit Logs
- To be implemented in audit.go
- No Update() or Delete() methods

### BR-018: One Quiz Attempt Per Student
- To be implemented in quiz_submission.go
- Similar pattern to enrollment (FOR UPDATE check)

## Error Handling

All repositories use consistent error handling:

```go
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
```

PostgreSQL errors are automatically converted:
- `23505` → `ErrDuplicateKey`
- `23503` → `ErrForeignKeyViolation`
- `23514` → `ErrCheckViolation`
- `23P01` → `ErrExclusionViolation`
- `40P01` → `ErrDeadlock`
- `40001` → `ErrSerializationFailure`
- `sql.ErrNoRows` → `ErrNotFound`

## Pagination

All List operations support cursor-based pagination:

```go
type PaginationParams struct {
    Cursor    *uuid.UUID // Last seen ID
    Limit     int        // Default: 50, Max: 100
    SortOrder string     // "ASC" or "DESC", default: "DESC"
}
```

## Transaction Usage

### When to Use Transactions

1. **Creating resources with constraint checks**:
   - Enrollment (BR-003)
   - Session activation (BR-007)
   - Term creation (BR-002)

2. **Multi-step operations**:
   - Transfer student (update enrollment + create new enrollment)
   - Swap periods (update both periods atomically)
   - Generate report card (aggregate from multiple tables)

3. **Billing operations**:
   - Create subscription with snapshot (BR-009)

### Transaction Pattern

```go
// In service layer
err := repo.WithTransaction(ctx, func(tx *sql.Tx) error {
    // Perform operations
    err := repo1.Operation(ctx, entity1, tx)
    if err != nil {
        return err
    }

    err = repo2.Operation(ctx, entity2, tx)
    if err != nil {
        return err
    }

    return nil
})
```

## Testing Considerations

Each repository should be tested with:

1. **Unit tests** (with mock database)
2. **Integration tests** (with PostgreSQL testcontainer)
3. **Race condition tests** (concurrent access)
4. **Transaction tests** (rollback scenarios)
5. **Constraint violation tests** (business rules)

## Performance Optimizations

1. **Indexes**: All foreign keys and commonly queried fields are indexed
2. **Prepared statements**: Can be added for frequently executed queries
3. **Connection pooling**: Configured in database setup
4. **Batch operations**: Provided for bulk inserts (e.g., period creation)
5. **JSONB indexing**: Can be added for frequently queried JSONB fields

## Next Steps

1. Implement remaining 15 repository files following established patterns
2. Add comprehensive test coverage
3. Implement prepared statement caching for hot paths
4. Add repository-level metrics and logging
5. Create repository factory for dependency injection
6. Document all repository interfaces for service layer

## Files Created

✅ **internal/repository/repository.go** (Base repository)
✅ **internal/repository/postgres/tenant.go**
✅ **internal/repository/postgres/user.go**
✅ **internal/repository/postgres/session.go**
✅ **internal/repository/postgres/term.go**
✅ **internal/repository/postgres/class.go**
✅ **internal/repository/postgres/course.go**
✅ **internal/repository/postgres/enrollment.go**
✅ **internal/repository/postgres/timetable.go**

## Total Progress

**Completed**: 9/24 repository files (37.5%)
**Critical business logic**: All major constraint enforcement patterns established
**Remaining**: 15 repository files to implement following the same patterns
