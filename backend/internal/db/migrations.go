package db

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// migrations contains all SQL migration files in order
var migrations = []string{
	`-- Users table
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		phone VARCHAR(20) UNIQUE NOT NULL,
		nickname VARCHAR(50) NOT NULL,
		gender VARCHAR(10),
		age INTEGER,
		avatar_url TEXT,
		bio TEXT,
		flirt_style VARCHAR(20) NOT NULL DEFAULT 'humorous',
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	)`,

	`-- Conversations table
	CREATE TABLE IF NOT EXISTS conversations (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user1_id UUID NOT NULL REFERENCES users(id),
		user2_id UUID NOT NULL REFERENCES users(id),
		last_message_at TIMESTAMP DEFAULT NOW(),
		UNIQUE(user1_id, user2_id)
	);

	CREATE INDEX idx_conversations_user1 ON conversations(user1_id);
	CREATE INDEX idx_conversations_user2 ON conversations(user2_id);`,

	`-- Messages table
	CREATE TABLE IF NOT EXISTS messages (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
		sender_id UUID NOT NULL REFERENCES users(id),
		content TEXT NOT NULL,
		message_type VARCHAR(20) DEFAULT 'text',
		status VARCHAR(20) DEFAULT 'sent',
		created_at TIMESTAMP DEFAULT NOW()
	);

	CREATE INDEX idx_messages_conversation ON messages(conversation_id, created_at);
	CREATE INDEX idx_messages_sender ON messages(sender_id);`,

	`-- Memory context table
	CREATE TABLE IF NOT EXISTS memory_context (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
		user_id UUID NOT NULL REFERENCES users(id),
		stage INTEGER DEFAULT 0,
		target_traits JSONB,
		successful_patterns JSONB,
		updated_at TIMESTAMP DEFAULT NOW(),
		UNIQUE(conversation_id, user_id)
	);

	CREATE INDEX idx_memory_conversation ON memory_context(conversation_id);`,

	`-- AI suggestions log table
	CREATE TABLE IF NOT EXISTS ai_suggestions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
		suggestion TEXT NOT NULL,
		was_used BOOLEAN DEFAULT FALSE,
		response_received BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT NOW()
	);

	CREATE INDEX idx_ai_suggestions_conversation ON ai_suggestions(conversation_id);`,

	`-- Function to update updated_at timestamp
	CREATE OR REPLACE FUNCTION update_updated_at_column()
	RETURNS TRIGGER AS $$
	BEGIN
		NEW.updated_at = NOW();
		RETURN NEW;
	END;
	$$ LANGUAGE plpgsql;`,

	`-- Triggers for auto-updating updated_at
	CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
	FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

	CREATE TRIGGER update_memory_updated_at BEFORE UPDATE ON memory_context
	FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();`,
}

func RunMigrations(db *sql.DB) error {
	// Create migrations table if it doesn't exist
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id SERIAL PRIMARY KEY,
			version INTEGER NOT NULL UNIQUE,
			applied_at TIMESTAMP DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current migration version
	var currentVersion int
	row := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations")
	if err := row.Scan(&currentVersion); err != nil {
		return fmt.Errorf("failed to get current migration version: %w", err)
	}

	// Run pending migrations
	for i, migration := range migrations {
		version := i + 1
		if version <= currentVersion {
			continue
		}

		// Start transaction
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}

		// Execute migration
		if _, err := tx.Exec(migration); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d failed: %w", version, err)
		}

		// Record migration
		if _, err := tx.Exec(
			"INSERT INTO schema_migrations (version) VALUES ($1)",
			version,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", version, err)
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", version, err)
		}

		fmt.Printf("Migration %d applied successfully\n", version)
	}

	return nil
}
