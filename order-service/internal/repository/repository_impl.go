package repository

import (
	"context"

	"github.com/alimikegami/point-of-sales/order-service/internal/domain"
	"github.com/alimikegami/point-of-sales/order-service/pkg/errs"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type OrderRepositoryImpl struct {
	db *gorm.DB
}

func CreateOrderRepository(db *gorm.DB) OrderRepository {
	return &OrderRepositoryImpl{
		db: db,
	}
}

func (r *OrderRepositoryImpl) AddOrder(ctx context.Context, data domain.Order) (id int64, err error) {
	err = r.db.WithContext(ctx).Create(&data).Error

	if err != nil {
		log.Error().Err(err).Str("component", "AddOrder").Msg("")
		return 0, errs.ErrInternalServer
	}

	return data.ID, nil
}

func (r *OrderRepositoryImpl) AddOrderDetails(ctx context.Context, data []domain.OrderDetail) (err error) {
	err = r.db.WithContext(ctx).Create(&data).Error

	if err != nil {
		log.Error().Err(err).Str("component", "AddOrderDetails").Msg("")
		return errs.ErrInternalServer
	}

	return nil
}

func (r *OrderRepositoryImpl) GetOrderByTransactionNumber(ctx context.Context, transactionNumber string) (data domain.Order, err error) {
	err = r.db.WithContext(ctx).Where("transaction_number = ?", transactionNumber).First(&data).Error

	if err != nil {
		log.Error().Err(err).Str("component", "GetOrderByTransactionNumber").Msg("")
		return data, errs.ErrInternalServer
	}

	return data, nil
}

func (r *OrderRepositoryImpl) UpdateOrderPaymentStatus(ctx context.Context, data domain.Order) (err error) {
	err = r.db.WithContext(ctx).Model(&data).Updates(data).Error

	if err != nil {
		log.Error().Err(err).Str("component", "UpdateOrderPaymentStatus").Msg("")
		return errs.ErrInternalServer
	}

	return nil
}

func (r *OrderRepositoryImpl) HandleTrx(ctx context.Context, fn func(repo OrderRepository) error) error {

	tx := r.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return err
	}

	newRepo := &OrderRepositoryImpl{
		db: tx,
	}

	err := fn(newRepo)
	if err != nil {
		return err
	}

	err = tx.Commit().Error

	return err
}
