package mocks

import (
	"context"

	"github.com/DanielDubon/Gimpy/internal/domain"
	"github.com/stretchr/testify/mock"
)

// MockUserRepository is a testify-based mock implementation of ports.UserRepository
type MockUserRepository struct {
	mock.Mock
}

// Create stores a new user in the repository
func (m *MockUserRepository) Create(ctx context.Context, user domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// FindByEmail retrieves a user by their email address
func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	args := m.Called(ctx, email)

	var user domain.User
	if args.Get(0) != nil {
		user = args.Get(0).(domain.User)
	}

	return user, args.Error(1)
}

// FindByID retrieves a user by their unique identifier
func (m *MockUserRepository) FindByID(ctx context.Context, id string) (domain.User, error) {
	args := m.Called(ctx, id)

	var user domain.User
	if args.Get(0) != nil {
		user = args.Get(0).(domain.User)
	}

	return user, args.Error(1)
}

// UpdateUserInjuries intelligently merges new injury descriptions with existing injuries
func (m *MockUserRepository) UpdateUserInjuries(ctx context.Context, userID string, newInjuryDescriptions []string) error {
	args := m.Called(ctx, userID, newInjuryDescriptions)
	return args.Error(0)
}
