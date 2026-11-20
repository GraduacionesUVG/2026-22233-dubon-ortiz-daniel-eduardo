package mocks

import (
	"context"

	"github.com/DanielDubon/Gimpy/internal/domain"
	"github.com/stretchr/testify/mock"
)

// MockChatRepository is a testify-based mock implementation of ports.ChatRepository
type MockChatRepository struct {
	mock.Mock
}

// SaveChatMessage saves a single chat message to the database
func (m *MockChatRepository) SaveChatMessage(ctx context.Context, userID string, role string, content string) error {
	args := m.Called(ctx, userID, role, content)
	return args.Error(0)
}

// GetRecentChatHistory retrieves the most recent chat messages for a user
func (m *MockChatRepository) GetRecentChatHistory(ctx context.Context, userID string, limit int) ([]domain.ChatMessage, error) {
	args := m.Called(ctx, userID, limit)

	var messages []domain.ChatMessage
	if args.Get(0) != nil {
		messages = args.Get(0).([]domain.ChatMessage)
	}

	return messages, args.Error(1)
}
