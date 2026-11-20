-- Migration: Convert injuries column from JSONB array of strings to JSONB array of objects with timestamps
-- This enables injury lifecycle tracking with proactive follow-up

-- Step 1: Add new column for structured injuries
ALTER TABLE users ADD COLUMN IF NOT EXISTS injuries_new JSONB DEFAULT '[]'::jsonb;

-- Step 2: Migrate existing data - convert string array to object array with current timestamp
UPDATE users
SET injuries_new = (
    SELECT jsonb_agg(
        jsonb_build_object(
            'description', injury_text,
            'reported_at', NOW()
        )
    )
    FROM jsonb_array_elements_text(injuries) AS injury_text
)
WHERE injuries IS NOT NULL AND jsonb_array_length(injuries) > 0;

-- Step 3: Drop old column
ALTER TABLE users DROP COLUMN IF EXISTS injuries;

-- Step 4: Rename new column to injuries
ALTER TABLE users RENAME COLUMN injuries_new TO injuries;

-- Add comment
COMMENT ON COLUMN users.injuries IS 'Array of injury objects with description and reported_at timestamp for lifecycle tracking';
