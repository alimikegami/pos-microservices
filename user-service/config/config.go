package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	PostgreSQLConfig PostgreSQLConfig
	JWTSecret        string
	KafkaConfig      KafkaConfig
}

func CreateNewConfig() *Config {
	godotenv.Load(".env")

	conf := Config{
		PostgreSQLConfig: PostgreSQLConfig{
			DBHost:     os.Getenv("DB_HOST"),
			DBName:     os.Getenv("DB_NAME"),
			DBPort:     os.Getenv("DB_PORT"),
			DBUsername: os.Getenv("DB_USERNAME"),
			DBPassword: os.Getenv("DB_PASSWORD"),
		},
		JWTSecret: os.Getenv("JWT_SECRET"),
		KafkaConfig: KafkaConfig{
			BrokerAddress: os.Getenv("BROKER_ADDRESS"),
			BrokerTopic:   os.Getenv("BROKER_TOPIC"),
		},
	}

	brokerPartition, err := strconv.Atoi(os.Getenv("BROKER_PARTITION"))
	if err != nil {
	}

	conf.KafkaConfig.BrokerPartition = brokerPartition

	return &conf
}
