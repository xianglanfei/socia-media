package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/socia-media/backend/internal/api"
	"github.com/socia-media/backend/internal/db"
	"github.com/socia-media/backend/internal/memory"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using defaults")
	}

	// Initialize database
	database, err := db.NewDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()
	log.Println("Connected to PostgreSQL database")

	// Run migrations
	if err := db.RunMigrations(database); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations completed")

	// Initialize Redis
	redis, err := db.NewRedis()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redis.Close()
	log.Println("Connected to Redis")

	// Initialize memory service
	memoryService := memory.NewService(database)

	// Start server
	app := api.NewApp(database, redis, memoryService)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Printf("API: http://localhost:%s", port)
	log.Printf("WebSocket: ws://localhost:%s/ws", port)
	log.Printf("Health check: http://localhost:%s/health", port)

	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
