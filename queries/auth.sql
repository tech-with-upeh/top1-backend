
-- name: CreateUser :one
INSERT INTO users (
    first_name,
    last_name,
    email,
    password_hash,
    phone_number,
    active,
    verified
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING
    id,
    first_name,
    last_name,
    email,
    password_hash,
    phone_number,
    active,
    verified,
    created_at,
    updated_at;


-- name: CreateEmailVerification :exec
INSERT INTO email_verifications (
    user_id,
    otp_hash,
    token_hash,
    expires_at
)
VALUES ($1, $2, $3, $4);


-- name: GetEmailVerificationByUserID :one
SELECT
    id,
    user_id,
    otp_hash,
    token_hash,
    expires_at,
    verified_at,
    created_at
FROM email_verifications
WHERE user_id = $1;


-- name: GetUserByEmail :one
SELECT
    id,
    first_name,
    last_name,
    email,
    password_hash,
    phone_number,
    active,
    verified,
    created_at,
    updated_at
FROM users
WHERE email = $1;


-- name: GetUserByID :one
SELECT
    id,
    first_name,
    last_name,
    email,
    password_hash,
    phone_number,
    active,
    verified,
    created_at,
    updated_at
FROM users
WHERE id = $1;


-- name: VerifyEmail :one
UPDATE email_verifications
SET verified_at = CURRENT_TIMESTAMP
WHERE user_id = $1
  AND (
      ($2::text <> '' AND otp_hash = $2)
      OR
      ($3::text <> '' AND token_hash = $3)
  )
  AND expires_at > CURRENT_TIMESTAMP
  AND verified_at IS NULL
RETURNING id;

-- name: UpdateEmailVerification :one
UPDATE email_verifications
SET
    otp_hash = $2,
    token_hash = $3,
    expires_at = $4,
    verified_at = NULL,
    created_at = CURRENT_TIMESTAMP
WHERE user_id = $1
RETURNING id::text;


-- name: ActivateVerifiedUser :exec
UPDATE users
SET
    verified = TRUE,
    active = TRUE,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;


-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens (
    user_id,
    token_hash,
    expires_at
)
VALUES ($1, $2, $3);


-- name: RevokeRefreshToken :one
UPDATE refresh_tokens
SET revoked_at = CURRENT_TIMESTAMP
WHERE user_id = $1
  AND token_hash = $2
  AND revoked_at IS NULL
  AND expires_at > CURRENT_TIMESTAMP
RETURNING id;


-- name: InsertRefreshToken :exec
INSERT INTO refresh_tokens (
    user_id,
    token_hash,
    expires_at
)
VALUES ($1, $2, $3);


-- name: RevokeRefreshTokenByHash :execrows
UPDATE refresh_tokens
SET revoked_at = CURRENT_TIMESTAMP
WHERE user_id = $1
  AND token_hash = $2
  AND revoked_at IS NULL;

-- name: ClaimEmailDelivery :execrows
INSERT INTO email_deliveries (
    idempotency_key,
    email_type,
    recipient,
    status,
    claim_token,
    claimed_at
)
VALUES (
    $1,
    $2,
    $3,
    'processing',
    $4,
    CURRENT_TIMESTAMP
)
ON CONFLICT (idempotency_key)
DO UPDATE SET
    status = 'processing',
    claim_token = EXCLUDED.claim_token,
    claimed_at = CURRENT_TIMESTAMP,
    last_error = NULL
WHERE
    email_deliveries.status = 'pending'
    OR (
        email_deliveries.status = 'processing'
        AND email_deliveries.claimed_at < CURRENT_TIMESTAMP - INTERVAL '5 minutes'
    );

-- name: GetEmailDeliveryStatus :one
SELECT status
FROM email_deliveries
WHERE idempotency_key = $1;

-- name: MarkEmailDeliverySent :execrows
UPDATE email_deliveries
SET
    status = 'sent',
    sent_at = CURRENT_TIMESTAMP,
    claimed_at = NULL,
    claim_token = NULL,
    last_error = NULL
WHERE idempotency_key = $1
  AND status = 'processing'
  AND claim_token = $2;

-- name: ReleaseEmailDeliveryClaim :execrows
UPDATE email_deliveries
SET
    status = 'pending',
    claimed_at = NULL,
    claim_token = NULL,
    last_error = $3
WHERE idempotency_key = $1
  AND status = 'processing'
  AND claim_token = $2;
