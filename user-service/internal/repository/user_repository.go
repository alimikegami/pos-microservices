package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/alimikegami/pos-microservices/user-service/internal/domain"
	pkgdto "github.com/alimikegami/pos-microservices/user-service/pkg/dto"
	"github.com/alimikegami/pos-microservices/user-service/pkg/errs"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

type UserRepository interface {
	GetUserByEmail(ctx context.Context, email string) (res domain.User, err error)
	AddUser(ctx context.Context, data domain.User) (id int64, err error)
	GetUserByID(ctx context.Context, id int64) (data domain.User, err error)
	UpdateUser(ctx context.Context, data domain.User) (err error)
	GetUsers(ctx context.Context, filter pkgdto.Filter) (data []domain.User, err error)
	CountUsers(ctx context.Context, filter pkgdto.Filter) (count int64, err error)
}

type UserRepositoryImpl struct {
	db *sqlx.DB
}

func CreateNewRepository(db *sqlx.DB) UserRepository {
	return &UserRepositoryImpl{db: db}
}

func (r *UserRepositoryImpl) GetUserByEmail(ctx context.Context, email string) (res domain.User, err error) {
	row := r.db.QueryRowxContext(ctx, "SELECT * FROM users WHERE email = $1 AND deleted_at IS NULL", email)
	err = row.StructScan(&res)
	if err != nil {
		log.Error().Err(err).Str("component", "GetUserByEmail").Msg("")
		if err == sql.ErrNoRows {
			return res, nil
		}
		return res, errs.ErrInternalServer
	}

	return
}

func (r *UserRepositoryImpl) AddUser(ctx context.Context, data domain.User) (id int64, err error) {
	tx := r.db.MustBegin()
	timestamp := time.Now().UnixMilli()
	data.CreatedAt = timestamp
	data.UpdatedAt = timestamp

	nstmt, err := tx.PrepareNamedContext(ctx, "INSERT INTO users(name, email, external_id, hashed_password, role_id, created_at, updated_at) VALUES (:name, :email, :external_id, :hashed_password, :role_id, :created_at, :updated_at) returning id")
	if err != nil {
		log.Error().Err(err).Str("component", "AddUser").Msg("")
		return
	}

	err = nstmt.GetContext(ctx, &data.ID, data)
	if err != nil {
		log.Error().Err(err).Str("component", "AddUser").Msg("")
		return
	}

	userHist := domain.UserHistory{
		Name:           data.Name,
		Email:          data.Email,
		HashedPassword: data.HashedPassword,
		UserID:         data.ID,
		ExternalID:     data.ExternalID,
		RoleID:         data.RoleID,
		CreatedAt:      timestamp,
		UpdatedAt:      timestamp,
	}

	_, err = tx.NamedExecContext(ctx, "INSERT INTO user_histories(name, email, external_id, hashed_password, role_id, user_id, created_at, updated_at) VALUES (:name, :email, :external_id, :hashed_password, :role_id, :user_id, :created_at, :updated_at)", userHist)
	if err != nil {
		log.Error().Err(err).Str("component", "AddUser").Msg("")
		return
	}

	err = tx.Commit()

	return
}

func (r *UserRepositoryImpl) GetUserByID(ctx context.Context, id int64) (data domain.User, err error) {
	row := r.db.QueryRowxContext(ctx, "SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL", id)
	err = row.StructScan(&data)
	if err != nil {
		log.Error().Err(err).Str("component", "GetUserByEmail").Msg("")
		if err == sql.ErrNoRows {
			return data, nil
		}
		return data, errs.ErrInternalServer
	}

	return
}

func (r *UserRepositoryImpl) UpdateUser(ctx context.Context, data domain.User) (err error) {
	tx := r.db.MustBegin()
	timestamp := time.Now().UnixMilli()
	data.CreatedAt = timestamp
	data.UpdatedAt = timestamp

	_, err = tx.NamedExecContext(ctx, "UPDATE users SET name=:name, hashed_password=:hashed_password WHERE id=:id AND deleted_at IS NULL", data)
	if err != nil {
		log.Error().Err(err).Str("component", "UpdateUser").Msg("")
		return
	}

	userHist := domain.UserHistory{
		Name:           data.Name,
		Email:          data.Email,
		HashedPassword: data.HashedPassword,
		UserID:         data.ID,
		ExternalID:     data.ExternalID,
		RoleID:         data.RoleID,
		CreatedAt:      timestamp,
		UpdatedAt:      timestamp,
	}

	_, err = tx.NamedExecContext(ctx, "INSERT INTO user_histories(name, email, external_id, hashed_password, role_id, user_id, created_at, updated_at) VALUES (:name, :email, :external_id, :hashed_password, :role_id, :user_id, :created_at, :updated_at)", userHist)
	if err != nil {
		log.Error().Err(err).Str("component", "UpdateUser").Msg("")
		return
	}

	err = tx.Commit()

	return
}

func (r *UserRepositoryImpl) GetUsers(ctx context.Context, filter pkgdto.Filter) (data []domain.User, err error) {
	query := "SELECT * FROM users WHERE deleted_at IS NULL"

	args := make(map[string]interface{})

	if filter.Limit != 0 && filter.Page != 0 {
		offset := (filter.Page - 1) * filter.Limit
		query += " LIMIT :limit OFFSET :offset"
		args["limit"] = filter.Limit
		args["offset"] = offset
	}

	nstmt, err := r.db.PrepareNamedContext(ctx, query)
	if err != nil {
		log.Error().Err(err).Str("component", "GetUsers").Msg("")
		return nil, err
	}

	err = nstmt.SelectContext(ctx, &data, args)
	if err != nil {
		log.Error().Err(err).Str("component", "GetUsers").Msg("")
		return nil, err
	}

	return data, nil
}

func (r *UserRepositoryImpl) CountUsers(ctx context.Context, filter pkgdto.Filter) (count int64, err error) {
	err = r.db.GetContext(ctx, &count, "SELECT COUNT(id) FROM users WHERE deleted_at IS NULL")
	if err != nil {
		log.Error().Err(err).Str("component", "CountUsers").Msg("")
		return 0, err
	}

	return
}
