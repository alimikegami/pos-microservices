package dto

type KafkaMessage struct {
	EventType string      `json:"event_type"`
	Data      interface{} `json:"data"`
}

type User struct {
	ID         int64  `json:"id"`
	ExternalID string `json:"external_id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
}

type Product struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Quantity    uint64  `json:"quantity"`
	Description string  `json:"description"`
	UserID      string  `json:"user_id"`
	UserName    string  `json:"user_name"`
	Price       float64 `json:"price"`
}
