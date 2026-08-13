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
	ErrWalletNotFound      = errors.New("wallet not found")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrInvalidAmount       = errors.New("amount must be greater than zero")
	ErrCurrencyMismatch    = errors.New("currency mismatch")
)

type WalletService struct {
	db                *pgx.Conn
	ledgerService     *LedgerService
	walletRepository  *repository.WalletRepository
	accountRepository *repository.AccountRepository
}

func NewWalletService(
	db *pgx.Conn,
	ledgerService *LedgerService,
	walletRepository *repository.WalletRepository,
	accountRepository *repository.AccountRepository,
) *WalletService {
	return &WalletService{
		db:                db,
		ledgerService:     ledgerService,
		walletRepository:  walletRepository,
		accountRepository: accountRepository,
	}
}

type DepositRequest struct {
	WalletID       int64
	Amount         int64
	IdempotencyKey string
}

type WithdrawRequest struct {
	WalletID       int64
	Amount         int64
	IdempotencyKey string
}

type TransferRequest struct {
	FromWalletID   int64
	ToWalletID     int64
	Amount         int64
	IdempotencyKey string
}

// Deposit adds money to a wallet.
//
// Platform perspective:
//
// DEBIT  -> platform asset account
// CREDIT -> user's wallet liability account
func (s *WalletService) Deposit(
	ctx context.Context,
	req DepositRequest,
) (int64, error) {

	if req.Amount <= 0 {
		return 0, ErrInvalidAmount
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	wallet, err := s.walletRepository.GetByIDForUpdate(
		ctx,
		tx,
		req.WalletID,
	)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrWalletNotFound, err)
	}

	transaction := PostTransactionRequest{
		IdempotencyKey: req.IdempotencyKey,
		Description:    "wallet deposit",
		Entries: []models.Entry{
			{
				AccountID: getPlatformAssetAccountID(),
				Direction: models.EntryDirectionDebit,
				Amount:    req.Amount,
			},
			{
				AccountID: wallet.AccountID,
				Direction: models.EntryDirectionCredit,
				Amount:    req.Amount,
			},
		},
	}

	transactionID, err := s.ledgerService.PostTransactionTx(
		ctx,
		tx,
		transaction,
	)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	return transactionID, nil
}

// Withdraw removes money from a wallet.
//
// Platform perspective:
//
// DEBIT  -> user's wallet liability account
// CREDIT -> platform asset account
func (s *WalletService) Withdraw(
	ctx context.Context,
	req WithdrawRequest,
) (int64, error) {

	if req.Amount <= 0 {
		return 0, ErrInvalidAmount
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	// Lock the wallet row.
	//
	// This is critical for concurrency safety.
	wallet, err := s.walletRepository.GetByIDForUpdate(
		ctx,
		tx,
		req.WalletID,
	)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrWalletNotFound, err)
	}

	balance, err := s.ledgerService.GetCurrentBalanceTx(
		ctx,
		tx,
		wallet.AccountID,
	)
	if err != nil {
		return 0, err
	}

	if balance < req.Amount {
		return 0, ErrInsufficientBalance
	}

	transaction := PostTransactionRequest{
		IdempotencyKey: req.IdempotencyKey,
		Description:    "wallet withdrawal",
		Entries: []models.Entry{
			{
				AccountID: wallet.AccountID,
				Direction: models.EntryDirectionDebit,
				Amount:    req.Amount,
			},
			{
				AccountID: getPlatformAssetAccountID(),
				Direction: models.EntryDirectionCredit,
				Amount:    req.Amount,
			},
		},
	}

	transactionID, err := s.ledgerService.PostTransactionTx(
		ctx,
		tx,
		transaction,
	)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	return transactionID, nil
}

// Transfer moves money from one wallet to another.
//
// DEBIT  -> destination wallet liability
// CREDIT -> source wallet liability
func (s *WalletService) Transfer(
	ctx context.Context,
	req TransferRequest,
) (int64, error) {

	if req.Amount <= 0 {
		return 0, ErrInvalidAmount
	}

	if req.FromWalletID == req.ToWalletID {
		return 0, errors.New("cannot transfer to the same wallet")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	// Always lock wallets in deterministic ID order.
	//
	// This helps prevent deadlocks when two transfers happen
	// in opposite directions at the same time.
	firstID := req.FromWalletID
	secondID := req.ToWalletID

	if firstID > secondID {
		firstID, secondID = secondID, firstID
	}

	firstWallet, err := s.walletRepository.GetByIDForUpdate(
		ctx,
		tx,
		firstID,
	)
	if err != nil {
		return 0, fmt.Errorf("%w: wallet %d", ErrWalletNotFound, firstID)
	}

	secondWallet, err := s.walletRepository.GetByIDForUpdate(
		ctx,
		tx,
		secondID,
	)
	if err != nil {
		return 0, fmt.Errorf("%w: wallet %d", ErrWalletNotFound, secondID)
	}

	var fromWallet models.Wallet
	var toWallet models.Wallet

	if firstWallet.ID == req.FromWalletID {
		fromWallet = firstWallet
		toWallet = secondWallet
	} else {
		fromWallet = secondWallet
		toWallet = firstWallet
	}

	if fromWallet.Currency != toWallet.Currency {
		return 0, ErrCurrencyMismatch
	}

	balance, err := s.ledgerService.GetCurrentBalanceTx(
		ctx,
		tx,
		fromWallet.AccountID,
	)
	if err != nil {
		return 0, err
	}

	if balance < req.Amount {
		return 0, ErrInsufficientBalance
	}

	transaction := PostTransactionRequest{
		IdempotencyKey: req.IdempotencyKey,
		Description:    "wallet transfer",
		Entries: []models.Entry{
			{
				AccountID: toWallet.AccountID,
				Direction: models.EntryDirectionDebit,
				Amount:    req.Amount,
			},
			{
				AccountID: fromWallet.AccountID,
				Direction: models.EntryDirectionCredit,
				Amount:    req.Amount,
			},
		},
	}

	transactionID, err := s.ledgerService.PostTransactionTx(
		ctx,
		tx,
		transaction,
	)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	return transactionID, nil
}

// Temporary platform asset account.
//
// This will later be replaced by a real system account lookup.
func getPlatformAssetAccountID() int64 {
	return 1
}
