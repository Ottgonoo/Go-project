package repository

import (
	"context"

	"github.com/jackc/pgx/v5"

	"ledger-wallet/internal/models"
)

type EntryRepository struct{}

func NewEntryRepository() *EntryRepository {
	return &EntryRepository{}
}

func (r *EntryRepository) Create(
	ctx context.Context,
	tx pgx.Tx,
	entry models.Entry,
) (models.Entry, error) {

	err := tx.QueryRow(
		ctx,
		`
		INSERT INTO entries (
			transaction_id,
			account_id,
			direction,
			amount
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
		`,
		entry.TransactionID,
		entry.AccountID,
		entry.Direction,
		entry.Amount,
	).Scan(
		&entry.ID,
		&entry.CreatedAt,
	)

	return entry, err
}