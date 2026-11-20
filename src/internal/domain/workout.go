package domain

import "time"

// Workout represents a training session for a user
type Workout struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	Date         time.Time  `json:"date"`
	Message      string     `json:"message"`
	Exercises    []Exercise `json:"exercises"`
	FatigueLevel int        `json:"fatigue_level"`
	Notes        string     `json:"notes"`
}
