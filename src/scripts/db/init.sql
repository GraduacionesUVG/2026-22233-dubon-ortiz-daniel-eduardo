-- Gimpy Database Schema
-- PostgreSQL/Supabase initialization script

-- users table
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(255) PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    fitness_goals JSONB DEFAULT '[]'::jsonb,
    injuries JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- index on email for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Create index on created_at for analytics
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);

-- workouts table
CREATE TABLE IF NOT EXISTS workouts (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date TIMESTAMP NOT NULL,
    message TEXT,
    fatigue_level INTEGER,
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- index on user_id for faster user-specific queries
CREATE INDEX IF NOT EXISTS idx_workouts_user_id ON workouts(user_id);

-- index on date for chronological queries
CREATE INDEX IF NOT EXISTS idx_workouts_date ON workouts(date);

-- workout_exercises table
CREATE TABLE IF NOT EXISTS workout_exercises (
    id SERIAL PRIMARY KEY,
    workout_id VARCHAR(255) NOT NULL REFERENCES workouts(id) ON DELETE CASCADE,
    exercise_id VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    sets INTEGER NOT NULL,
    reps INTEGER NOT NULL,
    weight DECIMAL(10, 2) NOT NULL,
    exercise_order INTEGER NOT NULL
);

-- index on workout_id for faster exercise lookups
CREATE INDEX IF NOT EXISTS idx_workout_exercises_workout_id ON workout_exercises(workout_id);

-- exercise_catalog table (allowed exercises the LLM can pick from; seeded by seed_exercises.sql)
CREATE TABLE IF NOT EXISTS exercise_catalog (
    id VARCHAR(100) PRIMARY KEY,
    name_es VARCHAR(100) NOT NULL,
    name_en VARCHAR(100) NOT NULL,
    muscle_group VARCHAR(50) NOT NULL
);
