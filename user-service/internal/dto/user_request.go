package dto

type UserRequest struct {
	ID       int64
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}