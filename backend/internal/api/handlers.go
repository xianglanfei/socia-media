package api

import (
	"crypto/rand"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/socia-media/backend/internal/auth"
	"github.com/socia-media/backend/internal/db"
	"github.com/socia-media/backend/internal/memory"
	"github.com/socia-media/backend/internal/models"
	"github.com/socia-media/backend/internal/sms"
)

type App struct {
	*fiber.App
	db         *db.DB
	redis      *redis.Client
	auth       *auth.JWTService
	smsService sms.SMSService
	memory     *memory.Service
}

func NewApp(db *db.DB, redis *redis.Client, memoryService *memory.Service) *App {
	app := &App{
		App:       fiber.New(fiber.Config{Immutable: true}),
		db:        db,
		redis:     redis,
		auth:      auth.NewJWTService("your-secret-key-change-in-production", 24*7),
		smsService: sms.NewMockSMSService(),
		memory:    memoryService,
	}

	// Middleware
	app.Use(requestID())
	app.Use(recovery())

	// Auth middleware
	app.Use("/api", authMiddleware(app.auth, app.db))

	// Routes
	api := app.Group("/api")

	// Auth routes
	authGroup := api.Group("/auth")
	authGroup.Post("/send-code", app.sendVerificationCode)
	authGroup.Post("/register", app.register)
	authGroup.Post("/login", app.login)
	authGroup.Post("/logout", app.logout)

	// Profile routes
	profileGroup := api.Group("/profile")
	profileGroup.Get("/me", app.getMyProfile)
	profileGroup.Put("/me", app.updateMyProfile)
	profileGroup.Put("/flirt-style", app.updateFlirtStyle)
	profileGroup.Get("/users/:userId", app.getOtherProfile)

	// Conversation routes
	conversationGroup := api.Group("/conversations")
	conversationGroup.Get("/", app.getConversations)
	conversationGroup.Get("/:id/messages", app.getMessages)
	conversationGroup.Post("/:id/messages", app.sendMessage)

	// AI routes
	aiGroup := api.Group("/ai")
	aiGroup.Get("/suggestions/:conversation_id", app.getAISuggestions)

	// WebSocket routes
	app.Use("/ws", func(c *fiber.Ctx) error {
		// Extract token from query param
		token := c.Query("token")
		if token == "" {
			return c.Status(http.StatusUnauthorized).SendString("Missing token")
		}

		// Validate token
		userID, err := app.auth.ValidateToken(token)
		if err != nil {
			return c.Status(http.StatusUnauthorized).SendString("Invalid token")
		}

		c.Locals("user_id", userID)
		return c.Next()
	})
	app.Get("/ws", websocket.New(app.HandleUpgrade, websocket.Config{
		HandshakeTimeout: 10,
		WriteBufferSize:  1024,
		ReadBufferSize:   1024,
		CheckOrigin: func(r *fiber.Ctx) bool {
			return true // Allow all origins in development
		},
	}))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"version": "1.0.0",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	return app
}

// Middleware
func requestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals("request_id", c.Get("X-Request-ID"))
		return c.Next()
	}
}

func recovery() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if err := recover(); err != nil {
				c.Status(http.StatusInternalServerError).JSON(fiber.Map{
					"error": "Internal server error",
				})
			}
		}()
		return c.Next()
	}
}

func authMiddleware(jwtService *auth.JWTService, db *db.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Get("Authorization")
		if token == "" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authorization header required",
			})
		}

		// Remove "Bearer " prefix
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		userID, err := jwtService.ValidateToken(token)
		if err != nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token",
			})
		}

		c.Locals("user_id", userID)
		return c.Next()
	}
}

// Auth handlers
func (a *App) sendVerificationCode(c *fiber.Ctx) error {
	var req sms.GenerateVerificationCodePayload
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if len(req.Phone) != 11 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid phone number",
		})
	}

	code := generateRandomCode()
	if err := a.smsService.SendCode(req.Phone, code); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to send verification code",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Verification code sent",
		"phone":   req.Phone,
	})
}

func (a *App) register(c *fiber.Ctx) error {
	var req models.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate verification code
	if err := a.smsService.VerifyCode(req.Phone, req.Code); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid verification code",
		})
	}

	// Check if user already exists
	var existingUser models.User
	err := a.db.QueryRow(`
		SELECT id, phone, nickname, gender, age, avatar_url, bio, flirt_style, created_at, updated_at
		FROM users WHERE phone = $1
	`, req.Phone).Scan(
		&existingUser.ID, &existingUser.Phone, &existingUser.Nickname,
		&existingUser.Gender, &existingUser.Age, &existingUser.AvatarURL,
		&existingUser.Bio, &existingUser.FlirtStyle, &existingUser.CreatedAt,
		&existingUser.UpdatedAt,
	)

	if err == nil {
		return c.Status(http.StatusConflict).JSON(fiber.Map{
			"error": "User already exists",
		})
	}

	// Create new user
	userID := uuid.New()
	flirtStyle := req.FlirtStyle
	if flirtStyle == "" {
		flirtStyle = "humorous"
	}

	_, err = a.db.Exec(`
		INSERT INTO users (id, phone, nickname, gender, age, avatar_url, bio, flirt_style)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, userID, req.Phone, req.Nickname, req.Gender, req.Age, nil, nil, flirtStyle)

	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user",
		})
	}

	// Generate JWT token
	token, err := a.auth.GenerateToken(userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	return c.Status(http.StatusCreated).JSON(models.AuthResponse{
		User: &models.User{
			ID:         userID,
			Phone:      req.Phone,
			Nickname:   req.Nickname,
			Gender:     req.Gender,
			Age:        req.Age,
			Bio:        nil,
			AvatarURL:  nil,
			FlirtStyle: flirtStyle,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		Token: token,
	})
}

func (a *App) login(c *fiber.Ctx) error {
	var req models.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Verify verification code
	if err := a.smsService.VerifyCode(req.Phone, req.Code); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid verification code",
		})
	}

	// Get user by phone
	var user models.User
	err := a.db.QueryRow(`
		SELECT id, phone, nickname, gender, age, avatar_url, bio, flirt_style, created_at, updated_at
		FROM users WHERE phone = $1
	`, req.Phone).Scan(
		&user.ID, &user.Phone, &user.Nickname,
		&user.Gender, &user.Age, &user.AvatarURL,
		&user.Bio, &user.FlirtStyle, &user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Generate JWT token
	token, err := a.auth.GenerateToken(user.ID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	return c.JSON(models.AuthResponse{
		User:  &user,
		Token: token,
	})
}

func (a *App) logout(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}

// Profile handlers
func (a *App) getMyProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)

	var user models.User
	err := a.db.QueryRow(`
		SELECT id, phone, nickname, gender, age, avatar_url, bio, flirt_style, created_at, updated_at
		FROM users WHERE id = $1
	`, userID).Scan(
		&user.ID, &user.Phone, &user.Nickname,
		&user.Gender, &user.Age, &user.AvatarURL,
		&user.Bio, &user.FlirtStyle, &user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	return c.JSON(user)
}

func (a *App) updateMyProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)

	var req models.UpdateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	result, err := a.db.Exec(`
		UPDATE users
		SET nickname = COALESCE($1, nickname),
		    gender = COALESCE($2, gender),
		    age = COALESCE($3, age),
		    bio = COALESCE($4, bio),
		    avatar_url = COALESCE($5, avatar_url)
		WHERE id = $6
	`, req.Nickname, req.Gender, req.Age, req.Bio, req.AvatarURL, userID)

	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update profile",
		})
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "Profile updated successfully",
	})
}

func (a *App) updateFlirtStyle(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(uuid.UUID)

	var req models.UpdateFlirtStyleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	result, err := a.db.Exec(`
		UPDATE users SET flirt_style = $1 WHERE id = $2
	`, req.FlirtStyle, userID)

	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update flirt style",
		})
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "Flirt style updated successfully",
	})
}

func (a *App) getOtherProfile(c *fiber.Ctx) error {
	userIDStr := c.Params("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	var user models.User
	err = a.db.QueryRow(`
		SELECT id, phone, nickname, gender, age, avatar_url, bio, flirt_style, created_at, updated_at
		FROM users WHERE id = $1
	`, userID).Scan(
		&user.ID, &user.Phone, &user.Nickname,
		&user.Gender, &user.Age, &user.AvatarURL,
		&user.Bio, &user.FlirtStyle, &user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	return c.JSON(user)
}

func generateRandomCode() string {
	const digits = "0123456789"
	const length = 6

	b := make([]byte, length)
	rand.Read(b)
	for i := range b {
		b[i] = digits[int(b[i])%len(digits)]
	}
	return string(b)
}
