package handler

import (
	"net/http"

	"arc-lms/internal/domain"
	"arc-lms/internal/pkg/errors"
	"arc-lms/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DashboardHandler handles dashboard HTTP requests
type DashboardHandler struct {
	dashboardService *service.DashboardService
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(dashboardService *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{
		dashboardService: dashboardService,
	}
}

// GetDashboard godoc
// @Summary Get dashboard data
// @Description Get role-specific dashboard data for the authenticated user
// @Tags Dashboard
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{} "Dashboard data (structure varies by role)"
// @Failure 401 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /dashboard [get]
func (h *DashboardHandler) GetDashboard(c *gin.Context) {
	// Get user info from JWT claims (set by auth middleware)
	userIDValue, exists := c.Get("user_id")
	if !exists {
		errors.Unauthorized(c, "user not authenticated")
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		errors.BadRequest(c, "invalid user ID format", nil)
		return
	}

	// Get role from context
	roleValue, exists := c.Get("role")
	if !exists {
		errors.Unauthorized(c, "role not found in token")
		return
	}

	role, ok := roleValue.(domain.Role)
	if !ok {
		errors.BadRequest(c, "invalid role format", nil)
		return
	}

	// Get tenant ID from context (optional for SUPER_ADMIN)
	var tenantID *uuid.UUID
	if tenantIDValue, exists := c.Get("tenant_id"); exists {
		if tid, ok := tenantIDValue.(uuid.UUID); ok {
			tenantID = &tid
		}
	}

	// Get dashboard data from service
	dashboard, err := h.dashboardService.GetDashboard(c.Request.Context(), userID, tenantID, role)
	if err != nil {
		errors.InternalError(c, "failed to fetch dashboard data")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"role":      role,
		"dashboard": dashboard,
	})
}
