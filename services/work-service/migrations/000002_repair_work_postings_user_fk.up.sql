ALTER TABLE work_postings
    DROP CONSTRAINT IF EXISTS work_postings_owner_profile_id_fkey;

ALTER TABLE work_postings
    DROP CONSTRAINT IF EXISTS work_postings_user_id_fkey;

ALTER TABLE work_postings
    ADD CONSTRAINT work_postings_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES profiles(user_id) ON DELETE CASCADE;