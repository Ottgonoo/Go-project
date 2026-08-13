package service

import (
	"testing"

	"ledger-wallet/internal/models"
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
					Amount:    10000,
				},
				{
					AccountID: 2,
					Direction: models.EntryDirectionCredit,
					Amount:    10000,
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
