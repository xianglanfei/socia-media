# Supabase Deployment Guide

## Step 1: Create Supabase Account & Project

1. Go to https://supabase.com
2. Sign up / Log in
3. Create a **New Project**:
   - Name: `socia-media`
   - Database Password: (generate a strong password)
   - Region: Choose a region near you (e.g., Singapore)

## Step 2: Get Project Credentials

After creating the project, note down:

| Credential | Where to find |
|-----------|--------------|
| **Project URL** | Settings → API |
| **anon public key** | Settings → API |
| **service_role key** | Settings → API |
| **Database URL** | Settings → Database → Connection String |
| **JWT Secret** | Settings → API |

## Step 3: Generate Personal Access Token (PAT)

1. Go to https://supabase.com/dashboard/account/tokens
2. Click **"Create new token"**
3. Name it: `socia-media-mcp`
4. Copy the token

## Step 4: Configure MCP Server

Run the Supabase MCP server with your token:

```bash
export SUPABASE_ACCESS_TOKEN=your_pat_token_here
node /home/eee/.local/share/fnm/node-versions/v24.12.0/installation/lib/node_modules/@supabase/mcp-server-supabase/dist/transports/stdio.js
```

## Step 5: Update Backend Code for Supabase

Modify `backend/internal/db/db.go`:

```go
package db

import (
    "database/sql"
    "fmt"
    "os"

    _ "github.com/lib/pq"
    "github.com/supabase/supabase-go" // Or use direct Postgres
)

func NewDB() (*sql.DB, error) {
    // Option 1: Use Supabase Go SDK
    // url := os.Getenv("SUPABASE_URL")
    // key := os.Getenv("SUPABASE_ANON_KEY")

    // Option 2: Use direct PostgreSQL connection
    postgresURL := getEnv("SUPABASE_POSTGRES_URL", "your_supabase_db_url_here")

    db, err := sql.Open("postgres", postgresURL)
    if err != nil {
        return nil, err
    }

    if err := db.Ping(); err != nil {
        return nil, err
    }

    return &DB{db}, nil
}
```

## Step 6: Add Supabase Realtime (instead of WebSocket)

Install Supabase Go SDK:

```bash
cd backend
go get github.com/supabase/supabase-go
```

Update `backend/internal/api/websocket.go`:

```go
package api

import (
    "context"
    "fmt"
    "sync"

    "github.com/supabase/supabase-go"
    "github.com/google/uuid"
)

type RealtimeManager struct {
    client   *supabase.Client
    channels map[string]*subscription
    mutex    sync.RWMutex
}

func NewRealtimeManager(projectURL, apiKey string) *RealtimeManager {
    client := supabase.NewClient(projectURL, apiKey)
    return &RealtimeManager{
        client:   client,
        channels: make(map[string]*subscription),
    }
}

// Subscribe to conversation updates
func (rm *RealtimeManager) SubscribeToConversation(ctx context.Context, conversationID uuid.UUID, callback func(msg map[string]interface{})) error {
    channel := fmt.Sprintf("conversation:%s", conversationID)

    sub, err := rm.client.Channel(channel).Subscribe()
    if err != nil {
        return err
    }

    go func() {
        for msg := range sub.Channel() {
            callback(msg.Payload)
        }
    }()

    rm.mutex.Lock()
    rm.channels[channel] = sub
    rm.mutex.Unlock()

    return nil
}
```

## Step 7: Deploy to Supabase Edge Functions

Create `supabase/functions/hello/hello.go`:

```go
package main

import (
    "net/http"
    "net/mail"
)

func HelloHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"message":"Hello from Supabase Edge Function!"}`))
}

func main() {
    http.ListenAndServe(":8080", nil)
}
```

Deploy:
```bash
npx supabase functions deploy hello
```

## Supabase Environment Variables

Add these to your Supabase project (Settings → Environment Variables):

| Key | Value |
|------|-------|
| `JWT_SECRET` | Your JWT secret |
| `LLM_API_KEY` | Your Qwen/DeepSeek API key |
| `REDIS_ENABLED` | `false` (Supabase handles caching) |

## Quick Start

```bash
# 1. Set environment variables
export SUPABASE_ACCESS_TOKEN=your_pat_token
export SUPABASE_URL=https://your-project.supabase.co
export SUPABASE_ANON_KEY=your_anon_key

# 2. Run MCP server
node ~/.npm-global/lib/node_modules/@supabase/mcp-server-supabase/dist/transports/stdio.js

# 3. Run backend (after updating for Supabase)
cd backend
go run cmd/server/main.go
```
