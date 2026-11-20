package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/DanielDubon/Gimpy/internal/domain"
	"github.com/DanielDubon/Gimpy/internal/mocks"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

var testJWTSecret = []byte("test-secret-key")

func TestAuthService_Register_Success(t *testing.T) {
	repo := new(mocks.MockUserRepository)
	service := NewAuthService(repo, testJWTSecret)

	repo.On("Create", mock.Anything, mock.MatchedBy(func(u domain.User) bool {
		return u.Email == "test@example.com" && u.Name == "Test User"
	})).Return(nil)

	err := service.Register(context.Background(), "test@example.com", "password123", "Test User", []string{"lose weight"}, []string{"knee pain"})

	assert.NoError(t, err)
	repo.AssertExpectations(t)
	repo.AssertCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestAuthService_Register_RepositoryError(t *testing.T) {
	repo := new(mocks.MockUserRepository)
	service := NewAuthService(repo, testJWTSecret)

	repo.On("Create", mock.Anything, mock.Anything).Return(domain.ErrEmailAlreadyExists)

	err := service.Register(context.Background(), "test@example.com", "password123", "Test User", nil, nil)

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrEmailAlreadyExists)
	repo.AssertExpectations(t)
}

func TestAuthService_Login_Success(t *testing.T) {
	repo := new(mocks.MockUserRepository)
	service := NewAuthService(repo, testJWTSecret)

	passwordHash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	assert.NoError(t, err)

	existingUser := domain.User{
		ID:           "user-123",
		Email:        "test@example.com",
		PasswordHash: string(passwordHash),
		Name:         "Test User",
	}

	repo.On("FindByEmail", mock.Anything, "test@example.com").Return(existingUser, nil)

	token, err := service.Login(context.Background(), "test@example.com", "password123")

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Validate token contents
	parsedToken, parseErr := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return testJWTSecret, nil
	})
	assert.NoError(t, parseErr)
	assert.True(t, parsedToken.Valid)

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	assert.True(t, ok)
	assert.Equal(t, "user-123", claims["user_id"])

	repo.AssertExpectations(t)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	repo := new(mocks.MockUserRepository)
	service := NewAuthService(repo, testJWTSecret)

	repo.On("FindByEmail", mock.Anything, "missing@example.com").Return(domain.User{}, domain.ErrUserNotFound)

	token, err := service.Login(context.Background(), "missing@example.com", "password123")

	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Equal(t, "invalid credentials", err.Error())
	repo.AssertExpectations(t)
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	repo := new(mocks.MockUserRepository)
	service := NewAuthService(repo, testJWTSecret)

	passwordHash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	assert.NoError(t, err)

	existingUser := domain.User{
		ID:           "user-123",
		Email:        "test@example.com",
		PasswordHash: string(passwordHash),
		Name:         "Test User",
	}

	repo.On("FindByEmail", mock.Anything, "test@example.com").Return(existingUser, nil)

	token, err := service.Login(context.Background(), "test@example.com", "wrong-password")

	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Equal(t, "invalid credentials", err.Error())
	repo.AssertExpectations(t)
}

func TestAuthService_Login_OtherRepositoryError(t *testing.T) {
	repo := new(mocks.MockUserRepository)
	service := NewAuthService(repo, testJWTSecret)

	repo.On("FindByEmail", mock.Anything, "test@example.com").Return(domain.User{}, errors.New("db connection failed"))

	token, err := service.Login(context.Background(), "test@example.com", "password123")

	assert.Error(t, err)
	assert.Empty(t, token)
	repo.AssertExpectations(t)
}
