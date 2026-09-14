-- Add is_active column to track which OTP is currently valid
ALTER TABLE otp_verifications
ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE;

-- Add index for querying active OTPs by phone number
CREATE INDEX idx_otp_phone_active
ON otp_verifications(phone_number, is_active)
WHERE is_active = TRUE;

-- Add index for rate limiting queries (recent requests)
CREATE INDEX idx_otp_phone_created
ON otp_verifications(phone_number, created_at DESC);
