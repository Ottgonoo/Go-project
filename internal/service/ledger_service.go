package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"ledger-wallet/internal/models"
	"ledger-wallet/internal/repository"
)

var (
	ErrInvalidTransaction    = errors.New("invalid transaction")
	ErrUnbalancedTransaction = errors.New("debits and credits must be equal")
	ErrIdempotencyConflict   = errors.New("idempotency key already used with different request")
)

type LedgerService struct {
	db                    *pgx.Conn
	accountRepository     *repository.AccountRepository
	transactionRepository *repository.TransactionRepository
	entryRepository       *repository.EntryRepository
}

func NewLedgerService(
	db *pgx.Conn,
	accountRepository *repository.AccountRepository,
	transactionRepository *repository.TransactionRepository,
	entryRepository *repository.EntryRepository,
) *LedgerService {
	return &LedgerService{
		db:                    db,
		accountRepository:     accountRepository,
		transactionRepository: transactionRepository,
		entryRepository:       entryRepository,
	}
}

// PostTransactionRequest contains everything required
// to create a double-entry ledger transaction.
type PostTransactionRequest struct {
	IdempotencyKey string
	Description    string
	Entries        []models.Entry
}

// validateEntries checks the core double-entry bookkeeping rules.
//
// Rules:
//   - A transaction must contain at least two entries.
//   - Every amount must be positive.
//   - Every entry must have a valid direction.
//   - Total debits must equal total credits.
func validateEntries(entries []models.Entry) error {
	if len(entries) < 2 {
		return ErrInvalidTransaction
	}

	var totalDebit int64
	var totalCredit int64

	for _, entry := range entries {
		if entry.Amount <= 0 {
			return fmt.Errorf(
				"%w: amount must be greater than zero",
				ErrInvalidTransaction,
			)
		}

		switch entry.Direction {
		case models.EntryDirectionDebit:
			totalDebit += entry.Amount

		case models.EntryDirectionCredit:
			totalCredit += entry.Amount

		default:
			return fmt.Errorf(
				"%w: invalid direction",
				ErrInvalidTransaction,
			)
		}
	}

	if totalDebit != totalCredit {
		return ErrUnbalancedTransaction
	}

	return nil
}

// PostTransaction creates an atomic double-entry transaction.
//
// If the idempotency key already exists, the existing transaction
// is returned and new entries are NOT created.
func (s *LedgerService) PostTransaction(
	ctx context.Context,
	req PostTransactionRequest,
) (int64, error) {

	if err := validateEntries(req.Entries); err != nil {
		return 0, err
	}

	if req.IdempotencyKey == "" {
		return 0, fmt.Errorf(
			"%w: idempotency key is required",
			ErrInvalidTransaction,
		)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}

	defer tx.Rollback(ctx)

	// Check whether this idempotency key already exists.
	existingTransaction, err := s.transactionRepository.GetByIdempotencyKey(
		ctx,
		tx,
		req.IdempotencyKey,
	)

	if err == nil {
		// Already processed.
		// Do not create entries again.
		if err := tx.Commit(ctx); err != nil {
			return 0, err
		}

		return existingTransaction.ID, nil
	}

	// Transaction does not exist.
	transaction := models.Transaction{
		IdempotencyKey: req.IdempotencyKey,
		Description:    req.Description,
	}

	createdTransaction, err := s.transactionRepository.Create(
		ctx,
		tx,
		transaction,
	)
	if err != nil {
		return 0, err
	}

	for _, entry := range req.Entries {
		entry.TransactionID = createdTransaction.ID

		if _, err := s.entryRepository.Create(
			ctx,
			tx,
			entry,
		); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	return createdTransaction.ID, nil
}

// PostTransactionTx creates a double-entry transaction
// using an existing PostgreSQL transaction.
//
// This is used by wallet operations so that wallet locking,
// balance checking and ledger posting happen atomically.
//
// If the idempotency key already exists, the existing transaction
// is returned and new entries are NOT created.
func (s *LedgerService) PostTransactionTx(
	ctx context.Context,
	tx pgx.Tx,
	req PostTransactionRequest,
) (int64, error) {

	if err := validateEntries(req.Entries); err != nil {
		return 0, err
	}

	if req.IdempotencyKey == "" {
		return 0, fmt.Errorf(
			"%w: idempotency key is required",
			ErrInvalidTransaction,
		)
	}

	// Check whether this idempotency key was already processed.
	existingTransaction, err := s.transactionRepository.GetByIdempotencyKey(
		ctx,
		tx,
		req.IdempotencyKey,
	)

	if err == nil {
		// Already exists.
		// IMPORTANT:
		// Do not create entries again.
		return existingTransaction.ID, nil
	}

	transaction := models.Transaction{
		IdempotencyKey: req.IdempotencyKey,
		Description:    req.Description,
	}

	createdTransaction, err := s.transactionRepository.Create(
		ctx,
		tx,
		transaction,
	)
	if err != nil {
		return 0, err
	}

	for _, entry := range req.Entries {
		entry.TransactionID = createdTransaction.ID

		if _, err := s.entryRepository.Create(
			ctx,
			tx,
			entry,
		); err != nil {
			return 0, err
		}
	}

	return createdTransaction.ID, nil
}

// GetCurrentBalance calculates the account balance
// directly from immutable ledger entries.
//
// ASSET / EXPENSE:
//
//	debit - credit
//
// LIABILITY / EQUITY / REVENUE:
//
//	credit - debit
func (s *LedgerService) GetCurrentBalance(
	ctx context.Context,
	accountID int64,
) (int64, error) {

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}

	defer tx.Rollback(ctx)

	return s.GetCurrentBalanceTx(ctx, tx, accountID)
}

// GetCurrentBalanceTx calculates an account balance
// using an existing PostgreSQL transaction.
func (s *LedgerService) GetCurrentBalanceTx(
	ctx context.Context,
	tx pgx.Tx,
	accountID int64,
) (int64, error) {

	account, err := s.accountRepository.GetByID(
		ctx,
		tx,
		accountID,
	)
	if err != nil {
		return 0, err
	}

	var debitTotal int64
	var creditTotal int64

	err = tx.QueryRow(
		ctx,
		`
		SELECT
			COALESCE(
				SUM(
					CASE
						WHEN direction = 'DEBIT'
						THEN amount
						ELSE 0
					END
				),
				0
			),
			COALESCE(
				SUM(
					CASE
						WHEN direction = 'CREDIT'
						THEN amount
						ELSE 0
					END
				),
				0
			)
		FROM entries
		WHERE account_id = $1
		`,
		accountID,
	).Scan(
		&debitTotal,
		&creditTotal,
	)

	if err != nil {
		return 0, err
	}

	switch account.Type {
	case models.AccountTypeAsset,
		models.AccountTypeExpense:

		return debitTotal - creditTotal, nil

	case models.AccountTypeLiability,
		models.AccountTypeEquity,
		models.AccountTypeRevenue:

		return creditTotal - debitTotal, nil

	default:
		return 0, fmt.Errorf(
			"%w: unknown account type %s",
			ErrInvalidTransaction,
			account.Type,
		)
	}
}
