ALTER TABLE worker_profiles
    ADD COLUMN IF NOT EXISTS user_id UUID,
    ADD COLUMN IF NOT EXISTS worker_type VARCHAR(32) NOT NULL DEFAULT 'individual',
    ADD COLUMN IF NOT EXISTS crew_name VARCHAR(200),
    ADD COLUMN IF NOT EXISTS crew_size INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS expected_rate NUMERIC,
    ADD COLUMN IF NOT EXISTS rate_type VARCHAR(32),
    ADD COLUMN IF NOT EXISTS availability_status VARCHAR(32) NOT NULL DEFAULT 'available',
    ADD COLUMN IF NOT EXISTS verification_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    ADD COLUMN IF NOT EXISTS bio TEXT,
    ADD COLUMN IF NOT EXISTS profile_completed BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE worker_profiles wp
SET user_id = p.user_id,
    bio = COALESCE(wp.bio, wp.skills),
    profile_completed = TRUE
FROM profiles p
WHERE wp.profile_id = p.id
  AND wp.user_id IS NULL;

ALTER TABLE worker_profiles
    ALTER COLUMN user_id SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_worker_profiles_user_id
    ON worker_profiles(user_id);

CREATE TABLE IF NOT EXISTS worker_profile_skills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    worker_profile_id UUID NOT NULL REFERENCES worker_profiles(id) ON DELETE CASCADE,
    work_type_id UUID NOT NULL REFERENCES work_types(id) ON DELETE CASCADE,
    experience_years NUMERIC,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (worker_profile_id, work_type_id)
);

INSERT INTO worker_profile_skills (worker_profile_id, work_type_id, experience_years)
SELECT wwt.worker_profile_id, wwt.work_type_id, wp.experience_years
FROM worker_work_types wwt
JOIN worker_profiles wp ON wp.id = wwt.worker_profile_id
ON CONFLICT (worker_profile_id, work_type_id) DO NOTHING;

ALTER TABLE worker_profiles
    DROP CONSTRAINT IF EXISTS worker_profiles_profile_id_key;

ALTER TABLE worker_profiles
    DROP COLUMN IF EXISTS profile_id,
    DROP COLUMN IF EXISTS skills;

DROP TABLE IF EXISTS worker_work_types;
