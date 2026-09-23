-- Revert worker_work_types back to worker_profile_skills.
CREATE TABLE IF NOT EXISTS worker_profile_skills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    worker_profile_id UUID NOT NULL REFERENCES worker_profiles(id) ON DELETE CASCADE,
    work_type_id UUID NOT NULL REFERENCES work_types(id) ON DELETE CASCADE,
    experience_years NUMERIC,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (worker_profile_id, work_type_id)
);

INSERT INTO worker_profile_skills (worker_profile_id, work_type_id, experience_years, created_at)
SELECT worker_profile_id, work_type_id, years_experience, created_at
FROM worker_work_types
ON CONFLICT (worker_profile_id, work_type_id) DO NOTHING;

DROP TABLE IF EXISTS worker_work_types;

DROP TABLE IF EXISTS work_type_config_versions;
DROP TABLE IF EXISTS work_type_field_options;
DROP TABLE IF EXISTS work_type_fields;

ALTER TABLE work_types
    DROP CONSTRAINT IF EXISTS uq_work_type_category_code;

DROP INDEX IF EXISTS idx_work_types_category_id;
DROP INDEX IF EXISTS idx_work_types_active;

ALTER TABLE work_types
    ALTER COLUMN name TYPE VARCHAR(100),
    ALTER COLUMN category_id DROP NOT NULL,
    ALTER COLUMN code DROP NOT NULL;

ALTER TABLE work_types
    DROP COLUMN IF EXISTS category_id,
    DROP COLUMN IF EXISTS code,
    DROP COLUMN IF EXISTS description,
    DROP COLUMN IF EXISTS display_order,
    DROP COLUMN IF EXISTS is_active,
    DROP COLUMN IF EXISTS updated_at;

ALTER TABLE work_types
    ADD CONSTRAINT work_types_name_key UNIQUE (name);

DROP TABLE IF EXISTS work_categories;
