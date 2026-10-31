package repository

import (
	"io"
	"sync"
	"errors"
	"context"
	"testing"
	"log/slog"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/jackc/pgx/v5/pgxpool"

	"wallet_test/internal/config"
	"wallet_test/internal/domain"
	"wallet_test/internal/migrate"
)

type testEnv struct {
	repo *Postgres
	pool *pgxpool.Pool
	t    *testing.T
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	_ = godotenv.Load("../../config.env")

	cfg := config.Load()
	cfg.DB.Host = "localhost"
	dsn := cfg.DB.DSN()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := migrate.Up(dsn, "../../migrations", log); err != nil {
		t.Fatalf("migrations: %v", err)
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	return &testEnv{repo: NewPostgres(pool), pool: pool, t: t}
}

func (e *testEnv) newWallet() uuid.UUID {
	id := uuid.New()
	e.t.Cleanup(func() {
		_, err := e.pool.Exec(context.Background(), `DELETE FROM wallets WHERE id = $1`, id)
		if err != nil {
			e.t.Errorf("the wallet could not be deleted. %s: %v", id, err)
		}
	})
	return id
}

func TestDepositAndWithdraw(t *testing.T) {
	env := newTestEnv(t)
	repo := env.repo
	ctx := context.Background()
	id := env.newWallet()

	balance, err := repo.Deposit(ctx, id, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if balance != 1000 {
		t.Fatalf("balance %d, expected 1000", balance)
	}

	balance, err = repo.Withdraw(ctx, id, 300)
	if err != nil {
		t.Fatal(err)
	}
	if balance != 700 {
		t.Fatalf("balance %d, expected 700", balance)
	}

	balance, err = repo.GetBalance(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if balance != 700 {
		t.Fatalf("GetBalance return %d, expected 700", balance)
	}
}

func TestWithdraw_InsufficientFunds(t *testing.T) {
	env := newTestEnv(t)
	repo := env.repo
	ctx := context.Background()
	id := env.newWallet()

	if _, err := repo.Deposit(ctx, id, 100); err != nil {
		t.Fatal(err)
	}

	_, err := repo.Withdraw(ctx, id, 500)
	if !errors.Is(err, domain.ErrInsufficientFunds) {
		t.Fatalf("error %v, expected ErrInsufficientFunds", err)
	}

	balance, err := repo.GetBalance(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if balance != 100 {
		t.Fatalf("balance %d, expected 100", balance)
	}
}

func TestWithdraw_WalletNotFound(t *testing.T) {
	env := newTestEnv(t)

	_, err := env.repo.Withdraw(context.Background(), uuid.New(), 10)
	if !errors.Is(err, domain.ErrWalletNotFound) {
		t.Fatalf("error %v, expected ErrWalletNotFound", err)
	}
}

func TestGetBalance_NotFound(t *testing.T) {
	env := newTestEnv(t)

	_, err := env.repo.GetBalance(context.Background(), uuid.New())
	if !errors.Is(err, domain.ErrWalletNotFound) {
		t.Fatalf("error %v, expected ErrWalletNotFound", err)
	}
}

func TestConcurrentDeposits(t *testing.T) {
	env := newTestEnv(t)
	repo := env.repo
	ctx := context.Background()
	id := env.newWallet()

	const n = 1000
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := repo.Deposit(ctx, id, 1); err != nil {
				t.Errorf("deposit: %v", err)
			}
		}()
	}
	wg.Wait() 

	balance, err := repo.GetBalance(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if balance != n {
		t.Fatalf("balance %d, expected %d (some of the deposits were lost)", balance, n)
	}
}