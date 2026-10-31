package handler

import (
	"io"
	"errors"
	"strings"
	"testing"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"

	"github.com/google/uuid"

	"wallet_test/internal/domain"
)

const testUUID = "11111111-1111-1111-1111-111111111111"

type fakeRepo struct {
	balance int64
	err     error
}

func (f *fakeRepo) Deposit(ctx context.Context, id uuid.UUID, amount int64) (int64, error) {
	return f.balance, f.err
}

func (f *fakeRepo) Withdraw(ctx context.Context, id uuid.UUID, amount int64) (int64, error) {
	return f.balance, f.err
}

func (f *fakeRepo) GetBalance(ctx context.Context, id uuid.UUID) (int64, error) {
	return f.balance, f.err
}

func send(repo *fakeRepo, method, url, body string) *httptest.ResponseRecorder {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := New(repo, log).Routes()

	req := httptest.NewRequest(method, url, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func checkStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status %d, expected %d, body: %s", rec.Code, want, rec.Body.String())
	}
}

func TestDeposit_OK(t *testing.T) {
	repo := &fakeRepo{balance: 1000}
	body := `{"walletId":"` + testUUID + `","operationType":"DEPOSIT","amount":1000}`

	rec := send(repo, "POST", "/api/v1/wallet", body)

	checkStatus(t, rec, http.StatusOK)
	if !strings.Contains(rec.Body.String(), `"balance":1000`) {
		t.Fatalf("the answer does not include a balance of 1000: %s", rec.Body.String())
	}
}

func TestWithdraw_InsufficientFunds(t *testing.T) {
	repo := &fakeRepo{err: domain.ErrInsufficientFunds}
	body := `{"walletId":"` + testUUID + `","operationType":"WITHDRAW","amount":900}`

	rec := send(repo, "POST", "/api/v1/wallet", body)

	checkStatus(t, rec, http.StatusUnprocessableEntity) 
}

func TestWithdraw_WalletNotFound(t *testing.T) {
	repo := &fakeRepo{err: domain.ErrWalletNotFound}
	body := `{"walletId":"` + testUUID + `","operationType":"WITHDRAW","amount":10}`

	rec := send(repo, "POST", "/api/v1/wallet", body)

	checkStatus(t, rec, http.StatusNotFound) 
}

func TestOperation_DBError(t *testing.T) {
	repo := &fakeRepo{err: errors.New("boom")}
	body := `{"walletId":"` + testUUID + `","operationType":"DEPOSIT","amount":10}`

	rec := send(repo, "POST", "/api/v1/wallet", body)

	checkStatus(t, rec, http.StatusInternalServerError) 
}

func TestOperation_BadAmount(t *testing.T) {
	repo := &fakeRepo{}

	for _, amount := range []string{"0", "-10"} {
		body := `{"walletId":"` + testUUID + `","operationType":"DEPOSIT","amount":` + amount + `}`
		rec := send(repo, "POST", "/api/v1/wallet", body)
		checkStatus(t, rec, http.StatusBadRequest) 
	}
}

func TestOperation_BadOperationType(t *testing.T) {
	body := `{"walletId":"` + testUUID + `","operationType":"TRANSFER","amount":10}`

	rec := send(&fakeRepo{}, "POST", "/api/v1/wallet", body)

	checkStatus(t, rec, http.StatusBadRequest)
}

func TestOperation_BadWalletID(t *testing.T) {
	body := `{"walletId":"123","operationType":"DEPOSIT","amount":10}`

	rec := send(&fakeRepo{}, "POST", "/api/v1/wallet", body)

	checkStatus(t, rec, http.StatusBadRequest)
}

func TestOperation_BrokenJSON(t *testing.T) {
	rec := send(&fakeRepo{}, "POST", "/api/v1/wallet", `{"walletId":`)

	checkStatus(t, rec, http.StatusBadRequest)
}

func TestGetBalance_OK(t *testing.T) {
	repo := &fakeRepo{balance: 500}

	rec := send(repo, "GET", "/api/v1/wallets/"+testUUID, "")

	checkStatus(t, rec, http.StatusOK)
	if !strings.Contains(rec.Body.String(), `"balance":500`) {
		t.Fatalf("в ответе нет баланса 500: %s", rec.Body.String())
	}
}

func TestGetBalance_NotFound(t *testing.T) {
	repo := &fakeRepo{err: domain.ErrWalletNotFound}

	rec := send(repo, "GET", "/api/v1/wallets/"+testUUID, "")

	checkStatus(t, rec, http.StatusNotFound)
}

func TestGetBalance_BadID(t *testing.T) {
	rec := send(&fakeRepo{}, "GET", "/api/v1/wallets/abc", "")

	checkStatus(t, rec, http.StatusBadRequest)
}