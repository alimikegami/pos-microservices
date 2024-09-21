package dto

// Response struct representing the entire response
type ProductResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    Data   `json:"data"`
}

// Data struct representing the "data" field
type Data struct {
	Metadata Metadata        `json:"_metadata"`
	Records  []ProductRecord `json:"records"`
}

// Metadata struct representing the "_metadata" field
type Metadata struct {
	TotalCount int `json:"total_count"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
}

// Record struct representing individual product records
type ProductRecord struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Quantity    int     `json:"quantity"`
	Description string  `json:"description"`
	UserID      string  `json:"user_id"`
	UserName    string  `json:"user_name"`
	Price       float64 `json:"price"`
}
