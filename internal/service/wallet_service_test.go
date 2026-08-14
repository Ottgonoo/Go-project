package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ledger-wallet/internal/database"
	"ledger-wallet/internal/repository"
)

func TestDepositInvalidAmount(t *testing.T) {
	service := &WalletService{}

	_, err := service.Deposit(
		nil,
		DepositRequest{
			WalletID:       1,
			Amount:         0,
			IdempotencyKey: "test-deposit-001",
		},
	)

	if err != ErrInvalidAmount {
		t.Fatalf("expected ErrInvalidAmount, got %v", err)
	}
}

func TestDepositNegativeAmount(t *testing.T) {
	service := &WalletService{}

	_, err := service.Deposit(
		nil,
		DepositRequest{
			WalletID:       1,
			Amount:         -100,
			IdempotencyKey: "test-deposit-002",
		},
	)

	if err != ErrInvalidAmount {
		t.Fatalf("expected ErrInvalidAmount, got %v", err)
	}
}

func TestWithdrawInvalidAmount(t *testing.T) {
	service := &WalletService{}

	_, err := service.Withdraw(
		nil,
		WithdrawRequest{
			WalletID:       1,
			Amount:         0,
			IdempotencyKey: "test-withdraw-001",
		},
	)

	if err != ErrInvalidAmount {
		t.Fatalf("expected ErrInvalidAmount, got %v", err)
	}
}

func TestWithdrawNegativeAmount(t *testing.T) {
	service := &WalletService{}

	_, err := service.Withdraw(
		nil,
		WithdrawRequest{
			WalletID:       1,
			Amount:         -500,
			IdempotencyKey: "test-withdraw-002",
		},
	)

	if err != ErrInvalidAmount {
		t.Fatalf("expected ErrInvalidAmount, got %v", err)
	}
}

func TestTransferInvalidAmount(t *testing.T) {
	service := &WalletService{}

	_, err := service.Transfer(
		nil,
		TransferRequest{
			FromWalletID:   1,
			ToWalletID:     2,
			Amount:         0,
			IdempotencyKey: "test-transfer-001",
		},
	)

	if err != ErrInvalidAmount {
		t.Fatalf("expected ErrInvalidAmount, got %v", err)
	}
}

func TestTransferNegativeAmount(t *testing.T) {
	service := &WalletService{}

	_, err := service.Transfer(
		nil,
		TransferRequest{
			FromWalletID:   1,
			ToWalletID:     2,
			Amount:         -100,
			IdempotencyKey: "test-transfer-002",
		},
	)

	if err != ErrInvalidAmount {
		t.Fatalf("expected ErrInvalidAmount, got %v", err)
	}
}

func TestTransferSameWallet(t *testing.T) {
	service := &WalletService{}

	_, err := service.Transfer(
		nil,
		TransferRequest{
			FromWalletID:   1,
			ToWalletID:     1,
			Amount:         100,
			IdempotencyKey: "test-transfer-003",
		},
	)

	if err == nil {
		t.Fatal("expected error for same wallet transfer")
	}

	if err.Error() != "cannot transfer to the same wallet" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWalletDepositWithdrawIntegration(t *testing.T) {
	ctx := context.Background()

	// Connect to real PostgreSQL
	conn, err := database.Connect()
	if err != nil {
		t.Fatalf("database connection failed: %v", err)
	}
	defer conn.Close(ctx)

	// Repositories
	accountRepository := repository.NewAccountRepository()
	transactionRepository := repository.NewTransactionRepository()
	entryRepository := repository.NewEntryRepository()
	walletRepository := repository.NewWalletRepository()

	// Services
	ledgerService := NewLedgerService(
		conn,
		accountRepository,
		transactionRepository,
		entryRepository,
	)

	walletService := NewWalletService(
		conn,
		ledgerService,
		walletRepository,
		accountRepository,
	)

	// Get current balance
	initialBalance, err := walletService.GetBalance(ctx, 1)
	if err != nil {
		t.Fatalf("failed to get initial balance: %v", err)
	}

	t.Logf("initial balance: %d", initialBalance)

	depositKey := fmt.Sprintf(
		"integration-deposit-%d",
		time.Now().UnixNano(),
	)

	depositRequest := DepositRequest{
		WalletID:       1,
		Amount:         100,
		IdempotencyKey: depositKey,
	}
	_, err = walletService.Deposit(ctx, depositRequest)
	if err != nil {
		t.Fatalf("deposit failed: %v", err)
	}

	// Check balance after deposit
	afterDeposit, err := walletService.GetBalance(ctx, 1)
	if err != nil {
		t.Fatalf("failed to get balance after deposit: %v", err)
	}

	expectedBalance := initialBalance + 100

	if afterDeposit != expectedBalance {
		t.Fatalf(
			"unexpected balance after deposit: got %d, want %d",
			afterDeposit,
			expectedBalance,
		)
	}

	// Repeat the exact same request.
	// Idempotency should prevent a second deposit.
	_, err = walletService.Deposit(ctx, depositRequest)
	if err != nil {
		t.Fatalf("idempotent deposit failed: %v", err)
	}

	afterDuplicate, err := walletService.GetBalance(ctx, 1)
	if err != nil {
		t.Fatalf("failed to get balance after duplicate deposit: %v", err)
	}

	if afterDuplicate != expectedBalance {
		t.Fatalf(
			"idempotency failed: got balance %d, want %d",
			afterDuplicate,
			expectedBalance,
		)
	}
	withdrawKey := fmt.Sprintf(
		"integration-withdraw-%d",
		time.Now().UnixNano(),
	)

	withdrawRequest := WithdrawRequest{
		WalletID:       1,
		Amount:         100,
		IdempotencyKey: withdrawKey,
	}

	_, err = walletService.Withdraw(ctx, withdrawRequest)
	if err != nil {
		t.Fatalf("withdraw failed: %v", err)
	}

	// Balance should return to the original amount.
	finalBalance, err := walletService.GetBalance(ctx, 1)
	if err != nil {
		t.Fatalf("failed to get final balance: %v", err)
	}

	if finalBalance != initialBalance {
		t.Fatalf(
			"unexpected final balance: got %d, want %d",
			finalBalance,
			initialBalance,
		)
	}

	t.Logf("final balance: %d", finalBalance)
}
