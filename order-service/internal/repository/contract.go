package repository

import (
	"context"

	"github.com/alimikegami/point-of-sales/order-service/internal/domain"
)

type OrderRepository interface {
	HandleTrx(ctx context.Context, fn func(repo OrderRepository) error) error

	AddOrder(ctx context.Context, data domain.Order) (id int64, err error)
	AddOrderDetails(ctx context.Context, data []domain.OrderDetail) (err error)
	GetOrderByTransactionNumber(ctx context.Context, transactionNumber string) (data domain.Order, err error)
	UpdateOrderPaymentStatus(ctx context.Context, data domain.Order) (err error)
}
