package kafka

import (
	"context"

	"github.com/alimikegami/e-commerce/user-service/config"
	"github.com/segmentio/kafka-go"
)

var KafkaConn *kafka.Conn

func CreateKafkaProducer(config *config.Config) *kafka.Conn {
	conn, err := kafka.DialLeader(context.Background(), "tcp", config.KafkaConfig.BrokerAddress, config.KafkaConfig.BrokerTopic, config.KafkaConfig.BrokerPartition)
	if err != nil {
		panic(err)
	}

	KafkaConn = conn
	return KafkaConn
}
