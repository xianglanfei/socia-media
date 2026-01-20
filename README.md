# Chinese Conversation & Flirt Assist App

A QQ-like mobile app for Chinese users to chat and receive AI-powered conversation/flirt assistance using memory-based context strategy (similar to Letta.com).

## Tech Stack

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Mobile App | Flutter | Cross-platform (iOS + Android), single codebase |
| Backend API | Go (Fiber) | High performance, excellent for real-time chat |
| Database | PostgreSQL | Reliable, supports complex queries |
| Cache | Redis | Session management, rate limiting |
| Real-time | WebSocket | Bi-directional messaging |
| AI/LLM | Qwen/DeepSeek API | Chinese-optimized models |
| Storage | MinIO/S3 | Media files |

## Project Structure

```
socia-media/
├── mobile/                      # Flutter app
│   ├── lib/
│   │   ├── models/             # Data models
│   │   ├── screens/            # UI screens
│   │   ├── services/           # API & WebSocket services
│   │   └── main.dart
│   └── pubspec.yaml
│
├── backend/                     # Go backend
│   ├── cmd/server/main.go      # Entry point
│   ├── internal/
│   │   ├── api/               # HTTP handlers
│   │   ├── websocket/         # Real-time chat
│   │   ├── auth/              # JWT auth
│   │   ├── memory/            # Memory agent (Letta-like)
│   │   ├── llm/               # LLM integration
│   │   ├── db/                # Database layer
│   │   └── models/            # Data models
│   └── go.mod
│
└── docs/                       # Documentation
    └── api.md
```

## Features

### 1. User Authentication
- Phone number + SMS verification
- JWT token-based authentication
- User registration with profile

### 2. Real-time Chat
- WebSocket connections
- Message types: text, image, voice
- Message status: sent, delivered, read
- Typing indicators
- Online/offline status

### 3. AI Conversation Assistant
- Memory-based context tracking
- Flirt stage detection (5 stages)
- 3 AI response suggestions per message
- Custom flirt styles:
  - 直球型
  - 幽默风趣
  - 温柔浪漫
  - 含蓄内敛

### 4. Flirt Stages
```
Stage 0: 冷启动 - First contact
Stage 1: 破冰 - Initial conversation
Stage 2: 热身 - Regular chatting
Stage 3: 暧昧 - Romantic tension
Stage 4: 深入 - Intimate connection
```

## Quick Start

### Prerequisites

- Flutter 3.0+
- Go 1.21+
- PostgreSQL 14+
- Redis 7+

### Backend Setup

1. **Copy environment variables:**
   ```bash
   cd backend
   cp .env.example .env
   ```

2. **Update .env with your values:**
   - Set `POSTGRES_URL` to your PostgreSQL connection string
   - Set `REDIS_URL` to your Redis address
   - Set `LLM_API_KEY` to your Qwen/DeepSeek API key

3. **Run migrations:**
   ```bash
   # The server will run migrations automatically on startup
   ```

4. **Start the server:**
   ```bash
   cd backend
   go run cmd/server/main.go
   ```

The server will start on `http://localhost:8080`

### Mobile App Setup

1. **Install dependencies:**
   ```bash
   cd mobile
   flutter pub get
   ```

2. **Update API URL:**
   Edit `lib/services/api_service.dart`:
   ```dart
   static const String baseUrl = 'http://localhost:8080';
   ```

3. **Run the app:**
   ```bash
   flutter run
   ```

## API Documentation

See [docs/api.md](docs/api.md) for complete API documentation.

## Development

### Running Tests

```bash
# Backend tests
cd backend
go test ./...

# Flutter tests
cd mobile
flutter test
```

### Building for Production

**Backend:**
```bash
cd backend
go build -o bin/server cmd/server/main.go
```

**Mobile:**
```bash
cd mobile
# Android
flutter build apk

# iOS
flutter build ios
```

## Configuration

### Flirt Styles

| Style | Description |
|--------|-------------|
| direct | Straightforward and confident |
| humorous | Witty and fun |
| romantic | Gentle and romantic |
| subtle | Low-key and meaningful |

### Message Status

- `sent` - Message sent
- `delivered` - Message delivered to recipient
- `read` - Message read by recipient

## License

MIT
