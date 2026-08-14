package models

import "time"

type EntryDirection string

const (
	EntryDirectionDebit  EntryDirection = "DEBIT"
	EntryDirectionCredit EntryDirection = "CREDIT"
)

type Entry struct {
	ID            int64          `json:"id"`
	TransactionID int64          `json:"transaction_id"`
	AccountID     int64          `json:"account_id"`
	Direction     EntryDirection `json:"direction"`
	Amount        int64          `json:"amount"`
	CreatedAt     time.Time      `json:"created_at"`
}
