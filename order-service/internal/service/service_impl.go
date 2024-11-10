package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/alimikegami/point-of-sales/order-service/config"
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
	config         *config.Config
}

func CreateOrderService(repository repository.OrderRepository, midtransClient *coreapi.Client, kafkaReader *kafka.Reader, kafkaProducer *kafka.Conn, config *config.Config) OrderService {
	return &OrderServiceImpl{
		repository:     repository,
		midtransClient: midtransClient,
		kafkaReader:    kafkaReader,
		kafkaProducer:  kafkaProducer,
		config:         config,
	}
}

func (s *OrderServiceImpl) AddOrder(ctx context.Context, req dto.OrderRequest) (orderResponse dto.OrderResponse, err error) {
	trxNumber, err := uuid.NewV7()
	if err != nil {
		return orderResponse, fmt.Errorf("error generating transaction number: %v", err)
	}

	var productReqPayload dto.OrderProductServiceRequest
	productReqPayload.TransactionNumber = trxNumber.String()
	for _, item := range req.OrderItems {
		productReqPayload.OrderItems = append(productReqPayload.OrderItems, dto.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	paymentMethod, err := s.repository.GetPaymentMethodByID(ctx, req.PaymentMethodID)
	if err != nil {
		return orderResponse, err
	}

	err = s.repository.HandleTrx(ctx, func(ctx context.Context, repo repository.OrderRepository) error {
		var paymentType coreapi.CoreapiPaymentType

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
			log.Error().Err(err).Str("component", "AddOrder").Msg("")
			return fmt.Errorf("error marshalling price info request: %v", err)
		}

		priceInfoHttpReq := httpclient.HttpRequest{
			URL:    fmt.Sprintf("%s/api/v1/products/prices", s.config.ProductQueryServiceHost),
			Method: "POST",
			Body:   priceInfoJsonBody,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}

		statusCode, priceInfoBody, err := httpclient.SendRequest(priceInfoHttpReq)
		if err != nil {
			log.Error().Err(err).Str("component", "AddOrder").Msg("")
			return fmt.Errorf("error calling product price info service: %v", err)
		}

		if statusCode != http.StatusOK {
			return fmt.Errorf("product price info service returned non-OK status: %d", statusCode)
		}

		// Parse the price info response
		var priceInfoResponse dto.ProductResponse
		if err := json.Unmarshal(priceInfoBody, &priceInfoResponse); err != nil {
			log.Error().Err(err).Str("component", "AddOrder").Msg("")
			return fmt.Errorf("error unmarshalling price info response: %v", err)
		}

		productReqPayloadJson, err := json.Marshal(productReqPayload)
		if err != nil {
			log.Error().Err(err).Str("component", "AddOrder").Msg("")
			return fmt.Errorf("error marshalling price info request: %v", err)
		}

		updateProductStock := httpclient.HttpRequest{
			URL:    fmt.Sprintf("%s/api/v1/products/quantity", s.config.ProductCommandServiceHost),
			Method: "PUT",
			Body:   productReqPayloadJson,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}

		statusCode, _, err = httpclient.SendRequest(updateProductStock)
		if err != nil {
			return fmt.Errorf("error calling product price info service: %v", err)
		}

		if statusCode != 200 {
			log.Info().Msg(fmt.Sprintf("%d", statusCode))
			return errs.ErrInternalServer
		}

		restoreProductMsg := dto.KafkaMessage{
			EventType: "restore_product_stock",
			Data:      productReqPayload,
		}

		restoreProductMsgParsed, err := json.Marshal(restoreProductMsg)
		if err != nil {
			log.Info().Msg("Restoring product stock")
			go func() {
				err = s.WriteKafkaMessageWithKey(restoreProductMsgParsed, trxNumber.String())
				if err != nil {
					log.Error().Err(err).Str("component", "AddOrder").Msg("")
				}
			}()

			log.Error().Err(err).Str("component", "AddOrder").Msg("")

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
						CreatedAt:   time.Now().Unix(),
						UpdatedAt:   time.Now().Unix(),
					})
					break
				}
			}

			if productInfo.ID == "" {
				log.Info().Msg("Restoring product stock")
				go func() {
					err = s.WriteKafkaMessageWithKey(restoreProductMsgParsed, trxNumber.String())
					if err != nil {
						log.Error().Err(err).Str("component", "AddOrder").Msg("")
					}
				}()

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

		if strings.ToLower(paymentMethod.Name) == "qris" {
			paymentType = coreapi.PaymentTypeQris
		}
		chargeReq := &coreapi.ChargeReq{
			PaymentType: paymentType,
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
			log.Info().Msg("Restoring product stock")
			go func() {
				err = s.WriteKafkaMessageWithKey(restoreProductMsgParsed, trxNumber.String())
				if err != nil {
					log.Error().Err(err).Str("component", "AddOrder").Msg("")
				}
			}()

			return fmt.Errorf("payment gateway returned non-200 status: %s", response.StatusCode)
		}

		fmt.Printf("%+v\n", response)

		if strings.ToLower(paymentMethod.Name) == "qris" {
			orderResponse.QRCode = &response.QRString
		}

		expiredAt, err := utils.ConvertDateTimeWibToUnixTimestamp(response.ExpiryTime)
		if err != nil {
			log.Info().Msg("Restoring product stock")
			go func() {
				err = s.WriteKafkaMessageWithKey(restoreProductMsgParsed, trxNumber.String())
				if err != nil {
					log.Error().Err(err).Str("component", "AddOrder").Msg("")
				}
			}()

			log.Error().Err(err).Str("component", "AddOrder").Msg("")
			return err
		}

		orderID, err := repo.AddOrder(ctx, domain.Order{
			PaymentMethodID:   int64(req.PaymentMethodID),
			Amount:            totalAmount,
			PaymentStatus:     "pending",
			TransactionNumber: trxNumber.String(),
			ExpiredAt:         expiredAt,
			CreatedAt:         time.Now().Unix(),
			UpdatedAt:         time.Now().Unix(),
		})
		if err != nil {
			log.Error().Err(err).Str("component", "AddOrder").Msg("")
			log.Info().Msg("Restoring product stock")
			go func() {
				err = s.WriteKafkaMessageWithKey(restoreProductMsgParsed, trxNumber.String())
				if err != nil {
					log.Error().Err(err).Str("component", "AddOrder").Msg("")
				}
			}()

			return err
		}

		for idx := range orderDetails {
			orderDetails[idx].OrderID = orderID
		}

		err = repo.AddOrderDetails(ctx, orderDetails)
		if err != nil {
			log.Error().Err(err).Str("component", "AddOrder").Msg("")
			log.Info().Msg("Restoring product stock")
			go func() {
				err = s.WriteKafkaMessageWithKey(restoreProductMsgParsed, trxNumber.String())
				if err != nil {
					log.Error().Err(err).Str("component", "AddOrder").Msg("")
				}
			}()
			return err
		}

		orderResponse.ID = orderID
		orderResponse.TransactionAmount = totalAmount
		orderResponse.PaymentStatus = "pending"
		orderResponse.PaymentMethodName = paymentMethod.Name
		orderResponse.PaymentExpiredAt = &expiredAt

		return nil
	})

	return orderResponse, err
}

func (s *OrderServiceImpl) MidtransPaymentWebhook(ctx context.Context, req dto.PaymentNotification) (err error) {
	order, err := s.repository.GetOrderByTransactionNumber(ctx, req.OrderID)
	if err != nil {
		return
	}

	if order.ExpiredAt < time.Now().Unix() {
		return errs.ErrPaymentExpired
	}

	if req.TransactionStatus == "capture" {
		if req.FraudStatus == "accept" {
			err = s.repository.UpdateOrderPaymentStatus(ctx, domain.Order{
				ID:            order.ID,
				PaymentStatus: "success",
			})
		}
	} else if req.TransactionStatus == "cancel" || req.TransactionStatus == "deny" || req.TransactionStatus == "expire" {
		err = s.repository.UpdateOrderPaymentStatus(ctx, domain.Order{
			ID:            order.ID,
			PaymentStatus: "expired",
		})
	}

	return
}

func (s *OrderServiceImpl) WriteKafkaMessageWithKey(msg []byte, key string) error {
	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		_, err := s.kafkaProducer.WriteMessages(
			kafka.Message{
				Key:   []byte(key),
				Value: msg,
			},
		)

		if err == nil {
			return nil
		}

		lastErr = err
		retryCount := attempt + 1
		log.Printf("Failed to write Kafka message (attempt %d/%d): %v", retryCount, maxRetries, err)

		if attempt < maxRetries-1 {
			// Exponential backoff: 1s, 2s, 4s...
			backoffDuration := time.Second * time.Duration(1<<attempt)
			time.Sleep(backoffDuration)
		}
	}

	return fmt.Errorf("failed to write Kafka message after %d attempts: %w", maxRetries, lastErr)
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
	if err != nil {
		return
	}

	for _, data := range datas {
		orderResponse = append(orderResponse, dto.OrderResponse{
			ID:                data.ID,
			PaymentStatus:     data.PaymentStatus,
			TransactionAmount: data.Amount,
			PaymentMethodName: data.PaymentMethod.Name,
			CreatedAt:         data.CreatedAt,
			TransactionNumber: data.TransactionNumber,
		})
	}

	response.Records = orderResponse

	return
}

func (s *OrderServiceImpl) GetOrderDetails(ctx context.Context, id int64) (response dto.OrderDetails, err error) {
	order, err := s.repository.GetOrderByOrderID(ctx, id)
	if err != nil {
		return
	}

	paymentMethod, err := s.repository.GetPaymentMethodByID(ctx, uint64(order.PaymentMethodID))
	if err != nil {
		return
	}

	response.ID = order.ID
	response.PaymentStatus = order.PaymentStatus
	response.TransactionAmount = order.Amount
	response.PaymentMethodName = paymentMethod.Name
	response.CreatedAt = order.CreatedAt
	response.TransactionNumber = order.TransactionNumber

	orderItems, err := s.repository.GetOrderDetailsByOrderID(ctx, id)
	if err != nil {
		return
	}

	for _, orderItem := range orderItems {
		response.OrderItems = append(response.OrderItems, dto.OrderItemResponse{
			ID:          orderItem.ID,
			ProductName: orderItem.ProductName,
			Quantity:    int(orderItem.Quantity),
			Price:       orderItem.Amount,
		})
	}

	return
}

func (s *OrderServiceImpl) RestoreExpiredPaymentItemStocks() {
	log.Info().Str("component", "RestoreExpiredPaymentItemStocks").Msg("cron starts")
	orders, err := s.repository.GetOrders(context.Background(), pkgdto.Filter{
		PaymentStatus: "pending",
		Expired:       true,
	})

	if err != nil {
		return
	}

	for _, order := range orders {
		order.PaymentStatus = "expired"
		err = s.repository.UpdateOrderPaymentStatus(context.Background(), order)
		if err != nil {
			return
		}

		orderDetails, err := s.repository.GetOrderDetailsByOrderID(context.Background(), order.ID)
		if err != nil {
			return
		}

		var orderRequest dto.OrderProductServiceRequest
		for _, item := range orderDetails {
			orderRequest.OrderItems = append(orderRequest.OrderItems, dto.OrderItem{
				ProductID: item.ProductID,
				Quantity:  int(item.Quantity),
			})
		}

		kafkaMsg := dto.KafkaMessage{
			EventType: "restore_product_stock",
			Data:      orderRequest,
		}

		fmt.Printf("%+v\n", kafkaMsg)
		jsonMsg, err := json.Marshal(kafkaMsg)
		if err != nil {
			log.Error().Err(err).Str("component", "RestoreExpiredPaymentItemStocks").Msg("")
			return
		}

		maxRetries := 3

		for i := 0; i < maxRetries; i++ {
			err = s.writeKafkaMessage(jsonMsg)
			if err == nil {
				break
			}
			log.Error().Err(err).Str("component", "RestoreExpiredPaymentItemStocks").Msg("")
			time.Sleep(time.Second * time.Duration(i+1)) // Exponential backoff
		}
	}

	log.Info().Str("component", "RestoreExpiredPaymentItemStocks").Msg("cron ends")
}
