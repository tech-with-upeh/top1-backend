package db

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(dbURL string) (*pgxpool.Pool, error) {
	var ctx context.Context = context.Background()

	var config *pgxpool.Config
	var err error

	config, err = pgxpool.ParseConfig(dbURL)
	if err != nil {
		slog.Error("Unable to parse DB URL")
		return nil, err
	}

	var pool *pgxpool.Pool
	pool, err = pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		slog.Error("Unable to create connection Pool: %v", err)
		return nil, err
	}

	if pool.Ping(ctx) != nil {
		slog.Error("Unable to Ping DATABASE")
		pool.Close()
		return nil, err
	}

	slog.Info("Successfully Connected to postgres DB")
	return pool, nil
}
