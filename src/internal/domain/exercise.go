package domain

// Exercise represents a single exercise within a workout
type Exercise struct {
	ExerciseID  string  `json:"exercise_id"`
	DisplayName string  `json:"display_name"`
	Sets        int     `json:"sets"`
	Reps        int     `json:"reps"`
	Weight      float64 `json:"weight"`
}
