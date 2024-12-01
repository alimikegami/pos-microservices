package posgres

import (
	"fmt"
	"sync"
	"time"

	"github.com/XSAM/otelsql"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"golang.org/x/crypto/bcrypt"
)

var lock = &sync.Mutex{}
var db *sqlx.DB

func GetDBInstance(user, password, host, port, dbName, environment string) (*sqlx.DB, error) {
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

		if environment == "test" {
			err = SeedRole(db)
			if err != nil {
				log.Error().Str("component", "GetDBInstance").Msg(err.Error())
			}

			err := SeedUser(db)
			if err != nil {
				log.Error().Str("component", "GetDBInstance").Msg(err.Error())
			}
		}
	} else {
		log.Info().Str("component", "GetDBInstance").Msg("instance is already created")
	}

	return db, nil
}

func SeedUser(db *sqlx.DB) error {
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte("testpassword"),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Prepare insert query
	query := `
	INSERT INTO users (name, email, hashed_password, role_id, external_id, created_at, updated_at) 
	VALUES ($1, $2, $3, $4, $5, $6, $7) 
	RETURNING id
`

	// Prepare and execute with returning the new user's ID
	var userID int
	err = db.QueryRow(
		query,
		"test",
		"testuser@example.com",
		string(hashedPassword),
		1, // Assuming role_id 1 is a default role
		"test",
		time.Now().Unix(),
		time.Now().Unix(),
	).Scan(&userID)

	if err != nil {
		return fmt.Errorf("failed to insert test user: %w", err)
	}

	return nil
}

func SeedRole(db *sqlx.DB) error {
	// Prepare insert query
	query := `
	INSERT INTO roles (id, name, created_at, updated_at) 
	VALUES ($1, $2, $3, $4) 
	RETURNING id
`

	// Prepare and execute with returning the new user's ID
	var roleID int
	err := db.QueryRow(
		query,
		1,
		"cashier",
		time.Now().Unix(),
		time.Now().Unix(),
	).Scan(&roleID)

	if err != nil {
		return fmt.Errorf("failed to insert role: %w", err)
	}

	return nil
}
