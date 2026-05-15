package main

import (
	"context"
	"encoding/json"
	"github.com/ClickHouse/clickhouse-go/v2" // ClickHouse драйвер
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/prometheus/client_golang/prometheus" // Prometheus библиотека
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"log"
	"net/http"
	"time"
)

var (
	ch     driver.Conn
	logger *zap.Logger
)

const version = "1"
const Version = "1.0.0"

var (
	paymentProcessed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "payments_processed_total",
		Help: "Total payments processed",
	})

	paymentsAmount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "payments_amount_total",
		Help: "Total payments amount",
	})
)

func init() {
	prometheus.MustRegister(paymentProcessed)
	prometheus.MustRegister(paymentsAmount)
}

func main() {
	var err error

	logger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}

	logger.Info("Analytics service version", zap.String("version", Version))

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{"clickhouse:9000"},
		Auth: clickhouse.Auth{
			Database: "lumia_analytics",
			Username: "default",
			Password: "",
		},
	})
	if err != nil {
		logger.Fatal("eror to connect to clickhouse:", zap.Error(err))
	}

	ch = conn
	defer ch.Close()

	if err := ch.Ping(context.Background()); err != nil {
		logger.Fatal("Clickhouse doesnt respond", zap.Error(err))
	}

	logger.Info("Connected to ClickHouse")

	go startKafkaConsumer()

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {

		w.Write([]byte(`{"status":"ok", "service": "anx"}`))
	})

	logger.Info("Analytics service running on :9091")
	if err := http.ListenAndServe(":9091", nil); err != nil {
		logger.Fatal("Server cant run", zap.Error(err))
	}
}

func startKafkaConsumer() {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{"kafka:9092"},
		Topic:       "payment_completed",
		GroupID:     "anx",
		StartOffset: kafka.LastOffset,
	})

	defer reader.Close()

	logger.Info("Kafka consumer is running")

	for {
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			logger.Error("error reading kafka", zap.Error(err))
			time.Sleep(1 * time.Second)
			continue
		}

		var payment map[string]interface{}
		if err := json.Unmarshal(msg.Value, &payment); err != nil {
			log.Println("Error parsing JSON:", err)
			continue
		}

		insertPaymentEvent(payment)

		paymentProcessed.Inc()

		if amount, ok := payment["amount"].(float64); ok {
			paymentsAmount.Add(amount)
		}

		logger.Info("Event processed:" + payment["payment_id"].(string))
	}
}

func insertPaymentEvent(payment map[string]interface{}) {
	paymentID := payment["payment_id"].(string)
	fromUserID := int64(payment["from_user_id"].(float64))
	toUserID := int64(payment["to_user_id"].(float64))
	amount := payment["amount"].(float64)
	status := payment["status"].(string)
	timestamp := time.Now()

	batch, err := ch.PrepareBatch(context.Background(), "INSERT INTO payments_analytics")
	if err != nil {
		logger.Error("batch error", zap.Error(err))
		return
	}

	err = batch.Append(
		paymentID,
		fromUserID,
		toUserID,
		amount,
		status,
		timestamp,
	)
	if err != nil {
		logger.Error("append error", zap.Error(err))
		return
	}

	err = batch.Send()
	if err != nil {
		logger.Error("send error", zap.Error(err))
	}
}
