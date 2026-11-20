package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/DanielDubon/Gimpy/internal/domain"
)

// PostgresWorkoutRepository implements WorkoutRepository interface with PostgreSQL storage
type PostgresWorkoutRepository struct {
	db *sql.DB
}

// NewPostgresWorkoutRepository creates a new PostgreSQL workout repository
func NewPostgresWorkoutRepository(db *sql.DB) *PostgresWorkoutRepository {
	return &PostgresWorkoutRepository{
		db: db,
	}
}

// Save stores a workout
func (r *PostgresWorkoutRepository) Save(ctx context.Context, userID string, workout domain.Workout) error {
	if len(workout.Exercises) == 0 {
		return fmt.Errorf("cannot save workout without exercises")
	}

	// Begin transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		slog.Error("Failed to begin transaction", "operation", "SaveWorkout", "user_id", userID, "error", err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	// Insert workout
	workoutQuery := `
		INSERT INTO workouts (id, user_id, date, message)
		VALUES ($1, $2, $3, $4)
	`
	_, err = tx.ExecContext(ctx, workoutQuery,
		workout.ID,
		userID, // use of userID from JWT, not from workout struct
		workout.Date,
		workout.Message,
	)
	if err != nil {
		slog.Error("Failed to insert workout", "operation", "SaveWorkout", "workout_id", workout.ID, "user_id", userID, "error", err)
		return fmt.Errorf("failed to insert workout: %w", err)
	}

	// Insert exercises
	exerciseQuery := `
		INSERT INTO workout_exercises (workout_id, exercise_id, display_name, sets, reps, weight, exercise_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	for i, exercise := range workout.Exercises {
		_, err = tx.ExecContext(ctx, exerciseQuery,
			workout.ID,
			exercise.ExerciseID,
			exercise.DisplayName,
			exercise.Sets,
			exercise.Reps,
			exercise.Weight,
			i, // exercise_order to maintain sequence
		)
		if err != nil {
			slog.Error("Failed to insert exercise", "operation", "SaveWorkout", "workout_id", workout.ID, "exercise_index", i, "exercise_name", exercise.DisplayName, "error", err)
			return fmt.Errorf("failed to insert exercise %d: %w", i, err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		slog.Error("Failed to commit transaction", "operation", "SaveWorkout", "workout_id", workout.ID, "user_id", userID, "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetHistory retrieves all workouts for a specific user
func (r *PostgresWorkoutRepository) GetHistory(ctx context.Context, userID string) ([]domain.Workout, error) {
	// the filter is by user_id to prevent IDOR attacks
	workoutQuery := `
		SELECT id, user_id, date, message, fatigue_level, notes, created_at
		FROM workouts
		WHERE user_id = $1
		ORDER BY date DESC
	`

	rows, err := r.db.QueryContext(ctx, workoutQuery, userID)
	if err != nil {
		slog.Error("Failed to query workouts", "operation", "GetHistory", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to query workouts: %w", err)
	}
	defer rows.Close()

	var workouts []domain.Workout
	workoutMap := make(map[string]*domain.Workout)

	for rows.Next() {
		var workout domain.Workout
		var createdAt sql.NullTime
		var fatigueLevel sql.NullInt64
		var notes sql.NullString

		err := rows.Scan(
			&workout.ID,
			&workout.UserID,
			&workout.Date,
			&workout.Message,
			&fatigueLevel,
			&notes,
			&createdAt,
		)
		if err != nil {
			slog.Error("Failed to scan workout row", "operation", "GetHistory", "user_id", userID, "error", err)
			return nil, fmt.Errorf("failed to scan workout: %w", err)
		}

		if fatigueLevel.Valid {
			workout.FatigueLevel = int(fatigueLevel.Int64)
		}
		if notes.Valid {
			workout.Notes = notes.String
		}

		workout.Exercises = []domain.Exercise{}
		workouts = append(workouts, workout)
	}

	// Build map with pointers to workouts in the slice
	for i := range workouts {
		workoutMap[workouts[i].ID] = &workouts[i]
	}

	if err := rows.Err(); err != nil {
		slog.Error("Error iterating workout rows", "operation", "GetHistory", "user_id", userID, "error", err)
		return nil, fmt.Errorf("error iterating workouts: %w", err)
	}

	if len(workouts) == 0 {
		return []domain.Workout{}, nil
	}

	// Build IN clause for workout IDs
	exerciseQuery := `
		SELECT workout_id, exercise_id, display_name, sets, reps, weight
		FROM workout_exercises
		WHERE workout_id = ANY($1::text[])
		ORDER BY workout_id, exercise_order
	`

	workoutIDs := make([]string, 0, len(workouts))
	for _, w := range workouts {
		workoutIDs = append(workoutIDs, w.ID)
	}

	// Use pq.Array for PostgreSQL array parameter
	exerciseRows, err := r.db.QueryContext(ctx, exerciseQuery, "{"+strings.Join(workoutIDs, ",")+"}")
	if err != nil {
		slog.Error("Failed to query exercises", "operation", "GetHistory", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to query exercises: %w", err)
	}
	defer exerciseRows.Close()

	for exerciseRows.Next() {
		var workoutID string
		var exercise domain.Exercise

		err := exerciseRows.Scan(
			&workoutID,
			&exercise.ExerciseID,
			&exercise.DisplayName,
			&exercise.Sets,
			&exercise.Reps,
			&exercise.Weight,
		)
		if err != nil {
			slog.Error("Failed to scan exercise row", "operation", "GetHistory", "user_id", userID, "error", err)
			return nil, fmt.Errorf("failed to scan exercise: %w", err)
		}

		// Trim any whitespace from workout ID
		workoutID = strings.TrimSpace(workoutID)

		if workout, ok := workoutMap[workoutID]; ok {
			workout.Exercises = append(workout.Exercises, exercise)
		}
	}

	if err := exerciseRows.Err(); err != nil {
		slog.Error("Error iterating exercise rows", "operation", "GetHistory", "user_id", userID, "error", err)
		return nil, fmt.Errorf("error iterating exercises: %w", err)
	}

	return workouts, nil
}

// UpdateWorkoutFeedback updates the fatigue level and notes of the user's most recent workout
func (r *PostgresWorkoutRepository) UpdateWorkoutFeedback(ctx context.Context, userID string, fatigueLevel int, notes string) error {
	query := `
		UPDATE workouts
		SET fatigue_level = $1, notes = $2
		WHERE id = (
			SELECT id
			FROM workouts
			WHERE user_id = $3
			ORDER BY date DESC
			LIMIT 1
		)
	`

	result, err := r.db.ExecContext(ctx, query, fatigueLevel, notes, userID)
	if err != nil {
		slog.Error("Failed to update workout feedback", "operation", "UpdateWorkoutFeedback", "user_id", userID, "error", err)
		return fmt.Errorf("failed to update workout feedback: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		slog.Error("Failed to get rows affected", "operation", "UpdateWorkoutFeedback", "user_id", userID, "error", err)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		slog.Warn("No workout found to update feedback", "operation", "UpdateWorkoutFeedback", "user_id", userID)
		return fmt.Errorf("no workout found for user %s", userID)
	}

	return nil
}

// GetAllowedExercises retrieves the catalog of allowed exercises for a given muscle group
func (r *PostgresWorkoutRepository) GetAllowedExercises(ctx context.Context, muscleGroup string) ([]domain.CatalogExercise, error) {
	normalizedMuscleGroup := strings.ToLower(strings.TrimSpace(muscleGroup))

	var (
		rows *sql.Rows
		err  error
	)

	if normalizedMuscleGroup == "full_body" || normalizedMuscleGroup == "all" {
		query := `
			SELECT id, name_es, name_en, muscle_group
			FROM exercise_catalog
		`
		rows, err = r.db.QueryContext(ctx, query)
	} else {
		query := `
			SELECT id, name_es, name_en, muscle_group
			FROM exercise_catalog
			WHERE muscle_group ILIKE $1
		`
		rows, err = r.db.QueryContext(ctx, query, muscleGroup)
	}
	if err != nil {
		slog.Error("Failed to query exercise catalog", "operation", "GetAllowedExercises", "muscle_group", muscleGroup, "error", err)
		return nil, fmt.Errorf("failed to query exercise catalog: %w", err)
	}
	defer rows.Close()

	exercises := []domain.CatalogExercise{}
	for rows.Next() {
		var exercise domain.CatalogExercise

		err := rows.Scan(
			&exercise.ID,
			&exercise.NameES,
			&exercise.NameEN,
			&exercise.MuscleGroup,
		)
		if err != nil {
			slog.Error("Failed to scan catalog exercise row", "operation", "GetAllowedExercises", "muscle_group", muscleGroup, "error", err)
			return nil, fmt.Errorf("failed to scan catalog exercise: %w", err)
		}

		exercises = append(exercises, exercise)
	}

	if err := rows.Err(); err != nil {
		slog.Error("Error iterating catalog exercise rows", "operation", "GetAllowedExercises", "muscle_group", muscleGroup, "error", err)
		return nil, fmt.Errorf("error iterating catalog exercises: %w", err)
	}

	return exercises, nil
}
