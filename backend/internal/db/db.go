package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

// DB wraps sql.DB with additional functionality
type DB struct {
	*sql.DB
}

// NewDB creates a new database connection
func NewDB() (*DB, error) {
	postgresURL := getEnv("POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/socia_media?sslmode=disable")

	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &DB{db}, nil
}

// NewRedis creates a new Redis client
func NewRedis() (*redis.Client, error) {
	redisURL := getEnv("REDIS_URL", "localhost:6379")

	return redis.NewClient(&redis.Options{
		Addr:     redisURL,
		Password: "", // no password set
		DB:       0,  // use default DB
	}), nil
}

// getEnv gets environment variable with fallback
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
