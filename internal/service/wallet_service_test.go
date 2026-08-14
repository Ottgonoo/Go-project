package service

import "testing"

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
