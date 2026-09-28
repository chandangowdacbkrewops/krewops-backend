CREATE TABLE IF NOT EXISTS worker_skills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    worker_profile_id UUID NOT NULL
        REFERENCES worker_profiles(id)
        ON DELETE CASCADE,
    skill_id UUID NOT NULL
        REFERENCES skills(id)
        ON DELETE RESTRICT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_worker_skill
        UNIQUE (worker_profile_id, skill_id)
);

CREATE INDEX IF NOT EXISTS idx_worker_skills_worker
    ON worker_skills(worker_profile_id);

CREATE INDEX IF NOT EXISTS idx_worker_skills_skill
    ON worker_skills(skill_id);
