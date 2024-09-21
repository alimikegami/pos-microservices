package dto

type KafkaMessage struct {
	EventType string      `json:"event_type"`
	Data      interface{} `json:"data"`
}
