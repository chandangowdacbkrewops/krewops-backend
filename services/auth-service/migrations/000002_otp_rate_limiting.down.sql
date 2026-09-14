-- Drop indexes
DROP INDEX IF EXISTS idx_otp_phone_created;
DROP INDEX IF EXISTS idx_otp_phone_active;

-- Remove is_active column
ALTER TABLE otp_verifications
DROP COLUMN is_active;
