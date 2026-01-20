package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/socia-media/backend/internal/models"
)

// Client handles LLM API calls
type Client struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// NewClient creates a new LLM client
func NewClient(baseURL, apiKey, model string) *Client {
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
	if model == "" {
		model = "qwen-turbo"
	}

	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		client:  &httpClient{},
	}
}

// httpClient is a minimal HTTP client interface
type httpClient struct{}

func (c *httpClient) Do(req *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(req)
}

// SuggestionRequest contains all context needed to generate suggestions
type SuggestionRequest struct {
	UserID            uuid.UUID
	ConversationID     uuid.UUID
	OtherUserID       uuid.UUID
	OtherUserGender   *string
	OtherUserNickname string
	Stage             int
	UserFlirtStyle    string
	ChatHistory       []map[string]interface{}
	TargetTraits      map[string]interface{}
	SuccessfulPatterns map[string]interface{}
}

// LLMResponse is the response from the LLM
type LLMResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message Message `json:"message"`
}

type Message struct {
	Content string `json:"content"`
}

// GenerateSuggestions generates AI-powered response suggestions
func GenerateSuggestions(ctx context.Context, req SuggestionRequest) ([]models.Suggestion, error) {
	// Check if LLM is configured
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("LLM API key not configured")
	}

	client := NewClient("", apiKey, "")

	// Build prompt
	prompt := buildPrompt(req)

	// Call LLM
	response, err := client.Call(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Parse response
	suggestions, err := parseSuggestions(response)
	if err != nil {
		return nil, err
	}

	return suggestions, nil
}

// buildPrompt creates the prompt for the LLM
func buildPrompt(req SuggestionRequest) string {
	// Stage name
	stageName := models.FlirtStageNames[req.Stage]
	if stageName == "" {
		stageName = "未知"
	}

	// User style name
	styleName := models.FlirtStyleNames[req.UserFlirtStyle]
	if styleName == "" {
		styleName = "幽默风趣"
	}

	// Target gender
	gender := "对方"
	if req.OtherUserGender != nil {
		switch *req.OtherUserGender {
		case "male":
			gender = "他"
		case "female":
			gender = "她"
		default:
			gender = "TA"
		}
	}

	// Build chat history string
	history := ""
	if len(req.ChatHistory) > 0 {
		for i, msg := range req.ChatHistory {
			isSelf, _ := msg["is_self"].(bool)
			content, _ := msg["content"].(string)
			prefix := "对方: "
			if isSelf {
				prefix = "你: "
			}
			history += fmt.Sprintf("%s%s\n", prefix, content)

			// Only include last 5 messages
			if i >= 4 {
				break
			}
		}
	}

	// Build traits string
	traits := ""
	if req.TargetTraits != nil {
		if interests, ok := req.TargetTraits["interests"].([]interface{}); ok && len(interests) > 0 {
			traits += "兴趣爱好: "
			for i, interest := range interests {
				if i > 0 {
					traits += ", "
				}
				traits += interest.(string)
			}
			traits += "\n"
		}
		if topics, ok := req.TargetTraits["topics"].([]interface{}); ok && len(topics) > 0 {
			traits += "话题: "
			for i, topic := range topics {
				if i > 0 {
					traits += ", "
				}
				traits += topic.(string)
			}
			traits += "\n"
		}
	}

	prompt := fmt.Sprintf(`你是一个专业的中文聊天和约会助手。根据以下信息生成3条回复建议：

【当前语境】
- 对话阶段: %s
- 你的风格: %s
- 对方: %s (%s)
- 对话历史:
%s
- 对方特点:
%s

【要求】
1. 必须生成恰好3条建议，每条风格不同
2. 第1条：符合你偏好的风格 (%s)
3. 第2条：幽默风趣 - 适合轻松氛围
4. 第3条：温柔浪漫 - 适合推进关系
5. 回复自然、不油腻
6. 符合当前对话阶段
7. 引导继续对话
8. 保持尊重和礼貌

请生成JSON格式回复:
{
  "suggestions": [
    {"text": "...", "style": "用户风格", "reason": "..."},
    {"text": "...", "style": "幽默风趣", "reason": "..."},
    {"text": "...", "style": "温柔浪漫", "reason": "..."}
  ]
}

只输出JSON，不要有任何其他文字。`,
		stageName,
		req.UserFlirtStyle,
		req.OtherUserNickname,
		gender,
		history,
		traits,
		req.UserFlirtStyle,
	)

	return prompt
}

// Call makes a request to the LLM API
func (c *Client) Call(ctx context.Context, prompt string) (string, error) {
	requestBody := map[string]interface{}{
		"model": c.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.8,
		"max_tokens":   1000,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(body))
	}

	var llmResponse LLMResponse
	if err := json.NewDecoder(resp.Body).Decode(&llmResponse); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(llmResponse.Choices) == 0 {
		return "", fmt.Errorf("no choices in LLM response")
	}

	return llmResponse.Choices[0].Message.Content, nil
}

// parseSuggestions parses the LLM response into suggestions
func parseSuggestions(response string) ([]models.Suggestion, error) {
	// Try to extract JSON from the response
	response = extractJSON(response)

	var result struct {
		Suggestions []models.Suggestion `json:"suggestions"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse suggestions: %w", err)
	}

	if len(result.Suggestions) == 0 {
		return nil, fmt.Errorf("no suggestions in response")
	}

	return result.Suggestions, nil
}

// extractJSON extracts JSON from a response that may have extra text
func extractJSON(s string) string {
	s = bytes.TrimSpace([]byte(s))
	start := bytes.Index([]byte(s), []byte("{"))
	if start == -1 {
		return s
	}
	end := bytes.LastIndex([]byte(s), []byte("}"))
	if end == -1 {
		return s
	}
	return string(s[start : end+1])
}

// StreamSuggestions streams suggestions from the LLM
func (c *Client) StreamSuggestions(ctx context.Context, prompt string, callback func(chunk string)) error {
	requestBody := map[string]interface{}{
		"model": c.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.8,
		"max_tokens":   1000,
		"stream":      true,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(body))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 6 && line[:6] == "data: " {
			data := line[6:]
			if data == "[DONE]" {
				break
			}

			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}

			if err := json.Unmarshal([]byte(data), &chunk); err == nil {
				if len(chunk.Choices) > 0 {
					callback(chunk.Choices[0].Delta.Content)
				}
			}
		}
	}

	return scanner.Err()
}
