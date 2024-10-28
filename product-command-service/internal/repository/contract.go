package repository

import (
	"context"

	"github.com/alimikegami/point-of-sales/product-command-service/internal/domain"
	pkgdto "github.com/alimikegami/point-of-sales/product-command-service/pkg/dto"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoDBProductRepository interface {
	AddProduct(ctx context.Context, data domain.Product) (id primitive.ObjectID, err error)
	GetProducts(ctx context.Context, param pkgdto.Filter) (data []domain.Product, err error)
	HandleTrx(ctx context.Context, fn func(ctx mongo.SessionContext) error) error
	GetProductByID(ctx context.Context, id string) (product domain.Product, err error)
	DeleteProduct(ctx context.Context, id string) (err error)
	UpdateProduct(ctx context.Context, data domain.Product) (err error)
	UpdateProductQuantity(ctx context.Context, data domain.Product) (err error)
	SetProductQuantity(ctx context.Context, data domain.Product) (err error)
}
