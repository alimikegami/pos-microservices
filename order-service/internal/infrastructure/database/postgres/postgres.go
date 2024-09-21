package postgres

import (
	"fmt"
	"sync"

	"github.com/alimikegami/point-of-sales/order-service/config"
	"github.com/alimikegami/point-of-sales/order-service/internal/domain"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var lock = &sync.Mutex{}
var db *gorm.DB

func GetDBInstance(config *config.Config) (*gorm.DB, error) {
	var err error

	if db == nil {
		lock.Lock()
		defer lock.Unlock()
		if db == nil {
			dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Singapore", config.PostgreSQLConfig.DBHost, config.PostgreSQLConfig.DBUsername, config.PostgreSQLConfig.DBPassword, config.PostgreSQLConfig.DBName, config.PostgreSQLConfig.DBPort)
			db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
			if err != nil {
				log.Error().Err(err).Str("component", "GetDBInstance").Msg("")
				return nil, err
			}
			MigrateDBSchema()
		} else {
			log.Info().Str("component", "GetDBInstance").Msg("single instance is created")
		}
	} else {
		log.Info().Str("component", "GetDBInstance").Msg("instance is already created")
	}

	return db, nil
}

func MigrateDBSchema() {
	db.AutoMigrate(&domain.PaymentMethod{}, &domain.Order{}, &domain.OrderDetail{})
}
