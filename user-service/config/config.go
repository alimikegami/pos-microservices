package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

type Config struct {
	Environment      string
	ServicePort      string
	PostgreSQLConfig PostgreSQLConfig
	JWTConfig        JWTConfig
	KafkaConfig      KafkaConfig
	TracingConfig    TracingConfig
}

func CreateNewConfig() *Config {
	possiblePaths := []string{
		".env",
		"../.env",
	}

	for _, path := range possiblePaths {
		if err := godotenv.Load(path); err == nil {
			break
		}
	}

	conf := Config{
		ServicePort: os.Getenv("SERVICE_PORT"),
		PostgreSQLConfig: PostgreSQLConfig{
			DBHost:     os.Getenv("DB_HOST"),
			DBName:     os.Getenv("DB_NAME"),
			DBPort:     os.Getenv("DB_PORT"),
			DBUsername: os.Getenv("DB_USERNAME"),
			DBPassword: os.Getenv("DB_PASSWORD"),
		},
		JWTConfig: JWTConfig{
			JWTSecret: os.Getenv("JWT_SECRET"),
			JWTKid:    os.Getenv("JWT_KID"),
		},
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
		log.Error().Err(err).Str("component", "CreateNewConfig").Msg("")
	}

	conf.KafkaConfig.BrokerPartition = brokerPartition

	return &conf
}
