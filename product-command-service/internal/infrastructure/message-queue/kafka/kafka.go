package kafka

import (
	"context"
	"time"

	"github.com/alimikegami/point-of-sales/product-command-service/config"
	"github.com/segmentio/kafka-go"
)

var (
	KafkaConn   *kafka.Conn
	KafkaReader *kafka.Reader
)

func CreateKafkaReader(config *config.Config) *kafka.Reader {
	KafkaReader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:          []string{config.KafkaConfig.BrokerAddress},
		Topic:            config.KafkaConfig.BrokerTopic,
		Partition:        config.KafkaConfig.BrokerPartition,
		MinBytes:         1e3, // 1KB
		MaxBytes:         1e6, // 1MB
		MaxWait:          100 * time.Millisecond,
		ReadLagInterval:  -1,
		StartOffset:      kafka.LastOffset,
		GroupID:          "product-command-service",
		QueueCapacity:    1000,
		ReadBatchTimeout: 10 * time.Millisecond,
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
