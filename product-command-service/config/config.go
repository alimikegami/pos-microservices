package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ServicePort   string
	MongoDBConfig MongoDBConfig
	KafkaConfig   KafkaConfig
	JWTSecret     string
	TracingConfig TracingConfig
}

func CreateNewConfig() *Config {
	godotenv.Load(".env")

	conf := Config{
		ServicePort: os.Getenv("SERVICE_PORT"),
		MongoDBConfig: MongoDBConfig{
			DBHost: os.Getenv("DB_HOST"),
			DBPort: os.Getenv("DB_PORT"),
		},
		JWTSecret: os.Getenv("JWT_SECRET"),
		KafkaConfig: KafkaConfig{
			BrokerAddress: os.Getenv("BROKER_ADDRESS"),
			BrokerTopic:   os.Getenv("BROKER_TOPIC"),
		},
		TracingConfig: TracingConfig{
			CollectorHost: os.Getenv("COLLECTOR_HOST"),
		},
	}

	brokerPartition, err := strconv.Atoi(os.Getenv("BROKER_PARTITION"))
	if err != nil {
	}

	conf.KafkaConfig.BrokerPartition = brokerPartition

	return &conf
}
