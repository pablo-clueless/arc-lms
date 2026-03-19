package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"arc-lms/internal/domain"
	"arc-lms/internal/repository"

	"github.com/google/uuid"
)

// UserService handles user management operations
type UserService struct {
	userRepo     repository.UserRepository
	auditService *AuditService
	// emailService would go here for sending invitations
}

// NewUserService creates a new user service
func NewUserService(
	userRepo repository.UserRepository,
	auditService *AuditService,
) *UserService {
	return &UserService{
		userRepo:     userRepo,
		auditService: auditService,
	}
}

// UpdateUserRequest represents user update data
type UpdateUserRequest struct {
	FirstName  *string `json:"first_name,omitempty" validate:"omitempty,min=1,max=100"`
	LastName   *string `json:"last_name,omitempty" validate:"omitempty,min=1,max=100"`
	MiddleName *string `json:"middle_name,omitempty" validate:"omitempty,max=100"`
	Phone      *string `json:"phone,omitempty" validate:"omitempty,min=10,max=20"`
	ProfilePhoto *string `json:"profile_photo,omitempty" validate:"omitempty,url"`
}

// InviteUserRequest represents user invitation data
type InviteUserRequest struct {
	Email      string      `json:"email" validate:"required,email"`
	Role       domain.Role `json:"role" validate:"required,oneof=ADMIN TUTOR STUDENT"`
	FirstName  string      `json:"first_name" validate:"required,min=1,max=100"`
	LastName   string      `json:"last_name" validate:"required,min=1,max=100"`
	MiddleName *string     `json:"middle_name,omitempty" validate:"omitempty,max=100"`
	Phone      *string     `json:"phone,omitempty" validate:"omitempty,min=10,max=20"`
	Permissions []string   `json:"permissions,omitempty"` // Only for ADMIN
}

// UserFilters represents filters for listing users
type UserFilters struct {
	Role       *domain.Role       `json:"role,omitempty"`
	Status     *domain.UserStatus `json:"status,omitempty"`
	SearchTerm *string            `json:"search_term,omitempty"`
}

// ChangePasswordRequest represents password change data
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required,min=8"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// GetUser gets a user by ID
func (s *UserService) GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Remove sensitive fields
	user.PasswordHash = ""

	return user, nil
}

// GetUserByEmail gets a user by email
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Remove sensitive fields
	user.PasswordHash = ""

	return user, nil
}

// UpdateUser updates a user profile
func (s *UserService) UpdateUser(
	ctx context.Context,
	id uuid.UUID,
	req *UpdateUserRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.User, error) {
	// Get existing user
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Store before state for audit
	beforeState := *user

	// Update fields
	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		user.LastName = *req.LastName
	}
	if req.MiddleName != nil {
		user.MiddleName = req.MiddleName
	}
	if req.Phone != nil {
		user.Phone = req.Phone
	}
	if req.ProfilePhoto != nil {
		user.ProfilePhoto = req.ProfilePhoto
	}
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionUserUpdated,
		actorID,
		actorRole,
		user.TenantID,
		domain.AuditResourceUser,
		user.ID,
		&beforeState,
		user,
		ipAddress,
	)

	// Remove sensitive fields
	user.PasswordHash = ""

	return user, nil
}

// DeactivateUser deactivates a user
func (s *UserService) DeactivateUser(
	ctx context.Context,
	id uuid.UUID,
	reason string,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.User, error) {
	// Get user
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Store before state
	beforeState := *user

	// Deactivate user
	user.Deactivate(reason)

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to deactivate user: %w", err)
	}

	// Audit log (marked as sensitive)
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionUserDeactivated,
		actorID,
		actorRole,
		user.TenantID,
		domain.AuditResourceUser,
		user.ID,
		&beforeState,
		user,
		ipAddress,
	)

	// Remove sensitive fields
	user.PasswordHash = ""

	return user, nil
}

// ReactivateUser reactivates a user
func (s *UserService) ReactivateUser(
	ctx context.Context,
	id uuid.UUID,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.User, error) {
	// Get user
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Store before state
	beforeState := *user

	// Reactivate user
	user.Reactivate()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to reactivate user: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionUserReactivated,
		actorID,
		actorRole,
		user.TenantID,
		domain.AuditResourceUser,
		user.ID,
		&beforeState,
		user,
		ipAddress,
	)

	// Remove sensitive fields
	user.PasswordHash = ""

	return user, nil
}

// ListUsers lists users with filters and pagination
func (s *UserService) ListUsers(
	ctx context.Context,
	tenantID uuid.UUID,
	filters *UserFilters,
	params repository.PaginationParams,
) ([]*domain.User, *repository.PaginatedResult, error) {
	users, pagination, err := s.userRepo.ListByTenant(ctx, tenantID, filters, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list users: %w", err)
	}

	// Remove sensitive fields from all users
	for _, user := range users {
		user.PasswordHash = ""
	}

	return users, pagination, nil
}

// InviteUser sends an invitation email to a new user
func (s *UserService) InviteUser(
	ctx context.Context,
	tenantID uuid.UUID,
	req *InviteUserRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.User, error) {
	// Check if email already exists
	existing, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil && err != repository.ErrNotFound {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if existing != nil {
		return nil, repository.ErrDuplicateKey
	}

	// Generate invitation token (32 bytes hex = 64 chars)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate invitation token: %w", err)
	}
	invitationToken := hex.EncodeToString(tokenBytes)

	// Set token expiry (7 days from now)
	expiry := time.Now().Add(7 * 24 * time.Hour)

	// Validate permissions for ADMIN role
	permissions := req.Permissions
	if req.Role == domain.RoleAdmin && len(permissions) == 0 {
		// Default permissions for ADMIN (can be customized)
		permissions = []string{
			"session:*",
			"term:*",
			"class:*",
			"course:*",
			"user:read",
			"user:invite",
			"user:update",
		}
	}

	// Create user in PENDING status
	user := &domain.User{
		ID:               uuid.New(),
		TenantID:         &tenantID,
		Role:             req.Role,
		Email:            req.Email,
		FirstName:        req.FirstName,
		LastName:         req.LastName,
		MiddleName:       req.MiddleName,
		Phone:            req.Phone,
		Status:           domain.UserStatusPending,
		Permissions:      permissions,
		InvitationToken:  &invitationToken,
		InvitationExpiry: &expiry,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// TODO: Send invitation email with token
	// This would be done via an email service (not implemented yet)
	fmt.Printf("Invitation token for %s: %s\n", user.Email, invitationToken)

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionUserCreated,
		actorID,
		actorRole,
		&tenantID,
		domain.AuditResourceUser,
		user.ID,
		nil,
		user,
		ipAddress,
	)

	// Remove sensitive fields
	user.PasswordHash = ""

	return user, nil
}

// UpdateUserPermissions updates ADMIN/SUPER_ADMIN permissions
func (s *UserService) UpdateUserPermissions(
	ctx context.Context,
	id uuid.UUID,
	permissions []string,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.User, error) {
	// Only SUPER_ADMIN and ADMIN can update permissions
	if actorRole != domain.RoleSuperAdmin && actorRole != domain.RoleAdmin {
		return nil, fmt.Errorf("insufficient permissions to update user permissions")
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Only ADMIN and SUPER_ADMIN can have permissions
	if user.Role != domain.RoleAdmin && user.Role != domain.RoleSuperAdmin {
		return nil, fmt.Errorf("only ADMIN and SUPER_ADMIN can have permissions")
	}

	// Store before state
	beforeState := *user

	// Update permissions
	user.Permissions = permissions
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionUserUpdated,
		actorID,
		actorRole,
		user.TenantID,
		domain.AuditResourceUser,
		user.ID,
		&beforeState,
		user,
		ipAddress,
	)

	// Remove sensitive fields
	user.PasswordHash = ""

	return user, nil
}

// RecordLogin records the last login timestamp
func (s *UserService) RecordLogin(ctx context.Context, userID uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	user.RecordLogin()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to record login: %w", err)
	}

	return nil
}

// ChangePassword changes a user's password
func (s *UserService) ChangePassword(
	ctx context.Context,
	userID uuid.UUID,
	req *ChangePasswordRequest,
	ipAddress string,
) error {
	// Get user (with password hash)
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Verify old password
	if !comparePassword(user.PasswordHash, req.OldPassword) {
		return fmt.Errorf("invalid old password")
	}

	// Hash new password
	hashedPassword, err := hashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	user.PasswordHash = hashedPassword
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionUserUpdated,
		user.ID,
		user.Role,
		user.TenantID,
		domain.AuditResourceUser,
		user.ID,
		nil,
		nil,
		ipAddress,
	)

	return nil
}

// Helper functions (these would import from crypto package)
func hashPassword(password string) (string, error) {
	// Import from crypto package
	return "", fmt.Errorf("not implemented - use crypto.HashPassword")
}

func comparePassword(hash, password string) bool {
	// Import from crypto package
	return false
}
