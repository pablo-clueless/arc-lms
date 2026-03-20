package handler

import (
	"log"
	"net/http"

	"arc-lms/internal/pkg/errors"
	"arc-lms/internal/pkg/jwt"
	ws "arc-lms/internal/pkg/websocket"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, validate the origin against allowed origins
		return true
	},
}

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	hub        *ws.Hub
	jwtManager *jwt.Manager
	logger     *log.Logger
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *ws.Hub, jwtManager *jwt.Manager, logger *log.Logger) *WebSocketHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &WebSocketHandler{
		hub:        hub,
		jwtManager: jwtManager,
		logger:     logger,
	}
}

// HandleConnection godoc
// @Summary WebSocket connection
// @Description Establish a WebSocket connection for real-time notifications
// @Tags WebSocket
// @Param token query string true "JWT access token"
// @Success 101 {string} string "Switching Protocols"
// @Failure 401 {object} errors.ErrorResponse
// @Router /ws [get]
func (h *WebSocketHandler) HandleConnection(c *gin.Context) {
	// Get token from query parameter (WebSocket can't use Authorization header easily)
	token := c.Query("token")
	if token == "" {
		errors.Unauthorized(c, "missing authentication token")
		return
	}

	// Validate token
	claims, err := h.jwtManager.ValidateAccessToken(token)
	if err != nil {
		errors.Unauthorized(c, "invalid or expired token")
		return
	}

	userID := claims.UserID

	// TenantID can be nil for SUPER_ADMIN
	var tenantID uuid.UUID
	if claims.TenantID != nil {
		tenantID = *claims.TenantID
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Printf("[WebSocket] Failed to upgrade connection: %v", err)
		return
	}

	// Create client and register with hub
	client := ws.NewClient(h.hub, conn, userID, tenantID, h.logger)
	h.hub.Register(client)

	// Start client read/write pumps
	go client.WritePump()
	go client.ReadPump()
}

// GetStats returns WebSocket connection statistics
func (h *WebSocketHandler) GetStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"online_users":      h.hub.GetOnlineUserCount(),
		"total_connections": h.hub.GetConnectionCount(),
	})
}
