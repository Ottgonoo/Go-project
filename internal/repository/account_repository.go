package repository

import (
	"context"

	"github.com/jackc/pgx/v5"

	"ledger-wallet/internal/models"
)

type AccountRepository struct{}

func NewAccountRepository() *AccountRepository {
	return &AccountRepository{}
}

func (r *AccountRepository) Create(
	ctx context.Context,
	tx pgx.Tx,
	account models.Account,
) (models.Account, error) {

	err := tx.QueryRow(
		ctx,
		`
		INSERT INTO accounts (
			name,
			type,
			currency
		)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
		`,
		account.Name,
		account.Type,
		account.Currency,
	).Scan(
		&account.ID,
		&account.CreatedAt,
	)

	return account, err
}