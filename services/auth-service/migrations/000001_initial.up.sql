CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    phone_number VARCHAR(20) NOT NULL UNIQUE,

    status VARCHAR(20) NOT NULL DEFAULT 'active',

    phone_verified BOOLEAN NOT NULL DEFAULT FALSE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE otp_verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    phone_number VARCHAR(20) NOT NULL,

    otp_hash TEXT NOT NULL,

    expires_at TIMESTAMPTZ NOT NULL,

    verified_at TIMESTAMPTZ,

    attempts INTEGER NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_otp_phone_number
ON otp_verifications(phone_number);

CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    token_hash TEXT NOT NULL,

    expires_at TIMESTAMPTZ NOT NULL,

    revoked_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user_id
ON refresh_tokens(user_id);

CREATE INDEX idx_refresh_tokens_token_hash
ON refresh_tokens(token_hash);
