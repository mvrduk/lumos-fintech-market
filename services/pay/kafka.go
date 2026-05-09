package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

var kafkaWriter *kafka.Writer

func InitKafka() {
	kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP("kafka:9092"),
		Topic:    "payments",
		Balancer: &kafka.LeastBytes{},
	}
}

func PublishEvent(event interface{}) error {

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	err = kafkaWriter.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(time.Now().String()),
			Value: data,
		},
	)

	if err != nil {
		logger.Warn("kafka publish failed", zap.Error(err))
		return err
	}

	return nil
}
