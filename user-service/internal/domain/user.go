package domain

type Role struct {
	ID        int64
	Name      string
	CreatedAt int64
	UpdatedAt int64
	DeletedAt *int64
}

type User struct {
	ID             int64
	Name           string
	Email          string
	ExternalID     string
	HashedPassword string
	RoleID         int64
	CreatedAt      int64
	UpdatedAt      int64
	DeletedAt      *int64
	Role           Role
}

type UserHistory struct {
	ID             int64
	Name           string
	Email          string
	HashedPassword string
	ExternalID     string
	UserID         int64
	RoleID         int64
	CreatedAt      int64
	UpdatedAt      int64
	DeletedAt      *int64
	User           User
	Role           Role
}
