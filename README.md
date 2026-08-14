# Ledger Wallet Service

A backend service written in Go that implements a double-entry ledger and a wallet system.

This project records all wallet activity as immutable ledger entries. Wallet balances are not stored as a mutable field on the wallet row; instead balances are computed from ledger entries so the ledger remains the single source of truth.

---

## Key Concepts

- Wallets do not store a separate `balance` field. Balances are derived from ledger entries.
- Each wallet is associated with a ledger `LIABILITY` account. A wallet represents funds the platform owes to users.
- All money movements are recorded as double-entry transactions: total debits must equal total credits.

### Wallet Balance

Because a wallet maps to a `LIABILITY` account, its balance is computed as:

```
Wallet Balance = Total Credit - Total Debit
```

This means credits increase the wallet balance and debits decrease it.

---

## 1. Account Types

The ledger supports the following account types:

- `ASSET`
- `LIABILITY`
- `EQUITY`
- `REVENUE`
- `EXPENSE`

Balance rules:

### ASSET / EXPENSE

```
Balance = Debit - Credit
```

### LIABILITY / EQUITY / REVENUE

```
Balance = Credit - Debit
```

---

## 2. Deposit (Example)

User deposits 1,000₮ into their wallet.

Ledger entries:

| Account | Type | Debit | Credit |
|---|---:|---:|---:|
| Platform Asset Account | ASSET | 1,000 | 0 |
| User Wallet | LIABILITY | 0 | 1,000 |

Why:

- The platform receives cash → platform `ASSET` increases (DEBIT).
- The platform now owes the user that money → user `LIABILITY` increases (CREDIT).

---

## 3. Withdrawal (Example)

User withdraws 500₮ from their wallet.

Ledger entries:

| Account | Type | Debit | Credit |
|---|---:|---:|---:|
| User Wallet | LIABILITY | 500 | 0 |
| Platform Asset Account | ASSET | 0 | 500 |

Why:

- The platform reduces its liability to the user (LIABILITY ↓) → DEBIT the wallet.
- The platform's cash asset decreases (ASSET ↓) → CREDIT the platform asset account.

---

## 4. Transfer (Wallet-to-Wallet)

Example: Wallet A → Wallet B, Amount = 300₮

Ledger entries:

| Account | Type | Debit | Credit |
|---|---:|---:|---:|
| Wallet A | LIABILITY | 300 | 0 |
| Wallet B | LIABILITY | 0 | 300 |

Why:

- Wallet A decreases (LIABILITY ↓) → DEBIT.
- Wallet B increases (LIABILITY ↑) → CREDIT.

---

## 5. Debit / Credit Summary

Every operation is recorded so that:

```
Total Debit == Total Credit
```

Summary table:

| Operation | Debit | Credit |
|---|---:|---:|
| Deposit | Platform Asset | User Wallet |
| Withdrawal | User Wallet | Platform Asset |
| Transfer | Source Wallet | Destination Wallet |

---

## 6. Idempotency

To protect against processing the same request multiple times, requests use an `IdempotencyKey`.

- If the same idempotency key and request payload are seen again, the system returns the existing transaction and does not create a new one.
- If the same key is submitted with a different payload (for example a different amount), the service returns an `ErrIdempotencyConflict` error.

---

## 7. Concurrency Safety

To prevent race conditions (for example, concurrent withdrawals consuming the same funds), wallet operations lock the wallet row using PostgreSQL `FOR UPDATE` within a database transaction:

1. Lock the wallet row.
2. Compute the current balance from ledger entries.
3. Verify sufficient funds.
4. Create the ledger transaction and entries.
5. Commit the database transaction.

This ensures that concurrent operations on the same wallet are serialized and prevents double-spending.

### Transfer Deadlock Prevention

When locking two wallets for a transfer, the service always locks wallets in ascending `id` order to reduce the risk of deadlocks.

---

## 8. Transaction Atomicity

Wallet state changes and ledger postings occur inside the same PostgreSQL transaction:

```
BEGIN
  Lock wallet
  Check balance
  Create transaction
  Create debit entry
  Create credit entry
COMMIT
```

If any step fails, the transaction is rolled back so partial state can't be persisted.

---

## 9. Immutable Ledger

Ledger entries are immutable. The system does not update historical entries to "apply" balances; instead, balances are recomputed from immutable entries.

---

## Project Structure

```text
ledger-wallet/
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── database/
│   │   └── database.go
│   ├── handler/
│   │   └── health_handler.go
│   ├── models/
│   ├── repository/
│   └── service/
├── migrations/
├── docker-compose.yml
├── go.mod
└── README.md
```

---

## Technology Stack

- Go
- PostgreSQL
- pgx / pgxpool
- Docker
- REST API

---

## Running the Project

Start PostgreSQL (via Docker Compose):

```bash
docker compose up -d
```

Install dependencies:

```bash
go mod tidy
```

Run tests:

```bash
go test ./...
```

Run the API server:

```bash
go run ./cmd/api
```

The API listens on `http://localhost:8080` by default.

---

If you'd like, I can also:

- Commit this updated `README.md` to the repository.
- Add a short diagram or Mermaid flow for the typical Deposit/Withdrawal flow.
- Add an example curl sequence for deposits and withdrawals.
