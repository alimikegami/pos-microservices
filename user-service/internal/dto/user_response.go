package dto

type LoginResponse struct {
	Token  string `json:"token"`
	UserID int64  `json:"user_id"`
}

type UserResponse struct {
	ID         int64  `json:"id"`
	ExternalID string `json:"external_id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
}
