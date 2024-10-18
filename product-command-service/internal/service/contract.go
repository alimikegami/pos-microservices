package service

import (
	"context"

	"github.com/alimikegami/point-of-sales/product-command-service/internal/dto"
)

type ProductService interface {
	AddProduct(ctx context.Context, data dto.ProductRequest) (err error)
	ConsumeEvent()
	UpdateProductsQuantity(ctx context.Context, req dto.OrderRequest) (err error)
	DeleteProduct(ctx context.Context, id string) (err error)
	UpdateProduct(ctx context.Context, data dto.ProductRequest) (err error)
	UpdateProductQuantity(ctx context.Context, req dto.ProductQuantityRequest) (err error)
}
