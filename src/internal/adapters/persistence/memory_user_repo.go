package persistence

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/DanielDubon/Gimpy/internal/domain"
)

// MemoryUserRepository implements UserRepository interface with in-memory storage
type MemoryUserRepository struct {
	users map[string]domain.User
	mu    sync.RWMutex
}

// NewMemoryUserRepository creates a new in-memory user repository
func NewMemoryUserRepository() *MemoryUserRepository {
	return &MemoryUserRepository{
		users: make(map[string]domain.User),
	}
}

// Create stores a new user in the repository
func (r *MemoryUserRepository) Create(ctx context.Context, user domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate if email already exists
	lowerEmail := strings.ToLower(user.Email)
	for _, existingUser := range r.users {
		if strings.ToLower(existingUser.Email) == lowerEmail {
			return domain.ErrEmailAlreadyExists
		}
	}

	// Generate unique ID if not provided
	if user.ID == "" {
		user.ID = fmt.Sprintf("user_%d", time.Now().UnixNano())
	}

	// Store user in map
	r.users[user.ID] = user

	return nil
}

// FindByEmail retrieves a user by their email address
func (r *MemoryUserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Search for user with matching email
	lowerEmail := strings.ToLower(email)
	for _, user := range r.users {
		if strings.ToLower(user.Email) == lowerEmail {
			return user, nil
		}
	}

	return domain.User{}, domain.ErrUserNotFound
}

// FindByID retrieves a user by their unique identifier
func (r *MemoryUserRepository) FindByID(ctx context.Context, id string) (domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Search for user by ID
	user, exists := r.users[id]
	if !exists {
		return domain.User{}, domain.ErrUserNotFound
	}

	return user, nil
}

// UpdateUserInjuries intelligently merges new injury descriptions with existing injuries
// - New injuries are added with current timestamp
// - Existing injuries (by description) are preserved with original timestamp
// - Empty array clears all injuries if user reports full recovery
func (r *MemoryUserRepository) UpdateUserInjuries(ctx context.Context, userID string, newInjuryDescriptions []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Find user
	user, exists := r.users[userID]
	if !exists {
		return domain.ErrUserNotFound
	}

	// If empty array, clear all injuries
	if len(newInjuryDescriptions) == 0 {
		user.Injuries = []domain.Injury{}
		r.users[userID] = user
		return nil
	}

	// Build map of existing injuries by description for lookup
	existingInjuriesMap := make(map[string]domain.Injury)
	for _, injury := range user.Injuries {
		existingInjuriesMap[injury.Description] = injury
	}

	// Merge logic: preserve existing timestamps, add new injuries with current timestamp
	mergedInjuries := []domain.Injury{}
	for _, desc := range newInjuryDescriptions {
		if existingInjury, exists := existingInjuriesMap[desc]; exists {
			// Preserve existing injury with original timestamp
			mergedInjuries = append(mergedInjuries, existingInjury)
		} else {
			// Add new injury with current timestamp
			mergedInjuries = append(mergedInjuries, domain.Injury{
				Description: desc,
				ReportedAt:  time.Now(),
			})
		}
	}

	// Update user with merged injuries
	user.Injuries = mergedInjuries
	r.users[userID] = user

	return nil
}
