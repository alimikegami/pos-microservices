package dto

type OrderResponse struct {
	ID                int64   `json:"id"`
	PaymentMethodName string  `json:"payment_method_name"`
	TransactionAmount float64 `json:"transaction_amount"`
	PaymentStatus     string  `json:"payment_status"`
	PaymentExpiredAt  *int64  `json:"payment_expired_at"`
	QRCode            *string `json:"qr_code"`
}
