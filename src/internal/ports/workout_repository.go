package ports

import (
	"context"

	"github.com/DanielDubon/Gimpy/internal/domain"
)

// WorkoutRepository defines the contract for workout data persistence
type WorkoutRepository interface {
	// Save stores a workout for a specific user
	Save(ctx context.Context, userID string, workout domain.Workout) error

	// GetHistory retrieves all workouts for a specific user
	GetHistory(ctx context.Context, userID string) ([]domain.Workout, error)

	// UpdateWorkoutFeedback updates the fatigue level and notes of the user's most recent workout
	UpdateWorkoutFeedback(ctx context.Context, userID string, fatigueLevel int, notes string) error

	// GetAllowedExercises retrieves the catalog of allowed exercises for a given muscle group
	GetAllowedExercises(ctx context.Context, muscleGroup string) ([]domain.CatalogExercise, error)
}
