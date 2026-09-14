DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'profiles'
          AND column_name = 'auth_user_id'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'profiles'
          AND column_name = 'user_id'
    ) THEN
        ALTER TABLE profiles RENAME COLUMN auth_user_id TO user_id;
    END IF;
END $$;

DROP INDEX IF EXISTS idx_profiles_auth_user_id;
CREATE INDEX IF NOT EXISTS idx_profiles_user_id ON profiles(user_id);