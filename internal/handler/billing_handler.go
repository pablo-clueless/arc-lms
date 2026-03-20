package handler

import (
	"net/http"

	"arc-lms/internal/domain"
	"arc-lms/internal/pkg/errors"
	"arc-lms/internal/pkg/validator"
	"arc-lms/internal/repository"
	"arc-lms/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// BillingHandler handles billing HTTP requests
type BillingHandler struct {
	billingService *service.BillingService
}

// NewBillingHandler creates a new billing handler
func NewBillingHandler(billingService *service.BillingService) *BillingHandler {
	return &BillingHandler{
		billingService: billingService,
	}
}

// ListInvoices godoc
// @Summary List invoices
// @Description List invoices for a tenant (admins see their tenant, superadmin sees all)
// @Tags Billing
// @Security BearerAuth
// @Produce json
// @Param status query string false "Filter by status (PENDING, PAID, OVERDUE, DISPUTED, VOIDED)"
// @Param cursor query string false "Pagination cursor"
// @Param limit query int false "Number of results"
// @Success 200 {object} map[string]interface{}
// @Router /billing/invoices [get]
func (h *BillingHandler) ListInvoices(c *gin.Context) {
	tenantID, _, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	var status *domain.InvoiceStatus
	if statusStr := c.Query("status"); statusStr != "" {
		s := domain.InvoiceStatus(statusStr)
		status = &s
	}

	params := repository.PaginationParams{Limit: 50, SortOrder: "DESC"}
	if cursorStr := c.Query("cursor"); cursorStr != "" {
		cursor, _ := uuid.Parse(cursorStr)
		params.Cursor = &cursor
	}

	// SuperAdmin sees all invoices, others see only their tenant
	var tenantIDPtr *uuid.UUID
	if role != domain.RoleSuperAdmin {
		tenantIDPtr = &tenantID
	}

	invoices, pagination, err := h.billingService.ListInvoices(c.Request.Context(), tenantIDPtr, status, params)
	if err != nil {
		errors.InternalError(c, "failed to list invoices")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": invoices, "pagination": pagination})
}

// GetInvoice godoc
// @Summary Get an invoice
// @Description Get invoice details by ID
// @Tags Billing
// @Security BearerAuth
// @Produce json
// @Param id path string true "Invoice ID"
// @Success 200 {object} domain.Invoice
// @Router /billing/invoices/{id} [get]
func (h *BillingHandler) GetInvoice(c *gin.Context) {
	tenantID, _, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid invoice ID", nil)
		return
	}

	invoice, err := h.billingService.GetInvoice(c.Request.Context(), id)
	if err != nil {
		errors.NotFound(c, "invoice not found")
		return
	}

	// Non-superadmins can only view their own tenant's invoices
	if role != domain.RoleSuperAdmin && invoice.TenantID != tenantID {
		errors.Forbidden(c, "you can only view your own invoices")
		return
	}

	c.JSON(http.StatusOK, invoice)
}

// MarkInvoiceAsPaid godoc
// @Summary Mark an invoice as paid
// @Description Mark an invoice as paid with payment details
// @Tags Billing
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Param request body service.MarkInvoicePaidRequest true "Payment data"
// @Success 200 {object} domain.Invoice
// @Router /billing/invoices/{id}/pay [post]
func (h *BillingHandler) MarkInvoiceAsPaid(c *gin.Context) {
	_, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid invoice ID", nil)
		return
	}

	var req service.MarkInvoicePaidRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	invoice, err := h.billingService.MarkInvoiceAsPaid(
		c.Request.Context(),
		id,
		&req,
		userID,
		role,
		c.ClientIP(),
	)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, invoice)
}

// DisputeInvoice godoc
// @Summary Dispute an invoice
// @Description Mark an invoice as disputed with a reason
// @Tags Billing
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Param request body service.DisputeInvoiceRequest true "Dispute reason"
// @Success 200 {object} domain.Invoice
// @Router /billing/invoices/{id}/dispute [post]
func (h *BillingHandler) DisputeInvoice(c *gin.Context) {
	tenantID, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid invoice ID", nil)
		return
	}

	// Only admins can dispute their tenant's invoices
	if role != domain.RoleAdmin && role != domain.RoleSuperAdmin {
		errors.Forbidden(c, "only admins can dispute invoices")
		return
	}

	// Verify invoice belongs to tenant
	invoice, err := h.billingService.GetInvoice(c.Request.Context(), id)
	if err != nil {
		errors.NotFound(c, "invoice not found")
		return
	}

	if role != domain.RoleSuperAdmin && invoice.TenantID != tenantID {
		errors.Forbidden(c, "you can only dispute your own invoices")
		return
	}

	var req service.DisputeInvoiceRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	updatedInvoice, err := h.billingService.DisputeInvoice(
		c.Request.Context(),
		id,
		&req,
		userID,
		role,
		c.ClientIP(),
	)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, updatedInvoice)
}

// VoidInvoice godoc
// @Summary Void an invoice
// @Description Void an invoice (SuperAdmin only)
// @Tags Billing
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Invoice ID"
// @Param request body service.VoidInvoiceRequest true "Void reason"
// @Success 200 {object} domain.Invoice
// @Router /billing/invoices/{id}/void [post]
func (h *BillingHandler) VoidInvoice(c *gin.Context) {
	_, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	// Only SuperAdmin can void invoices
	if role != domain.RoleSuperAdmin {
		errors.Forbidden(c, "only super admins can void invoices")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid invoice ID", nil)
		return
	}

	var req service.VoidInvoiceRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	invoice, err := h.billingService.VoidInvoice(
		c.Request.Context(),
		id,
		&req,
		userID,
		c.ClientIP(),
	)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, invoice)
}

// ApplyBillingAdjustment godoc
// @Summary Apply a billing adjustment
// @Description Apply a credit, discount, charge, or refund (SuperAdmin only)
// @Tags Billing
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body service.ApplyBillingAdjustmentRequest true "Adjustment data"
// @Success 201 {object} domain.BillingAdjustment
// @Router /billing/adjustments [post]
func (h *BillingHandler) ApplyBillingAdjustment(c *gin.Context) {
	_, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	// Only SuperAdmin can apply adjustments
	if role != domain.RoleSuperAdmin {
		errors.Forbidden(c, "only super admins can apply billing adjustments")
		return
	}

	var req service.ApplyBillingAdjustmentRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	adjustment, err := h.billingService.ApplyBillingAdjustment(
		c.Request.Context(),
		&req,
		userID,
		c.ClientIP(),
	)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusCreated, adjustment)
}

// ListBillingAdjustments godoc
// @Summary List billing adjustments
// @Description List billing adjustments for a tenant
// @Tags Billing
// @Security BearerAuth
// @Produce json
// @Param tenant_id query string false "Tenant ID (SuperAdmin only)"
// @Success 200 {object} map[string]interface{}
// @Router /billing/adjustments [get]
func (h *BillingHandler) ListBillingAdjustments(c *gin.Context) {
	tenantID, _, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	// SuperAdmin can query any tenant
	targetTenantID := tenantID
	if role == domain.RoleSuperAdmin {
		if tenantIDStr := c.Query("tenant_id"); tenantIDStr != "" {
			parsed, err := uuid.Parse(tenantIDStr)
			if err == nil {
				targetTenantID = parsed
			}
		}
	}

	params := repository.PaginationParams{Limit: 50}

	adjustments, err := h.billingService.ListBillingAdjustments(c.Request.Context(), targetTenantID, params)
	if err != nil {
		errors.InternalError(c, "failed to list adjustments")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": adjustments})
}

// GetBillingMetrics godoc
// @Summary Get billing metrics
// @Description Get platform-wide billing metrics (SuperAdmin only)
// @Tags Billing
// @Security BearerAuth
// @Produce json
// @Success 200 {object} postgres.BillingMetrics
// @Router /billing/metrics [get]
func (h *BillingHandler) GetBillingMetrics(c *gin.Context) {
	role := h.getUserRole(c)

	if role != domain.RoleSuperAdmin {
		errors.Forbidden(c, "only super admins can view billing metrics")
		return
	}

	metrics, err := h.billingService.GetBillingMetrics(c.Request.Context())
	if err != nil {
		errors.InternalError(c, "failed to get billing metrics")
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// GetTenantBillingStats godoc
// @Summary Get tenant billing statistics
// @Description Get billing statistics for a tenant
// @Tags Billing
// @Security BearerAuth
// @Produce json
// @Param tenant_id query string false "Tenant ID (SuperAdmin only)"
// @Success 200 {object} postgres.TenantBillingStats
// @Router /billing/stats [get]
func (h *BillingHandler) GetTenantBillingStats(c *gin.Context) {
	tenantID, _, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	targetTenantID := tenantID
	if role == domain.RoleSuperAdmin {
		if tenantIDStr := c.Query("tenant_id"); tenantIDStr != "" {
			parsed, err := uuid.Parse(tenantIDStr)
			if err == nil {
				targetTenantID = parsed
			}
		}
	}

	stats, err := h.billingService.GetTenantBillingStats(c.Request.Context(), targetTenantID)
	if err != nil {
		errors.InternalError(c, "failed to get billing stats")
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetUpcomingInvoices godoc
// @Summary Get upcoming invoices
// @Description Get invoices due within specified days
// @Tags Billing
// @Security BearerAuth
// @Produce json
// @Param days query int false "Days to look ahead (default 30)"
// @Success 200 {object} map[string]interface{}
// @Router /billing/invoices/upcoming [get]
func (h *BillingHandler) GetUpcomingInvoices(c *gin.Context) {
	role := h.getUserRole(c)

	if role != domain.RoleSuperAdmin {
		errors.Forbidden(c, "only super admins can view upcoming invoices")
		return
	}

	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		// Simple parse, ignore errors
		if _, err := c.GetQuery("days"); err {
			days = 30
		}
	}

	params := repository.PaginationParams{Limit: 50}

	invoices, err := h.billingService.GetUpcomingInvoices(c.Request.Context(), days, params)
	if err != nil {
		errors.InternalError(c, "failed to get upcoming invoices")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": invoices})
}

// GetOverdueInvoices godoc
// @Summary Get overdue invoices
// @Description Get all overdue invoices
// @Tags Billing
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /billing/invoices/overdue [get]
func (h *BillingHandler) GetOverdueInvoices(c *gin.Context) {
	role := h.getUserRole(c)

	if role != domain.RoleSuperAdmin {
		errors.Forbidden(c, "only super admins can view overdue invoices")
		return
	}

	params := repository.PaginationParams{Limit: 50}

	invoices, err := h.billingService.GetOverdueInvoices(c.Request.Context(), params)
	if err != nil {
		errors.InternalError(c, "failed to get overdue invoices")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": invoices})
}

// ListSubscriptions godoc
// @Summary List subscriptions
// @Description List all subscriptions (SuperAdmin only)
// @Tags Billing
// @Security BearerAuth
// @Produce json
// @Param status query string false "Filter by status (ACTIVE, PAYMENT_OVERDUE, SUSPENDED, CANCELLED)"
// @Success 200 {object} map[string]interface{}
// @Router /billing/subscriptions [get]
func (h *BillingHandler) ListSubscriptions(c *gin.Context) {
	role := h.getUserRole(c)

	if role != domain.RoleSuperAdmin {
		errors.Forbidden(c, "only super admins can view subscriptions")
		return
	}

	var status *domain.SubscriptionStatus
	if statusStr := c.Query("status"); statusStr != "" {
		s := domain.SubscriptionStatus(statusStr)
		status = &s
	}

	params := repository.PaginationParams{Limit: 50, SortOrder: "DESC"}

	subscriptions, pagination, err := h.billingService.ListSubscriptions(c.Request.Context(), status, params)
	if err != nil {
		errors.InternalError(c, "failed to list subscriptions")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": subscriptions, "pagination": pagination})
}

// GetSubscription godoc
// @Summary Get a subscription
// @Description Get subscription details by ID
// @Tags Billing
// @Security BearerAuth
// @Produce json
// @Param id path string true "Subscription ID"
// @Success 200 {object} domain.Subscription
// @Router /billing/subscriptions/{id} [get]
func (h *BillingHandler) GetSubscription(c *gin.Context) {
	tenantID, _, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid subscription ID", nil)
		return
	}

	subscription, err := h.billingService.GetSubscription(c.Request.Context(), id)
	if err != nil {
		errors.NotFound(c, "subscription not found")
		return
	}

	// Non-superadmins can only view their own tenant's subscription
	if role != domain.RoleSuperAdmin && subscription.TenantID != tenantID {
		errors.Forbidden(c, "you can only view your own subscription")
		return
	}

	c.JSON(http.StatusOK, subscription)
}

// GetTenantSubscription godoc
// @Summary Get tenant's subscription
// @Description Get the active subscription for a tenant
// @Tags Billing
// @Security BearerAuth
// @Produce json
// @Param tenant_id query string false "Tenant ID (SuperAdmin only)"
// @Success 200 {object} domain.Subscription
// @Router /billing/subscriptions/tenant [get]
func (h *BillingHandler) GetTenantSubscription(c *gin.Context) {
	tenantID, _, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	targetTenantID := tenantID
	if role == domain.RoleSuperAdmin {
		if tenantIDStr := c.Query("tenant_id"); tenantIDStr != "" {
			parsed, err := uuid.Parse(tenantIDStr)
			if err == nil {
				targetTenantID = parsed
			}
		}
	}

	subscription, err := h.billingService.GetTenantSubscription(c.Request.Context(), targetTenantID)
	if err != nil {
		errors.NotFound(c, "subscription not found")
		return
	}

	c.JSON(http.StatusOK, subscription)
}

// CancelSubscriptionRequest represents a request to cancel a subscription
type CancelSubscriptionRequest struct {
	Reason string `json:"reason" validate:"required,min=10,max=500"`
}

// CancelSubscription godoc
// @Summary Cancel a subscription
// @Description Cancel a subscription (SuperAdmin only)
// @Tags Billing
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Subscription ID"
// @Param request body CancelSubscriptionRequest true "Cancellation reason"
// @Success 200 {object} domain.Subscription
// @Router /billing/subscriptions/{id}/cancel [post]
func (h *BillingHandler) CancelSubscription(c *gin.Context) {
	_, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	if role != domain.RoleSuperAdmin {
		errors.Forbidden(c, "only super admins can cancel subscriptions")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid subscription ID", nil)
		return
	}

	var req CancelSubscriptionRequest
	if !validator.BindAndValidate(c, &req) {
		return
	}

	subscription, err := h.billingService.CancelSubscription(
		c.Request.Context(),
		id,
		req.Reason,
		userID,
		c.ClientIP(),
	)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, subscription)
}

// ReactivateSubscription godoc
// @Summary Reactivate a subscription
// @Description Reactivate a suspended subscription (SuperAdmin only)
// @Tags Billing
// @Security BearerAuth
// @Produce json
// @Param id path string true "Subscription ID"
// @Success 200 {object} domain.Subscription
// @Router /billing/subscriptions/{id}/reactivate [post]
func (h *BillingHandler) ReactivateSubscription(c *gin.Context) {
	_, userID, ok := h.getTenantAndUserID(c)
	if !ok {
		return
	}

	role := h.getUserRole(c)

	if role != domain.RoleSuperAdmin {
		errors.Forbidden(c, "only super admins can reactivate subscriptions")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errors.BadRequest(c, "invalid subscription ID", nil)
		return
	}

	subscription, err := h.billingService.ReactivateSubscription(
		c.Request.Context(),
		id,
		userID,
		c.ClientIP(),
	)
	if err != nil {
		errors.BadRequest(c, err.Error(), nil)
		return
	}

	c.JSON(http.StatusOK, subscription)
}

// GetSubscriptionStatistics godoc
// @Summary Get subscription statistics
// @Description Get subscription statistics (SuperAdmin only)
// @Tags Billing
// @Security BearerAuth
// @Produce json
// @Success 200 {object} postgres.SubscriptionStatistics
// @Router /billing/subscriptions/stats [get]
func (h *BillingHandler) GetSubscriptionStatistics(c *gin.Context) {
	role := h.getUserRole(c)

	if role != domain.RoleSuperAdmin {
		errors.Forbidden(c, "only super admins can view subscription statistics")
		return
	}

	stats, err := h.billingService.GetSubscriptionStatistics(c.Request.Context())
	if err != nil {
		errors.InternalError(c, "failed to get subscription statistics")
		return
	}

	c.JSON(http.StatusOK, stats)
}

// Helper method to get tenant and user IDs from context
// For SuperAdmins, tenant_id may not exist - they are platform-level users
func (h *BillingHandler) getTenantAndUserID(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	// Get user ID first (required for all users)
	userIDValue, exists := c.Get("user_id")
	if !exists {
		errors.Unauthorized(c, "user not found in token")
		return uuid.Nil, uuid.Nil, false
	}
	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		errors.BadRequest(c, "invalid user ID format", nil)
		return uuid.Nil, uuid.Nil, false
	}

	// Check if user is SuperAdmin - they don't have tenant_id
	role, _ := GetRoleFromContext(c)
	if role == domain.RoleSuperAdmin {
		// SuperAdmin doesn't have a tenant, return Nil for tenant_id
		return uuid.Nil, userID, true
	}

	// For non-SuperAdmin users, tenant_id is required
	tenantIDValue, exists := c.Get("tenant_id")
	if !exists {
		errors.Unauthorized(c, "tenant not found in token")
		return uuid.Nil, uuid.Nil, false
	}
	tenantID, ok := tenantIDValue.(uuid.UUID)
	if !ok {
		errors.BadRequest(c, "invalid tenant ID format", nil)
		return uuid.Nil, uuid.Nil, false
	}

	return tenantID, userID, true
}

// Helper method to get user role from context
func (h *BillingHandler) getUserRole(c *gin.Context) domain.Role {
	role, ok := GetRoleFromContext(c)
	if !ok {
		return domain.RoleStudent
	}
	return role
}
