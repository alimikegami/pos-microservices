package kafka

import (
	"context"

	"github.com/alimikegami/point-of-sales/product-service/config"
	"github.com/segmentio/kafka-go"
)

var (
	KafkaConn   *kafka.Conn
	KafkaReader *kafka.Reader
)

func CreateKafkaReader(config *config.Config) *kafka.Reader {
	KafkaReader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{config.KafkaConfig.BrokerAddress},
		Topic:       config.KafkaConfig.BrokerTopic,
		Partition:   config.KafkaConfig.BrokerPartition,
		MinBytes:    10e3, // 10KB
		MaxBytes:    10e6, // 10MB
		StartOffset: kafka.LastOffset,
		GroupID:     "product-service",
	})

	return KafkaReader
}

func CreateKafkaProducer(config *config.Config) *kafka.Conn {
	conn, err := kafka.DialLeader(context.Background(), "tcp", config.KafkaConfig.BrokerAddress, config.KafkaConfig.BrokerTopic, config.KafkaConfig.BrokerPartition)
	if err != nil {
		panic(err)
	}

	KafkaConn = conn
	return KafkaConn
}
