package main

import (
	"github.com/shopspring/decimal"
	"time"
)

type Wallet struct {
	ID       int             `json:"id"`
	UserId   int             `json:"user_id"`
	Balance  decimal.Decimal `json:"balance"`
	Currency string          `json:"currency"`
}

type CreateWalletRequest struct {
	UserID   int    `json:"user_id"`
	Currency string `json:"currency"`
}

type WalletBalanceCache struct {
	WalletId int             `json:"wallet_id"`
	Balance  decimal.Decimal `json:"balance"`
	Currency string          `json:"currency"`
	CachedAt time.Time       `json:"cachedAt"`
}

type TransactionRequest struct {
	WalletID int             `json:"wallet_id"`
	Amount   decimal.Decimal `json:"amount"`
	Type     string          `json:"type"`
	Reason   string          `json:"reason,optional"`
}

type Transaction struct {
	ID        int             `json:"id"`
	WalletID  int             `json:"wallet_id"`
	Amount    decimal.Decimal `json:"amount"`
	Type      string          `json:"type"`
	Reason    string          `json:"reason"`
	CreatedAt time.Time       `json:"created_at"`
}
