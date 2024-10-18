package dto

import "github.com/alimikegami/point-of-sales/product-command-service/internal/dto"

type ElasticsearchResponse struct {
	Took     int        `json:"took"`
	TimedOut bool       `json:"timed_out"`
	Shards   ShardsInfo `json:"_shards"`
	Hits     HitsInfo   `json:"hits"`
}

type ShardsInfo struct {
	Total      int `json:"total"`
	Successful int `json:"successful"`
	Skipped    int `json:"skipped"`
	Failed     int `json:"failed"`
}

type HitsInfo struct {
	Total    TotalHitsInfo `json:"total"`
	MaxScore float64       `json:"max_score"`
	Hits     []Hit         `json:"hits"`
}

type TotalHitsInfo struct {
	Value    int    `json:"value"`
	Relation string `json:"relation"`
}

type Hit struct {
	Index  string              `json:"_index"`
	ID     string              `json:"_id"`
	Score  float64             `json:"_score"`
	Source dto.ProductResponse `json:"_source"`
}
