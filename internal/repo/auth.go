package repo

import (
	"context"
	"time"

	"top1/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthRepository struct {
	pool *pgxpool.Pool
}

func NewAuthRepository(pool *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{
		pool: pool,
	}
}

func (r *AuthRepository) CreateWithVerification(
	ctx context.Context,
	user *models.User,
	verification *models.EmailVerification,
) (*models.User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	query := `
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
			id, first_name, last_name, email,
			password_hash, phone_number, active,
			verified, created_at, updated_at
	`

	err = tx.QueryRow(
		ctx,
		query,
		user.FirstName,
		user.LastName,
		user.Email,
		user.PasswordHash,
		user.PhoneNumber,
		user.Active,
		user.Verified,
	).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.PasswordHash,
		&user.PhoneNumber,
		&user.Active,
		&user.Verified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	verification.UserID = user.ID

	verificationQuery := `
		INSERT INTO email_verifications (
			user_id,
			otp_hash,
			token_hash,
			expires_at
		)
		VALUES ($1, $2, $3, $4)
	`

	_, err = tx.Exec(
		ctx,
		verificationQuery,
		verification.UserID,
		verification.OTPHash,
		verification.TokenHash,
		verification.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return user, nil
}

func (r *AuthRepository) GetEmailVerificationByUserID(
	ctx context.Context,
	userID string,
) (*models.EmailVerification, error) {
	query := `
		SELECT
			id,
			user_id,
			otp_hash,
			token_hash,
			expires_at,
			verified_at,
			created_at
		FROM email_verifications
		WHERE user_id = $1
	`

	var verification models.EmailVerification

	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&verification.ID,
		&verification.UserID,
		&verification.OTPHash,
		&verification.TokenHash,
		&verification.ExpiresAt,
		&verification.VerifiedAt,
		&verification.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &verification, nil
}

func (r *AuthRepository) GetUserByEmail(
	ctx context.Context,
	email string,
) (*models.User, error) {
	query := `
        SELECT
            id, first_name, last_name, email,
            password_hash, phone_number, active,
            verified, created_at, updated_at
        FROM users
        WHERE email = $1
    `

	var user models.User

	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.PasswordHash,
		&user.PhoneNumber,
		&user.Active,
		&user.Verified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *AuthRepository) VerifyEmail(
	ctx context.Context,
	userID string,
	otpHash string,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var verificationID string

	err = tx.QueryRow(ctx, `
        UPDATE email_verifications
        SET verified_at = CURRENT_TIMESTAMP
        WHERE user_id = $1
          AND otp_hash = $2
          AND expires_at > CURRENT_TIMESTAMP
          AND verified_at IS NULL
        RETURNING id
    `, userID, otpHash).Scan(&verificationID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
        UPDATE users
        SET verified = TRUE,
            active = TRUE,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = $1
    `, userID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func GetUserByID(pool *pgxpool.Pool, id string) (*models.User, error) {
	var ctx context.Context
	var cancel context.CancelFunc

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var query string = `
		SELECT id, first_name, last_name, email, password_hash, phone_number, active, verified,created_at, updated_at
		FROM users
		WHERE id = $1
        `

	var user models.User

	err := pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.PasswordHash,
		&user.PhoneNumber,
		&user.Active,
		&user.Verified,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
