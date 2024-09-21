package dto

type ProductResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Quantity    uint64  `json:"quantity"`
	Description string  `json:"description"`
	UserID      string  `json:"user_id"`
	UserName    string  `json:"user_name"`
	Price       float64 `json:"price"`
}
