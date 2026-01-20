# API Documentation

## Overview
This is the API documentation for the Chinese Conversation & Flirt Assist App.

**Base URL:** `http://localhost:8080`

## Authentication
All API endpoints (except `/api/auth/*`) require JWT authentication.

```
Authorization: Bearer <token>
```

## Endpoints

### Health Check
```http
GET /health
```

**Response:**
```json
{
  "status": "ok",
  "version": "1.0.0",
  "time": "2024-01-20T10:00:00Z"
}
```

---

### Authentication

#### Send Verification Code
```http
POST /api/auth/send-code
```

**Request Body:**
```json
{
  "phone": "13800138000"
}
```

**Response:**
```json
{
  "message": "Verification code sent",
  "phone": "13800138000"
}
```

#### Register
```http
POST /api/auth/register
```

**Request Body:**
```json
{
  "phone": "13800138000",
  "code": "123456",
  "nickname": "小明",
  "gender": "male",
  "age": 25,
  "flirt_style": "humorous"
}
```

**Response:**
```json
{
  "user": {
    "id": "uuid",
    "phone": "13800138000",
    "nickname": "小明",
    "gender": "male",
    "age": 25,
    "avatar_url": null,
    "bio": null,
    "flirt_style": "humorous",
    "created_at": "2024-01-20T10:00:00Z",
    "updated_at": "2024-01-20T10:00:00Z"
  },
  "token": "jwt-token"
}
```

#### Login
```http
POST /api/auth/login
```

**Request Body:**
```json
{
  "phone": "13800138000",
  "code": "123456"
}
```

**Response:**
```json
{
  "user": {...},
  "token": "jwt-token"
}
```

#### Logout
```http
POST /api/auth/logout
```

**Response:**
```json
{
  "message": "Logged out successfully"
}
```

---

### Profile

#### Get My Profile
```http
GET /api/profile/me
```

**Response:**
```json
{
  "id": "uuid",
  "phone": "13800138000",
  "nickname": "小明",
  "gender": "male",
  "age": 25,
  "avatar_url": null,
  "bio": "Personal bio",
  "flirt_style": "humorous",
  "created_at": "2024-01-20T10:00:00Z",
  "updated_at": "2024-01-20T10:00:00Z"
}
```

#### Update My Profile
```http
PUT /api/profile/me
```

**Request Body:**
```json
{
  "nickname": "新昵称",
  "gender": "male",
  "age": 26,
  "bio": "Updated bio",
  "avatar_url": "https://example.com/avatar.jpg"
}
```

#### Update Flirt Style
```http
PUT /api/profile/flirt-style
```

**Request Body:**
```json
{
  "flirt_style": "direct"
}
```

**Flirt Styles:**
- `direct` - 直球型
- `humorous` - 幽默风趣
- `romantic` - 温柔浪漫
- `subtle` - 含蓄内敛

#### Get Other User's Profile
```http
GET /api/profile/users/:userId
```

---

### Conversations

#### Get Conversations
```http
GET /api/conversations
```

**Response:**
```json
{
  "conversations": [
    {
      "id": "uuid",
      "user1_id": "uuid",
      "user2_id": "uuid",
      "last_message_at": "2024-01-20T10:00:00Z",
      "other_user": {
        "id": "uuid",
        "nickname": "小红",
        "avatar_url": "https://example.com/avatar.jpg"
      },
      "last_message": {
        "id": "uuid",
        "content": "Hello!",
        "message_type": "text",
        "status": "read",
        "created_at": "2024-01-20T10:00:00Z"
      },
      "unread_count": 2,
      "stage": 2
    }
  ]
}
```

**Stages:**
- `0` - 冷启动 (Cold Start)
- `1` - 破冰 (Breaking Ice)
- `2` - 热身 (Warm Up)
- `3` - 暧昧 (Flirty)
- `4` - 深入 (Deep)

#### Get Messages
```http
GET /api/conversations/:id/messages?limit=50
```

**Response:**
```json
{
  "messages": [
    {
      "id": "uuid",
      "conversation_id": "uuid",
      "sender_id": "uuid",
      "content": "Hello!",
      "message_type": "text",
      "status": "read",
      "created_at": "2024-01-20T10:00:00Z"
    }
  ]
}
```

#### Send Message
```http
POST /api/conversations/:id/messages
```

**Request Body:**
```json
{
  "content": "Hello!",
  "message_type": "text"
}
```

**Response:**
```json
{
  "id": "uuid",
  "conversation_id": "uuid",
  "sender_id": "uuid",
  "content": "Hello!",
  "message_type": "text",
  "status": "sent",
  "created_at": "2024-01-20T10:00:00Z"
}
```

---

### AI Suggestions

#### Get AI Suggestions
```http
GET /api/ai/suggestions/:conversation_id
```

**Response:**
```json
{
  "conversation_id": "uuid",
  "stage": 2,
  "suggestions": [
    {
      "text": "这就对啦，我就知道你懂的！",
      "style": "直球型",
      "reason": "肯定对方的观点，同时展现自信"
    },
    {
      "text": "哈哈，你这人说话真是又准又逗，跟你聊天很有意思",
      "style": "幽默风趣",
      "reason": "用轻松的语气赞美对方，增加互动趣味"
    },
    {
      "text": "和你聊天感觉很舒服，好像认识了很久一样",
      "style": "温柔浪漫",
      "reason": "表达对交流的愉悦感受，拉近心理距离"
    }
  ]
}
```

---

### WebSocket

#### Connect to WebSocket
```
ws://localhost:8080/ws?token=<jwt-token>
```

#### WebSocket Events

**Client → Server:**

Connect:
```json
{
  "type": "connect",
  "user_id": "uuid"
}
```

Send Message:
```json
{
  "type": "message",
  "conversation_id": "uuid",
  "content": "Hello!",
  "message_type": "text"
}
```

Typing:
```json
{
  "type": "typing",
  "conversation_id": "uuid",
  "is_typing": true
}
```

Read Receipt:
```json
{
  "type": "read",
  "conversation_id": "uuid",
  "message_ids": ["uuid1", "uuid2"]
}
```

Disconnect:
```json
{
  "type": "disconnect"
}
```

**Server → Client:**

Connect Confirmation:
```json
{
  "type": "connect",
  "data": {
    "user_id": "uuid"
  }
}
```

New Message:
```json
{
  "type": "message",
  "data": {
    "id": "uuid",
    "conversation_id": "uuid",
    "sender_id": "uuid",
    "content": "Hello!",
    "message_type": "text",
    "status": "delivered",
    "created_at": "2024-01-20T10:00:00Z"
  }
}
```

Typing Indicator:
```json
{
  "type": "typing",
  "data": {
    "conversation_id": "uuid",
    "is_typing": true
  }
}
```

Read Receipt:
```json
{
  "type": "read",
  "data": {
    "conversation_id": "uuid",
    "message_ids": ["uuid1", "uuid2"]
  }
}
```

## Error Responses

All endpoints may return the following error responses:

**400 Bad Request:**
```json
{
  "error": "Invalid request body"
}
```

**401 Unauthorized:**
```json
{
  "error": "Authorization header required"
}
```

**403 Forbidden:**
```json
{
  "error": "Access denied"
}
```

**404 Not Found:**
```json
{
  "error": "Resource not found"
}
```

**500 Internal Server Error:**
```json
{
  "error": "Internal server error"
}
```
