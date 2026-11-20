package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/DanielDubon/Gimpy/internal/ports"
)

type WorkoutHandler struct {
	workoutRepo ports.WorkoutRepository
}

func NewWorkoutHandler(workoutRepo ports.WorkoutRepository) *WorkoutHandler {
	return &WorkoutHandler{
		workoutRepo: workoutRepo,
	}
}

// GetHistory gets the workout history for the authenticated user
func (h *WorkoutHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract user ID from context
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: "User ID not found in context",
		})
		return
	}

	// Get workout history
	workouts, err := h.workoutRepo.GetHistory(r.Context(), userID)
	if err != nil {
		slog.Error("Error retrieving workout history", "user_id", userID, "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: "Failed to retrieve workout history",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(workouts); err != nil {
		slog.Error("Error encoding workout history response", "error", err)
	}
}
