package service

import (
	"context"

	pkgdto "github.com/alimikegami/point-of-sales/product-query-service/pkg/dto"
)

type ProductService interface {
	GetProducts(ctx context.Context, filter pkgdto.Filter) (responsePayload pkgdto.PaginationResponse, err error)
	ConsumeEvent()
}
