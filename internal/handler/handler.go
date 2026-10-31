package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"encoding/json"

	"github.com/google/uuid"

	"wallet_test/internal/domain"
)

type Repository interface {
	Deposit(ctx context.Context, id uuid.UUID, amount int64) (int64, error)
	Withdraw(ctx context.Context, id uuid.UUID, amount int64) (int64, error)
	GetBalance(ctx context.Context, id uuid.UUID) (int64, error)
}

type operationRequest struct {
	WalletID      uuid.UUID            `json:"walletId"`
	OperationType domain.OperationType `json:"operationType"`
	Amount        int64                `json:"amount"`
}

type balanceResponse struct {
	WalletID uuid.UUID `json:"walletId"`
	Balance  int64     `json:"balance"`
}

type errorResponse struct {
    Error string `json:"error"`
}

type Handler struct {
	repo Repository
	log  *slog.Logger
}

func New(repo Repository, log *slog.Logger) *Handler {
	return &Handler{repo: repo, log: log}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/wallet", h.operation)
	mux.HandleFunc("GET /api/v1/wallets/{id}", h.balance)
	return mux
}

func (h *Handler) operation(w http.ResponseWriter, r *http.Request) {
	var request operationRequest

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(&request)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON body or walletId")
		return
	}
	if request.WalletID == uuid.Nil {
		h.writeError(w, http.StatusBadRequest, "walletId is required")
		return
	}
	if request.Amount <= 0 {
		h.writeError(w, http.StatusBadRequest, "amount must be positive")
		return
	}

	var balance int64

	switch request.OperationType {
		case domain.Deposit:
			balance, err = h.repo.Deposit(r.Context(), request.WalletID, request.Amount)
		case domain.Withdraw:
			balance, err = h.repo.Withdraw(r.Context(), request.WalletID, request.Amount)
		default:
			h.writeError(w, http.StatusBadRequest, "operationType must be DEPOSIT or WITHDRAW")
			return
	}

	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, balanceResponse{WalletID: request.WalletID, Balance: balance})
}

func (h *Handler) balance(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid wallet id")
		return
	}

	balance, err := h.repo.GetBalance(r.Context(), id)	
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, balanceResponse{WalletID: id, Balance: balance})
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
		case errors.Is(err, domain.ErrInvalidAmount), errors.Is(err, domain.ErrInvalidOperation):
			h.writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrWalletNotFound):
			h.writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, domain.ErrInsufficientFunds):
			h.writeError(w, http.StatusUnprocessableEntity, err.Error())
		default:
			h.log.Error("internal error", "err", err)
			h.writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func (h *Handler) writeError(w http.ResponseWriter, status int, msg string) {
	h.writeJSON(w, status, errorResponse{Error: msg})
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.log.Error("write response", "err", err)
	}
}