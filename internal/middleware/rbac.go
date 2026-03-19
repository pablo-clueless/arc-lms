package middleware

import (
	"github.com/gin-gonic/gin"
	"arc-lms/internal/domain"
	"arc-lms/internal/pkg/errors"
)

// RequireRole checks if the authenticated user has one of the specified roles
func RequireRole(allowedRoles ...domain.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract role from context (set by AuthMiddleware)
		roleValue, exists := c.Get(ContextKeyRole)
		if !exists {
			errors.Forbidden(c, "Role information not found in request context")
			c.Abort()
			return
		}

		userRole := domain.Role(roleValue.(string))

		// Check if user has one of the allowed roles
		hasRole := false
		for _, allowedRole := range allowedRoles {
			if userRole == allowedRole {
				hasRole = true
				break
			}
		}

		if !hasRole {
			errors.Forbidden(c, "Insufficient role permissions to access this resource")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermission checks if the authenticated user has a specific permission
// Permission format: "resource:action" (e.g., "tenant:create", "user:delete")
// SUPER_ADMIN bypasses permission checks
func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract role from context
		roleValue, exists := c.Get(ContextKeyRole)
		if !exists {
			errors.Forbidden(c, "Role information not found in request context")
			c.Abort()
			return
		}

		userRole := domain.Role(roleValue.(string))

		// SUPER_ADMIN bypasses all permission checks
		if userRole == domain.RoleSuperAdmin {
			c.Next()
			return
		}

		// Extract permissions from context (set by AuthMiddleware)
		permissionsValue, exists := c.Get(ContextKeyPermissions)
		if !exists {
			errors.Forbidden(c, "Insufficient permissions to access this resource")
			c.Abort()
			return
		}

		permissions, ok := permissionsValue.([]string)
		if !ok {
			errors.Forbidden(c, "Invalid permissions format in request context")
			c.Abort()
			return
		}

		// Check if user has the required permission
		hasPermission := false
		for _, perm := range permissions {
			if perm == permission {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			errors.Forbidden(c, "Insufficient permissions to access this resource")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyPermission checks if the user has at least one of the specified permissions
// SUPER_ADMIN bypasses permission checks
func RequireAnyPermission(requiredPermissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract role from context
		roleValue, exists := c.Get(ContextKeyRole)
		if !exists {
			errors.Forbidden(c, "Role information not found in request context")
			c.Abort()
			return
		}

		userRole := domain.Role(roleValue.(string))

		// SUPER_ADMIN bypasses all permission checks
		if userRole == domain.RoleSuperAdmin {
			c.Next()
			return
		}

		// Extract permissions from context
		permissionsValue, exists := c.Get(ContextKeyPermissions)
		if !exists {
			errors.Forbidden(c, "Insufficient permissions to access this resource")
			c.Abort()
			return
		}

		permissions, ok := permissionsValue.([]string)
		if !ok {
			errors.Forbidden(c, "Invalid permissions format in request context")
			c.Abort()
			return
		}

		// Check if user has any of the required permissions
		hasPermission := false
		for _, perm := range permissions {
			for _, required := range requiredPermissions {
				if perm == required {
					hasPermission = true
					break
				}
			}
			if hasPermission {
				break
			}
		}

		if !hasPermission {
			errors.Forbidden(c, "Insufficient permissions to access this resource")
			c.Abort()
			return
		}

		c.Next()
	}
}
