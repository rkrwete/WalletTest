package repository

import (
	"fmt"
	"errors"
	"context"

	"wallet_test/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func (r *Postgres) Deposit(ctx context.Context, id uuid.UUID, amount int64) (int64, error) {
	var balance int64

	const query = `
		INSERT INTO wallets (id, balance) 
		VALUES ($1, $2)
		ON CONFLICT (id) 
		DO UPDATE SET balance = wallets.balance + EXCLUDED.balance, 
					  updated_at = now()
		RETURNING balance
	`

	err := r.pool.QueryRow(ctx, query, id, amount).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("deposit: %w", err)
	}
	
	return balance, nil
}

func (r *Postgres) Withdraw(ctx context.Context, id uuid.UUID, amount int64) (int64, error) {
	var balance int64

	const query = `
		UPDATE wallets
		SET balance = balance - $2,
			updated_at = now()
		WHERE id = $1
		AND balance >= $2
		RETURNING balance
	`

	err := r.pool.QueryRow(ctx, query, id, amount).Scan(&balance)
	if err == nil {
		return balance, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("withdraw: %w", err)
	}

	if _, err := r.GetBalance(ctx, id); err != nil {
		return 0, err
	}
	return 0, domain.ErrInsufficientFunds
}

func (r *Postgres) GetBalance(ctx context.Context, id uuid.UUID) (int64, error) {
	var balance int64
	const query = `SELECT balance FROM wallets WHERE id = $1`

	err := r.pool.QueryRow(ctx, query, id).Scan(&balance)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, domain.ErrWalletNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("get balance: %w", err)
	}

	return balance, nil
}