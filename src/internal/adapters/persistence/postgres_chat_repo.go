package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/DanielDubon/Gimpy/internal/domain"
	"github.com/google/uuid"
)

// PostgresChatRepository implements ChatRepository interface with PostgreSQL storage
type PostgresChatRepository struct {
	db *sql.DB
}

// NewPostgresChatRepository creates a new PostgreSQL chat repository
func NewPostgresChatRepository(db *sql.DB) *PostgresChatRepository {
	return &PostgresChatRepository{
		db: db,
	}
}

// SaveChatMessage saves a single chat message to the database
func (r *PostgresChatRepository) SaveChatMessage(ctx context.Context, userID string, role string, content string) error {
	query := `
		INSERT INTO chat_messages (id, user_id, role, content, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	messageID := uuid.New().String()
	createdAt := time.Now()

	_, err := r.db.ExecContext(ctx, query, messageID, userID, role, content, createdAt)
	if err != nil {
		slog.Error("Failed to save chat message", "operation", "SaveChatMessage", "user_id", userID, "role", role, "error", err)
		return fmt.Errorf("failed to save chat message: %w", err)
	}

	return nil
}

// GetRecentChatHistory retrieves the most recent chat messages for a user
// Returns messages in chronological order (oldest first)
func (r *PostgresChatRepository) GetRecentChatHistory(ctx context.Context, userID string, limit int) ([]domain.ChatMessage, error) {
	// Query to get recent messages in descending order
	query := `
		SELECT id, user_id, role, content, created_at
		FROM chat_messages
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		slog.Error("Failed to query chat history", "operation", "GetRecentChatHistory", "user_id", userID, "limit", limit, "error", err)
		return nil, fmt.Errorf("failed to query chat history: %w", err)
	}
	defer rows.Close()

	var messages []domain.ChatMessage
	for rows.Next() {
		var msg domain.ChatMessage
		err := rows.Scan(&msg.ID, &msg.UserID, &msg.Role, &msg.Content, &msg.CreatedAt)
		if err != nil {
			slog.Error("Failed to scan chat message", "operation", "GetRecentChatHistory", "user_id", userID, "error", err)
			return nil, fmt.Errorf("failed to scan chat message: %w", err)
		}
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		slog.Error("Error iterating chat messages", "operation", "GetRecentChatHistory", "user_id", userID, "error", err)
		return nil, fmt.Errorf("error iterating chat messages: %w", err)
	}

	// Reverse the slice to return messages in chronological order (oldest first)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}
