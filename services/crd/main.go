package main

import (
	"database/sql"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

var (
	db     *sql.DB
	logger *zap.Logger
)

const version = "1.0.0"

func main() {

	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		fmt.Printf("Cant initialize zap logger: %v\n:", err)
		os.Exit(1)
	}

	logger.Info("CRD service version", zap.String("version", Version))

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://lumia:lumia_password@localhost:5432/lumia?sslmode=disable"
	}
	db, err = sql.Open("postgres", dbURL)

	if err != nil {
		logger.Fatal("cant open postgres", zap.Error(err))
	}

	defer db.Close()

	if err = db.Ping(); err != nil {
		logger.Fatal("Database ping failed", zap.Error(err))
	}

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/crd/add", addCardHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8086"
	}

	logger.Info("Card service started on :" + port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		logger.Fatal("Server failed", zap.Error(err))
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
