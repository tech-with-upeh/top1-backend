package repo

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"
	"uuid"

	"top1/internal/db/sqlc"
	"top1/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewAuthRepository(pool *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

var ErrInvalidOrExpiredCode = errors.New("invalid or expired verification code")

// CreateWithVerification creates a user and their email verification
// record within a single transaction.
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

	q := r.queries.WithTx(tx)

	created, err := q.CreateUser(ctx, sqlc.CreateUserParams{
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		PhoneNumber:  user.PhoneNumber,
		Active:       &user.Active,
		Verified:     &user.Verified,
	})
	if err != nil {
		return nil, err
	}

	verification.UserID = uuid.UUID(created.ID.Bytes)

	err = q.CreateEmailVerification(ctx, sqlc.CreateEmailVerificationParams{
		UserID:    created.ID,
		OtpHash:   verification.OTPHash,
		TokenHash: verification.TokenHash,
		ExpiresAt: pgtype.Timestamptz{
			Time:  verification.ExpiresAt,
			Valid: true,
		},
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return mapCreatedUser(created), nil
}

func (r *AuthRepository) GetEmailVerificationByUserID(
	ctx context.Context,
	userID string,
) (*models.EmailVerification, error) {
	id, err := parseUUID(userID)
	if err != nil {
		return nil, err
	}

	verification, err := r.queries.GetEmailVerificationByUserID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &models.EmailVerification{
		ID:         uuid.UUID(verification.ID.Bytes),
		UserID:     uuid.UUID(verification.UserID.Bytes),
		OTPHash:    verification.OtpHash,
		TokenHash:  verification.TokenHash,
		ExpiresAt:  verification.ExpiresAt.Time,
		VerifiedAt: nullableTime(verification.VerifiedAt),
		CreatedAt:  verification.CreatedAt.Time,
	}, nil
}

func (r *AuthRepository) GetUserByEmail(
	ctx context.Context,
	email string,
) (*models.User, error) {
	row, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return mapUser(
		row.ID,
		row.FirstName,
		row.LastName,
		row.Email,
		row.PasswordHash,
		row.PhoneNumber,
		row.Active,
		row.Verified,
		row.CreatedAt,
		row.UpdatedAt,
	), nil
}

func (r *AuthRepository) VerifyEmail(
	ctx context.Context,
	userID string,
	otpHash string,
	tokenHash string,
) error {
	if otpHash == "" && tokenHash == "" {
		return errors.New("either otpHash or tokenHash must be provided")
	}

	id, err := parseUUID(userID)
	if err != nil {
		return err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	q := r.queries.WithTx(tx)

	_, err = q.VerifyEmail(ctx, sqlc.VerifyEmailParams{
		UserID:  id,
		Column2: otpHash,
		Column3: tokenHash,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidOrExpiredCode
	}
	if err != nil {
		return err
	}

	if err := q.ActivateVerifiedUser(ctx, id); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *AuthRepository) UpdateEmailVerification(
	ctx context.Context,
	userID string,
	otpHash string,
	tokenHash string,
	expiresAt time.Time,
) error {
	parsedID, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	rows, err := r.queries.UpdateEmailVerification(
		ctx,
		sqlc.UpdateEmailVerificationParams{
			UserID: pgtype.UUID{
				Bytes: parsedID,
				Valid: true,
			},
			OtpHash:   otpHash,
			TokenHash: tokenHash,
			ExpiresAt: pgtype.Timestamptz{
				Time:  expiresAt,
				Valid: true,
			},
		},
	)
	if err != nil {
		return err
	}

	if rows == "" {
		return pgx.ErrNoRows
	}

	return nil
}

func (r *AuthRepository) CreateRefreshToken(
	ctx context.Context,
	userID string,
	tokenHash string,
	expiresAt time.Time,
) error {
	id, err := parseUUID(userID)
	if err != nil {
		return err
	}

	return r.queries.CreateRefreshToken(ctx, sqlc.CreateRefreshTokenParams{
		UserID:    id,
		TokenHash: tokenHash,
		ExpiresAt: pgtype.Timestamptz{
			Time:  expiresAt,
			Valid: true,
		},
	})
}

func (r *AuthRepository) RotateRefreshToken(
	ctx context.Context,
	userID string,
	oldTokenHash string,
	newTokenHash string,
	newExpiresAt time.Time,
) error {
	id, err := parseUUID(userID)
	if err != nil {
		return err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	q := r.queries.WithTx(tx)

	_, err = q.RevokeRefreshToken(ctx, sqlc.RevokeRefreshTokenParams{
		UserID:    id,
		TokenHash: oldTokenHash,
	})
	if err != nil {
		return err
	}

	err = q.InsertRefreshToken(ctx, sqlc.InsertRefreshTokenParams{
		UserID:    id,
		TokenHash: newTokenHash,
		ExpiresAt: pgtype.Timestamptz{
			Time:  newExpiresAt,
			Valid: true,
		},
	})
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *AuthRepository) RevokeRefreshToken(
	ctx context.Context,
	userID string,
	tokenHash string,
) error {
	id, err := parseUUID(userID)
	if err != nil {
		return err
	}

	rowsAffected, err := r.queries.RevokeRefreshTokenByHash(
		ctx,
		sqlc.RevokeRefreshTokenByHashParams{
			UserID:    id,
			TokenHash: tokenHash,
		},
	)
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRefreshTokenNotFound
	}

	return nil
}

func (r *AuthRepository) GetUserByID(
	ctx context.Context,
	id string,
) (*models.User, error) {
	userID, err := parseUUID(id)
	if err != nil {
		return nil, err
	}

	row, err := r.queries.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return mapUser(
		row.ID,
		row.FirstName,
		row.LastName,
		row.Email,
		row.PasswordHash,
		row.PhoneNumber,
		row.Active,
		row.Verified,
		row.CreatedAt,
		row.UpdatedAt,
	), nil
}

// ClaimEmailDelivery attempts to acquire an email delivery.
func (r *AuthRepository) ClaimEmailDelivery(
	ctx context.Context,
	idempotencyKey string,
	emailType string,
	recipient string,
) (string, bool, error) {
	claimToken, err := generateClaimToken()
	if err != nil {
		return "", false, err
	}

	rows, err := r.queries.ClaimEmailDelivery(
		ctx,
		sqlc.ClaimEmailDeliveryParams{
			IdempotencyKey: idempotencyKey,
			EmailType:      emailType,
			Recipient:      recipient,
			ClaimToken:     &claimToken,
		},
	)
	if err != nil {
		return "", false, err
	}

	return claimToken, rows == 1, nil
}

// MarkEmailDeliverySent marks a successfully delivered email.
func (r *AuthRepository) MarkEmailDeliverySent(
	ctx context.Context,
	idempotencyKey string,
	claimToken string,
) (bool, error) {
	rows, err := r.queries.MarkEmailDeliverySent(
		ctx,
		sqlc.MarkEmailDeliverySentParams{
			IdempotencyKey: idempotencyKey,
			ClaimToken:     &claimToken,
		},
	)
	if err != nil {
		return false, err
	}

	return rows == 1, nil
}

// ReleaseEmailDeliveryClaim makes a failed delivery available for retry.
func (r *AuthRepository) ReleaseEmailDeliveryClaim(
	ctx context.Context,
	idempotencyKey string,
	claimToken string,
	lastError string,
) (bool, error) {
	rows, err := r.queries.ReleaseEmailDeliveryClaim(
		ctx,
		sqlc.ReleaseEmailDeliveryClaimParams{
			IdempotencyKey: idempotencyKey,
			ClaimToken:     &claimToken,
			LastError:      &lastError,
		},
	)
	if err != nil {
		return false, err
	}

	return rows == 1, nil
}

func (r *AuthRepository) GetEmailDeliveryStatus(
	ctx context.Context,
	idempotencyKey string,
) (string, error) {
	return r.queries.GetEmailDeliveryStatus(ctx, idempotencyKey)
}

func generateClaimToken() (string, error) {
	token := make([]byte, 32)

	if _, err := rand.Read(token); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(token), nil
}

// Helpers

func parseUUID(value string) (pgtype.UUID, error) {
	var id pgtype.UUID
	err := id.Scan(value)
	return id, err
}

func mapCreatedUser(row sqlc.CreateUserRow) *models.User {
	return mapUser(
		row.ID,
		row.FirstName,
		row.LastName,
		row.Email,
		row.PasswordHash,
		row.PhoneNumber,
		row.Active,
		row.Verified,
		row.CreatedAt,
		row.UpdatedAt,
	)
}

func mapUser(
	id pgtype.UUID,
	firstName string,
	lastName string,
	email string,
	passwordHash string,
	phoneNumber *string,
	active *bool,
	verified *bool,
	createdAt pgtype.Timestamp,
	updatedAt pgtype.Timestamp,
) *models.User {
	return &models.User{
		ID:           uuid.UUID(id.Bytes),
		FirstName:    firstName,
		LastName:     lastName,
		Email:        email,
		PasswordHash: passwordHash,
		PhoneNumber:  phoneNumber,
		Active:       boolValue(active),
		Verified:     boolValue(verified),
		CreatedAt:    createdAt.Time,
		UpdatedAt:    updatedAt.Time,
	}
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func nullableTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}

	t := value.Time
	return &t
}
