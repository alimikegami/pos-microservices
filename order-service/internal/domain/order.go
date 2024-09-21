package domain

type PaymentMethod struct {
	ID        uint64 `gorm:"primaryKey"`
	Name      string
	Channel   string
	MDR       float64
	MDRType   string
	ImgURL    string
	CreatedAt int64
	UpdatedAt int64
	DeletedAt *int64
}

type Order struct {
	ID                int64 `gorm:"primaryKey"`
	PaymentMethodID   int64
	Amount            float64
	MDRFee            float64
	PaidAt            *int64
	TransactionNumber string
	PaymentStatus     string
	ExpiredAt         int64
	CreatedAt         int64
	UpdatedAt         int64
	DeletedAt         *int64
	OrderDetail       []OrderDetail
	PaymentMethod     PaymentMethod
}

type OrderDetail struct {
	ID          int64 `gorm:"primaryKey"`
	ProductID   string
	OrderID     int64
	Quantity    int64
	Amount      float64
	ProductName string
	CreatedAt   int64
	UpdatedAt   int64
	DeletedAt   *int64
	Order       Order
}
