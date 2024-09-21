package elasticsearch

import (
	"log"
	"net/http"

	"github.com/alimikegami/point-of-sales/product-service/config"
	"github.com/elastic/go-elasticsearch"
)

// Declare a variable to hold the single instance of the client
var (
	esClientInstance *elasticsearch.Client
)

func CreateElasticsearchClient(config *config.Config) (*elasticsearch.Client, error) {
	var err error

	cfg := elasticsearch.Config{
		Addresses: []string{
			config.ElasticsearchConfig.DBHost,
		},
		Transport: http.DefaultTransport,
	}

	esClientInstance, err = elasticsearch.NewClient(cfg)
	if err != nil {
		log.Printf("Error creating Elasticsearch client: %s", err)
		return esClientInstance, err
	}

	res, err := esClientInstance.Info()
	if err != nil {
		log.Printf("Error connecting to Elasticsearch: %s", err)
		return esClientInstance, err
	}
	defer res.Body.Close()

	if res.IsError() {
		log.Printf("Error response from Elasticsearch: %s", res.String())
	}

	return esClientInstance, err
}
