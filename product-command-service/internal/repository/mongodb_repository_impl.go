package repository

import (
	"context"
	"fmt"

	"github.com/alimikegami/point-of-sales/product-command-service/internal/domain"
	pkgdto "github.com/alimikegami/point-of-sales/product-command-service/pkg/dto"
	"github.com/alimikegami/point-of-sales/product-command-service/pkg/errs"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBProductRepositoryImpl struct {
	db *mongo.Database
}

func CreateNewMongoDBRepository(db *mongo.Database) MongoDBProductRepository {
	return &MongoDBProductRepositoryImpl{db: db}
}

func (r *MongoDBProductRepositoryImpl) AddProduct(ctx context.Context, data domain.Product) (id primitive.ObjectID, err error) {
	result, err := r.db.Collection("products").InsertOne(ctx, data)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("component", "AddProduct").Msg("")
		return
	}

	return result.InsertedID.(primitive.ObjectID), err
}

func (r *MongoDBProductRepositoryImpl) GetProducts(ctx context.Context, param pkgdto.Filter) (data []domain.Product, err error) {
	var opts *options.FindOptions

	if param.Limit != 0 && param.Page != 0 {
		opts = options.Find().SetSkip((int64(param.Page) - 1) * int64(param.Limit))
	}

	cursor, err := r.db.Collection("products").Find(ctx, nil, opts)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("component", "GetProducts").Msg("")
		return
	}

	if err = cursor.All(ctx, &data); err != nil {
		log.Ctx(ctx).Error().Err(err).Str("component", "GetProducts").Msg("")
		return
	}

	return data, nil
}

func (r *MongoDBProductRepositoryImpl) GetProductByID(ctx context.Context, id string) (product domain.Product, err error) {
	productID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("component", "GetProductByID").Msg("")
		return
	}

	filter := bson.D{{Key: "_id", Value: productID}}
	opts := options.FindOne()

	err = r.db.Collection("products").FindOne(ctx, filter, opts).Decode(&product)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("component", "GetProductByID").Msg("")
		if err == mongo.ErrNoDocuments {
			return product, errs.ErrNotFound
		}

		return product, err
	}
	return product, nil
}

func (r *MongoDBProductRepositoryImpl) HandleTrx(ctx context.Context, fn func(ctx mongo.SessionContext) error) error {
	fmt.Println("starting session mongodb")
	session, err := r.db.Client().StartSession()
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("component", "HandleTrx").Msg("")
		panic(err)
	}

	// Defers ending the session after the transaction is committed or ended
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(ctx mongo.SessionContext) (interface{}, error) {
		err := fn(ctx)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Str("component", "HandleTrx").Msg("")
		}
		return nil, err
	})

	return err
}

func (r *MongoDBProductRepositoryImpl) DeleteProduct(ctx context.Context, id string) (err error) {
	productID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("component", "DeleteProduct").Msg("")
		return
	}

	filter := bson.D{{Key: "_id", Value: productID}}

	_, err = r.db.Collection("products").DeleteOne(ctx, filter)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("component", "DeleteProduct").Msg("")
		return
	}

	return
}

func (r *MongoDBProductRepositoryImpl) UpdateProduct(ctx context.Context, data domain.Product) (err error) {
	filter := bson.D{{Key: "_id", Value: data.ID}}

	update := bson.D{{Key: "$set", Value: bson.D{{Key: "name", Value: data.Name}, {Key: "description", Value: data.Description}}}}

	result, err := r.db.Collection("products").UpdateOne(ctx, filter, update)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("component", "UpdateProduct").Msg("Failed to update product")
		return
	}

	if result.MatchedCount == 0 {
		log.Ctx(ctx).Error().Err(err).Str("component", "UpdateProduct").Msg("Failed to update product")
		return errs.ErrNotFound
	}

	return nil
}

func (r *MongoDBProductRepositoryImpl) UpdateProductQuantity(ctx context.Context, data domain.Product) (err error) {
	filter := bson.D{{Key: "_id", Value: data.ID}}

	update := bson.D{{Key: "$inc", Value: bson.D{{Key: "quantity", Value: data.Quantity}}}}

	result, err := r.db.Collection("products").UpdateOne(ctx, filter, update)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("component", "SetProductQuantity").Msg("Failed to update product")
		return
	}

	if result.MatchedCount == 0 {
		log.Ctx(ctx).Error().Err(err).Str("component", "SetProductQuantity").Msg("Failed to update product")
		return errs.ErrNotFound
	}

	return
}

func (r *MongoDBProductRepositoryImpl) SetProductQuantity(ctx context.Context, data domain.Product) (err error) {
	filter := bson.D{{Key: "_id", Value: data.ID}}

	update := bson.D{{Key: "$set", Value: bson.D{{Key: "quantity", Value: data.Quantity}}}}

	result, err := r.db.Collection("products").UpdateOne(ctx, filter, update)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("component", "SetProductQuantity").Msg("Failed to update product")
		return
	}

	if result.MatchedCount == 0 {
		log.Ctx(ctx).Error().Err(err).Str("component", "SetProductQuantity").Msg("Failed to update product")
		return errs.ErrNotFound
	}

	return
}
