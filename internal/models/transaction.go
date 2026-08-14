package models

import "time"

type Transaction struct {
	ID             int64     `json:"id"`
	IdempotencyKey string    `json:"idempotency_key"`
	RequestHash    string    `json:"request_hash"`
	Description    string    `json:"description"`
	PostedAt       time.Time `json:"posted_at"`
	CreatedAt      time.Time `json:"created_at"`
}
