package repository

import (
	"context"

	"github.com/alimikegami/point-of-sales/product-service/internal/domain"
	"github.com/alimikegami/point-of-sales/product-service/internal/dto"
	pkgdto "github.com/alimikegami/point-of-sales/product-service/pkg/dto"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MongoDBProductRepository interface {
	AddProduct(ctx context.Context, data domain.Product) (id primitive.ObjectID, err error)
	GetProducts(ctx context.Context, param pkgdto.Filter) (data []domain.Product, err error)
	UpdateSellerDetails(ctx context.Context, data dto.User) (err error)
	GetProductByIDs(ctx context.Context, ids []string) (data []domain.Product, err error)
	HandleTrx(ctx context.Context, fn func(repo MongoDBProductRepository) error) error
	ReduceProductQuantity(ctx context.Context, productID string, quantity int) error
	GetProductByID(ctx context.Context, id string) (product domain.Product, err error)
}

type ElasticSearchProductRepository interface {
	AddProduct(ctx context.Context, index string, data dto.ProductResponse) (err error)
	GetProducts(ctx context.Context, filter pkgdto.Filter) ([]dto.ProductResponse, int, error)
	UpdateProductQuantities(ctx context.Context, products []domain.Product) error
}
