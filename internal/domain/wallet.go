package domain

import (
	"errors"
)

type OperationType string

const (
	Deposit OperationType = "DEPOSIT"
	Withdraw OperationType = "WITHDRAW"
)

func (ot OperationType) Valid() bool {
	return ot == Deposit || ot == Withdraw
}

var (
	ErrWalletNotFound = errors.New("wallet not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrInvalidAmount = errors.New("amount must be positive")
	ErrInvalidOperation = errors.New("invalid operation type")
)