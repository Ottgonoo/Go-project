package repository

import (
	"context"

	"github.com/jackc/pgx/v5"

	"ledger-wallet/internal/models"
)

type TransactionRepository struct{}

func NewTransactionRepository() *TransactionRepository {
	return &TransactionRepository{}
}

func (r *TransactionRepository) Create(
	ctx context.Context,
	tx pgx.Tx,
	transaction models.Transaction,
) (models.Transaction, error) {

	err := tx.QueryRow(
		ctx,
		`
		INSERT INTO transactions (
			idempotency_key,
			description,
			posted_at
		)
		VALUES ($1, $2, COALESCE($3, CURRENT_TIMESTAMP))
		RETURNING id, posted_at, created_at
		`,
		transaction.IdempotencyKey,
		transaction.Description,
		transaction.PostedAt,
	).Scan(
		&transaction.ID,
		&transaction.PostedAt,
		&transaction.CreatedAt,
	)

	return transaction, err
}

// GetByIdempotencyKey returns an existing transaction for an idempotency key.
func (r *TransactionRepository) GetByIdempotencyKey(
	ctx context.Context,
	tx pgx.Tx,
	key string,
) (models.Transaction, error) {

	var transaction models.Transaction

	err := tx.QueryRow(
		ctx,
		`
		SELECT
			id,
			idempotency_key,
			description,
			posted_at,
			created_at
		FROM transactions
		WHERE idempotency_key = $1
		`,
		key,
	).Scan(
		&transaction.ID,
		&transaction.IdempotencyKey,
		&transaction.Description,
		&transaction.PostedAt,
		&transaction.CreatedAt,
	)

	return transaction, err
}