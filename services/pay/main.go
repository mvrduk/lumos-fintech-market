package main

import (
	"net/http"

	"go.uber.org/zap"
)

var logger *zap.Logger

const version = "1"

func main() {
	var err error

	logger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {

		}
	}(logger)

	if err := InitDB(); err != nil {
		logger.Fatal("failed to connect DB", zap.Error(err))
	}

	InitKafka()

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/pay/p2p", p2pTransferHandler)

	logger.Info("Payment Service started on :8082")

	if err := http.ListenAndServe(":8082", nil); err != nil {
		logger.Fatal("server failed", zap.Error(err))
	}
}
