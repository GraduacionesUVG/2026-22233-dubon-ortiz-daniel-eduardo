-- Migration: Add fatigue_level and notes columns to workouts table
-- Enables the AI-native workout journal feature (log_workout_feedback tool)

ALTER TABLE workouts ADD COLUMN IF NOT EXISTS fatigue_level INTEGER;
ALTER TABLE workouts ADD COLUMN IF NOT EXISTS notes TEXT;

COMMENT ON COLUMN workouts.fatigue_level IS 'Perceived fatigue level (1-10) reported by the user after completing the workout';
COMMENT ON COLUMN workouts.notes IS 'Free-form feedback or notes reported by the user about how the workout went';
