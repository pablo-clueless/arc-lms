package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"arc-lms/internal/domain"
	"arc-lms/internal/pkg/crypto"
	"arc-lms/internal/pkg/jwt"
	"arc-lms/internal/repository"
	"arc-lms/internal/repository/postgres"

	"github.com/google/uuid"
)

// AuthService handles authentication operations
type AuthService struct {
	userRepo     *postgres.UserRepository
	jwtManager   *jwt.Manager
	auditService *AuditService
}

// NewAuthService creates a new authentication service
func NewAuthService(
	userRepo *postgres.UserRepository,
	jwtManager *jwt.Manager,
	auditService *AuditService,
) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		jwtManager:   jwtManager,
		auditService: auditService,
	}
}

// LoginRequest represents login credentials
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// LoginResponse contains authentication tokens and user info
type LoginResponse struct {
	User         *domain.User      `json:"user"`
	TokenPair    *jwt.TokenPair    `json:"token_pair"`
}

// RegisterRequest represents registration data
type RegisterRequest struct {
	Email       string      `json:"email" validate:"required,email"`
	Password    string      `json:"password" validate:"required,min=8"`
	FirstName   string      `json:"first_name" validate:"required,min=1,max=100"`
	LastName    string      `json:"last_name" validate:"required,min=1,max=100"`
	MiddleName  *string     `json:"middle_name,omitempty" validate:"omitempty,max=100"`
	Phone       *string     `json:"phone,omitempty" validate:"omitempty,min=10,max=20"`
	Role        domain.Role `json:"role" validate:"required,oneof=ADMIN TUTOR STUDENT"`
	TenantID    *uuid.UUID  `json:"tenant_id,omitempty" validate:"omitempty,uuid"`
	Permissions []string    `json:"permissions,omitempty"`
}

// PasswordResetRequest represents a password reset request
type PasswordResetRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ResetPasswordRequest represents password reset with token
type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// AcceptInvitationRequest represents invitation acceptance
type AcceptInvitationRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

// Login validates credentials and generates JWT tokens
func (s *AuthService) Login(ctx context.Context, req *LoginRequest, ipAddress string) (*LoginResponse, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, fmt.Errorf("invalid credentials")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user is active
	if !user.IsActive() {
		return nil, fmt.Errorf("user account is not active")
	}

	// Validate password
	if !crypto.ComparePassword(user.PasswordHash, req.Password) {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check if user's tenant is active (if applicable)
	if user.TenantID != nil {
		// Tenant status should be checked by the caller or via middleware
		// For now, we'll assume it's already been validated
	}

	// Generate JWT token pair
	tokenPair, err := s.jwtManager.GenerateTokenPair(
		user.ID,
		user.TenantID,
		string(user.Role),
		user.Permissions,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Record login
	user.RecordLogin()
	if err := s.userRepo.Update(ctx, user); err != nil {
		// Don't fail login if recording fails, just log
		fmt.Printf("failed to record login for user %s: %v\n", user.ID, err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionUserLogin,
		user.ID,
		user.Role,
		user.TenantID,
		domain.AuditResourceUser,
		user.ID,
		nil,
		nil,
		ipAddress,
	)

	// Remove sensitive fields
	user.PasswordHash = ""

	return &LoginResponse{
		User:      user,
		TokenPair: tokenPair,
	}, nil
}

// Register creates a new user account
func (s *AuthService) Register(ctx context.Context, req *RegisterRequest, actorID uuid.UUID, actorRole domain.Role, ipAddress string) (*domain.User, error) {
	// Check if email already exists
	existing, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil && err != repository.ErrNotFound {
		return nil, fmt.Errorf("failed to check email: %w", err)
	}
	if existing != nil {
		return nil, repository.ErrDuplicateKey
	}

	// Hash password
	hashedPassword, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &domain.User{
		ID:           uuid.New(),
		TenantID:     req.TenantID,
		Role:         req.Role,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		MiddleName:   req.MiddleName,
		Phone:        req.Phone,
		Status:       domain.UserStatusActive,
		Permissions:  req.Permissions,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionUserCreated,
		actorID,
		actorRole,
		req.TenantID,
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

// RefreshToken generates a new access token from a refresh token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*jwt.TokenPair, error) {
	// Validate refresh token
	claims, err := s.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Get user to ensure they still exist and are active
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if !user.IsActive() {
		return nil, fmt.Errorf("user account is not active")
	}

	// Generate new token pair
	tokenPair, err := s.jwtManager.GenerateTokenPair(
		user.ID,
		user.TenantID,
		string(user.Role),
		user.Permissions,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return tokenPair, nil
}

// RequestPasswordReset generates a password reset token and sends it via email
func (s *AuthService) RequestPasswordReset(ctx context.Context, req *PasswordResetRequest) error {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		// Don't reveal if email exists or not for security
		if err == repository.ErrNotFound {
			return nil
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Generate reset token (32 bytes hex = 64 chars)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}
	resetToken := hex.EncodeToString(tokenBytes)

	// Set token expiry (30 minutes from now)
	expiry := time.Now().Add(30 * time.Minute)
	user.PasswordResetToken = &resetToken
	user.PasswordResetExpiry = &expiry
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// TODO: Send password reset email with token
	// This would be done via an email service (not implemented yet)
	fmt.Printf("Password reset token for %s: %s\n", user.Email, resetToken)

	return nil
}

// ResetPassword resets a user's password using a reset token
func (s *AuthService) ResetPassword(ctx context.Context, req *ResetPasswordRequest, ipAddress string) error {
	// Get user by reset token
	user, err := s.userRepo.GetByPasswordResetToken(ctx, req.Token)
	if err != nil {
		if err == repository.ErrNotFound {
			return fmt.Errorf("invalid or expired reset token")
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Check if token is expired
	if user.PasswordResetExpiry == nil || time.Now().After(*user.PasswordResetExpiry) {
		return fmt.Errorf("reset token has expired")
	}

	// Hash new password
	hashedPassword, err := crypto.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password and clear reset token
	user.PasswordHash = hashedPassword
	user.PasswordResetToken = nil
	user.PasswordResetExpiry = nil
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
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

// ValidateInvitation validates an invitation token
func (s *AuthService) ValidateInvitation(ctx context.Context, token string) (*domain.User, error) {
	// Get user by invitation token
	user, err := s.userRepo.GetByInvitationToken(ctx, token)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, fmt.Errorf("invalid or expired invitation token")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if token is expired
	if user.InvitationExpiry == nil || time.Now().After(*user.InvitationExpiry) {
		return nil, fmt.Errorf("invitation token has expired")
	}

	// Check if user is in pending status
	if user.Status != domain.UserStatusPending {
		return nil, fmt.Errorf("invitation has already been accepted")
	}

	// Remove sensitive fields
	user.PasswordHash = ""

	return user, nil
}

// AcceptInvitation accepts an invitation and sets the user's password
func (s *AuthService) AcceptInvitation(ctx context.Context, req *AcceptInvitationRequest, ipAddress string) (*domain.User, error) {
	// Validate invitation
	user, err := s.ValidateInvitation(ctx, req.Token)
	if err != nil {
		return nil, err
	}

	// Hash password
	hashedPassword, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Get full user (ValidateInvitation removes password hash)
	fullUser, err := s.userRepo.GetByID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Update user
	fullUser.PasswordHash = hashedPassword
	fullUser.Status = domain.UserStatusActive
	fullUser.InvitationToken = nil
	fullUser.InvitationExpiry = nil
	fullUser.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, fullUser); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionUserUpdated,
		fullUser.ID,
		fullUser.Role,
		fullUser.TenantID,
		domain.AuditResourceUser,
		fullUser.ID,
		nil,
		fullUser,
		ipAddress,
	)

	// Remove sensitive fields
	fullUser.PasswordHash = ""

	return fullUser, nil
}
