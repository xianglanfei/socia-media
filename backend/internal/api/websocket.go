package api

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/socia-media/backend/internal/auth"
	"github.com/socia-media/backend/internal/models"
)

// WebSocket connection manager
type WebSocketManager struct {
	connections map[uuid.UUID]*WebSocketConnection
	mutex       sync.RWMutex
}

// WebSocketConnection represents a WebSocket connection
type WebSocketConnection struct {
	UserID      uuid.UUID
	Connection  *websocket.Conn
	ActiveConvs map[uuid.UUID]bool // Active conversations
	mutex       sync.RWMutex
}

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

// WSTyping represents a typing indicator
type WSTyping struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	IsTyping       bool       `json:"is_typing"`
}

// WSRead represents a read receipt
type WSRead struct {
	ConversationID uuid.UUID   `json:"conversation_id"`
	MessageIDs     []uuid.UUID `json:"message_ids"`
}

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager() *WebSocketManager {
	return &WebSocketManager{
		connections: make(map[uuid.UUID]*WebSocketConnection),
	}
}

// Global WebSocket manager instance
var wsManager = NewWebSocketManager()

// handleWebSocket handles WebSocket connections
func (a *App) handleWebSocket(c *fiber.Ctx) error {
	// Extract token from query param
	token := c.Query("token")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).SendString("Missing token")
	}

	// Validate token
	userID, err := a.auth.ValidateToken(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("Invalid token")
	}

	// Upgrade to WebSocket
	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("allowed", true)
		return c.Next()
	}

	return fiber.ErrUpgradeRequired
}

// HandleUpgrade handles the WebSocket upgrade
func (a *App) HandleUpgrade(c *websocket.Conn) {
	// Get user ID from context (set by JWT middleware)
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		c.Close()
		return
	}

	// Create connection
	conn := &WebSocketConnection{
		UserID:      userID,
		Connection:  c,
		ActiveConvs: make(map[uuid.UUID]bool),
	}

	// Register connection
	wsManager.mutex.Lock()
	wsManager.connections[userID] = conn
	wsManager.mutex.Unlock()

	log.Printf("WebSocket connected: user %s", userID)

	// Clean up on disconnect
	defer func() {
		wsManager.mutex.Lock()
		delete(wsManager.connections, userID)
		wsManager.mutex.Unlock()
		log.Printf("WebSocket disconnected: user %s", userID)
		c.Close()
	}()

	// Handle messages
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		go a.handleWSMessage(conn, message)
	}
}

// handleWSMessage processes incoming WebSocket messages
func (a *App) handleWSMessage(conn *WebSocketConnection, message []byte) {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Invalid WebSocket message: %v", err)
		return
	}

	msgType, ok := msg["type"].(string)
	if !ok {
		return
	}

	switch msgType {
	case "connect":
		// Connection confirmation
		_ = conn.Connection.WriteJSON(WSMessage{
			Type: "connect",
			Data: map[string]interface{}{
				"user_id": conn.UserID,
			},
		})

	case "message":
		a.handleWSMessageSend(conn, msg)

	case "typing":
		a.handleWSTyping(conn, msg)

	case "read":
		a.handleWSRead(conn, msg)

	case "disconnect":
		// Connection is closing
		break
	}
}

// handleWSMessageSend handles sending a message via WebSocket
func (a *App) handleWSMessageSend(conn *WebSocketConnection, msg map[string]interface{}) {
	conversationIDStr, ok := msg["conversation_id"].(string)
	if !ok {
		return
	}

	content, ok := msg["content"].(string)
	if !ok || content == "" {
		return
	}

	messageType := models.MessageTypeText
	if mt, ok := msg["message_type"].(string); ok {
		messageType = mt
	}

	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		return
	}

	// Verify user is part of this conversation
	var otherUserID uuid.UUID
	err = a.db.QueryRow(`
		SELECT CASE WHEN user1_id = $1 THEN user2_id ELSE user1_id END
		FROM conversations
		WHERE id = $2 AND (user1_id = $1 OR user2_id = $1)
	`, conn.UserID, conversationID).Scan(&otherUserID)

	if err != nil {
		return
	}

	// Create message
	msgID := uuid.New()
	_, err = a.db.Exec(`
		INSERT INTO messages (id, conversation_id, sender_id, content, message_type, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, msgID, conversationID, conn.UserID, content, messageType, models.MessageStatusSent)

	if err != nil {
		return
	}

	// Update conversation timestamp
	_, _ = a.db.Exec(`
		UPDATE conversations
		SET last_message_at = NOW()
		WHERE id = $1
	`, conversationID)

	// Get the created message
	var message models.Message
	_ = a.db.QueryRow(`
		SELECT id, conversation_id, sender_id, content, message_type, status, created_at
		FROM messages WHERE id = $1
	`, msgID).Scan(
		&message.ID, &message.ConversationID, &message.SenderID,
		&message.Content, &message.MessageType, &message.Status, &message.CreatedAt,
	)

	// Send to both users
	wsManager.mutex.RLock()
	defer wsManager.mutex.RUnlock()

	// Send to sender
	if senderConn, exists := wsManager.connections[conn.UserID]; exists {
		_ = senderConn.Connection.WriteJSON(WSMessage{
			Type: "message",
			Data: message,
		})
	}

	// Send to recipient
	if recipientConn, exists := wsManager.connections[otherUserID]; exists {
		_ = recipientConn.Connection.WriteJSON(WSMessage{
			Type: "message",
			Data: message,
		})

		// Update status to delivered
		_, _ = a.db.Exec(`
			UPDATE messages SET status = $1 WHERE id = $2
		`, models.MessageStatusDelivered, msgID)
	}
}

// handleWSTyping handles typing indicators
func (a *App) handleWSTyping(conn *WebSocketConnection, msg map[string]interface{}) {
	conversationIDStr, ok := msg["conversation_id"].(string)
	if !ok {
		return
	}

	isTyping, ok := msg["is_typing"].(bool)
	if !ok {
		return
	}

	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		return
	}

	// Get other user ID
	var otherUserID uuid.UUID
	_ = a.db.QueryRow(`
		SELECT CASE WHEN user1_id = $1 THEN user2_id ELSE user1_id END
		FROM conversations
		WHERE id = $2 AND (user1_id = $1 OR user2_id = $1)
	`, conn.UserID, conversationID).Scan(&otherUserID)

	// Send typing indicator to other user
	wsManager.mutex.RLock()
	defer wsManager.mutex.RUnlock()

	if recipientConn, exists := wsManager.connections[otherUserID]; exists {
		_ = recipientConn.Connection.WriteJSON(WSMessage{
			Type: "typing",
			Data: WSTyping{
				ConversationID: conversationID,
				IsTyping:       isTyping,
			},
		})
	}
}

// handleWSRead handles read receipts
func (a *App) handleWSRead(conn *WebSocketConnection, msg map[string]interface{}) {
	conversationIDStr, ok := msg["conversation_id"].(string)
	if !ok {
		return
	}

	messageIDsInterface, ok := msg["message_ids"].([]interface{})
	if !ok {
		return
	}

	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		return
	}

	messageIDs := make([]uuid.UUID, len(messageIDsInterface))
	for i, mid := range messageIDsInterface {
		if str, ok := mid.(string); ok {
			messageIDs[i], _ = uuid.Parse(str)
		}
	}

	if len(messageIDs) == 0 {
		return
	}

	// Update message status to read
	_, err = a.db.Exec(`
		UPDATE messages
		SET status = $1
		WHERE id = ANY($2) AND conversation_id = $3 AND sender_id != $4
	`, models.MessageStatusRead, messageIDs, conversationID, conn.UserID)

	if err != nil {
		return
	}

	// Get other user ID
	var otherUserID uuid.UUID
	_ = a.db.QueryRow(`
		SELECT CASE WHEN user1_id = $1 THEN user2_id ELSE user1_id END
		FROM conversations
		WHERE id = $2 AND (user1_id = $1 OR user2_id = $1)
	`, conn.UserID, conversationID).Scan(&otherUserID)

	// Notify other user
	wsManager.mutex.RLock()
	defer wsManager.mutex.RUnlock()

	if recipientConn, exists := wsManager.connections[otherUserID]; exists {
		_ = recipientConn.Connection.WriteJSON(WSMessage{
			Type: "read",
			Data: WSRead{
				ConversationID: conversationID,
				MessageIDs:     messageIDs,
			},
		})
	}
}

// RegisterWebSocketHandlers registers WebSocket upgrade route
func (a *App) RegisterWebSocketHandlers() {
	a.Get("/ws", websocket.New(a.HandleUpgrade, websocket.Config{
		HandshakeTimeout:  10,
		WriteBufferSize:   1024,
		ReadBufferSize:    1024,
		CheckOrigin: func(r *fiber.Ctx) bool {
			return true // Allow all origins in development
		},
	}))
}
