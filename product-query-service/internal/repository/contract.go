package repository

import (
	"context"

	"github.com/alimikegami/point-of-sales/product-query-service/internal/domain"
	"github.com/alimikegami/point-of-sales/product-query-service/internal/dto"
	pkgdto "github.com/alimikegami/point-of-sales/product-query-service/pkg/dto"
)

type ElasticSearchProductRepository interface {
	AddProduct(ctx context.Context, index string, data dto.ProductResponse) (err error)
	GetProducts(ctx context.Context, filter pkgdto.Filter) ([]dto.ProductResponse, int, error)
	DecreaseProductQuantities(ctx context.Context, products []domain.Product) error
	AddProductQuantities(ctx context.Context, products []domain.Product) error
	DeleteProduct(ctx context.Context, id string) error
	UpdateProduct(ctx context.Context, data domain.Product) (err error)
}
