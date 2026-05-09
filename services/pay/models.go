package main

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PaymentStatus string

const (
	StatusPending    PaymentStatus = "pending"
	StatusProcessing PaymentStatus = "processing"
	StatusCompleted  PaymentStatus = "completed"
	StatusFailed     PaymentStatus = "failed"
	StatusRolledBack PaymentStatus = "rolled_back"
)

type Payment struct {
	ID             uuid.UUID       `json:"id"`
	FromUserID     int             `json:"from_user_id"`
	ToUserID       int             `json:"to_user_id"`
	Amount         decimal.Decimal `json:"amount"`
	Status         PaymentStatus   `json:"status"`
	IdempotencyKey uuid.UUID       `json:"idempotency_key"`
	CreatedAt      time.Time       `json:"created_at"`
}

type P2PRequest struct {
	FromUserID     int             `json:"from_user_id"`
	ToUserID       int             `json:"to_user_id"`
	Amount         decimal.Decimal `json:"amount"`
	IdempotencyKey uuid.UUID       `json:"idempotency_key"`
}

type P2PResponse struct {
	PaymentID uuid.UUID     `json:"payment_id"`
	Status    PaymentStatus `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
}

type PaymentInitiatedEvent struct {
	EventType  string    `json:"event_type"`
	PaymentID  string    `json:"payment_id"`
	FromUserID int       `json:"from_user_id"`
	ToUserID   int       `json:"to_user_id"`
	Amount     string    `json:"amount"`
	Timestamp  time.Time `json:"timestamp"`
}

type PaymentCompletedEvent struct {
	EventType string    `json:"event_type"`
	PaymentID string    `json:"payment_id"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}
