package handler

import (
	"arc-lms/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetRoleFromContext extracts the role from the gin context.
// JWT stores role as string, so we need to convert it to domain.Role.
func GetRoleFromContext(c *gin.Context) (domain.Role, bool) {
	roleValue, exists := c.Get("role")
	if !exists {
		return "", false
	}
	// Role is stored as string in JWT claims
	roleStr, ok := roleValue.(string)
	if !ok {
		return "", false
	}
	return domain.Role(roleStr), true
}

// GetUserIDFromContext extracts the user ID from the gin context.
func GetUserIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, false
	}
	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		return uuid.Nil, false
	}
	return userID, true
}

// GetTenantIDFromContext extracts the tenant ID from the gin context.
func GetTenantIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	tenantIDValue, exists := c.Get("tenant_id")
	if !exists {
		return uuid.Nil, false
	}
	tenantID, ok := tenantIDValue.(uuid.UUID)
	if !ok {
		return uuid.Nil, false
	}
	return tenantID, true
}
