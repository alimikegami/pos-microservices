package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/alimikegami/point-of-sales/product-query-service/config"
	"github.com/alimikegami/point-of-sales/product-query-service/internal/domain"
	"github.com/alimikegami/point-of-sales/product-query-service/internal/dto"
	"github.com/alimikegami/point-of-sales/product-query-service/internal/repository"
	pkgdto "github.com/alimikegami/point-of-sales/product-query-service/pkg/dto"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProductServiceImpl struct {
	elasticSearchRepo repository.ElasticSearchProductRepository
	config            config.Config
	kafkaReader       *kafka.Reader
	kafkaProducer     *kafka.Conn
}

func CreateProductService(elasticSearchRepo repository.ElasticSearchProductRepository, config config.Config, kafkaReader *kafka.Reader, kafkaProducer *kafka.Conn) ProductService {
	return &ProductServiceImpl{elasticSearchRepo: elasticSearchRepo, config: config, kafkaReader: kafkaReader, kafkaProducer: kafkaProducer}
}

func (s *ProductServiceImpl) GetProducts(ctx context.Context, filter pkgdto.Filter) (responsePayload pkgdto.PaginationResponse, err error) {
	data, total, err := s.elasticSearchRepo.GetProducts(ctx, filter)
	if err != nil {
		return
	}

	responsePayload.Records = data
	responsePayload.Metadata.TotalCount = uint64(total)
	responsePayload.Metadata.Limit = filter.Limit
	responsePayload.Metadata.Page = uint64(filter.Page)
	return
}

func (s *ProductServiceImpl) AddProductToElasticsearch(ctx context.Context, data dto.ProductResponse) (err error) {
	err = s.elasticSearchRepo.AddProduct(ctx, "products", data)

	return
}

func (s *ProductServiceImpl) DecreaseElasticSearchProductQuantity(ctx context.Context, data []domain.Product) (err error) {
	err = s.elasticSearchRepo.DecreaseProductQuantities(ctx, data)

	return
}

func (s *ProductServiceImpl) AddElasticSearchProductQuantity(ctx context.Context, data []domain.Product) (err error) {
	err = s.elasticSearchRepo.AddProductQuantities(ctx, data)

	return
}

func (s *ProductServiceImpl) writeKafkaMessage(msg []byte) error {
	_, err := s.kafkaProducer.WriteMessages(
		kafka.Message{
			Value: msg,
		},
	)
	return err
}

func (s *ProductServiceImpl) ConsumeEvent() {
	for {
		msg, err := s.kafkaReader.ReadMessage(context.Background())
		if err != nil {
			log.Error().Err(err).Str("component", "ConsumeEvent").Msg("")
			continue
		}

		var receivedMsg dto.KafkaMessage
		if err := json.Unmarshal(msg.Value, &receivedMsg); err != nil {
			log.Error().Err(err).Str("component", "ConsumeEvent").Msg("")
			continue
		}

		fmt.Printf("Received message: %+v\n", receivedMsg)
		switch receivedMsg.EventType {
		case "add_product":
			var productData dto.ProductResponse
			dataBytes, err := json.Marshal(receivedMsg.Data)
			if err != nil {
				log.Error().Err(err).Str("component", "ConsumeEvent").Msg("")
				continue
			}
			if err := json.Unmarshal(dataBytes, &productData); err != nil {
				log.Error().Err(err).Str("component", "ConsumeEvent").Msg("")
				continue
			}

			err = s.AddProductToElasticsearch(context.Background(), productData)
			if err != nil {
				log.Error().Err(err).Str("component", "ConsumeEvent").Msg("")
				continue
			}

			fmt.Println("product data indexed successfully")
		case "decrease_product_quantity":
			var products []domain.Product
			dataBytes, err := json.Marshal(receivedMsg.Data)
			if err != nil {
				log.Error().Err(err).Str("component", "ConsumeEvent").Msg("")
				continue
			}

			if err := json.Unmarshal(dataBytes, &products); err != nil {
				log.Error().Err(err).Str("component", "ConsumeEvent").Msg("")
				continue
			}
			err = s.DecreaseElasticSearchProductQuantity(context.Background(), products)
			if err != nil {
				log.Error().Err(err).Str("component", "ConsumeEvent").Msg("")
				continue
			}

			fmt.Println("product data updated successfully")
		case "delete_product":
			var product dto.Product
			dataBytes, err := json.Marshal(receivedMsg.Data)
			if err != nil {
				log.Error().Err(err).Str("component", "ConsumeEvent").Msg("")
				continue
			}

			if err := json.Unmarshal(dataBytes, &product); err != nil {
				log.Error().Err(err).Str("component", "ConsumeEvent").Msg("")
				continue
			}

			err = s.DeleteElasticSearchProduct(context.Background(), product.ID)
			if err != nil {
				log.Error().Err(err).Str("component", "ConsumeEvent").Msg("")
				continue
			}

			fmt.Println("product data deleted successfully")
		case "update_product":
			var product dto.Product
			dataBytes, err := json.Marshal(receivedMsg.Data)
			if err != nil {
				log.Error().Err(err).Str("component", "ConsumeEvent").Msg("")
				continue
			}

			if err := json.Unmarshal(dataBytes, &product); err != nil {
				log.Error().Err(err).Str("component", "ConsumeEvent").Msg("")
				continue
			}

			err = s.UpdateElasticSearchProduct(context.Background(), product)
			if err != nil {
				log.Error().Err(err).Str("component", "ConsumeEvent").Msg("")
				continue
			}

			fmt.Println("product data updated successfully")
		case "restore_product_stock_es":
			var products []domain.Product
			dataBytes, err := json.Marshal(receivedMsg.Data)
			if err != nil {
				log.Error().Err(err).Str("component", "ConsumeEvent").Msg("")
				continue
			}

			if err := json.Unmarshal(dataBytes, &products); err != nil {
				log.Error().Err(err).Str("component", "ConsumeEvent").Msg("")
				continue
			}
			err = s.AddElasticSearchProductQuantity(context.Background(), products)
			if err != nil {
				log.Error().Err(err).Str("component", "ConsumeEvent").Msg("")
				continue
			}

			fmt.Println("product data updated successfully")
		default:
			fmt.Printf("Unknown event type: %s\n", receivedMsg.EventType)
		}
	}
}

func (s *ProductServiceImpl) UpdateElasticSearchProduct(ctx context.Context, data dto.Product) (err error) {
	objectID, err := primitive.ObjectIDFromHex(data.ID)
	if err != nil {
		return
	}

	err = s.elasticSearchRepo.UpdateProduct(ctx, domain.Product{
		ID:          objectID,
		Name:        data.Name,
		Description: data.Description,
		Quantity:    data.Quantity,
		Price:       data.Price,
	})

	return
}

func (s *ProductServiceImpl) DeleteElasticSearchProduct(ctx context.Context, id string) (err error) {
	err = s.elasticSearchRepo.DeleteProduct(ctx, id)

	return
}
