package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/alimikegami/point-of-sales/product-service/config"
	"github.com/alimikegami/point-of-sales/product-service/internal/domain"
	"github.com/alimikegami/point-of-sales/product-service/internal/dto"
	"github.com/alimikegami/point-of-sales/product-service/internal/repository"
	pkgdto "github.com/alimikegami/point-of-sales/product-service/pkg/dto"
	"github.com/alimikegami/point-of-sales/product-service/pkg/errs"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProductServiceImpl struct {
	mongoDBRepo       repository.MongoDBProductRepository
	elasticSearchRepo repository.ElasticSearchProductRepository
	config            config.Config
	kafkaReader       *kafka.Reader
	kafkaProducer     *kafka.Conn
}

func CreateProductService(mongoDBRepo repository.MongoDBProductRepository, elasticSearchRepo repository.ElasticSearchProductRepository, config config.Config, kafkaReader *kafka.Reader, kafkaProducer *kafka.Conn) ProductService {
	return &ProductServiceImpl{mongoDBRepo: mongoDBRepo, elasticSearchRepo: elasticSearchRepo, config: config, kafkaReader: kafkaReader, kafkaProducer: kafkaProducer}
}

func (s *ProductServiceImpl) AddProduct(ctx context.Context, data dto.ProductRequest) (err error) {
	productId, err := s.mongoDBRepo.AddProduct(ctx, domain.Product{
		Name:        data.Name,
		Description: data.Description,
		Quantity:    data.Quantity,
		UserID:      data.UserID,
		UserName:    data.UserName,
		Price:       data.Price,
	})

	if err != nil {
		return
	}

	kafkaMsg := dto.KafkaMessage{
		EventType: "add_product",
		Data: dto.ProductResponse{
			ID:          productId.Hex(),
			Name:        data.Name,
			Description: data.Description,
			Quantity:    data.Quantity,
			UserID:      data.UserID,
			UserName:    data.UserName,
			Price:       data.Price,
		},
	}
	fmt.Printf("%+v\n", kafkaMsg)
	jsonMsg, err := json.Marshal(kafkaMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal Kafka message: %w", err)
	}

	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err = s.writeKafkaMessage(jsonMsg)
		if err == nil {
			break
		}
		log.Printf("Failed to write Kafka message (attempt %d/%d): %v", i+1, maxRetries, err)
		time.Sleep(time.Second * time.Duration(i+1)) // Exponential backoff
	}

	if err != nil {
		return fmt.Errorf("failed to write Kafka message after %d attempts: %w", maxRetries, err)
	}

	return
}

func (s *ProductServiceImpl) GetProducts(ctx context.Context, filter pkgdto.Filter) (responsePayload pkgdto.PaginationResponse, err error) {
	data, total, err := s.elasticSearchRepo.GetProducts(ctx, filter)
	if err != nil {
		return
	}

	responsePayload.Records = data
	responsePayload.Metadata.TotalCount = uint64(total)
	return
}

func (s *ProductServiceImpl) UpdateSellerDetails(ctx context.Context, data dto.User) (err error) {
	err = s.mongoDBRepo.UpdateSellerDetails(context.Background(), data)

	return
}

func (s *ProductServiceImpl) AddProductToElasticsearch(ctx context.Context, data dto.ProductResponse) (err error) {
	err = s.elasticSearchRepo.AddProduct(ctx, "products", data)

	return
}

func (s *ProductServiceImpl) UpdateElasticSearchProductQuantity(ctx context.Context, data []domain.Product) (err error) {
	err = s.elasticSearchRepo.UpdateProductQuantities(ctx, data)

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
			fmt.Println("Error reading Kafka message:", err)
			continue
		}

		var receivedMsg dto.KafkaMessage
		if err := json.Unmarshal(msg.Value, &receivedMsg); err != nil {
			fmt.Println("Error unmarshalling Kafka message:", err)
			continue
		}

		fmt.Printf("Received message: %+v\n", receivedMsg)

		switch receivedMsg.EventType {
		case "user_update":
			var userData dto.User
			dataBytes, err := json.Marshal(receivedMsg.Data)
			if err != nil {
				fmt.Println("Error marshalling user data:", err)
				continue
			}
			if err := json.Unmarshal(dataBytes, &userData); err != nil {
				fmt.Println("Error unmarshalling user data:", err)
				continue
			}

			err = s.UpdateSellerDetails(context.Background(), userData)
			if err != nil {
				fmt.Println("Error:", err)
				continue
			}

			fmt.Println("User details updated successfully")
		case "user_delete":
			// Handle user delete event if needed
			fmt.Println("User delete event received - implement handling if required")
		case "add_product":
			var productData dto.ProductResponse
			dataBytes, err := json.Marshal(receivedMsg.Data)
			if err != nil {
				fmt.Println("Error marshalling user data:", err)
				continue
			}
			if err := json.Unmarshal(dataBytes, &productData); err != nil {
				fmt.Println("Error unmarshalling user data:", err)
				continue
			}

			err = s.AddProductToElasticsearch(context.Background(), productData)
			if err != nil {
				fmt.Println("Error:", err)
				continue
			}

			fmt.Println("product data indexed successfully")
		case "update_product_quantity":
			var products []domain.Product
			dataBytes, err := json.Marshal(receivedMsg.Data)
			if err != nil {
				fmt.Println("Error marshalling user data:", err)
				continue
			}

			if err := json.Unmarshal(dataBytes, &products); err != nil {
				fmt.Println("Error unmarshalling user data:", err)
				continue
			}

			err = s.UpdateElasticSearchProductQuantity(context.Background(), products)
			if err != nil {
				fmt.Println("Error:", err)
				continue
			}

			fmt.Println("product data updated successfully")
		default:
			fmt.Printf("Unknown event type: %s\n", receivedMsg.EventType)
		}
	}
}

func (s *ProductServiceImpl) UpdateProductsQuantity(ctx context.Context, req dto.OrderRequest) (err error) {
	err = s.mongoDBRepo.HandleTrx(ctx, func(repo repository.MongoDBProductRepository) error {
		for _, orderItem := range req.OrderItems {
			product, err := repo.GetProductByID(ctx, orderItem.ProductID)
			if err != nil {
				return err
			}

			if product.Quantity < uint64(orderItem.Quantity) {
				return errs.ErrConflict
			}

			err = repo.ReduceProductQuantity(ctx, orderItem.ProductID, orderItem.Quantity)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	var products []domain.Product

	for _, orderItem := range req.OrderItems {
		objectID, err := primitive.ObjectIDFromHex(orderItem.ProductID)
		if err != nil {
			return err
		}
		products = append(products, domain.Product{
			ID:       objectID,
			Quantity: uint64(orderItem.Quantity),
		})
	}

	kafkaMsg := dto.KafkaMessage{
		EventType: "update_product_quantity",
		Data:      products,
	}

	jsonMsg, err := json.Marshal(kafkaMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal Kafka message: %w", err)
	}

	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err = s.writeKafkaMessage(jsonMsg)
		if err == nil {
			break
		}
		log.Printf("Failed to write Kafka message (attempt %d/%d): %v", i+1, maxRetries, err)
		time.Sleep(time.Second * time.Duration(i+1)) // Exponential backoff
	}

	if err != nil {
		return fmt.Errorf("failed to write Kafka message after %d attempts: %w", maxRetries, err)
	}

	return
}
