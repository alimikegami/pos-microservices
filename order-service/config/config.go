package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServicePort        string
	PostgreSQLConfig   PostgreSQLConfig
	JWTSecret          string
	MidtransConfig     MidtransConfig
	KafkaConfig        KafkaConfig
	ProductServiceHost string
}

func CreateNewConfig() *Config {
	godotenv.Load(".env")

	conf := Config{
		ServicePort: os.Getenv("SERVICE_PORT"),
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
		ProductServiceHost: os.Getenv("PRODUCT_SERVICE_HOST"),
	}

	return &conf
}
