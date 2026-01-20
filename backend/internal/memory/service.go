package memory

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/socia-media/backend/internal/models"
)

// Service handles memory context operations
type Service struct {
	db *sql.DB
}

// NewService creates a new memory service
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// GetOrCreateContext gets or creates a memory context for a conversation
func (s *Service) GetOrCreateContext(ctx context.Context, conversationID, userID uuid.UUID) (*models.MemoryContext, error) {
	var ctx models.MemoryContext

	err := s.db.QueryRowContext(ctx, `
		SELECT id, conversation_id, user_id, stage, target_traits, successful_patterns, updated_at
		FROM memory_context
		WHERE conversation_id = $1 AND user_id = $2
	`, conversationID, userID).Scan(
		&ctx.ID, &ctx.ConversationID, &ctx.UserID,
		&ctx.Stage, &ctx.TargetTraits, &ctx.SuccessfulPatterns, &ctx.UpdatedAt,
	)

	if err == nil {
		return &ctx, nil
	}

	if err == sql.ErrNoRows {
		// Create new context
		ctxID := uuid.New()
		_, err = s.db.ExecContext(ctx, `
			INSERT INTO memory_context (id, conversation_id, user_id, stage, target_traits, successful_patterns)
			VALUES ($1, $2, $3, 0, '{}', '{}')
		`, ctxID, conversationID, userID)

		if err != nil {
			return nil, fmt.Errorf("failed to create memory context: %w", err)
		}

		return &models.MemoryContext{
			ID:                 ctxID,
			ConversationID:     conversationID,
			UserID:             userID,
			Stage:              0,
			TargetTraits:       make(models.Map),
			SuccessfulPatterns: make(models.Map),
		}, nil
	}

	return nil, fmt.Errorf("failed to get memory context: %w", err)
}

// UpdateContext updates the memory context with new message
func (s *Service) UpdateContext(ctx context.Context, conversationID, userID, otherUserID uuid.UUID, content string) error {
	memoryCtx, err := s.GetOrCreateContext(ctx, conversationID, userID)
	if err != nil {
		return err
	}

	// Extract information from the message
	newTraits := s.extractTraits(content)
	newStage := s.calculateStage(memoryCtx.Stage, content, memoryCtx.TargetTraits)

	// Update target traits
	updatedTraits := s.mergeTraits(memoryCtx.TargetTraits, newTraits)

	// Update successful patterns
	// In a real implementation, this would track which message types get positive responses
	updatedPatterns := s.updatePatterns(memoryCtx.SuccessfulPatterns, content)

	// Update in database
	_, err = s.db.ExecContext(ctx, `
		UPDATE memory_context
		SET stage = $1, target_traits = $2, successful_patterns = $3
		WHERE id = $4
	`, newStage, updatedTraits, updatedPatterns, memoryCtx.ID)

	return err
}

// extractTraits extracts personality traits and interests from messages
func (s *Service) extractTraits(content string) map[string]interface{} {
	traits := make(map[string]interface{})

	// Simple keyword-based trait extraction
	// In production, you would use NLP or an LLM

	interests := []string{}
	topics := []string{}

	contentLower := lowercase(content)

	// Interest keywords
	interestKeywords := map[string]string{
		"音乐": "music",
		"运动": "sports",
		"电影": "movies",
		"旅行": "travel",
		"美食": "food",
		"游戏": "gaming",
		"读书": "reading",
		"摄影": "photography",
		"健身": "fitness",
		"舞蹈": "dancing",
		"画画": "drawing",
		"唱歌": "singing",
	}

	for keyword, interest := range interestKeywords {
		if contains(contentLower, lowercase(keyword)) {
			interests = append(interests, interest)
		}
	}

	if len(interests) > 0 {
		traits["interests"] = interests
	}

	// Topic keywords
	topicKeywords := map[string]string{
		"工作": "work",
		"学校": "school",
		"家庭": "family",
		"朋友": "friends",
		"学习": "study",
		"梦想": "dreams",
	}

	for keyword, topic := range topicKeywords {
		if contains(contentLower, lowercase(keyword)) {
			topics = append(topics, topic)
		}
	}

	if len(topics) > 0 {
		traits["topics"] = topics
	}

	// Tone detection
	tone := s.detectTone(contentLower)
	traits["tone"] = tone

	// Sentiment detection
	sentiment := s.detectSentiment(contentLower)
	traits["sentiment"] = sentiment

	return traits
}

// detectTone detects the tone of a message
func (s *Service) detectTone(content string) string {
	// Simple keyword-based tone detection
	positiveKeywords := []string{"哈哈", "哈哈", "开心", "喜欢", "爱", "棒", "厉害"}
	negativeKeywords := []string{"难过", "伤心", "讨厌", "烦", "生气"}
	questionKeywords := []string{"吗", "呢", "什么", "如何", "怎么"}

	positiveCount := countKeywords(content, positiveKeywords)
	negativeCount := countKeywords(content, negativeKeywords)
	questionCount := countKeywords(content, questionKeywords)

	if questionCount > 0 {
		return "questioning"
	} else if positiveCount > negativeCount {
		return "positive"
	} else if negativeCount > positiveCount {
		return "negative"
	}
	return "neutral"
}

// detectSentiment detects the sentiment of a message
func (s *Service) detectSentiment(content string) string {
	positiveKeywords := []string{"哈哈", "哈哈", "开心", "喜欢", "爱", "棒", "厉害", "好", "漂亮", "帅"}
	negativeKeywords := []string{"难过", "伤心", "讨厌", "烦", "生气", "不好", "糟糕"}

	positiveCount := countKeywords(content, positiveKeywords)
	negativeCount := countKeywords(content, negativeKeywords)

	if positiveCount > 0 {
		return "positive"
	} else if negativeCount > 0 {
		return "negative"
	}
	return "neutral"
}

// calculateStage determines the flirt stage based on conversation
func (s *Service) calculateStage(currentStage int, content string, traits map[string]interface{}) int {
	// Stage progression logic
	// In production, this would be more sophisticated

	messageCount := 1
	if count, ok := traits["message_count"].(float64); ok {
		messageCount = int(count)
	}

	// Stage transitions
	switch currentStage {
	case models.FlirtStageColdStart:
		// Move to breaking ice after 1-2 messages
		if messageCount >= 1 {
			return models.FlirtStageBreakingIce
		}
	case models.FlirtStageBreakingIce:
		// Move to warm up after 5+ messages
		if messageCount >= 5 {
			return models.FlirtStageWarmUp
		}
	case models.FlirtStageWarmUp:
		// Move to flirty after 10+ messages and positive sentiment
		if messageCount >= 10 {
			sentiment := "neutral"
			if s, ok := traits["sentiment"].(string); ok {
				sentiment = s
			}
			if sentiment == "positive" || contains(lowercase(content), "喜欢") {
				return models.FlirtStageFlirty
			}
		}
	case models.FlirtStageFlirty:
		// Move to deep after 20+ messages with emotional keywords
		if messageCount >= 20 {
			emotionalKeywords := []string{"想", "想念", "在乎", "在意", "喜欢", "爱"}
			for _, kw := range emotionalKeywords {
				if contains(lowercase(content), lowercase(kw)) {
					return models.FlirtStageDeep
				}
			}
		}
	}

	return currentStage
}

// mergeTraits merges new traits into existing traits
func (s *Service) mergeTraits(existing map[string]interface{}, new map[string]interface{}) models.Map {
	result := make(models.Map)

	// Copy existing
	for k, v := range existing {
		result[k] = v
	}

	// Merge new traits
	for k, v := range new {
		switch k {
		case "interests":
			if existingInterests, ok := result[k].([]interface{}); ok {
				if newInterests, ok := v.([]string); ok {
					result[k] = mergeStringLists(existingInterests, newInterests)
				}
			} else {
				result[k] = v
			}
		case "topics":
			if existingTopics, ok := result[k].([]interface{}); ok {
				if newTopics, ok := v.([]string); ok {
					result[k] = mergeStringLists(existingTopics, newTopics)
				}
			} else {
				result[k] = v
			}
		default:
			result[k] = v
		}
	}

	return result
}

// updatePatterns updates successful conversation patterns
func (s *Service) updatePatterns(existing map[string]interface{}, content string) models.Map {
	result := make(models.Map)

	// Copy existing
	for k, v := range existing {
		result[k] = v
	}

	// Count messages
	messageCount := 1
	if count, ok := result["message_count"].(float64); ok {
		messageCount = int(count) + 1
	}
	result["message_count"] = float64(messageCount)

	// Track question vs statement
	messageType := "statement"
	if contains(lowercase(content), "?") || contains(lowercase(content), "吗") {
		messageType = "question"
	}

	if typeCount, ok := result[messageType].(float64); ok {
		result[messageType] = typeCount + 1
	} else {
		result[messageType] = 1.0
	}

	return result
}

// Helper functions
func lowercase(s string) string {
	// Simple lowercase conversion for ASCII
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c + 32
		}
		result[i] = c
	}
	return string(result)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}

func countKeywords(s string, keywords []string) int {
	count := 0
	lowerS := lowercase(s)
	for _, kw := range keywords {
		if contains(lowerS, lowercase(kw)) {
			count++
		}
	}
	return count
}

func mergeStringLists(existing []interface{}, new []string) []interface{} {
	seen := make(map[string]bool)
	result := make([]interface{}, 0)

	// Add existing
	for _, item := range existing {
		if str, ok := item.(string); ok {
			if !seen[str] {
				seen[str] = true
				result = append(result, str)
			}
		}
	}

	// Add new
	for _, item := range new {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
