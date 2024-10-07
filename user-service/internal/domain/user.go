package domain

type Role struct {
	ID        int64
	Name      string
	CreatedAt int64
	UpdatedAt int64
	DeletedAt *int64
}

type User struct {
	ID             int64  `db:"id"`
	Name           string `db:"name"`
	Email          string `db:"email"`
	ExternalID     string `db:"external_id"`
	HashedPassword string `db:"hashed_password"`
	RoleID         int64  `db:"role_id"`
	CreatedAt      int64  `db:"created_at"`
	UpdatedAt      int64  `db:"updated_at"`
	DeletedAt      *int64 `db:"deleted_at"`
	Role           Role
}

type UserHistory struct {
	ID             int64  `db:"id"`
	Name           string `db:"name"`
	Email          string `db:"email"`
	ExternalID     string `db:"external_id"`
	HashedPassword string `db:"hashed_password"`
	RoleID         int64  `db:"role_id"`
	CreatedAt      int64  `db:"created_at"`
	UpdatedAt      int64  `db:"updated_at"`
	DeletedAt      *int64 `db:"deleted_at"`
	UserID         int64  `db:"user_id"`
	User           User
	Role           Role
}
