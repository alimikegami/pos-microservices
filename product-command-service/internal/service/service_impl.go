package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/alimikegami/point-of-sales/product-command-service/config"
	"github.com/alimikegami/point-of-sales/product-command-service/internal/domain"
	"github.com/alimikegami/point-of-sales/product-command-service/internal/dto"
	"github.com/alimikegami/point-of-sales/product-command-service/internal/repository"
	"github.com/alimikegami/point-of-sales/product-command-service/pkg/errs"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ProductServiceImpl struct {
	mongoDBRepo   repository.MongoDBProductRepository
	config        config.Config
	kafkaReader   *kafka.Reader
	kafkaProducer *kafka.Conn
}

func CreateProductService(mongoDBRepo repository.MongoDBProductRepository, config config.Config, kafkaReader *kafka.Reader, kafkaProducer *kafka.Conn) ProductService {
	return &ProductServiceImpl{mongoDBRepo: mongoDBRepo, config: config, kafkaReader: kafkaReader, kafkaProducer: kafkaProducer}
}

func (s *ProductServiceImpl) AddProduct(ctx context.Context, data dto.ProductRequest) (err error) {
	productId, err := s.mongoDBRepo.AddProduct(ctx, domain.Product{
		Name:        data.Name,
		Description: data.Description,
		Quantity:    data.Quantity,
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
			Price:       data.Price,
		},
	}
	fmt.Printf("%+v\n", kafkaMsg)
	jsonMsg, err := json.Marshal(kafkaMsg)
	if err != nil {
		return err
	}

	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err = s.writeKafkaMessage(jsonMsg)
		if err == nil {
			break
		}
		log.Error().Err(err).Str("component", "AddProduct").Msg("")
		time.Sleep(time.Second * time.Duration(i+1)) // Exponential backoff
	}

	if err != nil {
		return err
	}

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

func (s *ProductServiceImpl) writeKafkaMessageWithKey(msg []byte, key string) error {
	_, err := s.kafkaProducer.WriteMessages(
		kafka.Message{
			Key:   []byte(key),
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
		case "order_created":
			var orderRequest dto.OrderRequest
			dataBytes, err := json.Marshal(receivedMsg.Data)
			if err != nil {
				log.Error().Err(err).Str("component", "ConsumeEvent").Msg("")
				continue
			}

			if err := json.Unmarshal(dataBytes, &orderRequest); err != nil {
				log.Error().Err(err).Str("component", "ConsumeEvent").Msg("")
				continue
			}

			stockUpdate := dto.StockUpdate{
				TransactionNumber: orderRequest.TransactionNumber,
				Status:            true,
			}

			err = s.UpdateProductsQuantity(context.Background(), orderRequest)
			if err != nil {
				stockUpdate.Status = false
			}

			kafkaMsg := dto.KafkaMessage{
				EventType: "stock_updated",
				Data:      stockUpdate,
			}

			jsonMsg, err := json.Marshal(kafkaMsg)
			if err != nil {
				log.Error().Err(err).Str("component", "ConsumeEvent").Msg("")
				continue
			}

			maxRetries := 3
			for i := 0; i < maxRetries; i++ {
				err = s.writeKafkaMessageWithKey(jsonMsg, orderRequest.TransactionNumber)
				if err == nil {
					break
				}
				log.Error().Err(err).Str("component", "ConsumeEvent").Msg("")
				time.Sleep(time.Second * time.Duration(i+1)) // Exponential backoff
			}

			if err != nil {
				log.Printf("failed to write Kafka message after %d attempts: %v\n", maxRetries, err)
				continue
			}

			fmt.Println("handled created order")
		case "restore_product_stock":
			var orderReq dto.OrderRequest
			dataBytes, err := json.Marshal(receivedMsg.Data)
			if err != nil {
				log.Error().Err(err).Str("component", "ConsumeEvent").Msg("")
				continue
			}

			if err := json.Unmarshal(dataBytes, &orderReq); err != nil {
				log.Error().Err(err).Str("component", "ConsumeEvent").Msg("")
				continue
			}
			err = s.RestoreProductStock(context.Background(), orderReq)
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

func (s *ProductServiceImpl) RestoreProductStock(ctx context.Context, req dto.OrderRequest) (err error) {
	// err = s.mongoDBRepo.HandleTrx(ctx, func(ctx mongo.SessionContext, repo repository.MongoDBProductRepository) error {
	for _, orderItem := range req.OrderItems {
		productID, err := primitive.ObjectIDFromHex(orderItem.ProductID)
		if err != nil {
			return err
		}

		err = s.mongoDBRepo.UpdateProductQuantity(ctx, domain.Product{
			ID:       productID,
			Quantity: uint64(orderItem.Quantity),
		})
		if err != nil {
			return err
		}
	}

	// return nil
	// })

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
		EventType: "restore_product_stock_es",
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

func (s *ProductServiceImpl) UpdateProductsQuantity(ctx context.Context, req dto.OrderRequest) (err error) {
	// TODO: handle transactions
	err = s.mongoDBRepo.HandleTrx(ctx, func(sessionCtx mongo.SessionContext) error {
		for _, orderItem := range req.OrderItems {
			product, err := s.mongoDBRepo.GetProductByID(sessionCtx, orderItem.ProductID)
			if err != nil {
				return err
			}

			if product.Quantity < uint64(orderItem.Quantity) {
				return errs.ErrConflict
			}

			err = s.mongoDBRepo.SetProductQuantity(sessionCtx, domain.Product{
				ID:       product.ID,
				Quantity: product.Quantity - uint64(orderItem.Quantity),
			})
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
		EventType: "decrease_product_quantity",
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

func (s *ProductServiceImpl) DeleteProduct(ctx context.Context, id string) (err error) {
	err = s.mongoDBRepo.DeleteProduct(ctx, id)

	if err != nil {
		return
	}

	kafkaMsg := dto.KafkaMessage{
		EventType: "delete_product",
		Data: dto.Product{
			ID: id,
		},
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

func (s *ProductServiceImpl) UpdateProduct(ctx context.Context, data dto.ProductRequest) (err error) {
	objectID, err := primitive.ObjectIDFromHex(data.ID)
	if err != nil {
		return fmt.Errorf("invalid product ID: %v", err)
	}

	updatedData := domain.Product{
		ID:          objectID,
		Name:        data.Name,
		Description: data.Description,
		Quantity:    data.Quantity,
		Price:       data.Price,
	}

	err = s.mongoDBRepo.UpdateProduct(ctx, updatedData)
	if err != nil {
		return
	}

	kafkaMsg := dto.KafkaMessage{
		EventType: "update_product",
		Data: dto.Product{
			ID:          data.ID,
			Name:        data.Name,
			Description: data.Description,
			Quantity:    data.Quantity,
			Price:       data.Price,
		},
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

func (s *ProductServiceImpl) UpdateProductQuantity(ctx context.Context, req dto.ProductQuantityRequest) (err error) {
	productData, err := s.mongoDBRepo.GetProductByID(ctx, req.ProductID)
	if err != nil {
		return
	}

	if req.Action == "add" {
		productData.Quantity += req.Quantity
	} else if req.Action == "reduce" {
		productData.Quantity -= req.Quantity
	} else {
		return errs.ErrClient
	}

	err = s.mongoDBRepo.UpdateProductQuantity(ctx, productData)
	if err != nil {
		return
	}

	kafkaMsg := dto.KafkaMessage{
		EventType: "update_product",
		Data:      productData,
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
