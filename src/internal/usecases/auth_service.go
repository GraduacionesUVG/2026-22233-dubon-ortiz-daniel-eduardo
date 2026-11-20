package usecases

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/DanielDubon/Gimpy/internal/domain"
	"github.com/DanielDubon/Gimpy/internal/ports"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles user authentication and authorization
type AuthService struct {
	repo      ports.UserRepository
	jwtSecret []byte
}

// NewAuthService creates a new AuthService with dependency injection
func NewAuthService(repo ports.UserRepository, jwtSecret []byte) *AuthService {
	return &AuthService{
		repo:      repo,
		jwtSecret: jwtSecret,
	}
}

// Register creates a new user account
func (s *AuthService) Register(ctx context.Context, email, password, name string, goals []string, injuries []string) error {
	// Generate password hash using bcrypt
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Convert injury strings to Injury objects with timestamps
	injuryObjects := make([]domain.Injury, len(injuries))
	for i, desc := range injuries {
		injuryObjects[i] = domain.Injury{
			Description: desc,
			ReportedAt:  time.Now(),
		}
	}

	// Build user domain object
	user := domain.User{
		Email:        email,
		PasswordHash: string(passwordHash),
		Name:         name,
		FitnessGoals: goals,
		Injuries:     injuryObjects,
	}

	// Create user in repository
	if err := s.repo.Create(ctx, user); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// Login authenticates a user and returns a JWT token
func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	// Find user by email
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", errors.New("invalid credentials")
		}
		return "", fmt.Errorf("failed to find user: %w", err)
	}

	// Compare password with hash
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", errors.New("invalid credentials")
	}

	// Generate JWT token
	expirationTime := time.Now().Add(72 * time.Hour)
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"exp":     expirationTime.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}
