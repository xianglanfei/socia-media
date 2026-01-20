package api

import (
	"context"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/socia-media/backend/internal/llm"
	"github.com/socia-media/backend/internal/models"
)

// getAISuggestions generates AI-powered response suggestions for a conversation
func (a *App) getAISuggestions(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)
	conversationIDStr := c.Params("conversation_id")
	conversationID, err := uuid.Parse(conversationIDStr)

	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid conversation ID",
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

	// Get user's flirt style
	var flirtStyle string
	err = a.db.QueryRow(`
		SELECT flirt_style FROM users WHERE id = $1
	`, userID).Scan(&flirtStyle)

	if err != nil {
		flirtStyle = "humorous" // Default
	}

	// Get target user info
	var targetGender *string
	var targetNickname string
	err = a.db.QueryRow(`
		SELECT nickname, gender FROM users WHERE id = $1
	`, otherUserID).Scan(&targetNickname, &targetGender)

	// Get conversation memory
	stage := 0
	targetTraits := make(map[string]interface{})
	successfulPatterns := make(map[string]interface{})

	var memoryContext models.MemoryContext
	err = a.db.QueryRow(`
		SELECT id, conversation_id, user_id, stage, target_traits, successful_patterns, updated_at
		FROM memory_context
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID).Scan(
		&memoryContext.ID, &memoryContext.ConversationID, &memoryContext.UserID,
		&memoryContext.Stage, &memoryContext.TargetTraits, &memoryContext.SuccessfulPatterns,
		&memoryContext.UpdatedAt,
	)

	if err == nil {
		stage = memoryContext.Stage
		if memoryContext.TargetTraits != nil {
			targetTraits = memoryContext.TargetTraits
		}
		if memoryContext.SuccessfulPatterns != nil {
			successfulPatterns = memoryContext.SuccessfulPatterns
		}
	}

	// Get recent messages for context
	rows, err := a.db.Query(`
		SELECT id, sender_id, content, created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at DESC
		LIMIT 10
	`, conversationID)

	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to load conversation history",
		})
	}
	defer rows.Close()

	chatHistory := []map[string]interface{}{}
	for rows.Next() {
		var msgID, senderID uuid.UUID
		var content string
		var createdAt string

		err := rows.Scan(&msgID, &senderID, &content, &createdAt)
		if err != nil {
			continue
		}

		chatHistory = append(chatHistory, map[string]interface{}{
			"sender_id":  senderID,
			"is_self":    senderID == userID,
			"content":    content,
			"created_at": createdAt,
		})
	}

	// Reverse to get chronological order
	for i, j := 0, len(chatHistory)-1; i < j; i, j = i+1, j-1 {
		chatHistory[i], chatHistory[j] = chatHistory[j], chatHistory[i]
	}

	// Generate suggestions using LLM
	suggestions, err := llm.GenerateSuggestions(context.Background(), llm.SuggestionRequest{
		UserID:          userID,
		ConversationID:   conversationID,
		OtherUserID:      otherUserID,
		OtherUserGender:  targetGender,
		OtherUserNickname: targetNickname,
		Stage:           stage,
		UserFlirtStyle:   flirtStyle,
		ChatHistory:      chatHistory,
		TargetTraits:     targetTraits,
		SuccessfulPatterns: successfulPatterns,
	})

	if err != nil {
		// Fallback to mock suggestions if LLM fails
		suggestions = getFallbackSuggestions(flirtStyle, stage, targetNickname)
	}

	// Store AI suggestions log (for analytics)
	for _, suggestion := range suggestions {
		a.db.Exec(`
			INSERT INTO ai_suggestions (id, conversation_id, suggestion, was_used, response_received)
			VALUES ($1, $2, $3, false, false)
		`, uuid.New(), conversationID, suggestion.Text)
	}

	return c.JSON(models.AISuggestionsResponse{
		ConversationID: conversationID.String(),
		Stage:          stage,
		Suggestions:    suggestions,
	})
}

// getFallbackSuggestions returns hardcoded suggestions when LLM is unavailable
func getFallbackSuggestions(flirtStyle string, stage int, targetName string) []models.Suggestion {
	stageName := models.FlirtStageNames[stage]
	if stageName == "" {
		stageName = "未知"
	}

	suggestions := []models.Suggestion{}

	// Suggestion 1: User's preferred style
	userStyleSuggestion := models.Suggestion{
		Style: models.FlirtStyleNames[flirtStyle],
	}

	switch flirtStyle {
	case models.FlirtStyleDirect:
		userStyleSuggestion.Text = "我想直接告诉你，和你聊天真的很开心"
		userStyleSuggestion.Reason = "直接表达情感，展现真诚态度"
	case models.FlirtStyleHumorous:
		userStyleSuggestion.Text = "哈哈，你这人说话真有意思，和你聊天特别放松"
		userStyleSuggestion.Reason = "用轻松愉快的语气，增加互动趣味"
	case models.FlirtStyleRomantic:
		userStyleSuggestion.Text = "感觉和你聊天就像认识很久的朋友一样，很舒服"
		userStyleSuggestion.Reason = "用温柔浪漫的语气，拉近心理距离"
	case models.FlirtStyleSubtle:
		userStyleSuggestion.Text = "每次和你聊天都觉得时间过得很快，可能是因为太投机了吧"
		userStyleSuggestion.Reason = "含蓄地表达对聊天的珍视"
	default:
		userStyleSuggestion.Text = "和你聊天感觉很棒"
		userStyleSuggestion.Reason = "表达聊天的愉悦感受"
	}

	suggestions = append(suggestions, userStyleSuggestion)

	// Suggestion 2: Humorous style
	suggestions = append(suggestions, models.Suggestion{
		Text:   "看来我们很有共同语言嘛，以后要多聊聊~",
		Style:  "幽默风趣",
		Reason: "用轻松的语气发现共同点，鼓励继续交流",
	})

	// Suggestion 3: Romantic style
	suggestions = append(suggestions, models.Suggestion{
		Text:   "感觉和你聊天的时候，心情都会变好",
		Style:  "温柔浪漫",
		Reason: "表达对方带来的正面影响，增进情感连接",
	})

	return suggestions
}
