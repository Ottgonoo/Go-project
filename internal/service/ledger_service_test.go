package service

import (
	"context"
	"testing"
	"time"

	"ledger-wallet/internal/database"
	"ledger-wallet/internal/models"
	"ledger-wallet/internal/repository"
)

func TestValidateEntries(t *testing.T) {
	tests := []struct {
		name    string
		entries []models.Entry
		wantErr bool
	}{
		{
			name: "balanced transaction",
			entries: []models.Entry{
				{
					AccountID: 1,
					Direction: models.EntryDirectionDebit,
					Amount:    100,
				},
				{
					AccountID: 2,
					Direction: models.EntryDirectionCredit,
					Amount:    100,
				},
			},
			wantErr: false,
		},
		{
			name: "unbalanced transaction",
			entries: []models.Entry{
				{
					AccountID: 1,
					Direction: models.EntryDirectionDebit,
					Amount:    10000,
				},
				{
					AccountID: 2,
					Direction: models.EntryDirectionCredit,
					Amount:    5000,
				},
			},
			wantErr: true,
		},
		{
			name: "negative amount",
			entries: []models.Entry{
				{
					AccountID: 1,
					Direction: models.EntryDirectionDebit,
					Amount:    -10000,
				},
				{
					AccountID: 2,
					Direction: models.EntryDirectionCredit,
					Amount:    10000,
				},
			},
			wantErr: true,
		},
		{
			name: "zero amount",
			entries: []models.Entry{
				{
					AccountID: 1,
					Direction: models.EntryDirectionDebit,
					Amount:    0,
				},
				{
					AccountID: 2,
					Direction: models.EntryDirectionCredit,
					Amount:    0,
				},
			},
			wantErr: true,
		},
		{
			name: "only one entry",
			entries: []models.Entry{
				{
					AccountID: 1,
					Direction: models.EntryDirectionDebit,
					Amount:    10000,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEntries(tt.entries)

			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}

func TestValidateEntriesInvalidDirection(t *testing.T) {
	entries := []models.Entry{
		{
			AccountID: 1,
			Direction: "INVALID",
			Amount:    10000,
		},
		{
			AccountID: 2,
			Direction: models.EntryDirectionCredit,
			Amount:    10000,
		},
	}

	err := validateEntries(entries)

	if err == nil {
		t.Fatal("expected invalid direction error")
	}
}

func TestHistoricalBalance(t *testing.T) {
	ctx := context.Background()

	// Connect to PostgreSQL.
	conn, err := database.Connect()
	if err != nil {
		t.Fatalf("database connection failed: %v", err)
	}
	defer conn.Close()

	// Repositories.
	accountRepository := repository.NewAccountRepository()
	transactionRepository := repository.NewTransactionRepository()
	entryRepository := repository.NewEntryRepository()

	// Ledger service.
	ledgerService := NewLedgerService(
		conn,
		accountRepository,
		transactionRepository,
		entryRepository,
	)

	// DB дээр:
	// account 1 = Platform Cash (ASSET)
	// account 2 = Test Wallet Account (LIABILITY)
	accountID := int64(2)

	// Get balance before test transaction.
	initialBalance, err := ledgerService.GetCurrentBalance(
		ctx,
		accountID,
	)
	if err != nil {
		t.Fatalf("failed to get initial balance: %v", err)
	}

	t.Logf("initial balance: %d", initialBalance)

	// Capture a timestamp safely before creating the transaction.
	// Use UTC and a larger margin.
	beforeDeposit := time.Now().UTC().Add(-2 * time.Second)

	// Unique idempotency key.
	depositKey := "historical-test-" +
		time.Now().Format("20060102150405.000000000")

	// Deposit:
	//
	// DEBIT  Platform Cash  (account 1)
	// CREDIT Wallet         (account 2)
	_, err = ledgerService.PostTransaction(
		ctx,
		PostTransactionRequest{
			IdempotencyKey: depositKey,
			Description:    "historical balance test",
			Entries: []models.Entry{
				{
					AccountID: 1,
					Direction: models.EntryDirectionDebit,
					Amount:    100,
				},
				{
					AccountID: accountID,
					Direction: models.EntryDirectionCredit,
					Amount:    100,
				},
			},
		},
	)

	if err != nil {
		t.Fatalf("failed to post test transaction: %v", err)
	}

	// Historical balance should NOT include
	// the transaction created after beforeDeposit.
	historicalBalance, err := ledgerService.GetHistoricalBalance(
		ctx,
		accountID,
		beforeDeposit,
	)
	if err != nil {
		t.Fatalf("failed to get historical balance: %v", err)
	}

	if historicalBalance != initialBalance {
		t.Fatalf(
			"unexpected historical balance: got %d, want %d",
			historicalBalance,
			initialBalance,
		)
	}

	// Current balance SHOULD include the 100 deposit.
	currentBalance, err := ledgerService.GetCurrentBalance(
		ctx,
		accountID,
	)
	if err != nil {
		t.Fatalf("failed to get current balance: %v", err)
	}

	expectedCurrentBalance := initialBalance + 100

	if currentBalance != expectedCurrentBalance {
		t.Fatalf(
			"unexpected current balance: got %d, want %d",
			currentBalance,
			expectedCurrentBalance,
		)
	}

	t.Logf("historical balance: %d", historicalBalance)
	t.Logf("current balance: %d", currentBalance)
}
