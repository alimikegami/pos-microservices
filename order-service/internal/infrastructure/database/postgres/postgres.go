package postgres

import (
	"fmt"
	"sync"

	"github.com/XSAM/otelsql"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

var lock = &sync.Mutex{}
var db *sqlx.DB

func GetDBInstance(user, password, host, port, dbName string) (*sqlx.DB, error) {
	if db == nil {
		lock.Lock()
		defer lock.Unlock()

		sqlDB, err := otelsql.Open("postgres",
			fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
				host, port, user, password, dbName),
			otelsql.WithAttributes(
				semconv.DBSystemPostgreSQL,
				semconv.DBNameKey.String(dbName),
			),
			otelsql.WithSpanOptions(otelsql.SpanOptions{
				DisableQuery: true,
			}),
		)
		if err != nil {
			return nil, err
		}

		db = sqlx.NewDb(sqlDB, "postgres")
		if err := db.Ping(); err != nil {
			return nil, err
		}
	} else {
		log.Info().Str("component", "GetDBInstance").Msg("instance is already created")
	}

	return db, nil
}
