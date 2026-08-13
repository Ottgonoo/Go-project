package models

import "time"

type AccountType string

const (
	AccountTypeAsset    AccountType = "ASSET"
	AccountTypeLiability AccountType = "LIABILITY"
	AccountTypeEquity   AccountType = "EQUITY"
	AccountTypeRevenue  AccountType = "REVENUE"
	AccountTypeExpense  AccountType = "EXPENSE"
)

type Account struct {
	ID        int64       `json:"id"`
	Name      string      `json:"name"`
	Type      AccountType `json:"type"`
	Currency  string      `json:"currency"`
	CreatedAt time.Time   `json:"created_at"`
}