package errors

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIError represents a standardized API error response
type APIError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ErrorResponse wraps the error in the standard envelope
type ErrorResponse struct {
	Error APIError `json:"error"`
}

// Error codes
const (
	ErrCodeValidation         = "VALIDATION_ERROR"
	ErrCodeUnauthorized       = "UNAUTHORIZED"
	ErrCodeForbidden          = "FORBIDDEN"
	ErrCodeNotFound           = "NOT_FOUND"
	ErrCodeConflict           = "CONFLICT"
	ErrCodeInternal           = "INTERNAL_ERROR"
	ErrCodeBadRequest         = "BAD_REQUEST"
	ErrCodeTermOverlap        = "TERM_OVERLAP"
	ErrCodeSessionActive      = "SESSION_ALREADY_ACTIVE"
	ErrCodeStudentEnrolled    = "STUDENT_ALREADY_ENROLLED"
	ErrCodeTutorDoubleBooking = "TUTOR_DOUBLE_BOOKING"
	ErrCodeClassConflict      = "CLASS_CONFLICT"
	ErrCodeExamWindowClosed   = "EXAMINATION_WINDOW_CLOSED"
	ErrCodeQuizAttemptExists  = "QUIZ_ATTEMPT_ALREADY_EXISTS"
	ErrCodeInvalidCredentials = "INVALID_CREDENTIALS"
	ErrCodeTokenExpired       = "TOKEN_EXPIRED"
	ErrCodeTenantSuspended    = "TENANT_SUSPENDED"
	ErrCodePaymentOverdue     = "PAYMENT_OVERDUE"
	ErrCodeIdempotencyConflict = "IDEMPOTENCY_CONFLICT"
)

// RespondWithError sends a standardized error response
func RespondWithError(c *gin.Context, statusCode int, code, message string, details map[string]interface{}) {
	c.JSON(statusCode, ErrorResponse{
		Error: APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// BadRequest sends a 400 error
func BadRequest(c *gin.Context, message string, details map[string]interface{}) {
	RespondWithError(c, http.StatusBadRequest, ErrCodeBadRequest, message, details)
}

// Unauthorized sends a 401 error
func Unauthorized(c *gin.Context, message string) {
	RespondWithError(c, http.StatusUnauthorized, ErrCodeUnauthorized, message, nil)
}

// Forbidden sends a 403 error
func Forbidden(c *gin.Context, message string) {
	RespondWithError(c, http.StatusForbidden, ErrCodeForbidden, message, nil)
}

// NotFound sends a 404 error
func NotFound(c *gin.Context, message string) {
	RespondWithError(c, http.StatusNotFound, ErrCodeNotFound, message, nil)
}

// Conflict sends a 409 error
func Conflict(c *gin.Context, code, message string, details map[string]interface{}) {
	RespondWithError(c, http.StatusConflict, code, message, details)
}

// InternalError sends a 500 error
func InternalError(c *gin.Context, message string) {
	RespondWithError(c, http.StatusInternalServerError, ErrCodeInternal, message, nil)
}

// ValidationError sends a 422 error
func ValidationError(c *gin.Context, details map[string]interface{}) {
	RespondWithError(c, http.StatusUnprocessableEntity, ErrCodeValidation, "Validation failed", details)
}
