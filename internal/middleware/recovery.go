package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"arc-lms/internal/pkg/errors"
)

// RecoveryMiddleware recovers from panics and returns a standard error response
func RecoveryMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Get stack trace
				stackTrace := string(debug.Stack())

				// Extract request information
				path := c.Request.URL.Path
				method := c.Request.Method
				clientIP := c.ClientIP()

				// Get request ID if available
				requestID := ""
				if reqID, exists := c.Get("request_id"); exists {
					if id, ok := reqID.(string); ok {
						requestID = id
					}
				}

				// Get user context if available
				var userID, tenantID string
				if userIDValue, exists := c.Get(ContextKeyUserID); exists {
					userID = fmt.Sprintf("%v", userIDValue)
				}
				if tenantIDValue, exists := c.Get(ContextKeyTenantID); exists {
					tenantID = fmt.Sprintf("%v", tenantIDValue)
				}

				// Log the panic with full context
				logger.WithFields(logrus.Fields{
					"request_id":  requestID,
					"method":      method,
					"path":        path,
					"ip":          clientIP,
					"user_id":     userID,
					"tenant_id":   tenantID,
					"panic":       err,
					"stack_trace": stackTrace,
				}).Error("Panic recovered in request handler")

				// Check if headers have already been written
				if !c.Writer.Written() {
					// Return a standard error response
					errors.InternalError(c, "An unexpected error occurred. Please try again later.")
				}

				// Abort the request to prevent further processing
				c.Abort()
			}
		}()

		c.Next()
	}
}

// RecoveryWithCustomHandler recovers from panics with a custom handler function
func RecoveryWithCustomHandler(logger *logrus.Logger, handler func(c *gin.Context, err interface{})) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Get stack trace
				stackTrace := string(debug.Stack())

				// Extract request information
				path := c.Request.URL.Path
				method := c.Request.Method
				clientIP := c.ClientIP()

				// Get request ID if available
				requestID := ""
				if reqID, exists := c.Get("request_id"); exists {
					if id, ok := reqID.(string); ok {
						requestID = id
					}
				}

				// Log the panic with full context
				logger.WithFields(logrus.Fields{
					"request_id":  requestID,
					"method":      method,
					"path":        path,
					"ip":          clientIP,
					"panic":       err,
					"stack_trace": stackTrace,
				}).Error("Panic recovered in request handler")

				// Call custom handler
				if handler != nil {
					handler(c, err)
				} else {
					// Fallback to default error response
					if !c.Writer.Written() {
						errors.InternalError(c, "An unexpected error occurred. Please try again later.")
					}
				}

				// Abort the request
				c.Abort()
			}
		}()

		c.Next()
	}
}
