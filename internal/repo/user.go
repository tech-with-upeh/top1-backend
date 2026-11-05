package repo

import (
	"context"
	"time"

	"top1/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateUser(pool *pgxpool.Pool, user *models.User) (*models.User, error) {
	var ctx context.Context
	var cancel context.CancelFunc

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var query string = `
		INSERT INTO users (first_name, last_name, email, password_hash, phone_number, active, verified)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, first_name, last_name, email, password_hash, phone_number, active, verified, created_at, updated_at
	`

	err := pool.QueryRow(ctx, query, user.FirstName, user.LastName, user.Email, user.PasswordHash, user.PhoneNumber, user.Active, user.Verified).Scan(
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
	return user, nil
}

func GetUserByMail(pool *pgxpool.Pool, email string) (*models.User, error) {
	var ctx context.Context
	var cancel context.CancelFunc

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var query string = `
		SELECT id, first_name, last_name, email, password_hash, phone_number, active, verified,created_at, updated_at
		FROM users
		WHERE email = $1
        `

	var user models.User

	err := pool.QueryRow(ctx, query, email).Scan(
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
