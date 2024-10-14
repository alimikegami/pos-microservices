package service

import (
	"context"

	"github.com/alimikegami/point-of-sales/order-service/internal/dto"
	pkgdto "github.com/alimikegami/point-of-sales/order-service/pkg/dto"
)

type OrderService interface {
	AddOrder(ctx context.Context, req dto.OrderRequest) (err error)
	MidtransPaymentWebhook(ctx context.Context, req dto.PaymentNotification) (err error)
	GetOrders(ctx context.Context, filter pkgdto.Filter) (response pkgdto.Pagination, err error)
	RestoreExpiredPaymentItemStocks()
}
