package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"arc-lms/internal/pkg/errors"
	"arc-lms/internal/pkg/idempotency"
)

// IdempotencyMiddleware implements idempotency for mutating requests using Redis
func IdempotencyMiddleware(store *idempotency.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only apply idempotency to mutating methods
		if c.Request.Method != http.MethodPost &&
			c.Request.Method != http.MethodPut &&
			c.Request.Method != http.MethodPatch &&
			c.Request.Method != http.MethodDelete {
			c.Next()
			return
		}

		// Extract Idempotency-Key header
		idempotencyKey := c.GetHeader("Idempotency-Key")
		if idempotencyKey == "" {
			// Idempotency key is optional; if not provided, proceed normally
			c.Next()
			return
		}

		// Validate idempotency key format (should be a valid identifier)
		if len(idempotencyKey) < 16 || len(idempotencyKey) > 128 {
			errors.BadRequest(c, "Invalid Idempotency-Key format", map[string]interface{}{
				"idempotency_key": "Must be between 16 and 128 characters",
			})
			c.Abort()
			return
		}

		// Extract tenant_id from context (set by AuthMiddleware)
		// For SUPER_ADMIN without tenant context, use a special tenant ID
		var tenantID uuid.UUID
		tenantIDValue, exists := c.Get(ContextKeyTenantID)
		if !exists {
			// Use a special UUID for SUPER_ADMIN global operations
			tenantID = uuid.MustParse("00000000-0000-0000-0000-000000000000")
		} else {
			tenantID = tenantIDValue.(uuid.UUID)
		}

		// Check if this idempotency key already exists
		storedResponse, err := store.Get(c.Request.Context(), tenantID, idempotencyKey)
		if err != nil {
			// Log error but continue processing (don't fail the request)
			c.Next()
			return
		}

		if storedResponse != nil {
			// If response is still being processed (should be rare due to locking)
			// Return 409 Conflict
			if storedResponse.StatusCode == 0 {
				errors.RespondWithError(c, http.StatusConflict, errors.ErrCodeIdempotencyConflict,
					"Request with this Idempotency-Key is already being processed", nil)
				c.Abort()
				return
			}

			// Return the cached response
			c.JSON(storedResponse.StatusCode, storedResponse.ResponseBody)
			c.Abort()
			return
		}

		// Try to acquire the idempotency key
		acquired, err := store.TryAcquire(c.Request.Context(), tenantID, idempotencyKey)
		if err != nil {
			// Log error but continue processing (don't fail the request)
			c.Next()
			return
		}

		if !acquired {
			// Another request is processing this key, return 409
			errors.RespondWithError(c, http.StatusConflict, errors.ErrCodeIdempotencyConflict,
				"Request with this Idempotency-Key is already being processed", nil)
			c.Abort()
			return
		}

		// Create a custom response writer to capture the response
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// Process the request
		c.Next()

		// After processing, store the response if successful (2xx or 3xx status)
		statusCode := c.Writer.Status()
		if statusCode >= 200 && statusCode < 400 {
			// Parse the response body
			var responseBody map[string]interface{}
			if err := json.Unmarshal(blw.body.Bytes(), &responseBody); err == nil {
				// Store the response in Redis
				if err := store.Store(c.Request.Context(), tenantID, idempotencyKey, statusCode, responseBody); err != nil {
					// Log error but don't fail the request (response already sent)
				}
			}
		} else {
			// If request failed, delete the idempotency key so client can retry
			store.Delete(c.Request.Context(), tenantID, idempotencyKey)
		}
	}
}

// bodyLogWriter is a custom response writer that captures the response body
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write captures the response body
func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// WriteString captures the response body
func (w *bodyLogWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// ReadFrom captures the response body
func (w *bodyLogWriter) ReadFrom(r io.Reader) (n int64, err error) {
	n, err = io.Copy(w.body, r)
	if err != nil {
		return n, err
	}
	_, err = w.ResponseWriter.Write(w.body.Bytes())
	return n, err
}
