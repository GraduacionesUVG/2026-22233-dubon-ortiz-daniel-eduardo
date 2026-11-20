package ports

import (
	"context"

	"github.com/DanielDubon/Gimpy/internal/domain"
)

// LLMService defines the contract for AI-powered workout generation
type LLMService interface {
	// GenerateRoutine creates a personalized workout routine based on user data and context
	// onTextChunk is called for each text fragment during streaming and can be nil for non-streaming
	GenerateRoutine(ctx context.Context, user domain.User, contextInfo string, onTextChunk func(string)) (domain.Workout, error)
}
