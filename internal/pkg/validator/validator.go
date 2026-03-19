package validator

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	apperrors "arc-lms/internal/pkg/errors"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// ValidateStruct validates a struct and returns validation errors
func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}

// BindAndValidate binds request body and validates it
func BindAndValidate(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		apperrors.BadRequest(c, "Invalid request body", map[string]interface{}{
			"error": err.Error(),
		})
		return false
	}

	if err := ValidateStruct(req); err != nil {
		validationErrors := make(map[string]interface{})
		if errs, ok := err.(validator.ValidationErrors); ok {
			for _, e := range errs {
				validationErrors[e.Field()] = fmt.Sprintf("validation failed on '%s' tag", e.Tag())
			}
		}
		apperrors.ValidationError(c, validationErrors)
		return false
	}

	return true
}
