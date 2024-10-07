package dto

type KafkaMessage struct {
	EventType string      `json:"event_type"`
	Data      interface{} `json:"data"`
}

type ProductServiceStockUpdate struct {
	TransactionNumber string `json:"transaction_number"`
	Status            bool   `json:"status"`
}
