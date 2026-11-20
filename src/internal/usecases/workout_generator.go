package usecases

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/DanielDubon/Gimpy/internal/domain"
	"github.com/DanielDubon/Gimpy/internal/ports"
)

// WorkoutGenerator orchestrates workout generation using the LLM service
type WorkoutGenerator struct {
	llmService  ports.LLMService
	userRepo    ports.UserRepository
	workoutRepo ports.WorkoutRepository
}

// NewWorkoutGenerator creates a new WorkoutGenerator with dependency injection
func NewWorkoutGenerator(llmService ports.LLMService, userRepo ports.UserRepository, workoutRepo ports.WorkoutRepository) *WorkoutGenerator {
	return &WorkoutGenerator{
		llmService:  llmService,
		userRepo:    userRepo,
		workoutRepo: workoutRepo,
	}
}

// Generate creates a personalized workout based on the user's prompt
// onTextChunk is an optional callback for streaming text fragments
func (wg *WorkoutGenerator) Generate(ctx context.Context, userID string, prompt string, onTextChunk func(string)) (domain.Workout, error) {

	// Fetch real user from repository
	user, err := wg.userRepo.FindByID(ctx, userID)
	if err != nil {
		slog.Error("Failed to find user", "operation", "Generate", "user_id", userID, "error", err)
		return domain.Workout{}, fmt.Errorf("failed to find user: %w", err)
	}

	// Call the LLM service to generate the workout with streaming support
	workout, err := wg.llmService.GenerateRoutine(ctx, user, prompt, onTextChunk)
	if err != nil {
		slog.Error("Failed to generate workout", "operation", "Generate", "user_id", userID, "error", err)
		return domain.Workout{}, fmt.Errorf("failed to generate workout: %w", err)
	}

	// Save the generated workout to database only if it has exercises
	if len(workout.Exercises) > 0 {
		if err := wg.workoutRepo.Save(ctx, userID, workout); err != nil {
			slog.Error("Failed to save workout", "operation", "Generate", "user_id", userID, "workout_id", workout.ID, "error", err)
			return domain.Workout{}, fmt.Errorf("failed to save workout: %w", err)
		}
		slog.Info("Workout generated successfully", "user_id", userID, "workout_id", workout.ID, "exercise_count", len(workout.Exercises))
	} else {
		fmt.Printf("Warning: Workout generated without exercises for user %s, not saving to database\n", userID)
	}

	return workout, nil
}
