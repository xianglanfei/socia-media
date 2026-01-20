package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID         uuid.UUID `json:"id" db:"id"`
	Phone      string    `json:"phone" db:"phone"`
	Nickname   string    `json:"nickname" db:"nickname"`
	Gender     *string   `json:"gender" db:"gender"`
	Age        *int      `json:"age" db:"age"`
	AvatarURL  *string   `json:"avatar_url" db:"avatar_url"`
	Bio        *string   `json:"bio" db:"bio"`
	FlirtStyle string    `json:"flirt_style" db:"flirt_style"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// FlirtStyle represents the user's preferred conversation style
const (
	FlirtStyleDirect    = "direct"
	FlirtStyleHumorous  = "humorous"
	FlirtStyleRomantic  = "romantic"
	FlirtStyleSubtle    = "subtle"
)

// FlirtStyleNames maps style codes to Chinese names
var FlirtStyleNames = map[string]string{
	FlirtStyleDirect:   "直球型",
	FlirtStyleHumorous: "幽默风趣",
	FlirtStyleRomantic: "温柔浪漫",
	FlirtStyleSubtle:   "含蓄内敛",
}

// FlirtStyleDescriptions provides descriptions for each style
var FlirtStyleDescriptions = map[string]string{
	FlirtStyleDirect:   "直接、自信 - 适合喜欢直来直去的人",
	FlirtStyleHumorous: "机智、有趣 - 适合喜欢轻松愉快氛围的人",
	FlirtStyleRomantic: "温暖、浪漫 - 适合喜欢浪漫氛围的人",
	FlirtStyleSubtle:   "含蓄、深情 - 适合不张扬但有意义的人",
}

// JSONB is a custom type for handling JSONB in PostgreSQL
type JSONB struct {
	Data interface{}
}

// Scan implements the sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		j.Data = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return driver.ErrBadConn
	}
	return json.Unmarshal(bytes, &j.Data)
}

// Value implements the driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j.Data == nil {
		return nil, nil
	}
	return json.Marshal(j.Data)
}

// Map is a helper for map[string]interface{} with JSONB
type Map map[string]interface{}

// Scan implements the sql.Scanner interface
func (m *Map) Scan(value interface{}) error {
	var j JSONB
	if err := j.Scan(value); err != nil {
		return err
	}
	if j.Data == nil {
		*m = make(Map)
		return nil
	}
	*m = j.Data.(map[string]interface{})
	return nil
}

// Value implements the driver.Valuer interface
func (m Map) Value() (driver.Value, error) {
	return JSONB{Data: m}.Value()
}

// Conversation represents a conversation between two users
type Conversation struct {
	ID           uuid.UUID `json:"id" db:"id"`
	User1ID      uuid.UUID `json:"user1_id" db:"user1_id"`
	User2ID      uuid.UUID `json:"user2_id" db:"user2_id"`
	LastMessageAt time.Time `json:"last_message_at" db:"last_message_at"`
	OtherUser    *User     `json:"other_user,omitempty" db:"-"`
	LastMessage  *Message  `json:"last_message,omitempty" db:"-"`
	UnreadCount  int       `json:"unread_count,omitempty" db:"-"`
	Stage        int       `json:"stage,omitempty" db:"-"`
}

// Message represents a chat message
type Message struct {
	ID             uuid.UUID `json:"id" db:"id"`
	ConversationID uuid.UUID `json:"conversation_id" db:"conversation_id"`
	SenderID       uuid.UUID `json:"sender_id" db:"sender_id"`
	Content        string    `json:"content" db:"content"`
	MessageType    string    `json:"message_type" db:"message_type"`
	Status         string    `json:"status" db:"status"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// Message status constants
const (
	MessageStatusSent      = "sent"
	MessageStatusDelivered = "delivered"
	MessageStatusRead      = "read"
)

// Message type constants
const (
	MessageTypeText  = "text"
	MessageTypeImage = "image"
	MessageTypeVoice = "voice"
)

// Flirt stages in Chinese
const (
	FlirtStageColdStart = 0 // 冷启动
	FlirtStageBreakingIce = 1 // 破冰
	FlirtStageWarmUp = 2 // 热身
	FlirtStageFlirty = 3 // 暧昧
	FlirtStageDeep = 4 // 深入
)

// FlirtStageNames maps stage codes to Chinese names
var FlirtStageNames = map[int]string{
	FlirtStageColdStart:   "冷启动",
	FlirtStageBreakingIce: "破冰",
	FlirtStageWarmUp:      "热身",
	FlirtStageFlirty:      "暧昧",
	FlirtStageDeep:        "深入",
}

// MemoryContext represents the AI memory for a conversation
type MemoryContext struct {
	ID                 uuid.UUID `json:"id" db:"id"`
	ConversationID     uuid.UUID `json:"conversation_id" db:"conversation_id"`
	UserID             uuid.UUID `json:"user_id" db:"user_id"`
	Stage              int       `json:"stage" db:"stage"`
	TargetTraits       Map       `json:"target_traits" db:"target_traits"`
	SuccessfulPatterns Map       `json:"successful_patterns" db:"successful_patterns"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

// AISuggestion represents an AI-generated response suggestion
type AISuggestion struct {
	ID                 uuid.UUID `json:"id" db:"id"`
	ConversationID     uuid.UUID `json:"conversation_id" db:"conversation_id"`
	Suggestion         string    `json:"suggestion" db:"suggestion"`
	WasUsed            bool      `json:"was_used" db:"was_used"`
	ResponseReceived   bool      `json:"response_received" db:"response_received"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
}

// AISuggestionsResponse is the API response for AI suggestions
type AISuggestionsResponse struct {
	ConversationID string       `json:"conversation_id"`
	Stage          int          `json:"stage"`
	Suggestions    []Suggestion `json:"suggestions"`
}

// Suggestion is a single AI suggestion
type Suggestion struct {
	Text   string `json:"text"`
	Style  string `json:"style"`
	Reason string `json:"reason"`
}

// VerificationCode represents a SMS verification code
type VerificationCode struct {
	Phone     string    `json:"phone"`
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expires_at"`
}

// RegisterRequest is the request payload for user registration
type RegisterRequest struct {
	Phone      string `json:"phone"`
	Code       string `json:"code"`
	Nickname   string `json:"nickname"`
	Gender     string `json:"gender,omitempty"`
	Age        int    `json:"age,omitempty"`
	FlirtStyle string `json:"flirt_style,omitempty"`
}

// LoginRequest is the request payload for user login
type LoginRequest struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

// AuthResponse is the response payload for authentication
type AuthResponse struct {
	User  *User  `json:"user"`
	Token string `json:"token"`
}

// UpdateProfileRequest is the request payload for profile update
type UpdateProfileRequest struct {
	Nickname  *string `json:"nickname,omitempty"`
	Gender    *string `json:"gender,omitempty"`
	Age       *int    `json:"age,omitempty"`
	Bio       *string `json:"bio,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

// UpdateFlirtStyleRequest is the request payload for flirt style update
type UpdateFlirtStyleRequest struct {
	FlirtStyle string `json:"flirt_style"`
}

// SendMessageRequest is the request payload for sending a message
type SendMessageRequest struct {
	Content     string `json:"content"`
	MessageType string `json:"message_type,omitempty"`
}
