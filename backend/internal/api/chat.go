package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/socia-media/backend/internal/memory"
	"github.com/socia-media/backend/internal/models"
)

// getConversations returns all conversations for the authenticated user
func (a *App) getConversations(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)

	// Get conversations where user is either user1 or user2
	rows, err := a.db.Query(`
		SELECT DISTINCT ON (c.id)
			c.id, c.user1_id, c.user2_id, c.last_message_at,
			CASE WHEN c.user1_id = $1 THEN c.user2_id ELSE c.user1_id END as other_user_id,
			u.nickname, u.avatar_url, u.gender, u.age,
			m.id as msg_id, m.content, m.message_type, m.status, m.created_at as msg_created_at,
			(SELECT COUNT(*) FROM messages WHERE conversation_id = c.id AND sender_id != $1 AND status != 'read') as unread_count,
			COALESCE(mc.stage, 0) as stage
		FROM conversations c
		LEFT JOIN users u ON (CASE WHEN c.user1_id = $1 THEN c.user2_id ELSE c.user1_id END) = u.id
		LEFT JOIN LATERAL (
			SELECT id, content, message_type, status, created_at
			FROM messages
			WHERE conversation_id = c.id
			ORDER BY created_at DESC
			LIMIT 1
		) m ON true
		LEFT JOIN memory_context mc ON mc.conversation_id = c.id AND mc.user_id = $1
		WHERE c.user1_id = $1 OR c.user2_id = $1
		ORDER BY c.last_message_at DESC
	`, userID)

	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load conversations",
		})
	}
	defer rows.Close()

	conversations := []models.Conversation{}
	for rows.Next() {
		var conv models.Conversation
		var otherUserID uuid.UUID
		var otherUser models.User
		var lastMessage models.Message
		var nickname, avatarURL, gender *string
		var age *int

		err := rows.Scan(
			&conv.ID, &conv.User1ID, &conv.User2ID, &conv.LastMessageAt,
			&otherUserID,
			&nickname, &avatarURL, &gender, &age,
			&lastMessage.ID, &lastMessage.Content, &lastMessage.MessageType, &lastMessage.Status, &lastMessage.CreatedAt,
			&conv.UnreadCount,
			&conv.Stage,
		)

		if err != nil {
			continue
		}

		// Build other user
		otherUser.ID = otherUserID
		otherUser.Nickname = ""
		if nickname != nil {
			otherUser.Nickname = *nickname
		}
		otherUser.AvatarURL = avatarURL
		otherUser.Gender = gender
		otherUser.Age = age

		conv.OtherUser = &otherUser

		if lastMessage.ID != uuid.Nil {
			lastMessage.ConversationID = conv.ID
			conv.LastMessage = &lastMessage
		}

		conversations = append(conversations, conv)
	}

	return c.JSON(fiber.Map{
		"conversations": conversations,
	})
}

// getMessages returns messages for a conversation
func (a *App) getMessages(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)
	conversationIDStr := c.Params("id")
	conversationID, err := uuid.Parse(conversationIDStr)

	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid conversation ID",
		})
	}

	// Verify user is part of this conversation
	var isParticipant bool
	err = a.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM conversations
			WHERE id = $1 AND (user1_id = $2 OR user2_id = $2)
		)
	`, conversationID, userID).Scan(&isParticipant)

	if err != nil || !isParticipant {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	// Get messages
	limit := 50
	if l := c.QueryInt("limit"); l > 0 && l <= 200 {
		limit = l
	}

	rows, err := a.db.Query(`
		SELECT id, conversation_id, sender_id, content, message_type, status, created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, conversationID, limit)

	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load messages",
		})
	}
	defer rows.Close()

	messages := []models.Message{}
	for rows.Next() {
		var msg models.Message
		err := rows.Scan(
			&msg.ID, &msg.ConversationID, &msg.SenderID,
			&msg.Content, &msg.MessageType, &msg.Status, &msg.CreatedAt,
		)
		if err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return c.JSON(fiber.Map{
		"messages": messages,
	})
}

// sendMessage sends a message to a conversation
func (a *App) sendMessage(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)
	conversationIDStr := c.Params("id")
	conversationID, err := uuid.Parse(conversationIDStr)

	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid conversation ID",
		})
	}

	var req models.SendMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Content == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Message content cannot be empty",
		})
	}

	// Verify user is part of this conversation
	var otherUserID uuid.UUID
	err = a.db.QueryRow(`
		SELECT CASE WHEN user1_id = $1 THEN user2_id ELSE user1_id END
		FROM conversations
		WHERE id = $2 AND (user1_id = $1 OR user2_id = $1)
	`, userID, conversationID).Scan(&otherUserID)

	if err != nil {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	// Create message
	msgID := uuid.New()
	messageType := req.MessageType
	if messageType == "" {
		messageType = models.MessageTypeText
	}

	_, err = a.db.Exec(`
		INSERT INTO messages (id, conversation_id, sender_id, content, message_type, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, msgID, conversationID, userID, req.Content, messageType, models.MessageStatusSent)

	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to send message",
		})
	}

	// Update conversation last_message_at
	_, err = a.db.Exec(`
		UPDATE conversations
		SET last_message_at = NOW()
		WHERE id = $1
	`, conversationID)

	if err != nil {
		// Log but don't fail
	}

	// Update memory context
	go func() {
		ctx := context.Background()
		_ = a.memory.UpdateContext(ctx, conversationID, userID, otherUserID, req.Content)
	}()

	// Get the created message
	var msg models.Message
	err = a.db.QueryRow(`
		SELECT id, conversation_id, sender_id, content, message_type, status, created_at
		FROM messages WHERE id = $1
	`, msgID).Scan(
		&msg.ID, &msg.ConversationID, &msg.SenderID,
		&msg.Content, &msg.MessageType, &msg.Status, &msg.CreatedAt,
	)

	return c.Status(http.StatusCreated).JSON(msg)
}
