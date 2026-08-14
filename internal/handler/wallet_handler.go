package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"ledger-wallet/internal/service"
)

type WalletHandler struct {
	walletService *service.WalletService
}

func NewWalletHandler(
	walletService *service.WalletService,
) *WalletHandler {
	return &WalletHandler{
		walletService: walletService,
	}
}

type createWalletRequest struct {
	UserID   int64  `json:"user_id"`
	Currency string `json:"currency"`
}

func (h *WalletHandler) CreateWallet(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req createWalletRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	wallet, err := h.walletService.CreateWallet(
		r.Context(),
		req.UserID,
		req.Currency,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(w).Encode(wallet)
}

// POST /wallets/{id}/deposit
func (h *WalletHandler) Deposit(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	walletID, err := walletIDFromPath(
		r.URL.Path,
		"/wallets/",
		"/deposit",
	)
	if err != nil {
		http.Error(w, "invalid wallet id", http.StatusBadRequest)
		return
	}

	var req depositRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	transactionID, err := h.walletService.Deposit(
		r.Context(),
		service.DepositRequest{
			WalletID:       walletID,
			Amount:         req.Amount,
			IdempotencyKey: req.IdempotencyKey,
		},
	)
	if err != nil {
		if errors.Is(err, service.ErrIdempotencyConflict) {
			http.Error(
				w,
				err.Error(),
				http.StatusConflict,
			)
			return
		}

		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"transaction_id": transactionID,
		"wallet_id":      walletID,
		"amount":         req.Amount,
	})
}

type depositRequest struct {
	Amount         int64  `json:"amount"`
	IdempotencyKey string `json:"idempotency_key"`
}

// POST /wallets/{id}/withdraw
func (h *WalletHandler) Withdraw(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	walletID, err := walletIDFromPath(
		r.URL.Path,
		"/wallets/",
		"/withdraw",
	)
	if err != nil {
		http.Error(w, "invalid wallet id", http.StatusBadRequest)
		return
	}

	var req withdrawRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	transactionID, err := h.walletService.Withdraw(
		r.Context(),
		service.WithdrawRequest{
			WalletID:       walletID,
			Amount:         req.Amount,
			IdempotencyKey: req.IdempotencyKey,
		},
	)
	if err != nil {
		if errors.Is(err, service.ErrIdempotencyConflict) {
			http.Error(
				w,
				err.Error(),
				http.StatusConflict,
			)
			return
		}

		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"transaction_id": transactionID,
		"wallet_id":      walletID,
		"amount":         req.Amount,
	})
}

type withdrawRequest struct {
	Amount         int64  `json:"amount"`
	IdempotencyKey string `json:"idempotency_key"`
}

// GET /wallets/balance?wallet_id=1
func (h *WalletHandler) GetBalance(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	walletIDStr := r.URL.Query().Get("wallet_id")

	if walletIDStr == "" {
		http.Error(w, "wallet_id is required", http.StatusBadRequest)
		return
	}

	walletID, err := strconv.ParseInt(walletIDStr, 10, 64)
	if err != nil || walletID <= 0 {
		http.Error(w, "invalid wallet_id", http.StatusBadRequest)
		return
	}

	balance, err := h.walletService.GetBalance(
		r.Context(),
		walletID,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"wallet_id": walletID,
		"balance":   balance,
	})
}

func walletIDFromPath(
	path string,
	prefix string,
	suffix string,
) (int64, error) {
	if !strings.HasPrefix(path, prefix) ||
		!strings.HasSuffix(path, suffix) {
		return 0, strconv.ErrSyntax
	}

	idPart := strings.TrimPrefix(path, prefix)
	idPart = strings.TrimSuffix(idPart, suffix)
	idPart = strings.Trim(idPart, "/")

	return strconv.ParseInt(idPart, 10, 64)
}

type transferRequest struct {
	FromWalletID   int64  `json:"from_wallet_id"`
	ToWalletID     int64  `json:"to_wallet_id"`
	Amount         int64  `json:"amount"`
	IdempotencyKey string `json:"idempotency_key"`
}

// POST /wallets/transfer
func (h *WalletHandler) Transfer(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	var req transferRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	transactionID, err := h.walletService.Transfer(
		r.Context(),
		service.TransferRequest{
			FromWalletID:   req.FromWalletID,
			ToWalletID:     req.ToWalletID,
			Amount:         req.Amount,
			IdempotencyKey: req.IdempotencyKey,
		},
	)

	if err != nil {
		if errors.Is(err, service.ErrIdempotencyConflict) {
			http.Error(
				w,
				err.Error(),
				http.StatusConflict,
			)
			return
		}

		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"transaction_id": transactionID,
		"amount":         req.Amount,
	})
}
