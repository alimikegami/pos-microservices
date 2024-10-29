package dto

type OrderResponse struct {
	ID                int64   `json:"id"`
	PaymentMethodName string  `json:"payment_method_name"`
	TransactionAmount float64 `json:"transaction_amount"`
	PaymentStatus     string  `json:"payment_status"`
	PaymentExpiredAt  *int64  `json:"payment_expired_at"`
	QRCode            *string `json:"qr_code"`
	CreatedAt         int64   `json:"created_at"`
	TransactionNumber string  `json:"transaction_number"`
}

type OrderItemResponse struct {
	ID          int64   `json:"id"`
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
}

type OrderDetails struct {
	ID                int64               `json:"id"`
	PaymentMethodName string              `json:"payment_method_name"`
	TransactionAmount float64             `json:"transaction_amount"`
	PaymentStatus     string              `json:"payment_status"`
	PaymentExpiredAt  *int64              `json:"payment_expired_at"`
	QRCode            *string             `json:"qr_code"`
	CreatedAt         int64               `json:"created_at"`
	TransactionNumber string              `json:"transaction_number"`
	OrderItems        []OrderItemResponse `json:"order_items"`
}
