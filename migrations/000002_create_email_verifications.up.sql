CREATE TABLE IF NOT EXISTS email_verifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
  user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  otp_hash VARCHAR(255) NOT NULL,
  token_hash VARCHAR(255) NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  verified_at TIMESTAMPTZ NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT unique_email_verification_user UNIQUE (user_id)
);

CREATE INDEX IF NOT EXISTS idx_email_verifications_expires_at ON email_verifications (expires_at);
