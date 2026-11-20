package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/DanielDubon/Gimpy/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

// PostgresUserRepository implements UserRepository interface with PostgreSQL storage
type PostgresUserRepository struct {
	db *sql.DB
}

// NewPostgresUserRepository creates a new PostgreSQL user repository
func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{
		db: db,
	}
}

// Create stores a new user in the database
func (r *PostgresUserRepository) Create(ctx context.Context, user domain.User) error {
	// Generate unique ID if not provided
	if user.ID == "" {
		user.ID = fmt.Sprintf("user_%d", time.Now().UnixNano())
	}

	// Convert string slices to JSON
	fitnessGoalsJSON, err := json.Marshal(user.FitnessGoals)
	if err != nil {
		slog.Error("Failed to marshal fitness goals", "error", err, "user_id", user.ID)
		return fmt.Errorf("failed to marshal fitness goals: %w", err)
	}

	injuriesJSON, err := json.Marshal(user.Injuries)
	if err != nil {
		slog.Error("Failed to marshal injuries", "error", err, "user_id", user.ID)
		return fmt.Errorf("failed to marshal injuries: %w", err)
	}

	// Insert user into database
	query := `
		INSERT INTO users (id, email, password_hash, name, fitness_goals, injuries)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = r.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Name,
		string(fitnessGoalsJSON), // string, not []byte: pgx would encode []byte as bytea and break the jsonb column
		string(injuriesJSON),
	)

	if err != nil {
		// Check for unique constraint violation on email
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// PostgreSQL error code 23505 is unique_violation
			if pgErr.Code == "23505" && strings.Contains(pgErr.Message, "email") {
				slog.Error("User creation failed - email already exists", "email", user.Email, "error", err)
				return domain.ErrEmailAlreadyExists
			}
		}
		slog.Error("Database insert failed", "operation", "CreateUser", "user_id", user.ID, "error", err)
		return fmt.Errorf("failed to insert user: %w", err)
	}

	return nil
}

// FindByEmail retrieves a user by their email address
func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	query := `
		SELECT id, email, password_hash, name, fitness_goals, injuries
		FROM users
		WHERE LOWER(email) = LOWER($1)
	`

	var user domain.User
	var fitnessGoalsJSON, injuriesJSON []byte

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&fitnessGoalsJSON,
		&injuriesJSON,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, domain.ErrUserNotFound
		}
		slog.Error("Database query failed", "operation", "FindByEmail", "email", email, "error", err)
		return domain.User{}, fmt.Errorf("failed to query user: %w", err)
	}

	// Unmarshal JSON arrays
	if err := json.Unmarshal(fitnessGoalsJSON, &user.FitnessGoals); err != nil {
		slog.Error("Failed to unmarshal fitness goals", "operation", "FindByEmail", "user_id", user.ID, "error", err)
		return domain.User{}, fmt.Errorf("failed to unmarshal fitness goals: %w", err)
	}

	if err := json.Unmarshal(injuriesJSON, &user.Injuries); err != nil {
		slog.Error("Failed to unmarshal injuries", "operation", "FindByEmail", "user_id", user.ID, "error", err)
		return domain.User{}, fmt.Errorf("failed to unmarshal injuries: %w", err)
	}

	return user, nil
}

// UpdateUserInjuries intelligently merges new injury descriptions with existing injuries
// - New injuries are added with current timestamp
// - Existing injuries (by description) are preserved with original timestamp
// - Empty array clears all injuries if user reports full recovery
func (r *PostgresUserRepository) UpdateUserInjuries(ctx context.Context, userID string, newInjuryDescriptions []string) error {
	// If empty array, clear all injuries (user reports full recovery)
	if len(newInjuryDescriptions) == 0 {
		emptyInjuries := []domain.Injury{}
		injuriesJSON, err := json.Marshal(emptyInjuries)
		if err != nil {
			slog.Error("Failed to marshal empty injuries", "operation", "UpdateUserInjuries", "user_id", userID, "error", err)
			return fmt.Errorf("failed to marshal empty injuries: %w", err)
		}

		query := `UPDATE users SET injuries = $1 WHERE id = $2`
		result, err := r.db.ExecContext(ctx, query, string(injuriesJSON), userID)
		if err != nil {
			slog.Error("Database update failed", "operation", "ClearUserInjuries", "user_id", userID, "error", err)
			return fmt.Errorf("failed to clear user injuries: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			slog.Error("Failed to get rows affected", "operation", "ClearUserInjuries", "user_id", userID, "error", err)
			return fmt.Errorf("failed to get rows affected: %w", err)
		}

		if rowsAffected == 0 {
			return domain.ErrUserNotFound
		}

		return nil
	}

	// Fetch current user to get existing injuries
	currentUser, err := r.FindByID(ctx, userID)
	if err != nil {
		slog.Error("Failed to find user for injury update", "operation", "UpdateUserInjuries", "user_id", userID, "error", err)
		return err
	}

	// Build map of existing injuries by description for lookup
	existingInjuriesMap := make(map[string]domain.Injury)
	for _, injury := range currentUser.Injuries {
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

	// Convert merged injuries to JSON
	injuriesJSON, err := json.Marshal(mergedInjuries)
	if err != nil {
		slog.Error("Failed to marshal merged injuries", "operation", "UpdateUserInjuries", "user_id", userID, "error", err)
		return fmt.Errorf("failed to marshal merged injuries: %w", err)
	}

	query := `UPDATE users SET injuries = $1 WHERE id = $2`
	result, err := r.db.ExecContext(ctx, query, string(injuriesJSON), userID)
	if err != nil {
		slog.Error("Database update failed", "operation", "UpdateUserInjuries", "user_id", userID, "error", err)
		return fmt.Errorf("failed to update user injuries: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		slog.Error("Failed to get rows affected", "operation", "UpdateUserInjuries", "user_id", userID, "error", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

// FindByID retrieves a user by their unique identifier
func (r *PostgresUserRepository) FindByID(ctx context.Context, id string) (domain.User, error) {
	query := `
		SELECT id, email, password_hash, name, fitness_goals, injuries
		FROM users
		WHERE id = $1
	`

	var user domain.User
	var fitnessGoalsJSON, injuriesJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&fitnessGoalsJSON,
		&injuriesJSON,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, domain.ErrUserNotFound
		}
		slog.Error("Database query failed", "operation", "FindByID", "user_id", id, "error", err)
		return domain.User{}, fmt.Errorf("failed to query user: %w", err)
	}

	// Unmarshal JSON arrays
	if err := json.Unmarshal(fitnessGoalsJSON, &user.FitnessGoals); err != nil {
		slog.Error("Failed to unmarshal fitness goals", "operation", "FindByID", "user_id", user.ID, "error", err)
		return domain.User{}, fmt.Errorf("failed to unmarshal fitness goals: %w", err)
	}

	if err := json.Unmarshal(injuriesJSON, &user.Injuries); err != nil {
		slog.Error("Failed to unmarshal injuries", "operation", "FindByID", "user_id", user.ID, "error", err)
		return domain.User{}, fmt.Errorf("failed to unmarshal injuries: %w", err)
	}

	return user, nil
}
