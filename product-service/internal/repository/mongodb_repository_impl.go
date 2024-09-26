package repository

import (
	"context"
	"fmt"

	"github.com/alimikegami/point-of-sales/product-service/internal/domain"
	"github.com/alimikegami/point-of-sales/product-service/internal/dto"
	pkgdto "github.com/alimikegami/point-of-sales/product-service/pkg/dto"
	"github.com/alimikegami/point-of-sales/product-service/pkg/errs"
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
	productResult, err := r.db.Collection("products").InsertOne(ctx, data)
	if err != nil {
		log.Error().Err(err).Str("component", "AddProduct").Msg("")
		return
	}

	id = productResult.InsertedID.(primitive.ObjectID)
	return
}

func (r *MongoDBProductRepositoryImpl) GetProducts(ctx context.Context, param pkgdto.Filter) (data []domain.Product, err error) {
	findOptions := options.Find()
	findOptions.SetLimit(int64(param.Limit))
	findOptions.SetSkip(int64((param.Page - 1) * param.Limit))

	filter := bson.D{}

	cursor, err := r.db.Collection("products").Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve documents: %v", err)
	}

	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &data); err != nil {
		return nil, fmt.Errorf("failed to decode documents: %v", err)
	}

	return data, nil
}

func (r *MongoDBProductRepositoryImpl) GetProductByIDs(ctx context.Context, ids []string) (data []domain.Product, err error) {
	objectIDs := make([]primitive.ObjectID, len(ids))
	for i, id := range ids {
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return nil, fmt.Errorf("invalid ID format: %v", err)
		}
		objectIDs[i] = objectID
	}

	filter := bson.M{"_id": bson.M{"$in": objectIDs}}

	cursor, err := r.db.Collection("products").Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve documents: %v", err)
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &data); err != nil {
		return nil, fmt.Errorf("failed to decode documents: %v", err)
	}

	return data, nil
}

func (r *MongoDBProductRepositoryImpl) UpdateSellerDetails(ctx context.Context, data dto.User) (err error) {
	filter := bson.M{"user_id": data.ExternalID}

	update := bson.M{
		"$set": bson.M{"user_name": data.Name},
	}

	_, err = r.db.Collection("products").UpdateOne(ctx, filter, update)
	if err != nil {
		fmt.Println(err)
		return
	}

	return
}

func (r *MongoDBProductRepositoryImpl) GetProductByID(ctx context.Context, id string) (product domain.Product, err error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return product, fmt.Errorf("invalid ID format: %v", err)
	}

	filter := bson.M{"_id": objectID}

	err = r.db.Collection("products").FindOne(ctx, filter).Decode(&product)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return product, fmt.Errorf("product not found")
		}
		return product, fmt.Errorf("failed to retrieve product: %v", err)
	}

	return product, nil
}

func (r *MongoDBProductRepositoryImpl) HandleTrx(ctx context.Context, fn func(repo MongoDBProductRepository) error) error {
	session, err := r.db.Client().StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %v", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		newRepo := &MongoDBProductRepositoryImpl{
			db: r.db,
		}

		err := fn(newRepo)
		if err != nil {
			return nil, err
		}

		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %v", err)
	}

	return nil
}

func (r *MongoDBProductRepositoryImpl) ReduceProductQuantity(ctx context.Context, productID string, quantity int) error {
	objectID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		return fmt.Errorf("invalid product ID: %v", err)
	}

	filter := bson.M{"_id": objectID}

	update := bson.M{
		"$inc": bson.M{"quantity": -quantity},
	}

	result, err := r.db.Collection("products").UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update product quantity: %v", err)
	}

	if result.ModifiedCount == 0 {
		return fmt.Errorf("no product found with ID %s", productID)
	}

	var product struct {
		Quantity int `bson:"quantity"`
	}

	err = r.db.Collection("products").FindOne(ctx, filter).Decode(&product)
	if err != nil {
		return fmt.Errorf("failed to retrieve updated product: %v", err)
	}

	if product.Quantity < 0 {
		return fmt.Errorf("product quantity cannot be negative")
	}

	return nil
}

func (r *MongoDBProductRepositoryImpl) DeleteProduct(ctx context.Context, id string) (err error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid product ID: %v", err)
	}

	filter := bson.M{"_id": objectID}
	opts := options.Delete().SetHint(bson.D{{Key: "_id", Value: 1}})
	result, err := r.db.Collection("products").DeleteOne(context.TODO(), filter, opts)
	if err != nil {
		return fmt.Errorf("failed to delete product: %v", err)
	}

	if result.DeletedCount == 0 {
		return errs.ErrNotFound
	}

	return
}

func (r *MongoDBProductRepositoryImpl) UpdateProduct(ctx context.Context, data domain.Product) (err error) {
	filter := bson.D{{Key: "_id", Value: data.ID}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "name", Value: data.Name},
		{Key: "description", Value: data.Description},
		{Key: "price", Value: data.Price}}}}

	_, err = r.db.Collection("products").UpdateOne(ctx, filter, update)
	if err != nil {
		return
	}

	return
}

func (r *MongoDBProductRepositoryImpl) UpdateProductQuantity(ctx context.Context, data domain.Product) (err error) {
	filter := bson.D{{Key: "_id", Value: data.ID}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "quantity", Value: data.Quantity}}}}

	_, err = r.db.Collection("products").UpdateOne(ctx, filter, update)
	if err != nil {
		return
	}

	return
}
