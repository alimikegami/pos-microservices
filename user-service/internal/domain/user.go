package domain

type User struct {
	ID             int64
	Name           string
	Email          string
	ExternalID     string
	HashedPassword string
	CreatedAt      int64
	UpdatedAt      int64
	DeletedAt      *int64
}

type UserHistory struct {
	ID             int64
	Name           string
	Email          string
	HashedPassword string
	ExternalID     string
	UserID         int64
	CreatedAt      int64
	UpdatedAt      int64
	DeletedAt      *int64
	User           User
}
