package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"regexp"
	"time"

	"arc-lms/internal/pkg/metrics"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var (
	// Sensitive fields to mask in logs
	sensitiveFields = []string{"password", "token", "secret", "authorization", "api_key", "access_token", "refresh_token"}
	// Regex patterns for masking sensitive data
	sensitivePatterns = []*regexp.Regexp{
		regexp.MustCompile(`("password"\s*:\s*)"[^"]*"`),
		regexp.MustCompile(`("token"\s*:\s*)"[^"]*"`),
		regexp.MustCompile(`("secret"\s*:\s*)"[^"]*"`),
		regexp.MustCompile(`("authorization"\s*:\s*)"[^"]*"`),
		regexp.MustCompile(`(Bearer\s+)[^\s]+`),
	}
)

// LoggerMiddleware provides structured request logging with JSON format
func LoggerMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate request ID for tracing
		requestID := uuid.New().String()
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// Start timer
		startTime := time.Now()

		// Extract initial request data
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery
		method := c.Request.Method
		clientIP := c.ClientIP()

		// Extract user context if available
		var userID, tenantID, role string
		if userIDValue, exists := c.Get(ContextKeyUserID); exists {
			if uid, ok := userIDValue.(uuid.UUID); ok {
				userID = uid.String()
			}
		}
		if tenantIDValue, exists := c.Get(ContextKeyTenantID); exists {
			if tid, ok := tenantIDValue.(uuid.UUID); ok {
				tenantID = tid.String()
			}
		}
		if roleValue, exists := c.Get(ContextKeyRole); exists {
			if r, ok := roleValue.(string); ok {
				role = r
			}
		}

		// Capture request body for error logging (only for non-GET requests)
		var requestBody []byte
		if method != "GET" && method != "DELETE" {
			if c.Request.Body != nil {
				requestBody, _ = io.ReadAll(c.Request.Body)
				// Restore the body for downstream handlers
				c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
			}
		}

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(startTime)
		statusCode := c.Writer.Status()

		// Record metrics
		isError := statusCode >= 400
		metrics.GetCollector().RecordRequest(latency.Milliseconds(), isError)

		// Build log fields
		fields := logrus.Fields{
			"request_id": requestID,
			"method":     method,
			"path":       path,
			"query":      rawQuery,
			"status":     statusCode,
			"latency_ms": latency.Milliseconds(),
			"ip":         clientIP,
			"user_agent": c.Request.UserAgent(),
		}

		// Add user context if available
		if userID != "" {
			fields["user_id"] = userID
		}
		if tenantID != "" {
			fields["tenant_id"] = tenantID
		}
		if role != "" {
			fields["role"] = role
		}

		// Log based on status code
		if statusCode >= 500 {
			// Server error - log with error level and include request body (masked)
			if len(requestBody) > 0 {
				maskedBody := maskSensitiveData(string(requestBody))
				fields["request_body"] = maskedBody
			}

			// Include error details if available
			if len(c.Errors) > 0 {
				fields["errors"] = c.Errors.String()
			}

			logger.WithFields(fields).Error("Internal server error")
		} else if statusCode >= 400 {
			// Client error - log with warning level
			if len(c.Errors) > 0 {
				fields["errors"] = c.Errors.String()
			}
			logger.WithFields(fields).Warn("Client error")
		} else {
			// Success - log with info level
			logger.WithFields(fields).Info("Request completed")
		}
	}
}

// maskSensitiveData masks sensitive fields in JSON strings
func maskSensitiveData(data string) string {
	masked := data

	// Apply regex patterns to mask sensitive data
	for _, pattern := range sensitivePatterns {
		masked = pattern.ReplaceAllString(masked, `$1"***MASKED***"`)
	}

	return masked
}

// RequestBodyLogger captures and logs request body for debugging (use sparingly)
func RequestBodyLogger(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only log for non-GET requests
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" {
			c.Next()
			return
		}

		// Read body
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			logger.WithError(err).Error("Failed to read request body")
			c.Next()
			return
		}

		// Restore the body for downstream handlers
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		// Try to parse as JSON for better logging
		var jsonBody map[string]interface{}
		if err := json.Unmarshal(body, &jsonBody); err == nil {
			// Mask sensitive fields
			maskSensitiveFields(jsonBody)
			logger.WithFields(logrus.Fields{
				"request_body": jsonBody,
				"path":         c.Request.URL.Path,
				"method":       c.Request.Method,
			}).Debug("Request body")
		} else {
			// Log as string if not JSON (masked)
			logger.WithFields(logrus.Fields{
				"request_body": maskSensitiveData(string(body)),
				"path":         c.Request.URL.Path,
				"method":       c.Request.Method,
			}).Debug("Request body")
		}

		c.Next()
	}
}

// maskSensitiveFields masks sensitive fields in a map
func maskSensitiveFields(data map[string]interface{}) {
	for key, value := range data {
		// Check if key is sensitive
		for _, sensitiveKey := range sensitiveFields {
			if key == sensitiveKey {
				data[key] = "***MASKED***"
				break
			}
		}

		// Recursively mask nested objects
		if nested, ok := value.(map[string]interface{}); ok {
			maskSensitiveFields(nested)
		}

		// Recursively mask arrays of objects
		if arr, ok := value.([]interface{}); ok {
			for _, item := range arr {
				if nestedMap, ok := item.(map[string]interface{}); ok {
					maskSensitiveFields(nestedMap)
				}
			}
		}
	}
}
