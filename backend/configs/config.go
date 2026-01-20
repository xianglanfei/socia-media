package configs

import (
	"os"
)

type Config struct {
	// Server
	Port string
	Env  string

	// Database
	PostgresURL string

	// Redis
	RedisURL string

	// JWT
	JWTSecret string
	JWTExpiry int // hours

	// SMS
	SMSProvider string
	SMSAPIKey   string

	// LLM
	LLMProvider string
	LLMBaseURL  string
	LLMAPIKey   string
	LLMModel    string
}

func LoadConfig() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		Env:         getEnv("ENV", "development"),
		PostgresURL: getEnv("POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/socia_media?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "localhost:6379"),
		JWTSecret:   getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		JWTExpiry:   24 * 7, // 7 days

		SMSProvider: getEnv("SMS_PROVIDER", "mock"),
		SMSAPIKey:   getEnv("SMS_API_KEY", ""),

		LLMProvider: getEnv("LLM_PROVIDER", "qwen"),
		LLMBaseURL:  getEnv("LLM_BASE_URL", "https://dashscope.aliyuncs.com/compatible-mode/v1"),
		LLMAPIKey:   getEnv("LLM_API_KEY", ""),
		LLMModel:    getEnv("LLM_MODEL", "qwen-turbo"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
