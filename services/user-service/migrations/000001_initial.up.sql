CREATE TABLE profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL UNIQUE,

    first_name VARCHAR(100),
    last_name VARCHAR(100),
    user_type VARCHAR(20),

    profile_completed BOOLEAN NOT NULL DEFAULT FALSE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE worker_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    profile_id UUID NOT NULL UNIQUE REFERENCES profiles(id) ON DELETE CASCADE,

    skills TEXT,
    experience_years INTEGER,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE owner_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    profile_id UUID NOT NULL UNIQUE REFERENCES profiles(id) ON DELETE CASCADE,

    business_name VARCHAR(200),
    business_description TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE work_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    name VARCHAR(100) NOT NULL UNIQUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE worker_work_types (
    worker_profile_id UUID NOT NULL REFERENCES worker_profiles(id) ON DELETE CASCADE,
    work_type_id UUID NOT NULL REFERENCES work_types(id) ON DELETE CASCADE,

    PRIMARY KEY (worker_profile_id, work_type_id)
);

CREATE TABLE business_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    name VARCHAR(100) NOT NULL UNIQUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE owner_business_types (
    owner_profile_id UUID NOT NULL REFERENCES owner_profiles(id) ON DELETE CASCADE,
    business_type_id UUID NOT NULL REFERENCES business_types(id) ON DELETE CASCADE,

    PRIMARY KEY (owner_profile_id, business_type_id)
);

CREATE INDEX idx_profiles_user_id
ON profiles(user_id);
