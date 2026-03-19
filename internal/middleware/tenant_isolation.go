package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"arc-lms/internal/domain"
	"arc-lms/internal/pkg/errors"
)

// TenantRepository interface for validating tenant status
type TenantRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error)
}

// TenantIsolationMiddleware ensures proper tenant isolation and validates tenant status
func TenantIsolationMiddleware(tenantRepo TenantRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract role from context (set by AuthMiddleware)
		roleValue, exists := c.Get(ContextKeyRole)
		if !exists {
			errors.Forbidden(c, "Role information not found in request context")
			c.Abort()
			return
		}

		userRole := domain.Role(roleValue.(string))

		// Handle SUPER_ADMIN cross-tenant operations
		if userRole == domain.RoleSuperAdmin {
			// SUPER_ADMIN can optionally specify a tenant_id via query param for cross-tenant operations
			tenantIDParam := c.Query("tenant_id")
			if tenantIDParam != "" {
				tenantID, err := uuid.Parse(tenantIDParam)
				if err != nil {
					errors.BadRequest(c, "Invalid tenant_id format", map[string]interface{}{
						"tenant_id": "Must be a valid UUID",
					})
					c.Abort()
					return
				}

				// Validate the tenant exists and is active
				tenant, err := tenantRepo.GetByID(c.Request.Context(), tenantID)
				if err != nil {
					errors.NotFound(c, "Tenant not found")
					c.Abort()
					return
				}

				if tenant.IsSuspended() {
					errors.RespondWithError(c, 403, errors.ErrCodeTenantSuspended,
						"Cannot access suspended tenant", map[string]interface{}{
							"tenant_id":        tenant.ID.String(),
							"suspension_reason": tenant.SuspensionReason,
						})
					c.Abort()
					return
				}

				// Store the specified tenant_id in context for cross-tenant operations
				c.Set(ContextKeyTenantID, tenantID)
			}
			// If no tenant_id specified, SUPER_ADMIN can proceed without tenant context
			c.Next()
			return
		}

		// For non-SUPER_ADMIN users, tenant_id must exist in the JWT claims
		tenantIDValue, exists := c.Get(ContextKeyTenantID)
		if !exists {
			errors.Forbidden(c, "Tenant information not found in request context")
			c.Abort()
			return
		}

		tenantID, ok := tenantIDValue.(uuid.UUID)
		if !ok {
			errors.Forbidden(c, "Invalid tenant information in request context")
			c.Abort()
			return
		}

		// Validate the tenant exists and is active
		tenant, err := tenantRepo.GetByID(c.Request.Context(), tenantID)
		if err != nil {
			errors.NotFound(c, "Tenant not found")
			c.Abort()
			return
		}

		// Block access if tenant is suspended
		if tenant.IsSuspended() {
			errors.RespondWithError(c, 403, errors.ErrCodeTenantSuspended,
				"Your school account has been suspended", map[string]interface{}{
					"tenant_id":        tenant.ID.String(),
					"suspension_reason": tenant.SuspensionReason,
					"suspended_at":      tenant.SuspendedAt,
				})
			c.Abort()
			return
		}

		// Tenant is valid and active, continue processing
		c.Next()
	}
}
