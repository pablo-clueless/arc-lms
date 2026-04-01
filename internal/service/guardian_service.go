package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"arc-lms/internal/domain"
	"arc-lms/internal/repository"
	"arc-lms/internal/repository/postgres"

	"github.com/google/uuid"
)

// GuardianService handles guardian-ward relationship operations
type GuardianService struct {
	guardianRepo   *postgres.GuardianRepository
	userRepo       *postgres.UserRepository
	enrollmentRepo *postgres.EnrollmentRepository
	classRepo      *postgres.ClassRepository
	sessionRepo    *postgres.SessionRepository
	progressRepo   *postgres.ProgressRepository
	invoiceRepo    *postgres.InvoiceRepository
	auditService   *AuditService
}

// NewGuardianService creates a new guardian service
func NewGuardianService(
	guardianRepo *postgres.GuardianRepository,
	userRepo *postgres.UserRepository,
	enrollmentRepo *postgres.EnrollmentRepository,
	classRepo *postgres.ClassRepository,
	sessionRepo *postgres.SessionRepository,
	progressRepo *postgres.ProgressRepository,
	invoiceRepo *postgres.InvoiceRepository,
	auditService *AuditService,
) *GuardianService {
	return &GuardianService{
		guardianRepo:   guardianRepo,
		userRepo:       userRepo,
		enrollmentRepo: enrollmentRepo,
		classRepo:      classRepo,
		sessionRepo:    sessionRepo,
		progressRepo:   progressRepo,
		invoiceRepo:    invoiceRepo,
		auditService:   auditService,
	}
}

// LinkWardRequest represents a request to link a guardian to a student
type LinkWardRequest struct {
	StudentID    uuid.UUID                   `json:"student_id" validate:"required,uuid"`
	Relationship domain.GuardianRelationship `json:"relationship" validate:"required,oneof=FATHER MOTHER GUARDIAN OTHER"`
	IsPrimary    bool                        `json:"is_primary"`
	Notes        *string                     `json:"notes,omitempty" validate:"omitempty,max=500"`
}

// CreateAndLinkGuardianRequest represents a request to create a guardian user and link to a student
type CreateAndLinkGuardianRequest struct {
	// Guardian user details
	Email       string  `json:"email" validate:"required,email"`
	FirstName   string  `json:"first_name" validate:"required,min=2,max=50"`
	LastName    string  `json:"last_name" validate:"required,min=2,max=50"`
	PhoneNumber *string `json:"phone_number,omitempty" validate:"omitempty,max=20"`

	// Relationship details
	StudentID    uuid.UUID                   `json:"student_id" validate:"required,uuid"`
	Relationship domain.GuardianRelationship `json:"relationship" validate:"required,oneof=FATHER MOTHER GUARDIAN OTHER"`
	IsPrimary    bool                        `json:"is_primary"`
	Notes        *string                     `json:"notes,omitempty" validate:"omitempty,max=500"`
}

// CreateAndLinkGuardianResponse contains the created guardian and relationship
type CreateAndLinkGuardianResponse struct {
	Guardian     *domain.GuardianWithDetails `json:"guardian"`
	TempPassword string                      `json:"temp_password"`
}

// CreateAndLinkGuardian creates a new guardian user and links them to a student in one operation
func (s *GuardianService) CreateAndLinkGuardian(
	ctx context.Context,
	tenantID uuid.UUID,
	req *CreateAndLinkGuardianRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*CreateAndLinkGuardianResponse, error) {
	// Verify student user exists and is a STUDENT
	studentUser, err := s.userRepo.GetByID(ctx, req.StudentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get student user: %w", err)
	}
	if studentUser.Role != domain.RoleStudent {
		return nil, fmt.Errorf("user is not a student")
	}
	if studentUser.TenantID == nil || *studentUser.TenantID != tenantID {
		return nil, fmt.Errorf("student does not belong to this tenant")
	}

	// Check if guardian email already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil && err != repository.ErrNotFound {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if existingUser != nil {
		return nil, fmt.Errorf("a user with this email already exists")
	}

	// Generate temporary password (12 characters)
	tempPassBytes := make([]byte, 6)
	if _, err := rand.Read(tempPassBytes); err != nil {
		return nil, fmt.Errorf("failed to generate temp password: %w", err)
	}
	tempPassword := hex.EncodeToString(tempPassBytes)

	// Hash the password
	passwordHash, err := hashPassword(tempPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now()

	// Create guardian user with PARENT role
	guardianUser := &domain.User{
		ID:                      uuid.New(),
		TenantID:                &tenantID,
		Role:                    domain.RoleParent,
		Email:                   req.Email,
		FirstName:               req.FirstName,
		LastName:                req.LastName,
		Phone:                   req.PhoneNumber,
		PasswordHash:            passwordHash,
		Status:                  domain.UserStatusActive,
		NotificationPreferences: domain.DefaultNotificationPreferences(),
		CreatedAt:               now,
		UpdatedAt:               now,
	}

	if err := s.userRepo.Create(ctx, guardianUser); err != nil {
		return nil, fmt.Errorf("failed to create guardian user: %w", err)
	}

	// Create guardian-student relationship
	guardian := &domain.Guardian{
		ID:           uuid.New(),
		TenantID:     tenantID,
		GuardianID:   guardianUser.ID,
		StudentID:    req.StudentID,
		Relationship: req.Relationship,
		IsPrimary:    req.IsPrimary,
		Status:       domain.GuardianStatusActive,
		Notes:        req.Notes,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.guardianRepo.Create(ctx, guardian); err != nil {
		return nil, fmt.Errorf("failed to create guardian relationship: %w", err)
	}

	// Audit log for user creation
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionUserCreated,
		actorID,
		actorRole,
		&tenantID,
		"user",
		guardianUser.ID,
		nil,
		guardianUser,
		ipAddress,
	)

	// Audit log for guardian linking
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionGuardianLinked,
		actorID,
		actorRole,
		&tenantID,
		"guardian",
		guardian.ID,
		nil,
		guardian,
		ipAddress,
	)

	// Remove sensitive data for response
	guardianUser.PasswordHash = ""
	studentUser.PasswordHash = ""

	return &CreateAndLinkGuardianResponse{
		Guardian: &domain.GuardianWithDetails{
			Guardian:     guardian,
			GuardianUser: guardianUser,
			StudentUser:  studentUser,
		},
		TempPassword: tempPassword,
	}, nil
}

// LinkWard links a guardian to a student (ward)
func (s *GuardianService) LinkWard(
	ctx context.Context,
	tenantID uuid.UUID,
	guardianID uuid.UUID,
	req *LinkWardRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.GuardianWithDetails, error) {
	// Verify guardian user exists and is a PARENT
	guardianUser, err := s.userRepo.GetByID(ctx, guardianID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guardian user: %w", err)
	}
	if guardianUser.Role != domain.RoleParent {
		return nil, fmt.Errorf("user is not a parent")
	}
	if guardianUser.TenantID == nil || *guardianUser.TenantID != tenantID {
		return nil, fmt.Errorf("guardian does not belong to this tenant")
	}

	// Verify student user exists and is a STUDENT
	studentUser, err := s.userRepo.GetByID(ctx, req.StudentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get student user: %w", err)
	}
	if studentUser.Role != domain.RoleStudent {
		return nil, fmt.Errorf("user is not a student")
	}
	if studentUser.TenantID == nil || *studentUser.TenantID != tenantID {
		return nil, fmt.Errorf("student does not belong to this tenant")
	}

	// Check if relationship already exists
	existing, err := s.guardianRepo.GetByGuardianAndStudent(ctx, guardianID, req.StudentID)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("guardian-student relationship already exists")
	}

	now := time.Now()
	guardian := &domain.Guardian{
		ID:           uuid.New(),
		TenantID:     tenantID,
		GuardianID:   guardianID,
		StudentID:    req.StudentID,
		Relationship: req.Relationship,
		IsPrimary:    req.IsPrimary,
		Status:       domain.GuardianStatusActive,
		Notes:        req.Notes,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.guardianRepo.Create(ctx, guardian); err != nil {
		return nil, fmt.Errorf("failed to create guardian relationship: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionGuardianLinked,
		actorID,
		actorRole,
		&tenantID,
		"guardian",
		guardian.ID,
		nil,
		guardian,
		ipAddress,
	)

	// Remove sensitive data
	guardianUser.PasswordHash = ""
	studentUser.PasswordHash = ""

	return &domain.GuardianWithDetails{
		Guardian:     guardian,
		GuardianUser: guardianUser,
		StudentUser:  studentUser,
	}, nil
}

// UnlinkWard removes a guardian-student relationship
func (s *GuardianService) UnlinkWard(
	ctx context.Context,
	tenantID uuid.UUID,
	relationshipID uuid.UUID,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) error {
	guardian, err := s.guardianRepo.Get(ctx, relationshipID)
	if err != nil {
		return fmt.Errorf("failed to get guardian relationship: %w", err)
	}

	if guardian.TenantID != tenantID {
		return fmt.Errorf("relationship does not belong to this tenant")
	}

	if err := s.guardianRepo.Delete(ctx, relationshipID); err != nil {
		return fmt.Errorf("failed to delete guardian relationship: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionGuardianUnlinked,
		actorID,
		actorRole,
		&tenantID,
		"guardian",
		relationshipID,
		guardian,
		nil,
		ipAddress,
	)

	return nil
}

// GetWards returns all wards (students) for a guardian
func (s *GuardianService) GetWards(
	ctx context.Context,
	guardianID uuid.UUID,
	params repository.PaginationParams,
) ([]*domain.WardSummary, *repository.PaginatedResult, error) {
	guardians, total, err := s.guardianRepo.ListByGuardian(ctx, guardianID, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list wards: %w", err)
	}

	wards := make([]*domain.WardSummary, 0, len(guardians))
	for _, g := range guardians {
		// Get student user
		student, err := s.userRepo.GetByID(ctx, g.StudentID)
		if err != nil {
			continue
		}
		student.PasswordHash = ""

		ward := &domain.WardSummary{
			Student: student,
		}

		// Get current enrollment for the student
		if student.TenantID != nil {
			enrollment, err := s.enrollmentRepo.GetCurrentByStudent(ctx, g.StudentID, *student.TenantID)
			if err == nil && enrollment != nil {
				ward.Enrollment = enrollment

				// Get class
				class, err := s.classRepo.Get(ctx, enrollment.ClassID)
				if err == nil {
					ward.Class = class
				}

				// Get session
				session, err := s.sessionRepo.Get(ctx, enrollment.SessionID)
				if err == nil {
					ward.Session = session
				}
			}
		}

		wards = append(wards, ward)
	}

	pagination := repository.BuildPaginatedResult(total, params)
	return wards, &pagination, nil
}

// GetGuardians returns all guardians for a student
func (s *GuardianService) GetGuardians(
	ctx context.Context,
	studentID uuid.UUID,
	params repository.PaginationParams,
) ([]*domain.GuardianWithDetails, *repository.PaginatedResult, error) {
	guardians, total, err := s.guardianRepo.ListByStudent(ctx, studentID, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list guardians: %w", err)
	}

	result := make([]*domain.GuardianWithDetails, 0, len(guardians))
	for _, g := range guardians {
		// Get guardian user
		guardianUser, err := s.userRepo.GetByID(ctx, g.GuardianID)
		if err != nil {
			continue
		}
		guardianUser.PasswordHash = ""

		// Get student user
		studentUser, err := s.userRepo.GetByID(ctx, g.StudentID)
		if err != nil {
			continue
		}
		studentUser.PasswordHash = ""

		result = append(result, &domain.GuardianWithDetails{
			Guardian:     g,
			GuardianUser: guardianUser,
			StudentUser:  studentUser,
		})
	}

	pagination := repository.BuildPaginatedResult(total, params)
	return result, &pagination, nil
}

// IsGuardianOfStudent checks if a user is a guardian of a specific student
func (s *GuardianService) IsGuardianOfStudent(ctx context.Context, guardianID, studentID uuid.UUID) (bool, error) {
	return s.guardianRepo.IsGuardianOfStudent(ctx, guardianID, studentID)
}

// GetWardIDs returns all student IDs that a guardian has access to
func (s *GuardianService) GetWardIDs(ctx context.Context, guardianID uuid.UUID) ([]uuid.UUID, error) {
	return s.guardianRepo.GetWardIDs(ctx, guardianID)
}

// WardProgressResponse contains progress information for a ward
type WardProgressResponse struct {
	Ward     *domain.WardSummary    `json:"ward"`
	Progress []*domain.Progress     `json:"progress"`
}

// GetWardProgress returns progress information for a specific ward
func (s *GuardianService) GetWardProgress(
	ctx context.Context,
	guardianID uuid.UUID,
	studentID uuid.UUID,
) (*WardProgressResponse, error) {
	// Verify guardian has access to this student
	isGuardian, err := s.guardianRepo.IsGuardianOfStudent(ctx, guardianID, studentID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify guardian access: %w", err)
	}
	if !isGuardian {
		return nil, fmt.Errorf("you are not a guardian of this student")
	}

	// Get student user
	student, err := s.userRepo.GetByID(ctx, studentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get student: %w", err)
	}
	student.PasswordHash = ""

	ward := &domain.WardSummary{
		Student: student,
	}

	// Get current enrollment
	if student.TenantID != nil {
		enrollment, err := s.enrollmentRepo.GetCurrentByStudent(ctx, studentID, *student.TenantID)
		if err == nil && enrollment != nil {
			ward.Enrollment = enrollment

			class, err := s.classRepo.Get(ctx, enrollment.ClassID)
			if err == nil {
				ward.Class = class
			}

			session, err := s.sessionRepo.Get(ctx, enrollment.SessionID)
			if err == nil {
				ward.Session = session
			}
		}
	}

	// Get progress records
	progress, _, err := s.progressRepo.ListByStudent(ctx, studentID, repository.PaginationParams{Limit: 100})
	if err != nil {
		progress = []*domain.Progress{}
	}

	return &WardProgressResponse{
		Ward:     ward,
		Progress: progress,
	}, nil
}

// WardInvoicesResponse contains invoice information for a ward
type WardInvoicesResponse struct {
	Ward     *domain.WardSummary `json:"ward"`
	Invoices []*domain.Invoice   `json:"invoices"`
}

// GetWardInvoices returns invoices for a specific ward
func (s *GuardianService) GetWardInvoices(
	ctx context.Context,
	guardianID uuid.UUID,
	studentID uuid.UUID,
) (*WardInvoicesResponse, error) {
	// Verify guardian has access to this student
	isGuardian, err := s.guardianRepo.IsGuardianOfStudent(ctx, guardianID, studentID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify guardian access: %w", err)
	}
	if !isGuardian {
		return nil, fmt.Errorf("you are not a guardian of this student")
	}

	// Get student user
	student, err := s.userRepo.GetByID(ctx, studentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get student: %w", err)
	}
	student.PasswordHash = ""

	ward := &domain.WardSummary{
		Student: student,
	}

	// Get invoices for this student
	invoices, _, err := s.invoiceRepo.ListByStudent(ctx, studentID, repository.PaginationParams{Limit: 100})
	if err != nil {
		invoices = []*domain.Invoice{}
	}

	return &WardInvoicesResponse{
		Ward:     ward,
		Invoices: invoices,
	}, nil
}

// ListRelationships lists all guardian relationships for a tenant (admin view)
func (s *GuardianService) ListRelationships(
	ctx context.Context,
	tenantID uuid.UUID,
	params repository.PaginationParams,
) ([]*domain.GuardianWithDetails, *repository.PaginatedResult, error) {
	guardians, total, err := s.guardianRepo.ListByTenant(ctx, tenantID, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list guardian relationships: %w", err)
	}

	result := make([]*domain.GuardianWithDetails, 0, len(guardians))
	for _, g := range guardians {
		// Get guardian user
		guardianUser, err := s.userRepo.GetByID(ctx, g.GuardianID)
		if err != nil {
			continue
		}
		guardianUser.PasswordHash = ""

		// Get student user
		studentUser, err := s.userRepo.GetByID(ctx, g.StudentID)
		if err != nil {
			continue
		}
		studentUser.PasswordHash = ""

		result = append(result, &domain.GuardianWithDetails{
			Guardian:     g,
			GuardianUser: guardianUser,
			StudentUser:  studentUser,
		})
	}

	pagination := repository.BuildPaginatedResult(total, params)
	return result, &pagination, nil
}
