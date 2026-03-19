package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"arc-lms/internal/pkg/errors"
	"arc-lms/internal/pkg/jwt"
)

// Context keys for storing authenticated user data
const (
	ContextKeyUserID      = "user_id"
	ContextKeyTenantID    = "tenant_id"
	ContextKeyRole        = "role"
	ContextKeyPermissions = "permissions"
)

// AuthMiddleware validates JWT access tokens and extracts claims
func AuthMiddleware(jwtManager *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			errors.Unauthorized(c, "Missing authorization header")
			c.Abort()
			return
		}

		// Check if it's a Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			errors.Unauthorized(c, "Invalid authorization header format. Expected: Bearer <token>")
			c.Abort()
			return
		}

		tokenString := parts[1]
		if tokenString == "" {
			errors.Unauthorized(c, "Missing access token")
			c.Abort()
			return
		}

		// Validate the access token
		claims, err := jwtManager.ValidateAccessToken(tokenString)
		if err != nil {
			// Check if token is expired
			if strings.Contains(err.Error(), "expired") {
				errors.RespondWithError(c, 401, errors.ErrCodeTokenExpired, "Access token has expired", nil)
			} else {
				errors.Unauthorized(c, "Invalid or malformed access token")
			}
			c.Abort()
			return
		}

		// Store claims in the Gin context for downstream middleware and handlers
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyRole, claims.Role)
		c.Set(ContextKeyPermissions, claims.Permissions)

		// TenantID is optional (NULL for SUPER_ADMIN)
		if claims.TenantID != nil {
			c.Set(ContextKeyTenantID, *claims.TenantID)
		}

		// Continue to the next handler
		c.Next()
	}
}
