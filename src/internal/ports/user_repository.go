package ports

import (
	"context"

	"github.com/DanielDubon/Gimpy/internal/domain"
)

// UserRepository defines the contract for user data persistence
type UserRepository interface {
	// Create stores a new user in the repository
	Create(ctx context.Context, user domain.User) error

	// FindByEmail retrieves a user by their email address
	FindByEmail(ctx context.Context, email string) (domain.User, error)

	// FindByID retrieves a user by their unique identifier
	FindByID(ctx context.Context, id string) (domain.User, error)

	// UpdateUserInjuries intelligently merges new injury descriptions with existing injuries
	UpdateUserInjuries(ctx context.Context, userID string, newInjuryDescriptions []string) error
}
