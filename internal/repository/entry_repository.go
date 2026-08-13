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

// GetBalance calculates an account balance entirely
// from immutable ledger entries.
//
// ASSET / EXPENSE:
//
//	DEBIT  increases balance
//	CREDIT decreases balance
//
// LIABILITY / EQUITY / REVENUE:
//
//	CREDIT increases balance
//	DEBIT  decreases balance
func (r *EntryRepository) GetBalance(
	ctx context.Context,
	tx pgx.Tx,
	accountID int64,
	accountType models.AccountType,
) (int64, error) {

	var balance int64

	err := tx.QueryRow(
		ctx,
		`
		SELECT COALESCE(
			SUM(
				CASE
					WHEN $2 IN ('ASSET', 'EXPENSE')
						AND direction = 'DEBIT'
						THEN amount

					WHEN $2 IN ('ASSET', 'EXPENSE')
						AND direction = 'CREDIT'
						THEN -amount

					WHEN $2 IN ('LIABILITY', 'EQUITY', 'REVENUE')
						AND direction = 'CREDIT'
						THEN amount

					WHEN $2 IN ('LIABILITY', 'EQUITY', 'REVENUE')
						AND direction = 'DEBIT'
						THEN -amount

					ELSE 0
				END
			),
			0
		)
		FROM entries
		WHERE account_id = $1
		`,
		accountID,
		string(accountType),
	).Scan(&balance)

	return balance, err
}
