package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SystemConfigCategory represents categories for grouping system configs
type SystemConfigCategory string

const (
	ConfigCategoryGeneral     SystemConfigCategory = "general"
	ConfigCategoryMaintenance SystemConfigCategory = "maintenance"
	ConfigCategoryBilling     SystemConfigCategory = "billing"
	ConfigCategoryEmail       SystemConfigCategory = "email"
	ConfigCategorySecurity    SystemConfigCategory = "security"
	ConfigCategoryFeatures    SystemConfigCategory = "features"
	ConfigCategoryRateLimit   SystemConfigCategory = "rate_limit"
	ConfigCategoryDefaults    SystemConfigCategory = "defaults"
)

// SystemConfig represents a platform-wide configuration setting
type SystemConfig struct {
	ID          uuid.UUID            `json:"id"`
	Key         string               `json:"key"`
	Value       json.RawMessage      `json:"value"`
	Description *string              `json:"description,omitempty"`
	Category    SystemConfigCategory `json:"category"`
	IsSensitive bool                 `json:"is_sensitive"`
	CreatedBy   *uuid.UUID           `json:"created_by,omitempty"`
	UpdatedBy   *uuid.UUID           `json:"updated_by,omitempty"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

// MaskedValue returns the value or a masked placeholder if sensitive
func (c *SystemConfig) MaskedValue() json.RawMessage {
	if c.IsSensitive {
		return json.RawMessage(`"********"`)
	}
	return c.Value
}

// GetString returns the value as a string (removes JSON quotes)
func (c *SystemConfig) GetString() string {
	var s string
	if err := json.Unmarshal(c.Value, &s); err != nil {
		return string(c.Value)
	}
	return s
}

// GetInt returns the value as an integer
func (c *SystemConfig) GetInt() int {
	var i int
	json.Unmarshal(c.Value, &i)
	return i
}

// GetBool returns the value as a boolean
func (c *SystemConfig) GetBool() bool {
	var b bool
	json.Unmarshal(c.Value, &b)
	return b
}

// GetFloat returns the value as a float64
func (c *SystemConfig) GetFloat() float64 {
	var f float64
	json.Unmarshal(c.Value, &f)
	return f
}

// GetStringSlice returns the value as a string slice
func (c *SystemConfig) GetStringSlice() []string {
	var s []string
	json.Unmarshal(c.Value, &s)
	return s
}

// ValidCategories returns all valid system config categories
func ValidCategories() []SystemConfigCategory {
	return []SystemConfigCategory{
		ConfigCategoryGeneral,
		ConfigCategoryMaintenance,
		ConfigCategoryBilling,
		ConfigCategoryEmail,
		ConfigCategorySecurity,
		ConfigCategoryFeatures,
		ConfigCategoryRateLimit,
		ConfigCategoryDefaults,
	}
}

// IsValidCategory checks if a category is valid
func IsValidCategory(category SystemConfigCategory) bool {
	for _, c := range ValidCategories() {
		if c == category {
			return true
		}
	}
	return false
}
