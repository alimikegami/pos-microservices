package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/alimikegami/point-of-sales/order-service/internal/domain"
	"github.com/alimikegami/point-of-sales/order-service/internal/dto"
	"github.com/alimikegami/point-of-sales/order-service/internal/repository"
	pkgdto "github.com/alimikegami/point-of-sales/order-service/pkg/dto"
	"github.com/alimikegami/point-of-sales/order-service/pkg/errs"
	"github.com/alimikegami/point-of-sales/order-service/pkg/httpclient"
	"github.com/alimikegami/point-of-sales/order-service/pkg/utils"
	"github.com/google/uuid"
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/coreapi"
	"github.com/segmentio/kafka-go"
)

type OrderServiceImpl struct {
	repository     repository.OrderRepository
	midtransClient *coreapi.Client
	kafkaReader    *kafka.Reader
	kafkaProducer  *kafka.Conn
}

func CreateOrderService(repository repository.OrderRepository, midtransClient *coreapi.Client, kafkaReader *kafka.Reader, kafkaProducer *kafka.Conn) OrderService {
	return &OrderServiceImpl{
		repository:     repository,
		midtransClient: midtransClient,
		kafkaReader:    kafkaReader,
		kafkaProducer:  kafkaProducer,
	}
}

func (s *OrderServiceImpl) AddOrder(ctx context.Context, req dto.OrderRequest) (err error) {
	trxNumber, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("error generating transaction number: %v", err)
	}

	var productReqPayload dto.OrderProductServiceRequest
	productReqPayload.TransactionNumber = trxNumber.String()
	for _, item := range req.OrderItems {
		productReqPayload.OrderItems = append(productReqPayload.OrderItems, dto.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	err = s.repository.HandleTrx(ctx, func(repo repository.OrderRepository) error {
		var orderDetails []domain.OrderDetail

		productIDs := make([]string, len(req.OrderItems))
		for i, item := range req.OrderItems {
			productIDs[i] = item.ProductID
		}

		priceInfoReq := dto.ProductRequest{
			ProductIds: productIDs,
		}
		priceInfoJsonBody, err := json.Marshal(priceInfoReq)
		if err != nil {
			return fmt.Errorf("error marshalling price info request: %v", err)
		}

		priceInfoHttpReq := httpclient.HttpRequest{
			URL:    "http://localhost:8081/api/v1/products/prices",
			Method: "POST",
			Body:   priceInfoJsonBody,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}

		statusCode, priceInfoBody, err := httpclient.SendRequest(priceInfoHttpReq)
		if err != nil {
			return fmt.Errorf("error calling product price info service: %v", err)
		}
		if statusCode != http.StatusOK {
			return fmt.Errorf("product price info service returned non-OK status: %d", statusCode)
		}

		// Parse the price info response
		var priceInfoResponse dto.ProductResponse
		if err := json.Unmarshal(priceInfoBody, &priceInfoResponse); err != nil {
			return fmt.Errorf("error unmarshalling price info response: %v", err)
		}

		kafkaMsg := dto.KafkaMessage{
			EventType: "order_created",
			Data:      productReqPayload,
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

		err = s.listenForProductStockUpdate(trxNumber.String())
		if err != nil {
			return err
		}

		var totalAmount float64
		chargeItems := make([]midtrans.ItemDetails, len(req.OrderItems))
		for i, item := range req.OrderItems {
			var productInfo dto.ProductRecord
			for _, p := range priceInfoResponse.Data.Records {
				if p.ID == item.ProductID {
					productInfo = p
					orderDetails = append(orderDetails, domain.OrderDetail{
						ProductID:   p.ID,
						Quantity:    int64(item.Quantity),
						Amount:      p.Price,
						ProductName: p.Name,
					})
					break
				}
			}

			if productInfo.ID == "" {
				return fmt.Errorf("product info not found for product ID: %s", item.ProductID)
			}

			itemTotal := float64(productInfo.Price) * float64(item.Quantity)
			totalAmount += itemTotal
			chargeItems[i] = midtrans.ItemDetails{
				ID:    item.ProductID,
				Price: int64(productInfo.Price),
				Qty:   int32(item.Quantity),
				Name:  productInfo.Name,
			}
		}

		chargeReq := &coreapi.ChargeReq{
			PaymentType: coreapi.PaymentTypeQris,
			TransactionDetails: midtrans.TransactionDetails{
				OrderID:  trxNumber.String(),
				GrossAmt: int64(totalAmount),
			},
			CustomerDetails: &midtrans.CustomerDetails{
				FName: "John",
				LName: "Doe",
				Email: "john@example.com",
				Phone: "081234567890",
			},
			Items: &chargeItems,
		}

		response, err := s.midtransClient.ChargeTransaction(chargeReq)
		if response.StatusCode != "201" {
			return fmt.Errorf("payment gateway returned non-200 status: %s", response.StatusCode)
		}

		expiredAt, err := utils.ConvertDateTimeWibToUnixTimestamp(response.ExpiryTime)
		if err != nil {
			return err
		}

		orderID, err := repo.AddOrder(ctx, domain.Order{
			PaymentMethodID:   int64(req.PaymentMethodID),
			Amount:            totalAmount,
			PaymentStatus:     "pending",
			TransactionNumber: trxNumber.String(),
			ExpiredAt:         expiredAt,
		})
		if err != nil {
			return err
		}

		for idx := range orderDetails {
			orderDetails[idx].OrderID = orderID
		}

		err = repo.AddOrderDetails(ctx, orderDetails)

		return err
	})

	if err != nil {
		fmt.Println(err)
		kafkaMsg := dto.KafkaMessage{
			EventType: "restore_product_stock",
			Data:      productReqPayload,
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

		return err
	}

	return nil
}

func (s *OrderServiceImpl) MidtransPaymentWebhook(ctx context.Context, req dto.PaymentNotification) (err error) {
	order, err := s.repository.GetOrderByTransactionNumber(ctx, req.OrderID)
	if err != nil {
		return
	}

	if order.ExpiredAt < time.Now().Unix() {
		return errs.ErrPaymentExpired
	}

	err = s.repository.UpdateOrderPaymentStatus(ctx, domain.Order{
		ID:            order.ID,
		PaymentStatus: "success",
	})

	return
}

func (s *OrderServiceImpl) listenForProductStockUpdate(transactionNumber string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	success := make(chan struct{})
	failed := make(chan struct{})

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := s.kafkaReader.ReadMessage(ctx)
				if err != nil {
					if err == context.DeadlineExceeded {
						return
					}
					log.Println("Error reading Kafka message:", err)
					continue
				}

				var receivedMsg dto.KafkaMessage
				if err := json.Unmarshal(msg.Value, &receivedMsg); err != nil {
					log.Println("Error unmarshalling Kafka message:", err)
					continue
				}

				log.Printf("Received message: %+v\n", receivedMsg)

				if receivedMsg.EventType == "stock_updated" {
					var stockUpdateData dto.ProductServiceStockUpdate
					dataBytes, err := json.Marshal(receivedMsg.Data)
					if err != nil {
						log.Println("Error marshalling user data:", err)
						continue
					}
					if err := json.Unmarshal(dataBytes, &stockUpdateData); err != nil {
						log.Println("Error unmarshalling user data:", err)
						continue
					}

					if stockUpdateData.TransactionNumber == transactionNumber {
						if stockUpdateData.Status {
							close(success)
							return
						} else {
							close(failed)
							return
						}
					}
				}
			}
		}
	}()

	select {
	case <-success:
		return nil
	case <-failed:
		return errs.ErrConflict
	case <-ctx.Done():
		return fmt.Errorf("timeout waiting for stock update")
	}
}

func (s *OrderServiceImpl) writeKafkaMessage(msg []byte) error {
	_, err := s.kafkaProducer.WriteMessages(
		kafka.Message{
			Value: msg,
		},
	)
	return err
}

func (s *OrderServiceImpl) GetOrders(ctx context.Context, filter pkgdto.Filter) (response pkgdto.Pagination, err error) {
	var orderResponse []dto.OrderResponse
	datas, err := s.repository.GetOrders(ctx, filter)

	for _, data := range datas {
		orderResponse = append(orderResponse, dto.OrderResponse{
			ID:                data.ID,
			PaymentStatus:     data.PaymentStatus,
			TransactionAmount: data.Amount,
			PaymentMethodName: data.PaymentMethod.Name,
		})
	}

	response.Records = orderResponse

	return
}
