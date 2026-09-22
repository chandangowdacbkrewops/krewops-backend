-- Supports SearchWorkers queries issued by search-service against the
-- shared profiles / worker_profiles / worker_profile_skills tables.

CREATE INDEX IF NOT EXISTS idx_profiles_city
ON profiles(LOWER(city));

CREATE INDEX IF NOT EXISTS idx_profiles_state
ON profiles(LOWER(state));

CREATE INDEX IF NOT EXISTS idx_worker_profiles_availability_status
ON worker_profiles(availability_status);

CREATE INDEX IF NOT EXISTS idx_worker_profiles_worker_type
ON worker_profiles(worker_type);

CREATE INDEX IF NOT EXISTS idx_worker_profile_skills_work_type_id
ON worker_profile_skills(work_type_id);
