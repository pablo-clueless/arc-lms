package service

import (
	"context"
	"fmt"
	"time"

	"arc-lms/internal/domain"
	"arc-lms/internal/pkg/crypto"
	"arc-lms/internal/repository/postgres"
	"arc-lms/internal/repository"

	"github.com/google/uuid"
)

// TenantService handles tenant management operations
type TenantService struct {
	tenantRepo   *postgres.TenantRepository
	userRepo     *postgres.UserRepository
	auditService *AuditService
}

// NewTenantService creates a new tenant service
func NewTenantService(
	tenantRepo *postgres.TenantRepository,
	userRepo *postgres.UserRepository,
	auditService *AuditService,
) *TenantService {
	return &TenantService{
		tenantRepo:   tenantRepo,
		userRepo:     userRepo,
		auditService: auditService,
	}
}

// CreateTenantRequest represents tenant creation data
type CreateTenantRequest struct {
	Name              string                       `json:"name" validate:"required,min=3,max=200"`
	SchoolType        domain.SchoolType            `json:"school_type" validate:"required,oneof=PRIMARY SECONDARY COMBINED"`
	ContactEmail      string                       `json:"contact_email" validate:"required,email"`
	Address           string                       `json:"address" validate:"required,min=10,max=500"`
	Logo              string                       `json:"logo" validate:"omitempty,url"`
	Configuration     domain.TenantConfiguration   `json:"configuration" validate:"required"`
	BillingContact    domain.BillingContact        `json:"billing_contact" validate:"required"`
	PrincipalAdmin    PrincipalAdminRequest        `json:"principal_admin" validate:"required"`
}

// PrincipalAdminRequest represents the principal admin user to be created
type PrincipalAdminRequest struct {
	Email      string  `json:"email" validate:"required,email"`
	FirstName  string  `json:"first_name" validate:"required,min=1,max=100"`
	LastName   string  `json:"last_name" validate:"required,min=1,max=100"`
	MiddleName *string `json:"middle_name,omitempty" validate:"omitempty,max=100"`
	Phone      *string `json:"phone,omitempty" validate:"omitempty,min=10,max=20"`
	Password   string  `json:"password" validate:"required,min=8"`
}

// UpdateTenantRequest represents tenant update data
type UpdateTenantRequest struct {
	Name           *string                `json:"name,omitempty" validate:"omitempty,min=3,max=200"`
	ContactEmail   *string                `json:"contact_email,omitempty" validate:"omitempty,email"`
	Address        *string                `json:"address,omitempty" validate:"omitempty,min=10,max=500"`
	Logo           *string                `json:"logo,omitempty" validate:"omitempty,url"`
	BillingContact *domain.BillingContact `json:"billing_contact,omitempty"`
}

// TenantFilters represents filters for listing tenants
type TenantFilters struct {
	Status     *domain.TenantStatus `json:"status,omitempty"`
	SchoolType *domain.SchoolType   `json:"school_type,omitempty"`
	SearchTerm *string              `json:"search_term,omitempty"`
}

// CreateTenant creates a new tenant with a principal admin (SUPER_ADMIN only)
func (s *TenantService) CreateTenant(
	ctx context.Context,
	req *CreateTenantRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Tenant, *domain.User, error) {
	// Verify actor is SUPER_ADMIN
	if actorRole != domain.RoleSuperAdmin {
		return nil, nil, fmt.Errorf("only SUPER_ADMIN can create tenants")
	}

	// Check if tenant with same name already exists
// 	existing, err := s.tenantRepo.GetByName(ctx, req.Name)
// 	if err != nil && err != repository.ErrNotFound {
// 		return nil, nil, fmt.Errorf("failed to check tenant name: %w", err)
// 	}
// 	if existing != nil {
// 		return nil, nil, repository.ErrDuplicateKey
// 	}

	// Check if admin email already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, req.PrincipalAdmin.Email)
	if err != nil && err != repository.ErrNotFound {
		return nil, nil, fmt.Errorf("failed to check admin email: %w", err)
	}
	if existingUser != nil {
		return nil, nil, fmt.Errorf("admin email already exists")
	}

	// Hash admin password
	hashedPassword, err := crypto.HashPassword(req.PrincipalAdmin.Password)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create tenant ID
	tenantID := uuid.New()
	adminID := uuid.New()

	// Create tenant
	tenant := &domain.Tenant{
		ID:               tenantID,
		Name:             req.Name,
		SchoolType:       req.SchoolType,
		ContactEmail:     req.ContactEmail,
		Address:          req.Address,
		Logo:             req.Logo,
		Status:           domain.TenantStatusActive,
		Configuration:    req.Configuration,
		BillingContact:   req.BillingContact,
		PrincipalAdminID: adminID,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Create principal admin with all permissions
	admin := &domain.User{
		ID:           adminID,
		TenantID:     &tenantID,
		Role:         domain.RoleAdmin,
		Email:        req.PrincipalAdmin.Email,
		PasswordHash: hashedPassword,
		FirstName:    req.PrincipalAdmin.FirstName,
		LastName:     req.PrincipalAdmin.LastName,
		MiddleName:   req.PrincipalAdmin.MiddleName,
		Phone:        req.PrincipalAdmin.Phone,
		Status:       domain.UserStatusActive,
		Permissions:  []string{"*:*"}, // Full permissions
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// TODO: Use transaction to create both tenant and admin
	// For now, create them separately
	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	if err := s.userRepo.Create(ctx, admin); err != nil {
		// TODO: Rollback tenant creation
		return nil, nil, fmt.Errorf("failed to create admin: %w", err)
	}

	if err != nil {
		return nil, nil, err
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionTenantCreated,
		actorID,
		actorRole,
		nil,
		domain.AuditResourceTenant,
		tenant.ID,
		nil,
		tenant,
		ipAddress,
	)

	// Remove sensitive fields
	admin.PasswordHash = ""

	return tenant, admin, nil
}

// GetTenant gets a tenant by ID
func (s *TenantService) GetTenant(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	tenant, err := s.tenantRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}
	return tenant, nil
}

// UpdateTenant updates a tenant
func (s *TenantService) UpdateTenant(
	ctx context.Context,
	id uuid.UUID,
	req *UpdateTenantRequest,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Tenant, error) {
	// Get existing tenant
	tenant, err := s.tenantRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Store before state for audit
	beforeState := *tenant

	// Update fields
	if req.Name != nil {
		tenant.Name = *req.Name
	}
	if req.ContactEmail != nil {
		tenant.ContactEmail = *req.ContactEmail
	}
	if req.Address != nil {
		tenant.Address = *req.Address
	}
	if req.Logo != nil {
		tenant.Logo = *req.Logo
	}
	if req.BillingContact != nil {
		tenant.BillingContact = *req.BillingContact
	}
	tenant.UpdatedAt = time.Now()

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, fmt.Errorf("failed to update tenant: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionTenantUpdated,
		actorID,
		actorRole,
		&tenant.ID,
		domain.AuditResourceTenant,
		tenant.ID,
		&beforeState,
		tenant,
		ipAddress,
	)

	return tenant, nil
}

// DeleteTenant deletes a tenant (with cascading)
func (s *TenantService) DeleteTenant(
	ctx context.Context,
	id uuid.UUID,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) error {
	// Verify actor is SUPER_ADMIN
	if actorRole != domain.RoleSuperAdmin {
		return fmt.Errorf("only SUPER_ADMIN can delete tenants")
	}

	// Get tenant for audit
	tenant, err := s.tenantRepo.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	// Delete tenant (will cascade to all related entities via database constraints)
	if err := s.tenantRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionTenantUpdated,
		actorID,
		actorRole,
		&tenant.ID,
		domain.AuditResourceTenant,
		tenant.ID,
		tenant,
		nil,
		ipAddress,
	)

	return nil
}

// ListTenants lists all tenants with filters and pagination
func (s *TenantService) ListTenants(
	ctx context.Context,
	filters *TenantFilters,
	params repository.PaginationParams,
) ([]*domain.Tenant, *repository.PaginatedResult, error) {
	var status *domain.TenantStatus
	if filters != nil {
		status = filters.Status
	}

	tenants, err := s.tenantRepo.List(ctx, status, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list tenants: %w", err)
	}

	// Build pagination result
	ids := make([]uuid.UUID, len(tenants))
	for i, tenant := range tenants {
		ids[i] = tenant.ID
	}
	pagination := repository.BuildPaginatedResult(ids, params.Limit)

	return tenants, &pagination, nil
}

// SuspendTenant suspends a tenant for non-payment/violation
func (s *TenantService) SuspendTenant(
	ctx context.Context,
	id uuid.UUID,
	reason string,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Tenant, error) {
	// Verify actor is SUPER_ADMIN
	if actorRole != domain.RoleSuperAdmin {
		return nil, fmt.Errorf("only SUPER_ADMIN can suspend tenants")
	}

	// Get tenant
	tenant, err := s.tenantRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Store before state
	beforeState := *tenant

	// Suspend tenant
	tenant.Suspend(reason)

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, fmt.Errorf("failed to suspend tenant: %w", err)
	}

	// Audit log (marked as sensitive)
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionTenantSuspended,
		actorID,
		actorRole,
		&tenant.ID,
		domain.AuditResourceTenant,
		tenant.ID,
		&beforeState,
		tenant,
		ipAddress,
	)

	return tenant, nil
}

// ReactivateTenant reactivates a suspended tenant
func (s *TenantService) ReactivateTenant(
	ctx context.Context,
	id uuid.UUID,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.Tenant, error) {
	// Verify actor is SUPER_ADMIN
	if actorRole != domain.RoleSuperAdmin {
		return nil, fmt.Errorf("only SUPER_ADMIN can reactivate tenants")
	}

	// Get tenant
	tenant, err := s.tenantRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Store before state
	beforeState := *tenant

	// Reactivate tenant
	tenant.Reactivate()

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, fmt.Errorf("failed to reactivate tenant: %w", err)
	}

	// Audit log (marked as sensitive)
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionTenantReactivated,
		actorID,
		actorRole,
		&tenant.ID,
		domain.AuditResourceTenant,
		tenant.ID,
		&beforeState,
		tenant,
		ipAddress,
	)

	return tenant, nil
}

// GetTenantConfiguration gets tenant configuration
func (s *TenantService) GetTenantConfiguration(ctx context.Context, id uuid.UUID) (*domain.TenantConfiguration, error) {
	tenant, err := s.tenantRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}
	return &tenant.Configuration, nil
}

// UpdateTenantConfiguration updates tenant configuration
func (s *TenantService) UpdateTenantConfiguration(
	ctx context.Context,
	id uuid.UUID,
	config *domain.TenantConfiguration,
	actorID uuid.UUID,
	actorRole domain.Role,
	ipAddress string,
) (*domain.TenantConfiguration, error) {
	// Get tenant
	tenant, err := s.tenantRepo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Store before state
	beforeState := *tenant

	// Update configuration
	tenant.Configuration = *config
	tenant.UpdatedAt = time.Now()

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, fmt.Errorf("failed to update tenant: %w", err)
	}

	// Audit log
	_ = s.auditService.LogAction(
		ctx,
		domain.AuditActionTenantUpdated,
		actorID,
		actorRole,
		&tenant.ID,
		domain.AuditResourceTenant,
		tenant.ID,
		&beforeState,
		tenant,
		ipAddress,
	)

	return &tenant.Configuration, nil
}
