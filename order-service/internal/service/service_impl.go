package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

	err = s.repository.HandleTrx(ctx, func(ctx context.Context, repo repository.OrderRepository) error {
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
		log.Info().Msg("making req to product service")

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
			return fmt.Errorf("error calling product price info service: %v", err)
		}

		if statusCode != http.StatusOK {
			return fmt.Errorf("product price info service returned non-OK status: %d", statusCode)
		}
		log.Info().Msg("item price has been retrieved")
		// Parse the price info response
		var priceInfoResponse dto.ProductResponse
		if err := json.Unmarshal(priceInfoBody, &priceInfoResponse); err != nil {
			return fmt.Errorf("error unmarshalling price info response: %v", err)
		}

		productReqPayloadJson, err := json.Marshal(productReqPayload)
		if err != nil {
			return fmt.Errorf("error marshalling price info request: %v", err)
		}

		log.Info().Msg("making req to product service")
		fmt.Printf("%+v\n", productReqPayload)
		updateProductStock := httpclient.HttpRequest{
			URL:    fmt.Sprintf("%s/api/v1/products/quantity", s.config.ProductCommandServiceHost),
			Method: "PUT",
			Body:   productReqPayloadJson,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}
		fmt.Println(updateProductStock.URL)
		statusCode, _, err = httpclient.SendRequest(updateProductStock)
		if err != nil {
			return fmt.Errorf("error calling product price info service: %v", err)
		}
		log.Info().Msg(fmt.Sprintf("%d", statusCode))

		if statusCode != 200 {
			return errs.ErrInternalServer
		}

		log.Info().Msg("Item stock has been reserved")
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

		log.Info().Msg("Making req to pg")
		response, err := s.midtransClient.ChargeTransaction(chargeReq)
		if response.StatusCode != "201" {
			return fmt.Errorf("payment gateway returned non-200 status: %s", response.StatusCode)
		}

		orderResponse.QRCode = &response.QRString
		expiredAt, err := utils.ConvertDateTimeWibToUnixTimestamp(response.ExpiryTime)
		if err != nil {
			log.Error().Err(err).Str("component", "AddOrder").Msg("")
			return err
		}
		orderResponse.PaymentExpiredAt = &expiredAt
		log.Info().Msg("Req to pg complete")

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
			return err
		}

		orderResponse.ID = orderID
		orderResponse.TransactionAmount = totalAmount
		orderResponse.PaymentStatus = "pending"

		for idx := range orderDetails {
			orderDetails[idx].OrderID = orderID
		}

		err = repo.AddOrderDetails(ctx, orderDetails)

		return err
	})

	if err != nil {
		fmt.Println(err)
		log.Info().Msg("Restoring product stock")
		kafkaMsg := dto.KafkaMessage{
			EventType: "restore_product_stock",
			Data:      productReqPayload,
		}

		jsonMsg, err := json.Marshal(kafkaMsg)
		if err != nil {
			return orderResponse, fmt.Errorf("failed to marshal Kafka message: %w", err)
		}

		maxRetries := 3
		for i := 0; i < maxRetries; i++ {
			err = s.writeKafkaMessageWithKey(jsonMsg, trxNumber.String())
			if err == nil {
				break
			}
			log.Printf("Failed to write Kafka message (attempt %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(time.Second * time.Duration(i+1)) // Exponential backoff
		}

		if err != nil {
			return orderResponse, fmt.Errorf("failed to write Kafka message after %d attempts: %w", maxRetries, err)
		}

		return orderResponse, err
	}

	return orderResponse, nil
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

func (s *OrderServiceImpl) writeKafkaMessageWithKey(msg []byte, key string) error {
	_, err := s.kafkaProducer.WriteMessages(
		kafka.Message{
			Key:   []byte(key),
			Value: msg,
		},
	)
	return err
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

		orderDetails, err := s.repository.GetOrderDetailsByOrderID(context.Background(), uint64(order.ID))
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
