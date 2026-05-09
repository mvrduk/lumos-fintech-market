package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func checkIdempotency(key uuid.UUID) (*P2PResponse, error) {
	var (
		paymentID uuid.UUID
		status    PaymentStatus
		createdAt time.Time
	)

	err := DB.QueryRow(
		"SELECT id, status, created_at FROM payments WHERE idempotency_key = $1",
		key,
	).Scan(&paymentID, &status, &createdAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("idempotency check failed: %w", err)
	}

	return &P2PResponse{
		PaymentID: paymentID,
		Status:    status,
		CreatedAt: createdAt,
	}, nil
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "pay",
	})
}

func p2pTransferHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "only POST requests", http.StatusMethodNotAllowed)
		return
	}

	var req P2PRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	existing, err := checkIdempotency(req.IdempotencyKey)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if existing != nil {
		writeJSON(w, existing)
		return
	}

	if err := validatePayment(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var (
		paymentID uuid.UUID
		createdAt time.Time
	)

	err = DB.QueryRow(
		`INSERT INTO payments (from_user_id, to_user_id, amount, status, idempotency_key)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at`,
		req.FromUserID,
		req.ToUserID,
		req.Amount,
		StatusPending,
		req.IdempotencyKey,
	).Scan(&paymentID, &createdAt)

	if err != nil {
		logger.Error("failed to create payment", zap.Error(err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := PublishEvent(PaymentInitiatedEvent{
		EventType:  "payment_initiated",
		PaymentID:  paymentID.String(),
		FromUserID: req.FromUserID,
		ToUserID:   req.ToUserID,
		Amount:     req.Amount.String(),
		Timestamp:  time.Now(),
	}); err != nil {
		logger.Warn("failed to publish initiated event", zap.Error(err))
	}

	_, _ = DB.Exec(
		`UPDATE payments SET status = $1 WHERE id = $2`,
		StatusProcessing,
		paymentID,
	)

	payment := Payment{
		ID:         paymentID,
		FromUserID: req.FromUserID,
		ToUserID:   req.ToUserID,
		Amount:     req.Amount,
		Status:     StatusProcessing,
	}

	err = executePaymentSaga(payment)

	var finalStatus PaymentStatus

	if err != nil {
		finalStatus = StatusRolledBack

		logger.Warn("saga failed", zap.Error(err), zap.String("payment_id", paymentID.String()))
	} else {
		finalStatus = StatusCompleted
	}

	_, _ = DB.Exec(
		`UPDATE payments SET status = $1 WHERE id = $2`,
		finalStatus,
		paymentID,
	)

	if err := PublishEvent(PaymentCompletedEvent{
		EventType: "payment_completed",
		PaymentID: paymentID.String(),
		Status:    string(finalStatus),
		Timestamp: time.Now(),
	}); err != nil {
		logger.Warn("failed to publish completed event", zap.Error(err))
	}

	writeJSON(w, P2PResponse{
		PaymentID: paymentID,
		Status:    finalStatus,
		CreatedAt: createdAt,
	})
}
func writeJSON(w http.ResponseWriter, resp interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func validatePayment(req P2PRequest) error {
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return errors.New("amount must be > 0")
	}
	if req.FromUserID == req.ToUserID {
		return errors.New("cannot send money to yourself")
	}

	balance, err := getWalletBalance(req.FromUserID)
	if err != nil {
		return fmt.Errorf("failed to check sender balance: %w", err)
	}
	if balance.LessThan(req.Amount) {
		return fmt.Errorf("insufficient funds: have %s, need %s", balance.String(), req.Amount.String())
	}

	return nil
}

func getWalletBalance(userID int) (decimal.Decimal, error) {
	walletURL := os.Getenv("WALLET_SERVICE_URL")
	if walletURL == "" {
		walletURL = "http://wlt-service:8084"
	}

	client := http.Client{Timeout: 3 * time.Second}

	resp, err := client.Get(fmt.Sprintf("%s/api/wlt/wallets?user_id=%d", walletURL, userID))
	if err != nil {
		return decimal.Zero, fmt.Errorf("wallet service unreachable: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode == http.StatusNotFound {
		return decimal.Zero, fmt.Errorf("no wallet found for user %d", userID)
	}
	if resp.StatusCode != http.StatusOK {
		return decimal.Zero, fmt.Errorf("wallet service returned %d", resp.StatusCode)
	}

	var wallets []struct {
		Balance decimal.Decimal `json:"balance"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&wallets); err != nil {
		return decimal.Zero, fmt.Errorf("failed to parse wallet response: %w", err)
	}

	if len(wallets) == 0 {
		return decimal.Zero, fmt.Errorf("user %d has no wallets", userID)
	}

	return wallets[0].Balance, nil
}

func getPaymentStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "only GET requests", http.StatusMethodNotAllowed)
		return
	}

	pathParts := strings.Split(r.URL.Path, "/")

	if len(pathParts) != 4 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	paymentIDStr := pathParts[3]

	paymentID, err := uuid.Parse(paymentIDStr)
	if err != nil {
		http.Error(w, "invalid payment_id", http.StatusBadRequest)
		return
	}

	var payment Payment

	err = DB.QueryRow(
		`SELECT id, from_user_id, to_user_id, amount, status, created_at
		 FROM payments WHERE id = $1`,
		paymentID,
	).Scan(
		&payment.ID,
		&payment.FromUserID,
		&payment.ToUserID,
		&payment.Amount,
		&payment.Status,
		&payment.CreatedAt,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "payment not found", http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, payment)
}
