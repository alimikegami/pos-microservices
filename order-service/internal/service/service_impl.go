package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/alimikegami/point-of-sales/order-service/internal/domain"
	"github.com/alimikegami/point-of-sales/order-service/internal/dto"
	"github.com/alimikegami/point-of-sales/order-service/internal/repository"
	"github.com/alimikegami/point-of-sales/order-service/pkg/errs"
	"github.com/alimikegami/point-of-sales/order-service/pkg/httpclient"
	"github.com/alimikegami/point-of-sales/order-service/pkg/utils"
	"github.com/google/uuid"
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/coreapi"
)

type OrderServiceImpl struct {
	repository     repository.OrderRepository
	midtransClient *coreapi.Client
}

func CreateOrderService(repository repository.OrderRepository, midtransClient *coreapi.Client) OrderService {
	return &OrderServiceImpl{
		repository:     repository,
		midtransClient: midtransClient,
	}
}

func (s *OrderServiceImpl) AddOrder(ctx context.Context, req dto.OrderRequest) (err error) {
	err = s.repository.HandleTrx(ctx, func(repo repository.OrderRepository) error {
		var orderDetails []domain.OrderDetail

		// Prepare product IDs for price info request
		productIDs := make([]string, len(req.OrderItems))
		for i, item := range req.OrderItems {
			productIDs[i] = item.ProductID
		}

		// Create and marshal the product price info request
		priceInfoReq := dto.ProductRequest{
			ProductIds: productIDs,
		}
		priceInfoJsonBody, err := json.Marshal(priceInfoReq)
		if err != nil {
			return fmt.Errorf("error marshalling price info request: %v", err)
		}

		// Make API call to get product price info
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

		// Prepare quantity reduction request
		var productReqPayload dto.OrderProductServiceRequest
		for _, item := range req.OrderItems {
			productReqPayload.OrderItems = append(productReqPayload.OrderItems, dto.OrderItem{
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
			})
		}

		// Marshal the quantity reduction request
		quantityReductionJsonBody, err := json.Marshal(productReqPayload)
		if err != nil {
			return fmt.Errorf("error marshalling quantity reduction request: %v", err)
		}

		// Make API call to reduce quantity
		quantityReductionReq := httpclient.HttpRequest{
			URL:    "http://localhost:8081/api/v1/products/quantity",
			Method: "PUT",
			Body:   quantityReductionJsonBody,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}

		statusCode, quantityReductionBody, err := httpclient.SendRequest(quantityReductionReq)
		if err != nil {
			return fmt.Errorf("error calling quantity reduction service: %v", err)
		}
		if statusCode != http.StatusOK {
			return fmt.Errorf("quantity reduction service returned non-OK status: %d", statusCode)
		}

		// Parse the quantity reduction response
		var quantityReductionResponse dto.ProductResponse
		if err := json.Unmarshal(quantityReductionBody, &quantityReductionResponse); err != nil {
			return fmt.Errorf("error unmarshalling quantity reduction response: %v", err)
		}

		// Calculate total amount and prepare items for charge request
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

		// Create transaction number
		trxNumber, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("error generating transaction number: %v", err)
		}

		// Create charge request
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

		fmt.Printf("%+v\n", chargeReq)
		// Process the charge
		response, err := s.midtransClient.ChargeTransaction(chargeReq)
		// if err !=  {
		// 	return fmt.Errorf("error processing charge: %v", err)
		// }

		// Check the response
		fmt.Printf("%+v\n", response)
		fmt.Println(response.StatusCode)
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

	return err
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
