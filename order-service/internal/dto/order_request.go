package dto

type OrderItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type OrderRequest struct {
	PaymentMethodID uint64 `json:"payment_method_id"`
	UserID          uint64
	OrderItems      []OrderItem `json:"order_items"`
}

type OrderProductServiceRequest struct {
	TransactionNumber string      `json:"transaction_number"`
	OrderItems        []OrderItem `json:"order_items"`
}
