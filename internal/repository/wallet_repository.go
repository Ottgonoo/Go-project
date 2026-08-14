package repository

import (
	"context"

	"github.com/jackc/pgx/v5"

	"ledger-wallet/internal/models"
)

type WalletRepository struct{}

func NewWalletRepository() *WalletRepository {
	return &WalletRepository{}
}

// Create creates a wallet linked to a user and a ledger account.
func (r *WalletRepository) Create(
	ctx context.Context,
	tx pgx.Tx,
	wallet models.Wallet,
) (models.Wallet, error) {

	err := tx.QueryRow(
		ctx,
		`
		INSERT INTO wallets (
			user_id,
			account_id,
			currency
		)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
		`,
		wallet.UserID,
		wallet.AccountID,
		wallet.Currency,
	).Scan(
		&wallet.ID,
		&wallet.CreatedAt,
	)

	return wallet, err
}

// GetByID returns a wallet by its ID.
func (r *WalletRepository) GetByID(
	ctx context.Context,
	tx pgx.Tx,
	walletID int64,
) (models.Wallet, error) {

	var wallet models.Wallet

	err := tx.QueryRow(
		ctx,
		`
		SELECT
			id,
			user_id,
			account_id,
			currency,
			created_at
		FROM wallets
		WHERE id = $1
		`,
		walletID,
	).Scan(
		&wallet.ID,
		&wallet.UserID,
		&wallet.AccountID,
		&wallet.Currency,
		&wallet.CreatedAt,
	)

	return wallet, err
}

// GetByIDForUpdate returns a wallet while locking its row.
//
// FOR UPDATE prevents concurrent operations from modifying
// the same wallet simultaneously inside separate transactions.
func (r *WalletRepository) GetByIDForUpdate(
	ctx context.Context,
	tx pgx.Tx,
	walletID int64,
) (models.Wallet, error) {

	var wallet models.Wallet

	err := tx.QueryRow(
		ctx,
		`
		SELECT
			id,
			user_id,
			account_id,
			currency,
			created_at
		FROM wallets
		WHERE id = $1
		FOR UPDATE
		`,
		walletID,
	).Scan(
		&wallet.ID,
		&wallet.UserID,
		&wallet.AccountID,
		&wallet.Currency,
		&wallet.CreatedAt,
	)

	return wallet, err
}

// GetByAccountID returns the wallet linked to a ledger account.
func (r *WalletRepository) GetByAccountID(
	ctx context.Context,
	tx pgx.Tx,
	accountID int64,
) (models.Wallet, error) {

	var wallet models.Wallet

	err := tx.QueryRow(
		ctx,
		`
		SELECT
			id,
			user_id,
			account_id,
			currency,
			created_at
		FROM wallets
		WHERE account_id = $1
		`,
		accountID,
	).Scan(
		&wallet.ID,
		&wallet.UserID,
		&wallet.AccountID,
		&wallet.Currency,
		&wallet.CreatedAt,
	)

	return wallet, err
}
