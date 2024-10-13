package repository

import (
	"context"
	"encoding/json"

	"github.com/alimikegami/point-of-sales/product-service/config"
	"github.com/alimikegami/point-of-sales/product-service/internal/domain"
	"github.com/alimikegami/point-of-sales/product-service/internal/dto"
	pkgdto "github.com/alimikegami/point-of-sales/product-service/pkg/dto"
	"github.com/alimikegami/point-of-sales/product-service/pkg/errs"
	"github.com/alimikegami/point-of-sales/product-service/pkg/httpclient"
)

type ElasticSearchProductRepositoryImpl struct {
	config *config.Config
}

func CreateNewElasticSearchRepository(config *config.Config) ElasticSearchProductRepository {
	return &ElasticSearchProductRepositoryImpl{config: config}
}

func (r *ElasticSearchProductRepositoryImpl) AddProduct(ctx context.Context, index string, data dto.ProductResponse) (err error) {
	requestPayload, err := json.Marshal(data)
	if err != nil {
		return
	}

	statusCode, _, err := httpclient.SendRequest(ctx, httpclient.HttpRequest{
		Body:   requestPayload,
		URL:    r.config.ElasticsearchConfig.DBHost + "/products/_doc/" + data.ID,
		Method: "POST",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	})

	if err != nil {
		return
	}

	if statusCode != 201 {
		return errs.ErrInternalServer
	}

	return
}

func (r *ElasticSearchProductRepositoryImpl) GetProducts(ctx context.Context, filter pkgdto.Filter) (data []dto.ProductResponse, count int, err error) {
	param := make(map[string]interface{})
	var parsedResponseBody pkgdto.ElasticsearchResponse

	if filter.Limit != 0 && filter.Page != 0 {
		param["size"] = filter.Limit
		param["from"] = (filter.Page - 1) * filter.Limit
	}

	requestPayload, err := json.Marshal(param)
	if err != nil {
		return
	}

	_, responseBody, err := httpclient.SendRequest(ctx, httpclient.HttpRequest{
		Body:   requestPayload,
		URL:    r.config.ElasticsearchConfig.DBHost + "/products/_search",
		Method: "GET",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	})
	if err != nil {
		return
	}
	err = json.Unmarshal(responseBody, &parsedResponseBody)
	if err != nil {
		return
	}
	for _, productData := range parsedResponseBody.Hits.Hits {
		data = append(data, productData.Source)
	}

	return data, parsedResponseBody.Hits.Total.Value, nil
}

func (r *ElasticSearchProductRepositoryImpl) DecreaseProductQuantities(ctx context.Context, products []domain.Product) error {
	for _, product := range products {
		param := make(map[string]interface{})

		param["script"] = map[string]interface{}{
			"lang":   "painless",
			"source": "ctx._source.quantity -= params.subtraction",
			"params": map[string]interface{}{
				"subtraction": product.Quantity,
			},
		}

		requestPayload, err := json.Marshal(param)
		if err != nil {
			return err
		}
		statusCode, _, err := httpclient.SendRequest(ctx, httpclient.HttpRequest{
			Body:   requestPayload,
			URL:    r.config.ElasticsearchConfig.DBHost + "/products/_update/" + product.ID.Hex(),
			Method: "POST",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		})
		if err != nil {
			return err
		}

		if statusCode == 200 {
			return errs.ErrInternalServer
		}
	}

	return nil
}

func (r *ElasticSearchProductRepositoryImpl) AddProductQuantities(ctx context.Context, products []domain.Product) error {
	for _, product := range products {
		param := make(map[string]interface{})

		param["script"] = map[string]interface{}{
			"language": "painless",
			"source":   "ctx._source.quantity += params.addition",
			"param": map[string]interface{}{
				"addition": product.Quantity,
			},
		}

		requestPayload, err := json.Marshal(param)
		if err != nil {
			return err
		}

		statusCode, _, err := httpclient.SendRequest(ctx, httpclient.HttpRequest{
			Body:   requestPayload,
			URL:    r.config.ElasticsearchConfig.DBHost + "/products/_update/" + product.ID.Hex(),
			Method: "POST",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		})
		if err != nil {
			return err
		}

		if statusCode != 200 {
			return err
		}
	}

	return nil
}

func (r *ElasticSearchProductRepositoryImpl) DeleteProduct(ctx context.Context, id string) error {
	statusCode, _, err := httpclient.SendRequest(ctx, httpclient.HttpRequest{
		Body:    nil,
		URL:     r.config.ElasticsearchConfig.DBHost + "/products/_doc/" + id,
		Method:  "DELETE",
		Headers: map[string]string{"Content-Type": "application/json"},
	})
	if err != nil {
		return err
	}

	if statusCode == 404 {
		return errs.ErrNotFound
	} else if statusCode != 200 {
		return errs.ErrInternalServer
	}

	return nil
}

func (r *ElasticSearchProductRepositoryImpl) UpdateProduct(ctx context.Context, data domain.Product) (err error) {
	requestPayload, err := json.Marshal(data)
	if err != nil {
		return
	}

	statusCode, _, err := httpclient.SendRequest(ctx, httpclient.HttpRequest{
		Body:   requestPayload,
		URL:    r.config.ElasticsearchConfig.DBHost + "/products/_doc/" + data.ID.Hex(),
		Method: "POST",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	})

	if err != nil {
		return
	}

	if statusCode != 200 {
		return errs.ErrInternalServer
	}

	return

}
