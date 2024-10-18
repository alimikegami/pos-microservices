package dto

type ProductQuantityRequest struct {
	ProductID string
	Action    string `json:"action"`
	Quantity  uint64 `json:"quantity"`
}
