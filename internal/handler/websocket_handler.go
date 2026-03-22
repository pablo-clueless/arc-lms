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
		return true
	},
}

type WebSocketHandler struct {
	hub        *ws.Hub
	jwtManager *jwt.Manager
	logger     *log.Logger
}

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
	token := c.Query("token")
	if token == "" {
		errors.Unauthorized(c, "missing authentication token")
		return
	}

	claims, err := h.jwtManager.ValidateAccessToken(token)
	if err != nil {
		errors.Unauthorized(c, "invalid or expired token")
		return
	}

	userID := claims.UserID

	var tenantID uuid.UUID
	if claims.TenantID != nil {
		tenantID = *claims.TenantID
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Printf("[WebSocket] Failed to upgrade connection: %v", err)
		return
	}

	client := ws.NewClient(h.hub, conn, userID, tenantID, h.logger)
	h.hub.Register(client)

	go client.WritePump()
	go client.ReadPump()
}

func (h *WebSocketHandler) GetStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"online_users":      h.hub.GetOnlineUserCount(),
		"total_connections": h.hub.GetConnectionCount(),
	})
}
