package handler

import (
	"fmt"
	"net/http"
	"time"

	"arc-lms/internal/domain"
	"arc-lms/internal/pkg/errors"
	"arc-lms/internal/repository"
	"arc-lms/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuditHandler handles audit log HTTP requests
type AuditHandler struct {
	auditService *service.AuditService
}

// NewAuditHandler creates a new audit handler
func NewAuditHandler(auditService *service.AuditService) *AuditHandler {
	return &AuditHandler{
		auditService: auditService,
	}
}

// ListAuditLogsRequest represents query parameters for listing audit logs
type ListAuditLogsRequest struct {
	TenantID     string `form:"tenant_id"`
	ActorUserID  string `form:"actor_user_id"`
	ActorRole    string `form:"actor_role"`
	Action       string `form:"action"`
	ResourceType string `form:"resource_type"`
	ResourceID   string `form:"resource_id"`
	IsSensitive  string `form:"is_sensitive"`
	StartDate    string `form:"start_date"`
	EndDate      string `form:"end_date"`
	Cursor       string `form:"cursor"`
	Limit        int    `form:"limit,default=50"`
}

// ListAuditLogs godoc
// @Summary List audit logs
// @Description Get paginated list of audit logs with optional filters (SUPER_ADMIN and ADMIN only)
// @Tags Audit
// @Security BearerAuth
// @Produce json
// @Param tenant_id query string false "Filter by tenant ID"
// @Param actor_user_id query string false "Filter by actor user ID"
// @Param actor_role query string false "Filter by actor role (SUPER_ADMIN, ADMIN, TUTOR, STUDENT)"
// @Param action query string false "Filter by action type"
// @Param resource_type query string false "Filter by resource type"
// @Param resource_id query string false "Filter by resource ID"
// @Param is_sensitive query bool false "Filter sensitive actions only"
// @Param start_date query string false "Filter from date (RFC3339 format)"
// @Param end_date query string false "Filter to date (RFC3339 format)"
// @Param cursor query string false "Pagination cursor"
// @Param limit query int false "Number of results (default 50, max 100)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Router /audit/logs [get]
func (h *AuditHandler) ListAuditLogs(c *gin.Context) {
	// Get role from context - only SUPER_ADMIN and ADMIN can view audit logs
	roleValue, exists := c.Get("role")
	if !exists {
		errors.Unauthorized(c, "role not found in token")
		return
	}

	roleStr, ok := roleValue.(string)
	if !ok {
		errors.BadRequest(c, "invalid role format", nil)
		return
	}
	role := domain.Role(roleStr)

	if role != domain.RoleSuperAdmin && role != domain.RoleAdmin {
		errors.Forbidden(c, "only SUPER_ADMIN and ADMIN can view audit logs")
		return
	}

	// Parse query parameters
	var req ListAuditLogsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		errors.BadRequest(c, "invalid query parameters", nil)
		return
	}

	// Build filters
	filters := &domain.AuditFilters{}

	// For ADMIN, restrict to their tenant only
	if role == domain.RoleAdmin {
		tenantIDValue, exists := c.Get("tenant_id")
		if !exists {
			errors.BadRequest(c, "tenant ID required for ADMIN", nil)
			return
		}
		tenantID, ok := tenantIDValue.(uuid.UUID)
		if !ok {
			errors.BadRequest(c, "invalid tenant ID format", nil)
			return
		}
		filters.TenantID = &tenantID
	} else if req.TenantID != "" {
		// SUPER_ADMIN can filter by tenant
		tenantID, err := uuid.Parse(req.TenantID)
		if err != nil {
			errors.BadRequest(c, "invalid tenant_id format", nil)
			return
		}
		filters.TenantID = &tenantID
	}

	if req.ActorUserID != "" {
		actorID, err := uuid.Parse(req.ActorUserID)
		if err != nil {
			errors.BadRequest(c, "invalid actor_user_id format", nil)
			return
		}
		filters.ActorUserID = &actorID
	}

	if req.ActorRole != "" {
		actorRole := domain.Role(req.ActorRole)
		filters.ActorRole = &actorRole
	}

	if req.Action != "" {
		action := domain.AuditAction(req.Action)
		filters.Action = &action
	}

	if req.ResourceType != "" {
		resourceType := domain.AuditResourceType(req.ResourceType)
		filters.ResourceType = &resourceType
	}

	if req.ResourceID != "" {
		resourceID, err := uuid.Parse(req.ResourceID)
		if err != nil {
			errors.BadRequest(c, "invalid resource_id format", nil)
			return
		}
		filters.ResourceID = &resourceID
	}

	if req.IsSensitive == "true" {
		isSensitive := true
		filters.IsSensitive = &isSensitive
	} else if req.IsSensitive == "false" {
		isSensitive := false
		filters.IsSensitive = &isSensitive
	}

	if req.StartDate != "" {
		startDate, err := time.Parse(time.RFC3339, req.StartDate)
		if err != nil {
			errors.BadRequest(c, "invalid start_date format, use RFC3339", nil)
			return
		}
		filters.StartDate = &startDate
	}

	if req.EndDate != "" {
		endDate, err := time.Parse(time.RFC3339, req.EndDate)
		if err != nil {
			errors.BadRequest(c, "invalid end_date format, use RFC3339", nil)
			return
		}
		filters.EndDate = &endDate
	}

	// Build pagination params
	params := repository.PaginationParams{
		Limit:     req.Limit,
		SortOrder: "DESC",
	}

	if req.Cursor != "" {
		cursor, err := uuid.Parse(req.Cursor)
		if err != nil {
			errors.BadRequest(c, "invalid cursor format", nil)
			return
		}
		params.Cursor = &cursor
	}

	// Get audit logs
	logs, pagination, err := h.auditService.GetAuditLogs(c.Request.Context(), filters, params)
	if err != nil {
		errors.InternalError(c, "failed to retrieve audit logs")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       logs,
		"pagination": pagination,
	})
}

// GetAuditLog godoc
// @Summary Get audit log by ID
// @Description Get a specific audit log entry (SUPER_ADMIN and ADMIN only)
// @Tags Audit
// @Security BearerAuth
// @Produce json
// @Param id path string true "Audit log ID"
// @Success 200 {object} domain.AuditLog
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Router /audit/logs/{id} [get]
func (h *AuditHandler) GetAuditLog(c *gin.Context) {
	// Get role from context
	roleValue, exists := c.Get("role")
	if !exists {
		errors.Unauthorized(c, "role not found in token")
		return
	}

	roleStr, ok := roleValue.(string)
	if !ok {
		errors.BadRequest(c, "invalid role format", nil)
		return
	}
	role := domain.Role(roleStr)

	if role != domain.RoleSuperAdmin && role != domain.RoleAdmin {
		errors.Forbidden(c, "only SUPER_ADMIN and ADMIN can view audit logs")
		return
	}

	// Parse audit log ID
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		errors.BadRequest(c, "invalid audit log ID format", nil)
		return
	}

	// Get audit log
	auditLog, err := h.auditService.GetAuditLog(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			errors.NotFound(c, "audit log not found")
			return
		}
		errors.InternalError(c, "failed to retrieve audit log")
		return
	}

	// For ADMIN, verify the audit log belongs to their tenant
	if role == domain.RoleAdmin {
		tenantIDValue, exists := c.Get("tenant_id")
		if !exists {
			errors.BadRequest(c, "tenant ID required for ADMIN", nil)
			return
		}
		tenantID, ok := tenantIDValue.(uuid.UUID)
		if !ok {
			errors.BadRequest(c, "invalid tenant ID format", nil)
			return
		}

		if auditLog.TenantID == nil || *auditLog.TenantID != tenantID {
			errors.Forbidden(c, "cannot access audit logs from other tenants")
			return
		}
	}

	c.JSON(http.StatusOK, auditLog)
}

// GetResourceAuditTrail godoc
// @Summary Get audit trail for a resource
// @Description Get all audit logs for a specific resource (SUPER_ADMIN and ADMIN only)
// @Tags Audit
// @Security BearerAuth
// @Produce json
// @Param resource_type query string true "Resource type (USER, TENANT, CLASS, etc.)"
// @Param resource_id query string true "Resource ID"
// @Param cursor query string false "Pagination cursor"
// @Param limit query int false "Number of results (default 50, max 100)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 403 {object} errors.ErrorResponse
// @Router /audit/logs/resource [get]
func (h *AuditHandler) GetResourceAuditTrail(c *gin.Context) {
	// Get role from context
	roleValue, exists := c.Get("role")
	if !exists {
		errors.Unauthorized(c, "role not found in token")
		return
	}

	roleStr, ok := roleValue.(string)
	if !ok {
		errors.BadRequest(c, "invalid role format", nil)
		return
	}
	role := domain.Role(roleStr)

	if role != domain.RoleSuperAdmin && role != domain.RoleAdmin {
		errors.Forbidden(c, "only SUPER_ADMIN and ADMIN can view audit logs")
		return
	}

	// Parse query parameters
	resourceTypeStr := c.Query("resource_type")
	resourceIDStr := c.Query("resource_id")

	if resourceTypeStr == "" || resourceIDStr == "" {
		errors.BadRequest(c, "resource_type and resource_id are required", nil)
		return
	}

	resourceType := domain.AuditResourceType(resourceTypeStr)
	resourceID, err := uuid.Parse(resourceIDStr)
	if err != nil {
		errors.BadRequest(c, "invalid resource_id format", nil)
		return
	}

	// Build pagination params
	params := repository.PaginationParams{
		Limit:     50,
		SortOrder: "DESC",
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		var limit int
		if _, err := fmt.Sscanf(limitStr, "%d", &limit); err == nil && limit > 0 {
			params.Limit = limit
		}
	}

	if cursorStr := c.Query("cursor"); cursorStr != "" {
		cursor, err := uuid.Parse(cursorStr)
		if err != nil {
			errors.BadRequest(c, "invalid cursor format", nil)
			return
		}
		params.Cursor = &cursor
	}

	// Get audit trail
	logs, pagination, err := h.auditService.GetResourceAuditTrail(c.Request.Context(), resourceType, resourceID, params)
	if err != nil {
		errors.InternalError(c, "failed to retrieve resource audit trail")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"resource_type": resourceType,
		"resource_id":   resourceID,
		"data":          logs,
		"pagination":    pagination,
	})
}
