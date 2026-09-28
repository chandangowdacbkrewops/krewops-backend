CREATE TABLE IF NOT EXISTS skills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    work_type_id UUID NOT NULL
        REFERENCES work_types(id)
        ON DELETE CASCADE,
    code VARCHAR(80) NOT NULL,
    name VARCHAR(150) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_skill_work_type_code
        UNIQUE (work_type_id, code)
);

CREATE INDEX IF NOT EXISTS idx_skills_work_type_id
    ON skills(work_type_id);

CREATE INDEX IF NOT EXISTS idx_skills_active
    ON skills(is_active);
