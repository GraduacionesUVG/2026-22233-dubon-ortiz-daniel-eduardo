package domain

import (
	"errors"
	"time"
)

// Injury represents a user injury with timestamp tracking
type Injury struct {
	Description string    `json:"description"`
	ReportedAt  time.Time `json:"reported_at"`
}

// User represents a gym member in the system
type User struct {
	ID           string   `json:"id"`
	Email        string   `json:"email"`
	PasswordHash string   `json:"-"` // no expose of the password hash in JSON
	Name         string   `json:"name"`
	FitnessGoals []string `json:"fitness_goals"`
	Injuries     []Injury `json:"injuries,omitempty"`
}

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
)
