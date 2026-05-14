package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func createWalletHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		logger.Warn("Method not allowed", zap.String("method", r.Method))
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateWalletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Failed to decode request body", zap.Error(err))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	logger.Info("Creating wallet",
		zap.Int("user_id", req.UserID),
		zap.String("currency", req.Currency),
	)

	wallet, err := CreateWallet(db, req.UserID, req.Currency)
	if err != nil {
		logger.Error("Failed to create wallet in DB",
			zap.Error(err),
			zap.Int("user_id", req.UserID),
		)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(wallet)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func getWalletHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		logger.Warn("Method not allowed", zap.String("method", r.Method), zap.String("path", r.RequestURI))
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	walletIDStr := r.URL.Query().Get("id")
	if walletIDStr == "" {
		logger.Warn("Missing wallet ID parameter")
		http.Error(w, "Missing Wallet ID parameter", http.StatusBadRequest)
		return
	}

	walletID, err := strconv.Atoi(walletIDStr)
	if err != nil {
		logger.Warn("Invalid wallet ID format", zap.String("id", walletIDStr))
		http.Error(w, "Invalid wallet ID format", http.StatusBadRequest)
		return
	}

	wallet, err := GetWallet(db, walletID)
	if err != nil {
		logger.Error("Failed to get Wallet",
			zap.Error(err),
			zap.Int("wallet_id", walletID))

		if err.Error() == "wallet not found" {
			http.Error(w, "Wallet not found", http.StatusNotFound)
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	logger.Info("Wallet retrieved successfully",
		zap.Int("wallet_id", wallet.ID),
		zap.Int("user_id", wallet.UserId),
		zap.String("balance", wallet.Balance.String()),
		zap.String("currency", wallet.Currency),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(wallet)
}

func getUserWalletsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		logger.Warn("Method not allowed", zap.String("method", r.Method))
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		logger.Warn("Missing user_id parameter")
		http.Error(w, "Missing user_id", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		logger.Warn("Invalid user ID format", zap.String("user_id", userIDStr))
		http.Error(w, "Invalid user ID format", http.StatusBadRequest)
		return
	}

	wallets, err := GetWalletsByUserID(db, userID)
	if err != nil {
		logger.Error("Failed to get user wallets",
			zap.Error(err),
			zap.Int("user_id", userID),
		)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if len(wallets) == 0 {
		logger.Info("No wallets found for user", zap.Int("user_id", userID))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
	}

	logger.Info("User wallets retrieved",
		zap.Int("user_id", userID),
		zap.Int("wallet_count", len(wallets)),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(wallets)
}

func getWalletBalanceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		logger.Warn("Method not allowed", zap.String("method", r.Method))
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	walletIDStr := r.URL.Query().Get("wallet_id")
	if walletIDStr == "" {
		logger.Warn("Missing wallet_id parameter")
		http.Error(w, "Missing wallet_id", http.StatusBadRequest)
		return
	}

	walletID, err := strconv.Atoi(walletIDStr)
	if err != nil {
		logger.Warn("Invalid wallet ID", zap.String("id", walletIDStr))
		http.Error(w, "Invalid wallet_id", http.StatusBadRequest)
		return
	}

	cachedBalance, err := getBalanceFromCache(walletID)
	if err == nil && cachedBalance != nil {
		logger.Info("Balance from cache", zap.Int("wallet_id", walletID))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cachedBalance)
		return
	}

	wallet, err := GetWallet(db, walletID)
	if err != nil {
		logger.Error("Failed to get wallet", zap.Error(err), zap.Int("wallet_id", walletID))
		if err.Error() == "wallet not found" {
			http.Error(w, "Wallet not found", http.StatusNotFound)
		} else {
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
		return
	}

	cacheData := &WalletBalanceCache{
		wallet.ID,
		wallet.Balance,
		wallet.Currency,
		time.Now(),
	}

	saveBalanceToCache(walletID, cacheData)

	logger.Info("Balance from database", zap.Int("wallet_id", walletID))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(wallet)
}

func debitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Invalid JSON", zap.Error(err))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Amount.LessThanOrEqual(decimal.Zero) {
		logger.Warn("Invalid amount", zap.String("amount", req.Amount.String()))
		http.Error(w, "Amount must be positive", http.StatusBadRequest)
		return
	}

	txn, err := DebitToWallet(db, req.WalletID, req.Amount, req.Reason)
	if err != nil {
		logger.Error("Deposit failed", zap.Error(err), zap.Int("wallet_id", req.WalletID))
		http.Error(w, "Deposit failed", http.StatusInternalServerError)
		return
	}

	invalidateWalletCache(req.WalletID)

	logger.Info("Deposit successful",
		zap.Int("wallet_id", req.WalletID),
		zap.String("amount", req.Amount.String()),
	)

	publishWalletEvent("wallet.debit", req.WalletID, req.Amount.String())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(txn)
}

func creditHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Invalid JSON", zap.Error(err))
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if req.Amount.LessThanOrEqual(decimal.Zero) {
		http.Error(w, "Amount must be positive", http.StatusBadRequest)
		return
	}

	txn, err := CreditFromWallet(db, req.WalletID, req.Amount, req.Reason)
	if err != nil {
		if err.Error() == "insufficient funds" {
			http.Error(w, "insufficient funds", http.StatusBadRequest)
		} else {
			logger.Error("Credit failed", zap.Error(err))
			http.Error(w, "Credit failed", http.StatusInternalServerError)
		}
		return
	}

	invalidateWalletCache(req.WalletID)

	logger.Info("Credit successful",
		zap.Int("wallet_id", req.WalletID),
		zap.String("amount", req.Amount.String()),
	)

	publishWalletEvent("wallet.credited", req.WalletID, req.Amount.String())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(txn)
}

func getTransactionHistoryhandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	walletIDStr := r.URL.Query().Get("wallet_id")
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	if walletIDStr == "" {
		http.Error(w, "Missing wallet_id", http.StatusBadRequest)
		return
	}

	walletID, err := strconv.Atoi(walletIDStr)
	if err != nil {
		http.Error(w, "invalid wallet id", http.StatusBadRequest)
		return
	}

	page := 1

	limit := 10

	if pageStr != "" {
		page, _ = strconv.Atoi(pageStr)
	}
	if limitStr != "" {
		limit, _ = strconv.Atoi(limitStr)
	}

	transactions, err := GetWalletTransactionHistory(db, walletID, page, limit)
	if err != nil {
		logger.Error("Failed to get history", zap.Error(err))
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	logger.Info("Transaction history retrieved",
		zap.Int("wallet_id", walletID),
		zap.Int("count", len(transactions)),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(transactions)
}
