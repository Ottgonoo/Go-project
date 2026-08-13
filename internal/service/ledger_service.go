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

	// Core double-entry invariant.
	if totalDebit != totalCredit {
		return ErrUnbalancedTransaction
	}

	return nil
}

// PostTransaction creates an atomic double-entry transaction.
//
// The transaction and all ledger entries are written inside
// one PostgreSQL transaction. If any operation fails,
// everything is rolled back.
func (s *LedgerService) PostTransaction(
	ctx context.Context,
	req PostTransactionRequest,
) (int64, error) {

	// Validate the complete transaction before writing anything.
	if err := validateEntries(req.Entries); err != nil {
		return 0, err
	}

	// Start PostgreSQL transaction.
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}

	// If anything fails before Commit, rollback everything.
	defer tx.Rollback(ctx)

	transaction := models.Transaction{
		IdempotencyKey: req.IdempotencyKey,
		Description:    req.Description,
	}

	// Create the transaction record.
	createdTransaction, err := s.transactionRepository.Create(
		ctx,
		tx,
		transaction,
	)
	if err != nil {
		return 0, err
	}

	// Create all ledger entries.
	for _, entry := range req.Entries {
		entry.TransactionID = createdTransaction.ID

		_, err := s.entryRepository.Create(
			ctx,
			tx,
			entry,
		)
		if err != nil {
			return 0, err
		}
	}

	// Only a complete transaction is committed.
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	return createdTransaction.ID, nil
}
