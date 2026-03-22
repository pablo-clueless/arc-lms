package service

import (
	"context"
	"fmt"
	"time"

	"arc-lms/internal/domain"
	"arc-lms/internal/repository"
	"arc-lms/internal/repository/postgres"

	"github.com/google/uuid"
)

// EnrollmentService handles student enrollment operations
type EnrollmentService struct {
	enrollmentRepo *postgres.EnrollmentRepository
	classRepo      *postgres.ClassRepository
	userRepo       *postgres.UserRepository
	sessionRepo    *postgres.SessionRepository
	termRepo       *postgres.TermRepository
	auditService   *AuditService
}

// NewEnrollmentService creates a new enrollment service
func NewEnrollmentService(
	enrollmentRepo *postgres.EnrollmentRepository,
	classRepo *postgres.ClassRepository,
	userRepo *postgres.UserRepository,
	sessionRepo *postgres.SessionRepository,
	termRepo *postgres.TermRepository,
	auditService *AuditService,
) *EnrollmentService {
	return &EnrollmentService{
		enrollmentRepo: enrollmentRepo,
		classRepo:      classRepo,
		userRepo:       userRepo,
		sessionRepo:    sessionRepo,
		termRepo:       termRepo,
		auditService:   auditService,
	}
}

// EnrollStudentRequest represents student enrollment data
type EnrollStudentRequest struct {
	StudentID uuid.UUID `json:"student_id" validate:"required,uuid"`
	ClassID   uuid.UUID `json:"class_id" validate:"required,uuid"`
	SessionID uuid.UUID `json:"session_id" validate:"required,uuid"`
}

// TransferStudentRequest represents student transfer data
type TransferStudentRequest struct {
	NewClassID uuid.UUID `json:"new_class_id" validate:"required,uuid"`
	Reason     string    `json:"reason" validate:"required,min=10,max=500"`
}

// CreateAndEnrollStudentRequest represents data for creating a new student and enrolling them
type CreateAndEnrollStudentRequest struct {
	// Student details
	Email      string  `json:"email" validate:"required,email"`
	FirstName  string  `json:"first_name" validate:"required,min=1,max=100"`
	LastName   string  `json:"last_name" validate:"required,min=1,max=100"`
	MiddleName *string `json:"middle_name,omitempty" validate:"omitempty,max=100"`
	Phone      *string `json:"phone,omitempty" validate:"omitempty,min=10,max=20"`
	// Enrollment details - only class_id is required
	ClassID uuid.UUID `json:"class_id" validate:"required,uuid"`
}

// CreateAndEnrollStudentResponse contains both the created user and enrollment
type CreateAndEnrollStudentResponse struct {
	User       *domain.User       `json:"user"`
	Enrollment *domain.Enrollment `json:"enrollment"`
}

// EnrollStudent enrolls a student in a class (enforces BR-003)
func (s *EnrollmentService) EnrollStudent(
	ctx context.Context,
	tenantID uuid.UUID,
	req *EnrollStudentRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Enrollment, error) {
	// Verify student exists and belongs to tenant
	student, err := s.userRepo.GetByID(ctx, req.StudentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get student: %w", err)
	}

	if student.TenantID == nil || *student.TenantID != tenantID {
		return nil, fmt.Errorf("student does not belong to this tenant")
	}

	if student.Role != domain.RoleStudent {
		return nil, fmt.Errorf("user is not a student")
	}

	// Verify class exists and belongs to tenant
	class, err := s.classRepo.Get(ctx, req.ClassID)
	if err != nil {
		return nil, fmt.Errorf("failed to get class: %w", err)
	}

	if class.TenantID != tenantID {
		return nil, fmt.Errorf("class does not belong to this tenant")
	}

	// Verify session exists
	session, err := s.sessionRepo.Get(ctx, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if session.TenantID != tenantID {
		return nil, fmt.Errorf("session does not belong to this tenant")
	}

	// BR-003: Database constraint enforces one enrollment per student per session
	// Create enrollment
	enrollment := &domain.Enrollment{
		ID:             uuid.UUID{},
		TenantID:       tenantID,
		StudentID:      req.StudentID,
		ClassID:        req.ClassID,
		SessionID:      req.SessionID,
		Status:         domain.EnrollmentStatusActive,
		EnrollmentDate: time.Now(),
	}

	if err := s.enrollmentRepo.Enroll(ctx, enrollment, nil); err != nil {
		return nil, fmt.Errorf("failed to create enrollment: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionEnrollmentCreated,
		actorID,
		actorRole,
		&tenantID,
		domain.AuditResourceEnrollment,
		enrollment.ID,
		nil,
		enrollment,
		ipAddress,
	)

	return enrollment, nil
}

// CreateAndEnrollStudent creates a new student user and enrolls them in a class
// Only requires class_id - automatically uses the active session and term
func (s *EnrollmentService) CreateAndEnrollStudent(
	ctx context.Context,
	tenantID uuid.UUID,
	req *CreateAndEnrollStudentRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*CreateAndEnrollStudentResponse, error) {
	// Check if email already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with this email already exists")
	}

	// Verify class exists and belongs to tenant
	class, err := s.classRepo.Get(ctx, req.ClassID)
	if err != nil {
		return nil, fmt.Errorf("failed to get class: %w", err)
	}
	if class.TenantID != tenantID {
		return nil, fmt.Errorf("class does not belong to this tenant")
	}

	// Get the active session for this tenant
	activeSession, err := s.sessionRepo.GetActiveSession(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("no active session found: %w", err)
	}

	// Verify there's an active term (enrollment requires an active term)
	_, err = s.termRepo.GetActiveTerm(ctx, activeSession.ID)
	if err != nil {
		return nil, fmt.Errorf("no active term found: %w", err)
	}

	now := time.Now()

	// Create the student user
	student := &domain.User{
		ID:        uuid.New(),
		TenantID:  &tenantID,
		Role:      domain.RoleStudent,
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		MiddleName: req.MiddleName,
		Phone:     req.Phone,
		Status:    domain.UserStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.userRepo.Create(ctx, student); err != nil {
		return nil, fmt.Errorf("failed to create student: %w", err)
	}

	// Audit log for user creation
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionUserCreated,
		actorID,
		actorRole,
		&tenantID,
		domain.AuditResourceUser,
		student.ID,
		nil,
		student,
		ipAddress,
	)

	// Create the enrollment
	enrollment := &domain.Enrollment{
		ID:             uuid.New(),
		TenantID:       tenantID,
		StudentID:      student.ID,
		ClassID:        req.ClassID,
		SessionID:      activeSession.ID,
		Status:         domain.EnrollmentStatusActive,
		EnrollmentDate: now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.enrollmentRepo.Enroll(ctx, enrollment, nil); err != nil {
		// TODO: Consider rolling back user creation in production
		return nil, fmt.Errorf("failed to create enrollment: %w", err)
	}

	// Audit log for enrollment
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionEnrollmentCreated,
		actorID,
		actorRole,
		&tenantID,
		domain.AuditResourceEnrollment,
		enrollment.ID,
		nil,
		enrollment,
		ipAddress,
	)

	// Remove sensitive fields
	student.PasswordHash = ""

	return &CreateAndEnrollStudentResponse{
		User:       student,
		Enrollment: enrollment,
	}, nil
}

// GetEnrollment gets an enrollment by ID
func (s *EnrollmentService) GetEnrollment(ctx context.Context, id uuid.UUID) (*domain.Enrollment, error) {
	enrollment, err := s.enrollmentRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get enrollment: %w", err)
	}
	return enrollment, nil
}

// ListEnrollments lists enrollments with filters
func (s *EnrollmentService) ListEnrollments(
	ctx context.Context,
	tenantID uuid.UUID,
	classID *uuid.UUID,
	sessionID *uuid.UUID,
	status *domain.EnrollmentStatus,
	params repository.PaginationParams,
) ([]*domain.Enrollment, *repository.PaginatedResult, error) {
	var enrollments []*domain.Enrollment
	var total int
	var err error

	if sessionID != nil {
		enrollments, total, err = s.enrollmentRepo.ListBySession(ctx, *sessionID, params)
	} else if classID != nil {
		enrollments, total, err = s.enrollmentRepo.ListByClass(ctx, *classID, params)
	} else {
		return nil, nil, fmt.Errorf("at least one filter (classID or sessionID) is required")
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to list enrollments: %w", err)
	}

	// TODO: Filter by status if provided

	pagination := repository.BuildPaginatedResult(total, params)

	return enrollments, &pagination, nil
}

// TransferStudent transfers a student to a different class (FR-ACA-006)
func (s *EnrollmentService) TransferStudent(
	ctx context.Context,
	enrollmentID uuid.UUID,
	req *TransferStudentRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Enrollment, error) {
	// Get enrollment
	enrollment, err := s.enrollmentRepo.Get(ctx, enrollmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get enrollment: %w", err)
	}

	// Verify new class exists and belongs to tenant
	newClass, err := s.classRepo.Get(ctx, req.NewClassID)
	if err != nil {
		return nil, fmt.Errorf("failed to get new class: %w", err)
	}

	if newClass.TenantID != enrollment.TenantID {
		return nil, fmt.Errorf("new class does not belong to this tenant")
	}

	// Ensure new class is in the same session
	if newClass.SessionID != enrollment.SessionID {
		return nil, fmt.Errorf("cannot transfer to a class in a different session")
	}

	// Store before state
	beforeState := *enrollment

	// Transfer student
	enrollment.Transfer(req.NewClassID, req.Reason)

	if err := s.enrollmentRepo.Update(ctx, enrollment, nil); err != nil {
		return nil, fmt.Errorf("failed to transfer student: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionEnrollmentUpdated,
		actorID,
		actorRole,
		&enrollment.TenantID,
		domain.AuditResourceEnrollment,
		enrollment.ID,
		&beforeState,
		enrollment,
		ipAddress,
	)

	return enrollment, nil
}

// WithdrawStudent withdraws a student from their class
func (s *EnrollmentService) WithdrawStudent(
	ctx context.Context,
	enrollmentID uuid.UUID,
	reason string,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Enrollment, error) {
	// Get enrollment
	enrollment, err := s.enrollmentRepo.Get(ctx, enrollmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get enrollment: %w", err)
	}

	// Store before state
	beforeState := *enrollment

	// Withdraw student
	enrollment.Withdraw(reason)

	if err := s.enrollmentRepo.Update(ctx, enrollment, nil); err != nil {
		return nil, fmt.Errorf("failed to withdraw student: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionEnrollmentWithdrawn,
		actorID,
		actorRole,
		&enrollment.TenantID,
		domain.AuditResourceEnrollment,
		enrollment.ID,
		&beforeState,
		enrollment,
		ipAddress,
	)

	return enrollment, nil
}

// SuspendEnrollment suspends a student's enrollment
func (s *EnrollmentService) SuspendEnrollment(
	ctx context.Context,
	enrollmentID uuid.UUID,
	reason string,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Enrollment, error) {
	// Get enrollment
	enrollment, err := s.enrollmentRepo.Get(ctx, enrollmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get enrollment: %w", err)
	}

	// Store before state
	beforeState := *enrollment

	// Suspend enrollment
	enrollment.Suspend(reason)

	if err := s.enrollmentRepo.Update(ctx, enrollment, nil); err != nil {
		return nil, fmt.Errorf("failed to suspend enrollment: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionEnrollmentSuspended,
		actorID,
		actorRole,
		&enrollment.TenantID,
		domain.AuditResourceEnrollment,
		enrollment.ID,
		&beforeState,
		enrollment,
		ipAddress,
	)

	return enrollment, nil
}

// ReactivateEnrollment reactivates a suspended enrollment
func (s *EnrollmentService) ReactivateEnrollment(
	ctx context.Context,
	enrollmentID uuid.UUID,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Enrollment, error) {
	// Get enrollment
	enrollment, err := s.enrollmentRepo.Get(ctx, enrollmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get enrollment: %w", err)
	}

	// Store before state
	beforeState := *enrollment

	// Reactivate enrollment
	enrollment.Reactivate()

	if err := s.enrollmentRepo.Update(ctx, enrollment, nil); err != nil {
		return nil, fmt.Errorf("failed to reactivate enrollment: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionEnrollmentReactivated,
		actorID,
		actorRole,
		&enrollment.TenantID,
		domain.AuditResourceEnrollment,
		enrollment.ID,
		&beforeState,
		enrollment,
		ipAddress,
	)

	return enrollment, nil
}

// CountActiveStudents counts active students for billing
func (s *EnrollmentService) CountActiveStudents(
	ctx context.Context,
	tenantID uuid.UUID,
	sessionID uuid.UUID,
) (int, error) {
	count, err := s.enrollmentRepo.CountActiveStudents(ctx, tenantID, sessionID)
	if err != nil {
		return 0, fmt.Errorf("failed to count active students: %w", err)
	}
	return count, nil
}
