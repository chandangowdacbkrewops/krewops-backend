DROP TABLE IF EXISTS worker_profile_skills;

ALTER TABLE worker_profiles
    ADD COLUMN IF NOT EXISTS profile_id UUID REFERENCES profiles(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS skills TEXT;

ALTER TABLE worker_profiles
    DROP COLUMN IF EXISTS user_id,
    DROP COLUMN IF EXISTS worker_type,
    DROP COLUMN IF EXISTS crew_name,
    DROP COLUMN IF EXISTS crew_size,
    DROP COLUMN IF EXISTS expected_rate,
    DROP COLUMN IF EXISTS rate_type,
    DROP COLUMN IF EXISTS availability_status,
    DROP COLUMN IF EXISTS verification_status,
    DROP COLUMN IF EXISTS bio,
    DROP COLUMN IF EXISTS profile_completed;

DROP INDEX IF EXISTS idx_worker_profiles_user_id;
