package usecases

import (
	"context"
	"testing"
	"time"

	"github.com/DanielDubon/Gimpy/internal/domain"
	"github.com/DanielDubon/Gimpy/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestWorkoutGenerator_Success(t *testing.T) {
	userRepo := new(mocks.MockUserRepository)
	workoutRepo := new(mocks.MockWorkoutRepository)
	llmService := new(mocks.MockLLMService)

	generator := NewWorkoutGenerator(llmService, userRepo, workoutRepo)

	existingUser := domain.User{
		ID:           "user-123",
		Email:        "test@example.com",
		Name:         "Test User",
		FitnessGoals: []string{"build muscle", "lose weight"},
		Injuries: []domain.Injury{
			{Description: "knee pain", ReportedAt: time.Now()},
		},
	}

	userRepo.On("FindByID", mock.Anything, "user-123").Return(existingUser, nil)

	expectedWorkout := domain.Workout{
		ID:      "workout-456",
		UserID:  "user-123",
		Date:    time.Now(),
		Message: "Here is your personalized workout plan!",
		Exercises: []domain.Exercise{
			{ExerciseID: "ex-1", DisplayName: "Squat", Sets: 3, Reps: 10, Weight: 50},
			{ExerciseID: "ex-2", DisplayName: "Bench Press", Sets: 3, Reps: 8, Weight: 40},
		},
	}

	// Simulated streaming chunks that the mock will replay through the onTextChunk callback
	streamChunks := []string{"Generating", " your", " workout", "..."}

	llmService.On("GenerateRoutine", mock.Anything, existingUser, "Give me a workout for building muscle", mock.AnythingOfType("func(string)")).
		Return(expectedWorkout, nil, streamChunks)

	workoutRepo.On("Save", mock.Anything, "user-123", expectedWorkout).Return(nil)

	// Track streaming callback invocations to prove streaming logic works
	var receivedChunks []string
	onTextChunk := func(chunk string) {
		receivedChunks = append(receivedChunks, chunk)
	}

	result, err := generator.Generate(context.Background(), "user-123", "Give me a workout for building muscle", onTextChunk)

	assert.NoError(t, err)
	assert.Equal(t, expectedWorkout, result)
	assert.NotEmpty(t, receivedChunks, "expected streaming callback to be triggered")
	assert.Equal(t, streamChunks, receivedChunks)

	userRepo.AssertExpectations(t)
	llmService.AssertExpectations(t)
	workoutRepo.AssertExpectations(t)
}
