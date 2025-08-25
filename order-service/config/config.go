package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServicePort               string
	MetricsPort               string
	PostgreSQLConfig          PostgreSQLConfig
	JWTSecret                 string
	MidtransConfig            MidtransConfig
	KafkaConfig               KafkaConfig
	ProductQueryServiceHost   string
	ProductCommandServiceHost string
	TracingConfig             TracingConfig
}

func CreateNewConfig() *Config {
	godotenv.Load(".env")

	conf := Config{
		ServicePort: os.Getenv("SERVICE_PORT"),
		MetricsPort: os.Getenv("METRICS_PORT"),
		PostgreSQLConfig: PostgreSQLConfig{
			DBHost:     os.Getenv("DB_HOST"),
			DBName:     os.Getenv("DB_NAME"),
			DBPort:     os.Getenv("DB_PORT"),
			DBUsername: os.Getenv("DB_USERNAME"),
			DBPassword: os.Getenv("DB_PASSWORD"),
		},
		KafkaConfig: KafkaConfig{
			BrokerAddress: os.Getenv("BROKER_ADDRESS"),
			BrokerTopic:   os.Getenv("BROKER_TOPIC"),
		},
		JWTSecret: os.Getenv("JWT_SECRET"),
		MidtransConfig: MidtransConfig{
			ServerKey: os.Getenv("MIDTRANS_SERVER_KEY"),
		},
		ProductQueryServiceHost:   os.Getenv("PRODUCT_QUERY_SERVICE_HOST"),
		ProductCommandServiceHost: os.Getenv("PRODUCT_COMMAND_SERVICE_HOST"),
		TracingConfig: TracingConfig{
			CollectorHost: os.Getenv("COLLECTOR_HOST"),
		},
	}

	return &conf
}
