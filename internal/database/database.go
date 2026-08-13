package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func Connect() (*pgx.Conn, error) {
	conn, err := pgx.Connect(
		context.Background(),
		"postgres://postgres:postgres@localhost:5432/ledger_wallet",
	)

	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	return conn, nil
}