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
)

type ElasticSearchProductRepositoryImpl struct {
	elasticsearch *elasticsearch.Client
}

func CreateNewElasticSearchRepository(elasticsearch *elasticsearch.Client) ElasticSearchProductRepository {
	return &ElasticSearchProductRepositoryImpl{elasticsearch: elasticsearch}
}

func (r *ElasticSearchProductRepositoryImpl) AddProduct(ctx context.Context, index string, data dto.ProductResponse) (err error) {
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

func (r *ElasticSearchProductRepositoryImpl) GetProducts(ctx context.Context, filter pkgdto.Filter) ([]dto.ProductResponse, int, error) {
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

func (r *ElasticSearchProductRepositoryImpl) DecreaseProductQuantities(ctx context.Context, products []domain.Product) error {
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

func (r *ElasticSearchProductRepositoryImpl) AddProductQuantities(ctx context.Context, products []domain.Product) error {
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
				"source": "ctx._source.quantity = Math.max(0, ctx._source.quantity + params.add)",
				"lang":   "painless",
				"params": map[string]interface{}{
					"add": product.Quantity,
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

func (r *ElasticSearchProductRepositoryImpl) UpdateProductQuantities(ctx context.Context, product domain.Product) error {
	var buf bytes.Buffer

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
			"source": "ctx._source.quantity = params.newQuantity",
			"lang":   "painless",
			"params": map[string]interface{}{
				"newQuantity": product.Quantity,
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

func (r *ElasticSearchProductRepositoryImpl) DeleteProduct(ctx context.Context, id string) error {
	res, err := r.elasticsearch.Delete(
		"products",
		id,
		r.elasticsearch.Delete.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("error deleting document: %w", err)
	}
	defer res.Body.Close()

	// Check if the operation was successful
	if res.IsError() {
		return fmt.Errorf("error deleting document: %s", res.String())
	}

	log.Printf("Document deleted successfully with ID: %s", id)

	return nil
}

func (r *ElasticSearchProductRepositoryImpl) UpdateProduct(ctx context.Context, data domain.Product) (err error) {
	docBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshaling document: %w", err)
	}

	res, err := r.elasticsearch.Index(
		"products",
		bytes.NewReader(docBytes),
		r.elasticsearch.Index.WithDocumentID(data.ID.Hex()),
		r.elasticsearch.Index.WithContext(ctx),
	)

	if err != nil {
		return fmt.Errorf("error indexing document: %w", err)
	}

	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error indexing document: %s", res.String())
	}

	log.Printf("Document indexed successfully with ID: %s", data.ID)

	return nil
}
