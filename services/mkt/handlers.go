package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var logger *zap.Logger

func initLogger() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}
}

func productsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		logger.Warn("invalid method for productsHandler", zap.String("method", r.Method))
		http.Error(w, "only GET method", http.StatusMethodNotAllowed)
		return
	}

	rows, err := db.Query("SELECT id, name, price FROM products")
	if err != nil {
		logger.Error("db query failed", zap.Error(err))
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var products []Product

	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price); err != nil {
			logger.Error("row scan failed", zap.Error(err))
			continue
		}
		products = append(products, p)
	}

	if len(products) == 0 {
		products = []Product{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(products); err != nil {
		logger.Error("failed to encode response", zap.Error(err))
	}
}

func orderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		logger.Warn("invalid method for orderHandler", zap.String("method", r.Method))
		http.Error(w, "only POST method", http.StatusMethodNotAllowed)
		return
	}

	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("invalid json", zap.Error(err))
		http.Error(w, "incorrect JSON", http.StatusBadRequest)
		return
	}

	if req.UserID == 0 || req.ProductID == 0 || req.Quantity == 0 {
		logger.Warn("missing required fields",
			zap.Int("user_id", req.UserID),
			zap.Int("product_id", req.ProductID),
			zap.Int("quantity", req.Quantity),
		)
		http.Error(w, "not enough data", http.StatusBadRequest)
		return
	}

	var productPrice string
	err := db.QueryRow("SELECT price FROM products WHERE id = $1", req.ProductID).Scan(&productPrice)
	if err == sql.ErrNoRows {
		logger.Warn("product not found", zap.Int("product_id", req.ProductID))
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}
	if err != nil {
		logger.Error("db error on product lookup", zap.Error(err))
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	price := parsePrice(productPrice)
	total := price * float64(req.Quantity)
	totalStr := fmt.Sprintf("%.2f", total)

	var orderID int
	err = db.QueryRow(`
        INSERT INTO orders (user_id, product_id, quantity, total, status)
        VALUES ($1,$2,$3,$4,$5) RETURNING id
    `, req.UserID, req.ProductID, req.Quantity, totalStr, "pending").Scan(&orderID)

	if err != nil {
		logger.Error("failed to create order",
			zap.Error(err),
			zap.Int("user_id", req.UserID),
			zap.Int("product_id", req.ProductID),
		)
		http.Error(w, "Ошибка БД", http.StatusInternalServerError)
		return
	}

	logger.Info("order created",
		zap.Int("order_id", orderID),
		zap.String("total", totalStr),
	)

	paymentStatus := callPaymentService(req.UserID, 999, total)

	_, err = db.Exec("UPDATE orders SET status=$1 WHERE id=$2", paymentStatus, orderID)
	if err != nil {
		logger.Error("failed to update order status",
			zap.Error(err),
			zap.Int("order_id", orderID),
			zap.String("status", paymentStatus),
		)
	}

	resp := CreateOrderResponse{
		OrderID: orderID,
		Status:  paymentStatus,
		Total:   totalStr,
	}

	status := http.StatusCreated
	if paymentStatus != "paid" {
		status = http.StatusBadRequest
	}

	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Error("failed to encode response", zap.Error(err))
	}
}

func callPaymentService(fromUserID int, toUserID int, amount float64) string {
	url := "http://pay-service:8082/api/pay/p2p"

	payload := map[string]interface{}{
		"from_user_id":    fromUserID,
		"to_user_id":      toUserID,
		"amount":          fmt.Sprintf("%.2f", amount),
		"idempotency_key": uuid.New().String(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		logger.Error("failed to marshal payment payload", zap.Error(err))
		return "failed"
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		logger.Error("payment request failed",
			zap.Error(err),
			zap.Int("from_user_id", fromUserID),
			zap.Float64("amount", amount),
		)
		return "failed"
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var r map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
			logger.Error("failed to decode payment response", zap.Error(err))
			return "failed"
		}

		if s, ok := r["status"].(string); ok && s == "completed" {
			logger.Info("payment successful",
				zap.Int("from_user_id", fromUserID),
				zap.Float64("amount", amount),
			)
			return "paid"
		}
	}

	logger.Warn("payment failed or unexpected response",
		zap.Int("status_code", resp.StatusCode),
	)

	return "failed"
}

func parsePrice(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		logger.Warn("failed to parse price", zap.String("value", s))
		return 0
	}
	return f
}
