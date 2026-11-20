package mocks

import (
	"context"

	"github.com/DanielDubon/Gimpy/internal/domain"
	"github.com/stretchr/testify/mock"
)

// MockWorkoutRepository is a testify-based mock implementation of ports.WorkoutRepository
type MockWorkoutRepository struct {
	mock.Mock
}

// Save stores a workout for a specific user
func (m *MockWorkoutRepository) Save(ctx context.Context, userID string, workout domain.Workout) error {
	args := m.Called(ctx, userID, workout)
	return args.Error(0)
}

// GetHistory retrieves all workouts for a specific user
func (m *MockWorkoutRepository) GetHistory(ctx context.Context, userID string) ([]domain.Workout, error) {
	args := m.Called(ctx, userID)

	var workouts []domain.Workout
	if args.Get(0) != nil {
		workouts = args.Get(0).([]domain.Workout)
	}

	return workouts, args.Error(1)
}

// UpdateWorkoutFeedback updates the fatigue level and notes of the user's most recent workout
func (m *MockWorkoutRepository) UpdateWorkoutFeedback(ctx context.Context, userID string, fatigueLevel int, notes string) error {
	args := m.Called(ctx, userID, fatigueLevel, notes)
	return args.Error(0)
}

// GetAllowedExercises retrieves the catalog of allowed exercises for a given muscle group
func (m *MockWorkoutRepository) GetAllowedExercises(ctx context.Context, muscleGroup string) ([]domain.CatalogExercise, error) {
	args := m.Called(ctx, muscleGroup)

	var exercises []domain.CatalogExercise
	if args.Get(0) != nil {
		exercises = args.Get(0).([]domain.CatalogExercise)
	}

	return exercises, args.Error(1)
}
