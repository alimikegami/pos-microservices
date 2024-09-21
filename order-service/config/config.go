package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	PostgreSQLConfig PostgreSQLConfig
	JWTSecret        string
	MidtransConfig   MidtransConfig
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
		MidtransConfig: MidtransConfig{
			ServerKey: os.Getenv("MIDTRANS_SERVER_KEY"),
		},
	}

	return &conf
}
