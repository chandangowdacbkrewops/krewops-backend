ALTER TABLE profiles
    DROP COLUMN IF EXISTS date_of_birth,
    DROP COLUMN IF EXISTS email,
    DROP COLUMN IF EXISTS preferred_language,
    DROP COLUMN IF EXISTS country,
    DROP COLUMN IF EXISTS state,
    DROP COLUMN IF EXISTS city,
    DROP COLUMN IF EXISTS postal_code;

ALTER TABLE owner_profiles
    DROP COLUMN IF EXISTS registration_type,
    DROP COLUMN IF EXISTS business_size,
    DROP COLUMN IF EXISTS gst_registration_number;

DELETE FROM business_types WHERE name IN (
    'Construction',
    'Manufacturing',
    'Hospitality',
    'Retail',
    'Logistics & Transport',
    'Facility Management',
    'Agriculture',
    'Healthcare',
    'Real Estate',
    'Automotive',
    'Other'
);
