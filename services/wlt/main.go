package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	_ "go.uber.org/zap"
)

var (
	db     *sql.DB
	logger *zap.Logger
)

const version = "1"

var (
	walletBalanceGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wallet_balance",
			Help: "Current wallet balance",
		},
		[]string{"wallet_id"},
	)
	walletTransactions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wallet_transactions_total",
			Help: "Total wallet transactions",
		},
		[]string{"type"},
	)
)

func init() {
	prometheus.MustRegister(walletBalanceGauge)
	prometheus.MustRegister(walletTransactions)
}

func main() {

	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		fmt.Printf("Cant initialize zap logger: %v\n:", err)
		os.Exit(1)
	}
	defer logger.Sync()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://lumia:lumia_password@localhost:5432/lumia?sslmode=disable"
	}

	var err2 error
	db, err2 = sql.Open("postgres", dbURL)
	if err2 != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err2))
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatal("Database ping failed", zap.Error(err))
	}

	logger.Info("Successfully connected to PostgreSQL")

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}
	err = initRedis(redisURL)
	if err != nil {
		logger.Fatal("Failed to init Redis", zap.Error(err))
	}

	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("/api/wlt/health", healthHandler)
	http.HandleFunc("/api/wlt/create", createWalletHandler)

	http.HandleFunc("/api/wlt/get", getWalletHandler)
	http.HandleFunc("api/wlt/balance", getWalletBalanceHandler)
	http.HandleFunc("api/wlt/debit", debitHandler)
	http.HandleFunc("api/wlt/credit", creditHandler)
	http.HandleFunc("api/wlt/history", getTransactionHistoryhandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}
	logger.Info("Wallet Service Started", zap.String("port", port))

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		logger.Fatal("Server failed", zap.Error(err))
	}
}
