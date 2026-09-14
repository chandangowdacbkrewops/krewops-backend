ALTER TABLE profiles
    ADD COLUMN IF NOT EXISTS onboarding_step VARCHAR(32) NOT NULL DEFAULT 'role_selection';

ALTER TABLE profiles
    ADD CONSTRAINT profiles_onboarding_step_check
    CHECK (onboarding_step IN (
        'role_selection',
        'profile_details',
        'worker_work_types',
        'work_owner_business',
        'completed'
    ));
