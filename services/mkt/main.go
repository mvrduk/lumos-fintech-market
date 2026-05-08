package main

import (
	"database/sql"
	"net/http"
	"os"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

var db *sql.DB

const version = "1"

func main() {
	initLogger()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://lumia:lumia_password@localhost:5432/lumia?sslmode=disable"
	}

	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		logger.Fatal("db open error", zap.Error(err))
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		logger.Fatal("db ping error", zap.Error(err))
	}

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/mkt/products", productsHandler)
	http.HandleFunc("/api/mkt/order", orderHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8085"
	}

	logger.Info("Marketplace Service started", zap.String("port", port))
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		logger.Fatal("server error", zap.Error(err))
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
