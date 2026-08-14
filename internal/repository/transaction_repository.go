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
			request_hash,
			description,
			posted_at
		)
		VALUES ($1, $2, $3, COALESCE($4, CURRENT_TIMESTAMP))
		RETURNING
			id,
			idempotency_key,
			request_hash,
			description,
			posted_at,
			created_at
		`,
		transaction.IdempotencyKey,
		transaction.RequestHash,
		transaction.Description,
		transaction.PostedAt,
	).Scan(
		&transaction.ID,
		&transaction.IdempotencyKey,
		&transaction.RequestHash,
		&transaction.Description,
		&transaction.PostedAt,
		&transaction.CreatedAt,
	)

	return transaction, err
}

// CreateIdempotent creates a transaction only if the
// idempotency key does not already exist.
//
// If the key already exists, it returns the existing transaction.
func (r *TransactionRepository) CreateIdempotent(
	ctx context.Context,
	tx pgx.Tx,
	transaction models.Transaction,
) (models.Transaction, error) {

	var created models.Transaction

	err := tx.QueryRow(
		ctx,
		`
		INSERT INTO transactions (
			idempotency_key,
			request_hash,
			description,
			posted_at
		)
		VALUES ($1, $2, $3, COALESCE($4, CURRENT_TIMESTAMP))
		ON CONFLICT (idempotency_key)
		DO UPDATE SET idempotency_key = EXCLUDED.idempotency_key
		RETURNING
			id,
			idempotency_key,
			request_hash,
			description,
			posted_at,
			created_at
		`,
		transaction.IdempotencyKey,
		transaction.RequestHash,
		transaction.Description,
		transaction.PostedAt,
	).Scan(
		&created.ID,
		&created.IdempotencyKey,
		&created.RequestHash,
		&created.Description,
		&created.PostedAt,
		&created.CreatedAt,
	)

	return created, err
}

// GetByIdempotencyKey returns an existing transaction
// for an idempotency key.
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
			request_hash,
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
		&transaction.RequestHash,
		&transaction.Description,
		&transaction.PostedAt,
		&transaction.CreatedAt,
	)

	return transaction, err
}
