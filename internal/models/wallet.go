package models

import "time"

type Wallet struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	AccountID int64     `json:"account_id"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
}
