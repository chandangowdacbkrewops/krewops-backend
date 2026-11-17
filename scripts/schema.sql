-- Consolidated fresh schema for the KrewOps local dev database.
--
-- This replaces the per-service migration files (deleted) as the single
-- source of truth for the local Postgres schema. The database is treated
-- as disposable dev data: run `go run scripts/reset_db.go` (drops and
-- recreates the public schema) and then apply this file, e.g.:
--
--   go run scripts/init_db.go
--   go run scripts/reset_db.go
--   psql "postgres://postgres:root@localhost:5432/krewops?sslmode=disable" -f scripts/schema.sql
--
-- Architecture notes (see KrewOps_Category_Architecture_and_Dynamic_Work_Config.docx):
--   * `categories` is the single polymorphic Sector/Group/Work Type tree
--     (replaces the old work_categories / work_segments / work_types
--     tables). category_id / segment_id are denormalized convenience
--     pointers to the row's ancestors (NULL where not applicable).
--   * `skills` is global, reusable master data; `work_type_skills` maps
--     skills onto a WORK_TYPE category row (replaces the old 1:1
--     skills.work_type_id column).
--   * `common_roles` + `work_type_common_roles` are new (previously
--     entirely missing).
--   * `work_type_fields` gained a `context` column so the same framework
--     can describe WORK_POST fields, WORKER_PROFILE fields, or BOTH.
--   * `rate_bases` / `rate_units` / `work_rate_type_mappings` are kept as
--     they already exceed the doc (scoped-fallback resolution); only the
--     FK targets move to the unified `categories` table.
--   * `worker_categories` is a single generic join table (replaces the old
--     worker_work_categories / worker_work_segments / worker_work_types
--     tables), keyed against `categories` regardless of node_type.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ===========================================================================
-- Auth (shared database with auth-service)
-- ===========================================================================

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_number VARCHAR(20) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    phone_verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE otp_verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_number VARCHAR(20) NOT NULL,
    otp_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    verified_at TIMESTAMPTZ,
    attempts INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_otp_phone_number ON otp_verifications(phone_number);
CREATE INDEX idx_otp_phone_active ON otp_verifications(phone_number, is_active) WHERE is_active = TRUE;
CREATE INDEX idx_otp_phone_created ON otp_verifications(phone_number, created_at DESC);

CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);

-- ===========================================================================
-- Profiles
-- ===========================================================================

CREATE TABLE profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE,

    first_name VARCHAR(100),
    last_name VARCHAR(100),
    gender VARCHAR(20),
    user_type VARCHAR(20),

    date_of_birth DATE,
    email VARCHAR(255),
    preferred_language VARCHAR(50),
    country VARCHAR(100),
    state VARCHAR(100),
    city VARCHAR(100),
    postal_code VARCHAR(20),

    onboarding_step VARCHAR(32) NOT NULL DEFAULT 'role_selection'
        CHECK (onboarding_step IN (
            'role_selection', 'profile_details', 'worker_work_types',
            'work_owner_business', 'completed'
        )),

    profile_completed BOOLEAN NOT NULL DEFAULT FALSE,
    onboarding_completed BOOLEAN NOT NULL DEFAULT FALSE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_profiles_user_id ON profiles(user_id);
CREATE INDEX idx_profiles_city ON profiles(LOWER(city));
CREATE INDEX idx_profiles_state ON profiles(LOWER(state));

CREATE TABLE business_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE owner_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL UNIQUE REFERENCES profiles(id) ON DELETE CASCADE,

    business_name VARCHAR(200),
    business_description TEXT,
    registration_type VARCHAR(20),
    business_size VARCHAR(50),
    gst_registration_number VARCHAR(100),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE owner_business_types (
    owner_profile_id UUID NOT NULL REFERENCES owner_profiles(id) ON DELETE CASCADE,
    business_type_id UUID NOT NULL REFERENCES business_types(id) ON DELETE CASCADE,
    PRIMARY KEY (owner_profile_id, business_type_id)
);

CREATE TABLE worker_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE,

    worker_type VARCHAR(32) NOT NULL DEFAULT 'individual',
    crew_name VARCHAR(200),
    crew_size INTEGER NOT NULL DEFAULT 0,
    experience_years NUMERIC,

    expected_rate NUMERIC,
    expected_rate_min NUMERIC,
    expected_rate_max NUMERIC,
    rate_basis VARCHAR(80),
    rate_unit VARCHAR(80),

    availability_status VARCHAR(32) NOT NULL DEFAULT 'available',
    verification_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    bio TEXT,
    profile_completed BOOLEAN NOT NULL DEFAULT FALSE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_worker_profiles_availability_status ON worker_profiles(availability_status);
CREATE INDEX idx_worker_profiles_worker_type ON worker_profiles(worker_type);

-- ===========================================================================
-- Unified category tree (Sector / Group / Work Type)
-- ===========================================================================

CREATE TYPE category_node_type AS ENUM ('SECTOR', 'GROUP', 'WORK_TYPE');

CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    parent_id UUID REFERENCES categories(id) ON DELETE RESTRICT,
    node_type category_node_type NOT NULL,

    -- Denormalized ancestor pointers for convenient filtering/joins.
    -- category_id = the row's own SECTOR ancestor (NULL for SECTOR rows).
    -- segment_id  = the row's own GROUP ancestor (set only for WORK_TYPE rows).
    category_id UUID REFERENCES categories(id) ON DELETE RESTRICT,
    segment_id UUID REFERENCES categories(id) ON DELETE RESTRICT,

    code VARCHAR(80) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,
    description TEXT,

    display_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT categories_no_self_parent CHECK (parent_id IS NULL OR parent_id <> id),
    CONSTRAINT categories_sector_has_no_parent CHECK (
        node_type <> 'SECTOR' OR (parent_id IS NULL AND category_id IS NULL AND segment_id IS NULL)
    ),
    CONSTRAINT categories_group_has_category CHECK (
        node_type <> 'GROUP' OR (parent_id IS NOT NULL AND category_id IS NOT NULL AND segment_id IS NULL)
    ),
    CONSTRAINT categories_work_type_has_segment CHECK (
        node_type <> 'WORK_TYPE' OR (parent_id IS NOT NULL AND category_id IS NOT NULL AND segment_id IS NOT NULL)
    )
);

CREATE INDEX idx_categories_parent_id ON categories(parent_id);
CREATE INDEX idx_categories_node_type ON categories(node_type);
CREATE INDEX idx_categories_parent_active ON categories(parent_id, is_active);
CREATE INDEX idx_categories_category_id ON categories(category_id);
CREATE INDEX idx_categories_segment_id ON categories(segment_id);

-- ===========================================================================
-- Skills (global reusable master data) + common roles
-- ===========================================================================

CREATE TABLE skills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(80) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE work_type_skills (
    work_type_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    skill_id UUID NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    is_required BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INT NOT NULL DEFAULT 0,
    PRIMARY KEY (work_type_id, skill_id)
);

CREATE INDEX idx_work_type_skills_skill_id ON work_type_skills(skill_id);

CREATE TABLE common_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(80) NOT NULL UNIQUE,
    name VARCHAR(120) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE work_type_common_roles (
    work_type_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    common_role_id UUID NOT NULL REFERENCES common_roles(id) ON DELETE CASCADE,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INT NOT NULL DEFAULT 0,
    PRIMARY KEY (work_type_id, common_role_id)
);

CREATE INDEX idx_work_type_common_roles_role_id ON work_type_common_roles(common_role_id);

-- ===========================================================================
-- Rate types (kept as-is structurally; FKs repointed to `categories`)
-- ===========================================================================

CREATE TABLE rate_bases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(80) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE rate_units (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(80) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,
    unit_kind VARCHAR(32) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE work_rate_type_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    work_category_id UUID REFERENCES categories(id) ON DELETE CASCADE,
    work_segment_id UUID REFERENCES categories(id) ON DELETE CASCADE,
    work_type_id UUID REFERENCES categories(id) ON DELETE CASCADE,
    rate_basis_id UUID NOT NULL REFERENCES rate_bases(id) ON DELETE CASCADE,
    rate_unit_id UUID REFERENCES rate_units(id) ON DELETE CASCADE,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_work_rate_type_mapping_scope CHECK (
        (
            (work_type_id IS NOT NULL)::int +
            (work_segment_id IS NOT NULL)::int +
            (work_category_id IS NOT NULL)::int
        ) IN (0, 1)
    )
);

CREATE UNIQUE INDEX uq_work_rate_type_mapping
    ON work_rate_type_mappings (
        COALESCE(work_category_id, '00000000-0000-0000-0000-000000000000'),
        COALESCE(work_segment_id, '00000000-0000-0000-0000-000000000000'),
        COALESCE(work_type_id, '00000000-0000-0000-0000-000000000000'),
        rate_basis_id,
        COALESCE(rate_unit_id, '00000000-0000-0000-0000-000000000000')
    );

CREATE UNIQUE INDEX uq_work_rate_type_mapping_default_type
    ON work_rate_type_mappings (work_type_id) WHERE is_default AND work_type_id IS NOT NULL;
CREATE UNIQUE INDEX uq_work_rate_type_mapping_default_segment
    ON work_rate_type_mappings (work_segment_id) WHERE is_default AND work_segment_id IS NOT NULL;
CREATE UNIQUE INDEX uq_work_rate_type_mapping_default_category
    ON work_rate_type_mappings (work_category_id) WHERE is_default AND work_category_id IS NOT NULL;
CREATE UNIQUE INDEX uq_work_rate_type_mapping_default_global
    ON work_rate_type_mappings ((TRUE))
    WHERE is_default AND work_type_id IS NULL AND work_segment_id IS NULL AND work_category_id IS NULL;

CREATE INDEX idx_work_rate_type_mappings_work_type ON work_rate_type_mappings(work_type_id);
CREATE INDEX idx_work_rate_type_mappings_segment ON work_rate_type_mappings(work_segment_id);
CREATE INDEX idx_work_rate_type_mappings_category ON work_rate_type_mappings(work_category_id);

-- ===========================================================================
-- Dynamic attribute framework (work_type_fields gains `context`)
-- ===========================================================================

CREATE TABLE work_type_fields (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    work_type_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,

    field_key VARCHAR(100) NOT NULL,
    label VARCHAR(150) NOT NULL,
    field_type VARCHAR(30) NOT NULL,

    -- WORKER_PROFILE: captured once on the worker's work-type selection.
    -- WORK_POST: captured per job posting (default, matches legacy behavior).
    -- BOTH: relevant in either context.
    context VARCHAR(30) NOT NULL DEFAULT 'WORK_POST',

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

    CONSTRAINT uq_work_type_field_key UNIQUE (work_type_id, field_key),
    CONSTRAINT chk_work_type_field_type CHECK (
        field_type IN ('TEXT', 'TEXTAREA', 'NUMBER', 'SELECT', 'MULTI_SELECT', 'BOOLEAN', 'RADIO', 'DATE')
    ),
    CONSTRAINT chk_work_type_field_context CHECK (
        context IN ('WORKER_PROFILE', 'WORK_POST', 'BOTH')
    )
);

CREATE INDEX idx_work_type_fields_work_type_id ON work_type_fields(work_type_id);
CREATE INDEX idx_work_type_fields_active ON work_type_fields(is_active);
CREATE INDEX idx_work_type_fields_context ON work_type_fields(context);

CREATE TABLE work_type_field_options (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    field_id UUID NOT NULL REFERENCES work_type_fields(id) ON DELETE CASCADE,
    value VARCHAR(100) NOT NULL,
    label VARCHAR(150) NOT NULL,
    display_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_work_type_field_option_value UNIQUE (field_id, value)
);

CREATE INDEX idx_work_type_field_options_field_id ON work_type_field_options(field_id);

CREATE TABLE work_type_config_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    work_type_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    version INT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_work_type_config_version UNIQUE (work_type_id, version)
);

-- ===========================================================================
-- Worker selections: a single generic join table against the unified
-- `categories` tree (replaces the old worker_work_categories /
-- worker_work_segments / worker_work_types tables). One row per
-- SECTOR/GROUP/WORK_TYPE node the worker has selected; the node's
-- `category_id` / `segment_id` ancestors (on `categories`) make it cheap
-- to query "all WORK_TYPE rows under GROUP X the worker has selected"
-- without needing separate tables per level.
-- ===========================================================================

CREATE TABLE worker_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    worker_profile_id UUID NOT NULL REFERENCES worker_profiles(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,

    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    experience_years NUMERIC(4, 1),

    -- Values for this node's WORKER_PROFILE / BOTH context dynamic
    -- attributes (see work_type_fields.context), e.g. years of experience
    -- in a specific picking method. Only meaningful for WORK_TYPE rows.
    -- Validated at the service layer against work_type_fields; never
    -- affects the master configuration.
    dynamic_attributes JSONB NOT NULL DEFAULT '{}'::jsonb,

    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_worker_category UNIQUE (worker_profile_id, category_id)
);

CREATE INDEX idx_worker_categories_worker ON worker_categories(worker_profile_id);
CREATE INDEX idx_worker_categories_category ON worker_categories(category_id);
CREATE INDEX idx_worker_categories_category_active
    ON worker_categories(category_id, worker_profile_id) WHERE is_active = TRUE;

CREATE TABLE worker_skills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    worker_profile_id UUID NOT NULL REFERENCES worker_profiles(id) ON DELETE CASCADE,
    skill_id UUID NOT NULL REFERENCES skills(id) ON DELETE RESTRICT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_worker_skill UNIQUE (worker_profile_id, skill_id)
);

CREATE INDEX idx_worker_skills_worker ON worker_skills(worker_profile_id);
CREATE INDEX idx_worker_skills_skill ON worker_skills(skill_id);

-- ===========================================================================
-- Seed data: Sector -> Group -> Work Type catalog
-- ===========================================================================

INSERT INTO categories (code, name, description, node_type, display_order)
VALUES
    ('agriculture', 'Agriculture', 'Farming, plantation and crop-related work', 'SECTOR', 1),
    ('construction', 'Construction', 'Building, infrastructure and civil work', 'SECTOR', 2),
    ('housekeeping', 'Housekeeping', 'Cleaning, maintenance and care services', 'SECTOR', 3);

INSERT INTO categories (parent_id, category_id, code, name, node_type, display_order)
SELECT sector.id, sector.id, seed.code, seed.name, 'GROUP', seed.display_order
FROM categories sector
JOIN (
    VALUES
        ('agriculture', 'coffee', 'Coffee', 1),
        ('agriculture', 'tea', 'Tea', 2),
        ('agriculture', 'paddy', 'Paddy', 3),
        ('construction', 'masonry', 'Masonry', 1),
        ('construction', 'electrical', 'Electrical', 2),
        ('construction', 'painting', 'Painting', 3),
        ('housekeeping', 'home_cleaning', 'Home cleaning', 1),
        ('housekeeping', 'facility_maintenance', 'Facility Maintenance', 2),
        ('housekeeping', 'laundry_care', 'Laundry & Care', 3)
) AS seed(sector_code, code, name, display_order)
    ON seed.sector_code = sector.code
WHERE sector.node_type = 'SECTOR';

INSERT INTO categories (parent_id, category_id, segment_id, code, name, node_type, display_order)
SELECT group_row.id, group_row.category_id, group_row.id, seed.code, seed.name, 'WORK_TYPE', seed.display_order
FROM categories group_row
JOIN (
    VALUES
        ('coffee', 'harvesting', 'Harvesting', 1),
        ('coffee', 'pruning', 'Pruning', 2),
        ('tea', 'plucking', 'Plucking', 1),
        ('paddy', 'transplanting', 'Transplanting', 1),
        ('masonry', 'brick_work', 'Brick Work', 1),
        ('electrical', 'wiring', 'Wiring', 1),
        ('painting', 'wall_painting', 'Wall painting', 1),
        ('home_cleaning', 'deep_cleaning', 'Deep cleaning', 1),
        ('facility_maintenance', 'office_cleaning', 'Office cleaning', 1),
        ('laundry_care', 'ironing_washing', 'Ironing/Washing', 1)
) AS seed(group_code, code, name, display_order)
    ON seed.group_code = group_row.code
WHERE group_row.node_type = 'GROUP';

-- Skills: global master data, deduped by code, then mapped per work type.
INSERT INTO skills (code, name)
VALUES
    ('selective_picking', 'Selective picking'),
    ('cherry_collection', 'Cherry collection'),
    ('manual_pruning', 'Manual pruning'),
    ('branch_cutting', 'Branch cutting'),
    ('hand_plucking', 'Hand plucking'),
    ('fine_plucking', 'Fine plucking'),
    ('seedling_handling', 'Seedling handling'),
    ('row_transplanting', 'Row transplanting'),
    ('brick_laying', 'Brick laying'),
    ('mortar_mixing', 'Mortar mixing'),
    ('conduit_wiring', 'Conduit wiring'),
    ('cable_pulling', 'Cable pulling'),
    ('roller_painting', 'Roller painting'),
    ('surface_preparation', 'Surface preparation'),
    ('kitchen_cleaning', 'Kitchen cleaning'),
    ('bathroom_sanitization', 'Bathroom sanitization'),
    ('floor_cleaning', 'Floor cleaning'),
    ('waste_handling', 'Waste handling'),
    ('fabric_handling', 'Fabric handling'),
    ('stain_treatment', 'Stain treatment');

INSERT INTO work_type_skills (work_type_id, skill_id, is_required, sort_order)
SELECT wt.id, s.id, seed.is_required, seed.sort_order
FROM categories wt
JOIN (
    VALUES
        ('harvesting', 'selective_picking', TRUE, 1),
        ('harvesting', 'cherry_collection', FALSE, 2),
        ('pruning', 'manual_pruning', TRUE, 1),
        ('pruning', 'branch_cutting', FALSE, 2),
        ('plucking', 'hand_plucking', TRUE, 1),
        ('plucking', 'fine_plucking', FALSE, 2),
        ('transplanting', 'seedling_handling', TRUE, 1),
        ('transplanting', 'row_transplanting', FALSE, 2),
        ('brick_work', 'brick_laying', TRUE, 1),
        ('brick_work', 'mortar_mixing', FALSE, 2),
        ('wiring', 'conduit_wiring', TRUE, 1),
        ('wiring', 'cable_pulling', FALSE, 2),
        ('wall_painting', 'roller_painting', TRUE, 1),
        ('wall_painting', 'surface_preparation', FALSE, 2),
        ('deep_cleaning', 'kitchen_cleaning', TRUE, 1),
        ('deep_cleaning', 'bathroom_sanitization', FALSE, 2),
        ('office_cleaning', 'floor_cleaning', TRUE, 1),
        ('office_cleaning', 'waste_handling', FALSE, 2),
        ('ironing_washing', 'fabric_handling', TRUE, 1),
        ('ironing_washing', 'stain_treatment', FALSE, 2)
) AS seed(work_type_code, skill_code, is_required, sort_order)
    ON seed.work_type_code = wt.code
JOIN skills s ON s.code = seed.skill_code
WHERE wt.node_type = 'WORK_TYPE';

-- Common roles (previously missing entirely).
INSERT INTO common_roles (code, name)
VALUES
    ('picker', 'Picker'),
    ('farm_worker', 'Farm Worker'),
    ('mason', 'Mason'),
    ('helper', 'Helper'),
    ('electrician', 'Electrician'),
    ('painter', 'Painter'),
    ('housekeeper', 'Housekeeper');

INSERT INTO work_type_common_roles (work_type_id, common_role_id, is_primary)
SELECT wt.id, cr.id, seed.is_primary
FROM categories wt
JOIN (
    VALUES
        ('harvesting', 'picker', TRUE),
        ('harvesting', 'farm_worker', FALSE),
        ('pruning', 'farm_worker', TRUE),
        ('plucking', 'picker', TRUE),
        ('transplanting', 'farm_worker', TRUE),
        ('brick_work', 'mason', TRUE),
        ('brick_work', 'helper', FALSE),
        ('wiring', 'electrician', TRUE),
        ('wall_painting', 'painter', TRUE),
        ('wall_painting', 'helper', FALSE),
        ('deep_cleaning', 'housekeeper', TRUE),
        ('office_cleaning', 'housekeeper', TRUE),
        ('ironing_washing', 'housekeeper', TRUE)
) AS seed(work_type_code, role_code, is_primary)
    ON seed.work_type_code = wt.code
JOIN common_roles cr ON cr.code = seed.role_code
WHERE wt.node_type = 'WORK_TYPE';

-- Dynamic attribute fields (WORK_POST context by default, matching legacy
-- behavior) and their options.
INSERT INTO work_type_fields (work_type_id, field_key, label, field_type, context, is_required, unit, display_order)
SELECT wt.id, seed.field_key, seed.label, seed.field_type, seed.context, seed.is_required, seed.unit, seed.display_order
FROM categories wt
JOIN (
    VALUES
        ('harvesting', 'harvesting_method', 'Harvesting method', 'SELECT', 'WORK_POST', TRUE, NULL, 1),
        ('harvesting', 'estimated_quantity', 'Estimated quantity', 'NUMBER', 'WORK_POST', FALSE, 'kg', 2),
        ('harvesting', 'area_acres', 'Area', 'NUMBER', 'WORK_POST', TRUE, 'acres', 3),
        ('harvesting', 'years_experience', 'Years of harvesting experience', 'NUMBER', 'WORKER_PROFILE', FALSE, 'years', 4),
        ('pruning', 'plant_age', 'Plant age', 'NUMBER', 'WORK_POST', FALSE, 'years', 1),
        ('pruning', 'area', 'Area', 'NUMBER', 'WORK_POST', TRUE, 'acres', 2),
        ('pruning', 'duration', 'Duration', 'NUMBER', 'WORK_POST', FALSE, 'days', 3),
        ('pruning', 'tools_required', 'Tools required', 'BOOLEAN', 'WORK_POST', FALSE, NULL, 4),
        ('plucking', 'plucking_standard', 'Plucking standard', 'SELECT', 'WORK_POST', TRUE, NULL, 1),
        ('plucking', 'daily_target', 'Daily target', 'NUMBER', 'WORK_POST', FALSE, 'kg', 2),
        ('transplanting', 'area', 'Area', 'NUMBER', 'WORK_POST', TRUE, 'acres', 1),
        ('transplanting', 'planting_method', 'Planting method', 'SELECT', 'WORK_POST', TRUE, NULL, 2),
        ('transplanting', 'seedling_age', 'Seedling age', 'NUMBER', 'WORK_POST', FALSE, 'days', 3),
        ('brick_work', 'wall_length', 'Wall length', 'NUMBER', 'WORK_POST', TRUE, 'ft', 1),
        ('brick_work', 'height', 'Height', 'NUMBER', 'WORK_POST', TRUE, 'ft', 2),
        ('brick_work', 'brick_type', 'Brick type', 'SELECT', 'WORK_POST', FALSE, NULL, 3),
        ('wiring', 'building_type', 'Building type', 'SELECT', 'WORK_POST', TRUE, NULL, 1),
        ('wiring', 'points_count', 'Points count', 'NUMBER', 'WORK_POST', TRUE, NULL, 2),
        ('wiring', 'wire_type', 'Wire type', 'SELECT', 'WORK_POST', FALSE, NULL, 3),
        ('wiring', 'tools_required', 'Tools required', 'BOOLEAN', 'WORK_POST', FALSE, NULL, 4),
        ('wall_painting', 'area', 'Area', 'NUMBER', 'WORK_POST', TRUE, 'sqft', 1),
        ('wall_painting', 'interior_exterior', 'Interior / exterior', 'SELECT', 'WORK_POST', TRUE, NULL, 2),
        ('wall_painting', 'coat_count', 'Coat count', 'NUMBER', 'WORK_POST', FALSE, NULL, 3),
        ('wall_painting', 'paint_type', 'Paint type', 'SELECT', 'WORK_POST', FALSE, NULL, 4),
        ('deep_cleaning', 'property_type', 'Property type', 'SELECT', 'WORK_POST', TRUE, NULL, 1),
        ('deep_cleaning', 'room_count', 'Room count', 'NUMBER', 'WORK_POST', TRUE, NULL, 2),
        ('deep_cleaning', 'cleaning_supplies_provided', 'Cleaning supplies provided', 'BOOLEAN', 'WORK_POST', FALSE, NULL, 3),
        ('office_cleaning', 'office_size', 'Office size', 'NUMBER', 'WORK_POST', TRUE, 'sqft', 1),
        ('office_cleaning', 'shift_timing', 'Shift timing', 'SELECT', 'WORK_POST', FALSE, NULL, 2),
        ('office_cleaning', 'frequency', 'Frequency', 'SELECT', 'WORK_POST', TRUE, NULL, 3),
        ('ironing_washing', 'item_count', 'Item count', 'NUMBER', 'WORK_POST', TRUE, NULL, 1),
        ('ironing_washing', 'service_type', 'Service type', 'SELECT', 'WORK_POST', TRUE, NULL, 2),
        ('ironing_washing', 'pickup_drop_needed', 'Pickup / drop needed', 'BOOLEAN', 'WORK_POST', FALSE, NULL, 3)
) AS seed(work_type_code, field_key, label, field_type, context, is_required, unit, display_order)
    ON seed.work_type_code = wt.code
WHERE wt.node_type = 'WORK_TYPE';

INSERT INTO work_type_field_options (field_id, value, label, display_order)
SELECT f.id, seed.value, seed.label, seed.display_order
FROM work_type_fields f
JOIN categories wt ON wt.id = f.work_type_id
JOIN (
    VALUES
        ('harvesting', 'harvesting_method', 'selective', 'Selective', 1),
        ('harvesting', 'harvesting_method', 'mechanical', 'Mechanical', 2),
        ('harvesting', 'harvesting_method', 'mixed', 'Mixed', 3),
        ('plucking', 'plucking_standard', 'fine', 'Fine', 1),
        ('plucking', 'plucking_standard', 'standard', 'Standard', 2),
        ('plucking', 'plucking_standard', 'coarse', 'Coarse', 3),
        ('transplanting', 'planting_method', 'manual', 'Manual', 1),
        ('transplanting', 'planting_method', 'machine', 'Machine', 2),
        ('brick_work', 'brick_type', 'red_clay', 'Red clay', 1),
        ('brick_work', 'brick_type', 'fly_ash', 'Fly ash', 2),
        ('brick_work', 'brick_type', 'concrete', 'Concrete', 3),
        ('wiring', 'building_type', 'residential', 'Residential', 1),
        ('wiring', 'building_type', 'commercial', 'Commercial', 2),
        ('wiring', 'building_type', 'industrial', 'Industrial', 3),
        ('wiring', 'wire_type', 'copper', 'Copper', 1),
        ('wiring', 'wire_type', 'aluminum', 'Aluminum', 2),
        ('wall_painting', 'interior_exterior', 'interior', 'Interior', 1),
        ('wall_painting', 'interior_exterior', 'exterior', 'Exterior', 2),
        ('wall_painting', 'interior_exterior', 'both', 'Both', 3),
        ('wall_painting', 'paint_type', 'emulsion', 'Emulsion', 1),
        ('wall_painting', 'paint_type', 'enamel', 'Enamel', 2),
        ('wall_painting', 'paint_type', 'distemper', 'Distemper', 3),
        ('deep_cleaning', 'property_type', 'apartment', 'Apartment', 1),
        ('deep_cleaning', 'property_type', 'house', 'House', 2),
        ('deep_cleaning', 'property_type', 'villa', 'Villa', 3),
        ('office_cleaning', 'shift_timing', 'morning', 'Morning', 1),
        ('office_cleaning', 'shift_timing', 'evening', 'Evening', 2),
        ('office_cleaning', 'shift_timing', 'night', 'Night', 3),
        ('office_cleaning', 'frequency', 'daily', 'Daily', 1),
        ('office_cleaning', 'frequency', 'weekly', 'Weekly', 2),
        ('office_cleaning', 'frequency', 'monthly', 'Monthly', 3),
        ('ironing_washing', 'service_type', 'wash_only', 'Wash only', 1),
        ('ironing_washing', 'service_type', 'iron_only', 'Iron only', 2),
        ('ironing_washing', 'service_type', 'wash_and_iron', 'Wash and iron', 3)
) AS seed(work_type_code, field_key, value, label, display_order)
    ON seed.work_type_code = wt.code AND seed.field_key = f.field_key;

-- Rate catalog + scoped mappings.
INSERT INTO rate_bases (code, name, description)
VALUES
    ('per_time', 'Per time', 'Priced by time worked'),
    ('per_output', 'Per output', 'Priced by quantity produced'),
    ('per_area', 'Per area', 'Priced by area covered'),
    ('per_entity', 'Per entity', 'Priced per tree, pipe, or similar unit'),
    ('fixed', 'Fixed', 'Fixed contract for the full job'),
    ('negotiable', 'Negotiable', 'Rate to be discussed');

INSERT INTO rate_units (code, name, unit_kind)
VALUES
    ('hour', 'Hour', 'time'),
    ('day', 'Day', 'time'),
    ('kg', 'Kilogram', 'output'),
    ('bag', 'Bag', 'output'),
    ('acre', 'Acre', 'area'),
    ('tree', 'Tree', 'entity'),
    ('sq_ft', 'Square foot', 'area'),
    ('job', 'Job', 'fixed');

-- Global defaults (no scope).
INSERT INTO work_rate_type_mappings (rate_basis_id, rate_unit_id, is_default)
SELECT rb.id, ru.id, seed.is_default
FROM (
    VALUES
        ('per_time', 'day', TRUE),
        ('fixed', 'job', FALSE),
        ('negotiable', NULL, FALSE)
) AS seed(basis_code, unit_code, is_default)
JOIN rate_bases rb ON rb.code = seed.basis_code
LEFT JOIN rate_units ru ON ru.code = seed.unit_code;

-- Work-type-scoped overrides for the Agriculture / Coffee example and a few
-- construction work types.
INSERT INTO work_rate_type_mappings (work_type_id, rate_basis_id, rate_unit_id, is_default)
SELECT wt.id, rb.id, ru.id, seed.is_default
FROM categories wt
JOIN (
    VALUES
        ('harvesting', 'per_time', 'day', TRUE),
        ('harvesting', 'per_output', 'kg', FALSE),
        ('harvesting', 'negotiable', NULL, FALSE),
        ('pruning', 'per_time', 'day', TRUE),
        ('pruning', 'per_entity', 'tree', FALSE),
        ('pruning', 'per_area', 'acre', FALSE),
        ('pruning', 'fixed', 'job', FALSE),
        ('pruning', 'negotiable', NULL, FALSE),
        ('brick_work', 'per_time', 'day', TRUE),
        ('brick_work', 'per_area', 'sq_ft', FALSE),
        ('brick_work', 'fixed', 'job', FALSE),
        ('brick_work', 'negotiable', NULL, FALSE),
        ('wall_painting', 'per_time', 'day', TRUE),
        ('wall_painting', 'per_area', 'sq_ft', FALSE),
        ('wall_painting', 'fixed', 'job', FALSE)
) AS seed(type_code, basis_code, unit_code, is_default)
    ON seed.type_code = wt.code
JOIN rate_bases rb ON rb.code = seed.basis_code
LEFT JOIN rate_units ru ON ru.code = seed.unit_code
WHERE wt.node_type = 'WORK_TYPE' AND wt.is_active = TRUE;
