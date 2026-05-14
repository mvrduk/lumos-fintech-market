package main

import (
	"encoding/json"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
	"time"
)

type WalletEvent struct {
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"`
	WalletID  int       `json:"wallet_id"`
	Amount    string    `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
}

var kafkaProducer *kafka.Producer

func initKafkaProducer(bootstrapServers string) error {
	if bootstrapServers == "" {
		bootstrapServers = "localhost:9092"
	}

	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":                     bootstrapServers,
		"acks":                                  "all",
		"retries":                               3,
		"max.in.flight.requests.per.connection": 5,
	})

	if err != nil {
		return fmt.Errorf("failed to create wlt:kafka producer: %v", err)
	}

	kafkaProducer = p
	logger.Info("Kafka producer initialized", zap.String("bootstrap.servers", bootstrapServers))
	return nil

}

func publishWalletEvent(eventType string, walletID int, amount string) error {
	event := &WalletEvent{
		EventID:   fmt.Sprintf("wallet-%d-%d", walletID, time.Now().UnixNano()),
		EventType: eventType,
		WalletID:  walletID,
		Amount:    amount,
		Timestamp: time.Now(),
	}

	eventData, err := json.Marshal(event)

	if err != nil {
		logger.Error("failed to marshal event", zap.Error(err))
		return err
	}

	topic := "wallet-events"
	partitionKey := fmt.Sprintf("%d", walletID)

	kafkaProducer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},

		Key:   []byte(partitionKey),
		Value: eventData,
	},
		nil)

	logger.Info("Event published",
		zap.String("event_type", eventType),
		zap.Int("wallet_id", walletID),
	)

	return nil
}

func closeKafkaProducer() {
	if kafkaProducer != nil {
		remaining := kafkaProducer.Flush(10000)
		if remaining > 0 {
			logger.Warn("failed to flush all messages", zap.Int("remaining", remaining))
		}

		kafkaProducer.Close()
		logger.Info("kafka prodcer closed")
	}
}
