ALTER TABLE profiles
    ADD COLUMN IF NOT EXISTS date_of_birth DATE,
    ADD COLUMN IF NOT EXISTS email VARCHAR(255),
    ADD COLUMN IF NOT EXISTS preferred_language VARCHAR(50),
    ADD COLUMN IF NOT EXISTS country VARCHAR(100),
    ADD COLUMN IF NOT EXISTS state VARCHAR(100),
    ADD COLUMN IF NOT EXISTS city VARCHAR(100),
    ADD COLUMN IF NOT EXISTS postal_code VARCHAR(20);

ALTER TABLE owner_profiles
    ADD COLUMN IF NOT EXISTS registration_type VARCHAR(20),
    ADD COLUMN IF NOT EXISTS business_size VARCHAR(50),
    ADD COLUMN IF NOT EXISTS gst_registration_number VARCHAR(100);

INSERT INTO business_types (name) VALUES
    ('Construction'),
    ('Manufacturing'),
    ('Hospitality'),
    ('Retail'),
    ('Logistics & Transport'),
    ('Facility Management'),
    ('Agriculture'),
    ('Healthcare'),
    ('Real Estate'),
    ('Automotive'),
    ('Other')
ON CONFLICT (name) DO NOTHING;
