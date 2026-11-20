-- Cleanup script for Gimpy Database
-- Removes workouts that have no associated exercises

-- First, identify workouts without exercises
-- This query shows how many orphaned workouts exist
SELECT COUNT(*) as orphaned_workouts_count
FROM workouts w
WHERE NOT EXISTS (
    SELECT 1 
    FROM workout_exercises we 
    WHERE we.workout_id = w.id
);

-- Delete all workouts that have no exercises
-- The CASCADE constraint will handle any potential dependencies
DELETE FROM workouts
WHERE id IN (
    SELECT w.id
    FROM workouts w
    WHERE NOT EXISTS (
        SELECT 1 
        FROM workout_exercises we 
        WHERE we.workout_id = w.id
    )
);

-- Verify cleanup
SELECT COUNT(*) as remaining_orphaned_workouts
FROM workouts w
WHERE NOT EXISTS (
    SELECT 1 
    FROM workout_exercises we 
    WHERE we.workout_id = w.id
);
