-- Introduces a category hierarchy for work_types, brings work_types up to
-- the richer shape (code/description/display_order/is_active), adds the
-- dynamic custom-fields engine tables for future work-type configuration
-- (schema only - no application code uses these yet), and replaces
-- worker_profile_skills with the richer worker_work_types table.

CREATE TABLE work_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    code VARCHAR(80) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,

    description TEXT,

    display_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_work_categories_active
    ON work_categories(is_active);

-- Backfill category for existing work_types rows (created before categories
-- existed) so category_id can become NOT NULL below.
INSERT INTO work_categories (code, name, display_order)
VALUES ('general', 'General', 0);

-- ---------------------------------------------------------------------
-- work_types: add the new columns.
-- ---------------------------------------------------------------------
ALTER TABLE work_types
    ADD COLUMN IF NOT EXISTS category_id UUID
        REFERENCES work_categories(id) ON DELETE RESTRICT,
    ADD COLUMN IF NOT EXISTS code VARCHAR(80),
    ADD COLUMN IF NOT EXISTS description TEXT,
    ADD COLUMN IF NOT EXISTS display_order INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE work_types
SET category_id = (SELECT id FROM work_categories WHERE code = 'general')
WHERE category_id IS NULL;

UPDATE work_types
SET code = lower(regexp_replace(regexp_replace(trim(name), '[^a-zA-Z0-9]+', '_', 'g'), '^_+|_+$', '', 'g'))
WHERE code IS NULL;

ALTER TABLE work_types
    ALTER COLUMN category_id SET NOT NULL,
    ALTER COLUMN code SET NOT NULL,
    ALTER COLUMN name TYPE VARCHAR(150);

ALTER TABLE work_types
    DROP CONSTRAINT IF EXISTS work_types_name_key;

ALTER TABLE work_types
    ADD CONSTRAINT uq_work_type_category_code
        UNIQUE (category_id, code);

CREATE INDEX idx_work_types_category_id
    ON work_types(category_id);

CREATE INDEX idx_work_types_active
    ON work_types(is_active);

-- ---------------------------------------------------------------------
-- Dynamic custom-fields engine (schema only for now).
-- ---------------------------------------------------------------------
CREATE TABLE work_type_fields (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    work_type_id UUID NOT NULL
        REFERENCES work_types(id)
        ON DELETE CASCADE,

    field_key VARCHAR(100) NOT NULL,
    label VARCHAR(150) NOT NULL,

    field_type VARCHAR(30) NOT NULL,

    placeholder VARCHAR(255),
    help_text TEXT,

    is_required BOOLEAN NOT NULL DEFAULT FALSE,

    unit VARCHAR(50),

    min_value NUMERIC,
    max_value NUMERIC,

    min_length INT,
    max_length INT,

    display_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_work_type_field_key
        UNIQUE (work_type_id, field_key),

    CONSTRAINT chk_work_type_field_type
        CHECK (
            field_type IN (
                'TEXT',
                'TEXTAREA',
                'NUMBER',
                'SELECT',
                'MULTI_SELECT',
                'BOOLEAN',
                'RADIO',
                'DATE'
            )
        )
);

CREATE INDEX idx_work_type_fields_work_type_id
    ON work_type_fields(work_type_id);

CREATE INDEX idx_work_type_fields_active
    ON work_type_fields(is_active);

CREATE TABLE work_type_field_options (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    field_id UUID NOT NULL
        REFERENCES work_type_fields(id)
        ON DELETE CASCADE,

    value VARCHAR(100) NOT NULL,
    label VARCHAR(150) NOT NULL,

    display_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_work_type_field_option_value
        UNIQUE (field_id, value)
);

CREATE INDEX idx_work_type_field_options_field_id
    ON work_type_field_options(field_id);

CREATE TABLE work_type_config_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    work_type_id UUID NOT NULL
        REFERENCES work_types(id)
        ON DELETE CASCADE,

    version INT NOT NULL,

    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_work_type_config_version
        UNIQUE (work_type_id, version)
);

-- ---------------------------------------------------------------------
-- Replace worker_profile_skills (added in 000007) with the richer
-- worker_work_types table (is_primary / years_experience / is_active).
-- ---------------------------------------------------------------------
CREATE TABLE worker_work_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    worker_profile_id UUID NOT NULL,

    work_type_id UUID NOT NULL
        REFERENCES work_types(id)
        ON DELETE RESTRICT,

    is_primary BOOLEAN NOT NULL DEFAULT FALSE,

    years_experience NUMERIC(4,1),

    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_worker_work_type
        UNIQUE (worker_profile_id, work_type_id)
);

INSERT INTO worker_work_types (
    worker_profile_id, work_type_id, years_experience, created_at, updated_at
)
SELECT worker_profile_id, work_type_id, experience_years, created_at, created_at
FROM worker_profile_skills
ON CONFLICT (worker_profile_id, work_type_id) DO NOTHING;

DROP TABLE IF EXISTS worker_profile_skills;

CREATE INDEX idx_worker_work_types_worker
    ON worker_work_types(worker_profile_id);

CREATE INDEX idx_worker_work_types_type
    ON worker_work_types(work_type_id);
