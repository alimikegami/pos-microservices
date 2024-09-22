package dto

type Filter struct {
	Limit int    `query:"limit"`
	Page  int    `query:"page"`
	Q     string `query:"q"`
}
