package posgres

import (
	"fmt"
	"sync"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

var lock = &sync.Mutex{}
var db *sqlx.DB

func GetDBInstance(user, password, host, port, dbName string) (*sqlx.DB, error) {
	var err error

	if db == nil {
		lock.Lock()
		defer lock.Unlock()

		db, err = sqlx.Connect("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbName))
		if err != nil {
			return db, err
		}
	} else {
		log.Info().Str("component", "GetDBInstance").Msg("instance is already created")
	}

	return db, nil
}
