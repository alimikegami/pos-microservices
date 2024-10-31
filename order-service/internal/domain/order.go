package domain

type PaymentMethod struct {
	ID        uint64  `db:"id"`
	Name      string  `db:"name"`
	Channel   *string `db:"channel"`
	MDR       float64 `db:"mdr"`
	MDRType   string  `db:"mdr_type"`
	ImgURL    *string `db:"img_url"`
	CreatedAt int64   `db:"created_at"`
	UpdatedAt int64   `db:"updated_at"`
	DeletedAt *int64  `db:"deleted_at"`
}

type Order struct {
	ID                int64   `db:"id"`
	PaymentMethodID   int64   `db:"payment_method_id"`
	Amount            float64 `db:"amount"`
	MDRFee            float64 `db:"mdr_fee"`
	PaidAt            *int64  `db:"paid_at"`
	TransactionNumber string  `db:"transaction_number"`
	PaymentStatus     string  `db:"payment_status"`
	ExpiredAt         int64   `db:"expired_at"`
	CreatedAt         int64   `db:"created_at"`
	UpdatedAt         int64   `db:"updated_at"`
	DeletedAt         *int64  `db:"deleted_at"`
	OrderDetail       []OrderDetail
	PaymentMethod     PaymentMethod
}

type OrderDetail struct {
	ID          int64   `db:"id"`
	ProductID   string  `db:"product_id"`
	OrderID     int64   `db:"order_id"`
	Quantity    int64   `db:"quantity"`
	Amount      float64 `db:"amount"`
	ProductName string  `db:"product_name"`
	CreatedAt   int64   `db:"created_at"`
	UpdatedAt   int64   `db:"updated_at"`
	DeletedAt   *int64  `db:"deleted_at"`
	Order       Order
}
