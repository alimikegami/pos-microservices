package service

import (
	"context"

	"github.com/alimikegami/point-of-sales/order-service/internal/dto"
)

type OrderService interface {
	AddOrder(ctx context.Context, req dto.OrderRequest) (err error)
	MidtransPaymentWebhook(ctx context.Context, req dto.PaymentNotification) (err error)
}
