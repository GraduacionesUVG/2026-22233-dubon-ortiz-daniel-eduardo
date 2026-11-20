package ports

import (
	"context"

	"github.com/DanielDubon/Gimpy/internal/domain"
)

// ChatRepository defines the interface for chat message persistence
type ChatRepository interface {
	// SaveChatMessage saves a single chat message to the database
	SaveChatMessage(ctx context.Context, userID string, role string, content string) error

	// GetRecentChatHistory retrieves the most recent chat messages for a user
	// Returns messages in chronological order (oldest first)
	GetRecentChatHistory(ctx context.Context, userID string, limit int) ([]domain.ChatMessage, error)
}
