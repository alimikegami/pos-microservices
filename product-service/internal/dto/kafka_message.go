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
