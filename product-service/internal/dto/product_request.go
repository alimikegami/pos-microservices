package dto

type ProductRequest struct {
	ID          string
	Name        string `json:"name"`
	Quantity    uint64 `json:"quantity"`
	Description string `json:"description"`
	UserID      string
	UserName    string
	Price       float64 `json:"price"`
}

type OrderItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type OrderRequest struct {
	TransactionNumber string      `json:"transaction_number"`
	OrderItems        []OrderItem `json:"order_items"`
}
