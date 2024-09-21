package repository

import (
	"context"
	"time"

	"github.com/alimikegami/e-commerce/user-service/internal/domain"
	"github.com/alimikegami/e-commerce/user-service/pkg/errs"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type UserRepository interface {
	GetUserByEmail(ctx context.Context, email string) (res domain.User, err error)
	AddUser(ctx context.Context, data domain.User) (id int64, err error)
	GetUserByID(ctx context.Context, id int64) (data domain.User, err error)
	UpdateUser(ctx context.Context, data domain.User) (err error)
}

type UserRepositoryImpl struct {
	db *gorm.DB
}

func CreateNewRepository(db *gorm.DB) UserRepository {
	return &UserRepositoryImpl{db: db}
}

func (r *UserRepositoryImpl) AddUser(ctx context.Context, data domain.User) (id int64, err error) {
	tx := r.db.Begin()
	timestamp := time.Now().UnixMilli()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return 0, errs.ErrInternalServer
	}

	data.CreatedAt = timestamp
	data.UpdatedAt = timestamp

	err = tx.WithContext(ctx).Create(&data).Error

	if err != nil {
		log.Error().Err(err).Str("component", "AddUser").Msg("")
		return 0, errs.ErrInternalServer
	}

	err = tx.WithContext(ctx).Create(&domain.UserHistory{
		Name:           data.Name,
		Email:          data.Email,
		HashedPassword: data.HashedPassword,
		UserID:         data.ID,
		ExternalID:     data.ExternalID,
		CreatedAt:      timestamp,
		UpdatedAt:      timestamp,
	}).Error

	if err != nil {
		log.Error().Err(err).Str("component", "AddUser").Msg("")
		return 0, errs.ErrInternalServer
	}

	err = tx.Commit().Error
	if err != nil {
		log.Error().Err(err).Str("component", "AddUser").Msg("")
		return 0, errs.ErrInternalServer
	}

	return data.ID, nil
}

func (r *UserRepositoryImpl) GetUserByEmail(ctx context.Context, email string) (res domain.User, err error) {
	err = r.db.WithContext(ctx).Where("deleted_at IS NULL").Where("email = ?", email).First(&res).Error

	if err != nil {
		log.Error().Err(err).Str("component", "GetUserByEmail").Msg("")
		if err == gorm.ErrRecordNotFound {
			return res, nil
		}
		return
	}

	return
}

func (r *UserRepositoryImpl) UpdateUser(ctx context.Context, data domain.User) (err error) {
	tx := r.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	timestamp := time.Now().UnixMilli()

	data.UpdatedAt = timestamp
	err = r.db.WithContext(ctx).Model(&data).Updates(data).Error
	if err != nil {
		log.Error().Err(err).Str("component", "UpdateUser").Msg("")
		return errs.ErrInternalServer
	}

	err = r.db.WithContext(ctx).Model(&domain.UserHistory{}).Where("user_id = ?", data.ID).Where("deleted_at IS NULL").UpdateColumn("deleted_at", timestamp).Error
	if err != nil {
		log.Error().Err(err).Str("component", "UpdateUser").Msg("")
		return errs.ErrInternalServer
	}

	err = r.db.WithContext(ctx).Create(&domain.UserHistory{
		Name:       data.Name,
		Email:      data.Email,
		UserID:     data.ID,
		ExternalID: data.ExternalID,
		CreatedAt:  timestamp,
		UpdatedAt:  timestamp,
	}).Error

	if err != nil {
		log.Error().Err(err).Str("component", "UpdateUser").Msg("")
		return errs.ErrInternalServer
	}

	err = tx.Commit().Error
	return
}

func (r *UserRepositoryImpl) GetUserByID(ctx context.Context, id int64) (data domain.User, err error) {
	err = r.db.WithContext(ctx).Where("deleted_at IS NULL").First(&data, id).Error

	if err != nil {
		log.Error().Err(err).Str("component", "GetUserByID").Msg("")
		if err == gorm.ErrRecordNotFound {
			return data, nil
		}
		return
	}

	return
}
