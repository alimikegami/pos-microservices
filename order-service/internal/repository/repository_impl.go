package repository

import (
	"context"
	"database/sql"

	"github.com/alimikegami/point-of-sales/order-service/internal/domain"
	pkgdto "github.com/alimikegami/point-of-sales/order-service/pkg/dto"
	"github.com/alimikegami/point-of-sales/order-service/pkg/errs"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

type OrderRepositoryImpl struct {
	db *sqlx.DB
	tx *sqlx.Tx
}

func CreateOrderRepository(db *sqlx.DB) OrderRepository {
	return &OrderRepositoryImpl{
		db: db,
	}
}

func (r *OrderRepositoryImpl) AddOrder(ctx context.Context, data domain.Order) (id int64, err error) {
	nstmt, err := r.tx.PrepareNamedContext(ctx, "INSERT INTO orders(payment_method_id, amount, mdr_fee, transaction_number, payment_status, expired_at, created_at, updated_at) VALUES (:payment_method_id, :amount, :mdr_fee, :transaction_number, :payment_status, :expired_at, :created_at, :updated_at) returning id")
	if err != nil {
		log.Error().Err(err).Str("component", "AddOrder").Msg("")
		return
	}

	err = nstmt.GetContext(ctx, &data.ID, data)
	if err != nil {
		log.Error().Err(err).Str("component", "AddOrder").Msg("")
		return
	}

	return data.ID, nil
}

func (r *OrderRepositoryImpl) AddOrderDetails(ctx context.Context, data []domain.OrderDetail) (err error) {
	_, err = r.tx.NamedExecContext(ctx, "INSERT INTO order_details(product_id, order_id, quantity, amount, product_name, created_at, updated_at) VALUES (:product_id, :order_id, :quantity, :amount, :product_name, :created_at, :updated_at)", data)
	if err != nil {
		log.Error().Err(err).Str("component", "AddOrderDetails").Msg("")
		return
	}

	return nil
}

func (r *OrderRepositoryImpl) GetOrderByTransactionNumber(ctx context.Context, transactionNumber string) (data domain.Order, err error) {
	row := r.db.QueryRowxContext(ctx, "SELECT * FROM orders WHERE transaction_number = $1 AND deleted_at IS NULL", transactionNumber)
	err = row.StructScan(&data)
	if err != nil {
		log.Error().Err(err).Str("component", "GetUserByEmail").Msg("")
		if err == sql.ErrNoRows {
			return data, nil
		}
		return data, errs.ErrInternalServer
	}

	return
}

func (r *OrderRepositoryImpl) UpdateOrderPaymentStatus(ctx context.Context, data domain.Order) (err error) {
	_, err = r.db.NamedExecContext(ctx, "UPDATE orders SET payment_status = :payment_status WHERE id=:id AND deleted_at IS NULL", data)
	if err != nil {
		log.Error().Err(err).Str("component", "UpdateOrderPaymentStatus").Msg("")
		return
	}

	return nil
}

func (r *OrderRepositoryImpl) GetOrders(ctx context.Context, filter pkgdto.Filter) (data []domain.Order, err error) {
	query := "SELECT * FROM orders WHERE deleted_at IS NULL"

	args := make(map[string]interface{})

	if filter.PaymentStatus != "" {
		query += " AND payment_status = :payment_status"
		args["payment_status"] = filter.PaymentStatus
	}

	if filter.Expired {
		query += " AND expired_at < EXTRACT(EPOCH FROM NOW())"
	}

	if filter.Limit != 0 && filter.Page != 0 {
		offset := (filter.Page - 1) * filter.Limit
		query += " LIMIT :limit OFFSET :offset"
		args["limit"] = filter.Limit
		args["offset"] = offset
	}

	nstmt, err := r.db.PrepareNamedContext(ctx, query)
	if err != nil {
		log.Error().Err(err).Str("component", "GetOrders").Msg("")
		return nil, err
	}

	err = nstmt.SelectContext(ctx, &data, args)
	if err != nil {
		log.Error().Err(err).Str("component", "GetOrders").Msg("")
		return nil, err
	}

	return
}

func (r *OrderRepositoryImpl) GetOrderDetailsByOrderID(ctx context.Context, id uint64) (data []domain.OrderDetail, err error) {
	query := "SELECT * FROM order_details WHERE order_id = :order_id"
	args := make(map[string]interface{})
	args["order_id"] = id

	nstmt, err := r.db.PrepareNamedContext(ctx, query)
	if err != nil {
		log.Error().Err(err).Str("component", "GetOrderDetailsByOrderID").Msg("")
		return nil, err
	}

	err = nstmt.SelectContext(ctx, &data, args)
	if err != nil {
		log.Error().Err(err).Str("component", "GetOrderDetailsByOrderID").Msg("")
		return nil, err
	}

	return
}

func (r *OrderRepositoryImpl) HandleTrx(ctx context.Context, fn func(ctx context.Context, repo OrderRepository) error) error {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	dqRepo := &OrderRepositoryImpl{
		tx: tx,
	}

	err = fn(ctx, dqRepo)

	if err != nil {
		return err
	}

	return nil
}
