package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect() (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(
		context.Background(),
		"postgres://postgres:postgres@localhost:5432/ledger_wallet",
	)
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	return pool, nil
}
