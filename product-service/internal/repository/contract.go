package repository

import (
	"context"

	"github.com/alimikegami/point-of-sales/product-service/internal/domain"
	"github.com/alimikegami/point-of-sales/product-service/internal/dto"
	pkgdto "github.com/alimikegami/point-of-sales/product-service/pkg/dto"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoDBProductRepository interface {
	AddProduct(ctx context.Context, data domain.Product) (id primitive.ObjectID, err error)
	GetProducts(ctx context.Context, param pkgdto.Filter) (data []domain.Product, err error)
	HandleTrx(ctx context.Context, fn func(ctx mongo.SessionContext, repo MongoDBProductRepository) error) error
	GetProductByID(ctx context.Context, id string) (product domain.Product, err error)
	DeleteProduct(ctx context.Context, id string) (err error)
	UpdateProduct(ctx context.Context, data domain.Product) (err error)
	UpdateProductQuantity(ctx context.Context, data domain.Product) (err error)
	SetProductQuantity(ctx context.Context, data domain.Product) (err error)
}

type ElasticSearchProductRepository interface {
	AddProduct(ctx context.Context, index string, data dto.ProductResponse) (err error)
	GetProducts(ctx context.Context, filter pkgdto.Filter) ([]dto.ProductResponse, int, error)
	DecreaseProductQuantities(ctx context.Context, products []domain.Product) error
	AddProductQuantities(ctx context.Context, products []domain.Product) error
	DeleteProduct(ctx context.Context, id string) error
	UpdateProduct(ctx context.Context, data domain.Product) (err error)
}
