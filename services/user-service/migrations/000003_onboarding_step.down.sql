ALTER TABLE profiles DROP CONSTRAINT IF EXISTS profiles_onboarding_step_check;
ALTER TABLE profiles DROP COLUMN IF EXISTS onboarding_step;
