package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alimikegami/point-of-sales/product-service/internal/domain"
	"github.com/alimikegami/point-of-sales/product-service/internal/dto"
	pkgdto "github.com/alimikegami/point-of-sales/product-service/pkg/dto"
	"github.com/elastic/go-elasticsearch"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ProductRepositoryImpl struct {
	db            *mongo.Database
	elasticsearch *elasticsearch.Client
}

func CreateNewRepository(db *mongo.Database, elasticsearch *elasticsearch.Client) ProductRepository {
	return &ProductRepositoryImpl{db: db, elasticsearch: elasticsearch}
}

func (r *ProductRepositoryImpl) AddProduct(ctx context.Context, data domain.Product) (id primitive.ObjectID, err error) {
	productResult, err := r.db.Collection("products").InsertOne(ctx, data)
	if err != nil {
		log.Error().Err(err).Str("component", "AddProduct").Msg("")
		return
	}

	id = productResult.InsertedID.(primitive.ObjectID)
	return
}

func (r *ProductRepositoryImpl) GetProducts(ctx context.Context, param pkgdto.Filter) (data []domain.Product, err error) {
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

func (r *ProductRepositoryImpl) GetProductByIDs(ctx context.Context, ids []string) (data []domain.Product, err error) {
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

func (r *ProductRepositoryImpl) UpdateSellerDetails(ctx context.Context, data dto.User) (err error) {
	filter := bson.M{"user_id": data.ExternalID}

	update := bson.M{
		"$set": bson.M{"user_name": data.Name},
	}

	_, err = r.db.Collection("products").UpdateMany(ctx, filter, update)
	if err != nil {
		fmt.Println(err)
		return
	}

	return
}

func (r *ProductRepositoryImpl) AddProductToElasticsearch(ctx context.Context, index string, data dto.ProductResponse) (err error) {
	docBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshaling document: %w", err)
	}

	res, err := r.elasticsearch.Index(
		index,
		bytes.NewReader(docBytes),
		r.elasticsearch.Index.WithDocumentID(data.ID),
		r.elasticsearch.Index.WithContext(context.Background()),
	)
	if err != nil {
		return fmt.Errorf("error indexing document: %w", err)
	}
	defer res.Body.Close()

	// Check if the operation was successful
	if res.IsError() {
		return fmt.Errorf("error indexing document: %s", res.String())
	}

	log.Printf("Document indexed successfully with ID: %s", data.ID)

	return nil
}

func (r *ProductRepositoryImpl) GetProductsFromElasticsearch(ctx context.Context, filter pkgdto.Filter) ([]dto.ProductResponse, int, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{},
			},
		},
	}

	boolQuery := query["query"].(map[string]interface{})["bool"].(map[string]interface{})

	// Add text search if Q is provided
	if filter.Q != "" {
		boolQuery["must"] = append(boolQuery["must"].([]interface{}), map[string]interface{}{
			"match": map[string]interface{}{
				"name": filter.Q,
			},
		})
	}

	// Add product ID filtering if ProductIds are provided
	if len(filter.ProductIds) > 0 {
		boolQuery["filter"] = map[string]interface{}{
			"terms": map[string]interface{}{
				"_id": filter.ProductIds,
			},
		}
	}

	// If no conditions are added, use match_all
	if len(boolQuery["must"].([]interface{})) == 0 && boolQuery["filter"] == nil {
		query["query"] = map[string]interface{}{
			"match_all": map[string]interface{}{},
		}
	}

	// Add pagination
	if filter.Limit != 0 && filter.Page != 0 {
		from := (filter.Page - 1) * filter.Limit
		query["from"] = from
		query["size"] = filter.Limit
	}

	var buf strings.Builder
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, 0, fmt.Errorf("error encoding query: %w", err)
	}

	res, err := r.elasticsearch.Search(
		r.elasticsearch.Search.WithContext(ctx),
		r.elasticsearch.Search.WithIndex("products"),
		r.elasticsearch.Search.WithBody(strings.NewReader(buf.String())),
		r.elasticsearch.Search.WithTrackTotalHits(true),
		r.elasticsearch.Search.WithPretty(),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("error performing search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, 0, fmt.Errorf("error searching documents: %s", res.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("error parsing the response body: %w", err)
	}

	hits, found := result["hits"].(map[string]interface{})["hits"].([]interface{})
	if !found {
		return nil, 0, fmt.Errorf("hits not found in the response")
	}

	// Get total number of hits
	total := int(result["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64))

	var products []dto.ProductResponse
	for _, hit := range hits {
		source, found := hit.(map[string]interface{})["_source"]
		if !found {
			continue
		}
		productJSON, err := json.Marshal(source)
		if err != nil {
			return nil, 0, fmt.Errorf("error marshaling product: %w", err)
		}
		var product dto.ProductResponse
		if err := json.Unmarshal(productJSON, &product); err != nil {
			return nil, 0, fmt.Errorf("error unmarshaling product: %w", err)
		}
		products = append(products, product)
	}

	return products, total, nil
}

func (r *ProductRepositoryImpl) GetProductByID(ctx context.Context, id string) (product domain.Product, err error) {
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

func (r *ProductRepositoryImpl) HandleTrx(ctx context.Context, fn func(repo ProductRepository) error) error {
	// Start a session
	session, err := r.db.Client().StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %v", err)
	}
	defer session.EndSession(ctx)

	// Start a transaction
	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		// Create a new repository instance with the transaction session
		newRepo := &ProductRepositoryImpl{
			db: r.db,
		}

		// Execute the provided function
		err := fn(newRepo)
		if err != nil {
			return nil, err // This will cause the transaction to be aborted
		}

		return nil, nil // This will cause the transaction to be committed
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %v", err)
	}

	return nil
}

func (r *ProductRepositoryImpl) ReduceProductQuantity(ctx context.Context, productID string, quantity int) error {
	// Convert string ID to ObjectID
	objectID, err := primitive.ObjectIDFromHex(productID)
	if err != nil {
		return fmt.Errorf("invalid product ID: %v", err)
	}

	// Define the filter to find the product
	filter := bson.M{"_id": objectID}

	// Define the update to reduce the quantity
	update := bson.M{
		"$inc": bson.M{"quantity": -quantity},
	}

	// Perform the update
	result, err := r.db.Collection("products").UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update product quantity: %v", err)
	}

	// Check if a document was actually modified
	if result.ModifiedCount == 0 {
		return fmt.Errorf("no product found with ID %s", productID)
	}

	// Optionally, you might want to check if the quantity didn't go below zero
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

func (r *ProductRepositoryImpl) UpdateProductQuantities(ctx context.Context, products []domain.Product) error {
	var buf bytes.Buffer

	for _, product := range products {
		// Create the update action
		action := map[string]interface{}{
			"update": map[string]interface{}{
				"_index": "products",
				"_id":    product.ID.Hex(),
			},
		}

		// Create the update script
		script := map[string]interface{}{
			"script": map[string]interface{}{
				"source": "ctx._source.quantity = Math.max(0, ctx._source.quantity - params.decreaseBy)",
				"lang":   "painless",
				"params": map[string]interface{}{
					"decreaseBy": product.Quantity,
				},
			},
		}

		// Encode the action and script
		if err := json.NewEncoder(&buf).Encode(action); err != nil {
			return fmt.Errorf("error encoding action: %w", err)
		}
		if err := json.NewEncoder(&buf).Encode(script); err != nil {
			return fmt.Errorf("error encoding script: %w", err)
		}
	}

	// Perform the bulk update
	res, err := r.elasticsearch.Bulk(bytes.NewReader(buf.Bytes()),
		r.elasticsearch.Bulk.WithContext(ctx),
		r.elasticsearch.Bulk.WithIndex("products"),
	)
	if err != nil {
		return fmt.Errorf("error performing bulk update: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("bulk update failed: %s", res.String())
	}

	// Check for individual failures
	var bulkResponse map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&bulkResponse); err != nil {
		return fmt.Errorf("error parsing the response body: %w", err)
	}

	if bulkResponse["errors"].(bool) {
		return fmt.Errorf("some updates failed: %v", bulkResponse)
	}

	return nil
}
