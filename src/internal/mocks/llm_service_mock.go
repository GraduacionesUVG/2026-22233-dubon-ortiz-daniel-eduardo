package mocks

import (
	"context"

	"github.com/DanielDubon/Gimpy/internal/domain"
	"github.com/stretchr/testify/mock"
)

// MockLLMService is a testify-based mock implementation of ports.LLMService
type MockLLMService struct {
	mock.Mock
}

func (m *MockLLMService) GenerateRoutine(ctx context.Context, user domain.User, contextInfo string, onTextChunk func(string)) (domain.Workout, error) {
	args := m.Called(ctx, user, contextInfo, onTextChunk)

	// If chunks were provided as an extra argument via mock.Anything setup, the caller
	// can trigger them manually using the onTextChunk passed here. To simulate streaming,
	// we invoke onTextChunk with any chunks configured through the "Chunks" mock argument.
	if onTextChunk != nil {
		if chunks, ok := args.Get(2).([]string); ok {
			for _, chunk := range chunks {
				onTextChunk(chunk)
			}
		}
	}

	var workout domain.Workout
	if args.Get(0) != nil {
		workout = args.Get(0).(domain.Workout)
	}

	return workout, args.Error(1)
}
