package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"ledger-wallet/internal/database"
	"ledger-wallet/internal/handler"
	"ledger-wallet/internal/repository"
	"ledger-wallet/internal/service"
)

func main() {
	conn, err := database.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	fmt.Println("Database connected successfully!")

	// Repositories
	accountRepository := repository.NewAccountRepository()
	transactionRepository := repository.NewTransactionRepository()
	entryRepository := repository.NewEntryRepository()
	walletRepository := repository.NewWalletRepository()

	// Services
	ledgerService := service.NewLedgerService(
		conn,
		accountRepository,
		transactionRepository,
		entryRepository,
	)

	walletService := service.NewWalletService(
		conn,
		ledgerService,
		walletRepository,
		accountRepository,
	)

	// Handlers
	healthHandler := handler.NewHealthHandler()
	walletHandler := handler.NewWalletHandler(walletService)

	// Router
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc(
		"/health",
		healthHandler.Health,
	)

	// Create wallet
	// POST /wallets
	mux.HandleFunc(
		"/wallets",
		walletHandler.CreateWallet,
	)

	// Get wallet balance
	// GET /wallets/balance?wallet_id=1
	mux.HandleFunc(
		"/wallets/balance",
		walletHandler.GetBalance,
	)

	// Transfer between wallets
	// POST /wallets/transfer
	//
	// Body:
	// {
	//   "from_wallet_id": 1,
	//   "to_wallet_id": 5,
	//   "amount": 2000,
	//   "idempotency_key": "transfer-003"
	// }
	mux.HandleFunc(
		"/wallets/transfer",
		walletHandler.Transfer,
	)

	// Wallet-specific operations
	//
	// POST /wallets/1/deposit
	// POST /wallets/1/withdraw
	mux.HandleFunc(
		"/wallets/",
		func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.HasSuffix(r.URL.Path, "/deposit"):
				walletHandler.Deposit(w, r)

			case strings.HasSuffix(r.URL.Path, "/withdraw"):
				walletHandler.Withdraw(w, r)

			default:
				http.NotFound(w, r)
			}
		},
	)

	// HTTP server
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	fmt.Println("API server running on http://localhost:8080")

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
