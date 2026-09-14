CREATE TABLE work_postings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL REFERENCES profiles(user_id) ON DELETE CASCADE,

    title VARCHAR(80),
    description VARCHAR(500),
    work_type_id UUID REFERENCES work_types(id),

    address VARCHAR(255),
    city VARCHAR(100),
    state VARCHAR(100),
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,

    workers_needed INTEGER,
    experience_level VARCHAR(30),
    skills TEXT[] NOT NULL DEFAULT '{}',
    tools_provided BOOLEAN,
    materials_provided BOOLEAN,

    start_date DATE,
    duration_value INTEGER,
    duration_unit VARCHAR(20),
    shift_timing VARCHAR(100),

    payment_type VARCHAR(20),
    budget_rate NUMERIC(12, 2),
    payment_notes VARCHAR(300),
    accommodation_provided BOOLEAN,
    meals_provided BOOLEAN,

    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    published_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_work_postings_user_id
ON work_postings(user_id);

CREATE INDEX idx_work_postings_status
ON work_postings(status);
