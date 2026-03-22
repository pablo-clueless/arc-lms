package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/google/uuid"
)

type MessageType string

const (
	MessageTypeNotification     MessageType = "NOTIFICATION"
	MessageTypeNotificationRead MessageType = "NOTIFICATION_READ"
	MessageTypePing             MessageType = "PING"
	MessageTypePong             MessageType = "PONG"
)

type Message struct {
	Type    MessageType `json:"type"`
	Payload interface{} `json:"payload"`
}

type Hub struct {
	clients    map[uuid.UUID]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	logger     *log.Logger
}

func NewHub(logger *log.Logger) *Hub {
	if logger == nil {
		logger = log.Default()
	}
	return &Hub{
		clients:    make(map[uuid.UUID]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.UserID] == nil {
				h.clients[client.UserID] = make(map[*Client]bool)
			}
			h.clients[client.UserID][client] = true
			h.mu.Unlock()
			h.logger.Printf("[WebSocket] Client connected: user=%s", client.UserID)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.UserID]; ok {
				if _, exists := clients[client]; exists {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.clients, client.UserID)
					}
				}
			}
			h.mu.Unlock()
			h.logger.Printf("[WebSocket] Client disconnected: user=%s", client.UserID)
		}
	}
}

func (h *Hub) SendToUser(userID uuid.UUID, message *Message) {
	data, err := json.Marshal(message)
	if err != nil {
		h.logger.Printf("[WebSocket] Failed to marshal message: %v", err)
		return
	}

	h.mu.RLock()
	clients, ok := h.clients[userID]
	h.mu.RUnlock()

	if !ok || len(clients) == 0 {
		return
	}

	h.mu.RLock()
	for client := range clients {
		select {
		case client.send <- data:
		default:

			h.mu.RUnlock()
			h.mu.Lock()
			close(client.send)
			delete(h.clients[userID], client)
			h.mu.Unlock()
			h.mu.RLock()
		}
	}
	h.mu.RUnlock()
}

func (h *Hub) SendToUsers(userIDs []uuid.UUID, message *Message) {
	for _, userID := range userIDs {
		h.SendToUser(userID, message)
	}
}

func (h *Hub) Broadcast(message *Message) {
	data, err := json.Marshal(message)
	if err != nil {
		h.logger.Printf("[WebSocket] Failed to marshal broadcast message: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, clients := range h.clients {
		for client := range clients {
			select {
			case client.send <- data:
			default:

			}
		}
	}
}

func (h *Hub) IsUserOnline(userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	clients, ok := h.clients[userID]
	return ok && len(clients) > 0
}

func (h *Hub) GetOnlineUserCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) GetConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	count := 0
	for _, clients := range h.clients {
		count += len(clients)
	}
	return count
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}
