package repository

import (
	"context"
	"fmt"
	"go-gin-high-concurrency/internal/model"
	apperrors "go-gin-high-concurrency/pkg/app_errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepository interface {
	List(ctx context.Context) ([]*model.Order, error)
	FindByID(ctx context.Context, id int) (*model.Order, error)
	FindByUserID(ctx context.Context, userID int) ([]*model.Order, error)
	FindByTicketID(ctx context.Context, ticketID int) ([]*model.Order, error)
	Delete(ctx context.Context, id int) error

	// Transaction methods
	Create(ctx context.Context, tx pgx.Tx, order *model.Order) (*model.Order, error)
	UpdateStatusWithLock(ctx context.Context, tx pgx.Tx, id int, status model.OrderStatus) (*model.Order, error)
	GetUserTicketOrderCount(ctx context.Context, tx pgx.Tx, userID int, ticketID int) (int, error)
}

type OrderRepositoryImpl struct {
	pool *pgxpool.Pool
}

func NewOrderRepository(pool *pgxpool.Pool) OrderRepository {
	return &OrderRepositoryImpl{
		pool: pool,
	}
}

func (r *OrderRepositoryImpl) Create(ctx context.Context, tx pgx.Tx, order *model.Order) (*model.Order, error) {
	query := `
		INSERT INTO orders (
			user_id, ticket_id, quantity, total_price, status
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, ticket_id, quantity, total_price, status, created_at, updated_at
	`

	err := tx.QueryRow(ctx, query,
		order.UserID, order.TicketID, order.Quantity, order.TotalPrice, order.Status,
	).Scan(
		&order.ID,
		&order.UserID,
		&order.TicketID,
		&order.Quantity,
		&order.TotalPrice,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	return order, nil
}

func (r *OrderRepositoryImpl) List(ctx context.Context) ([]*model.Order, error) {
	query := `
		SELECT id, user_id, ticket_id, quantity, total_price, status,
		       created_at, updated_at, deleted_at
		FROM orders
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*model.Order

	for rows.Next() {
		var order model.Order
		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.TicketID,
			&order.Quantity,
			&order.TotalPrice,
			&order.Status,
			&order.CreatedAt,
			&order.UpdatedAt,
			&order.DeletedAt,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, &order)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func (r *OrderRepositoryImpl) FindByID(ctx context.Context, id int) (*model.Order, error) {
	query := `
		SELECT id, user_id, ticket_id, quantity, total_price, status,
		       created_at, updated_at, deleted_at
		FROM orders
		WHERE id = $1 AND deleted_at IS NULL
	`

	var order model.Order
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&order.ID,
		&order.UserID,
		&order.TicketID,
		&order.Quantity,
		&order.TotalPrice,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
		&order.DeletedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.ErrOrderNotFound
		}
		return nil, err
	}

	return &order, nil
}

func (r *OrderRepositoryImpl) FindByUserID(ctx context.Context, userID int) ([]*model.Order, error) {
	query := `
		SELECT id, user_id, ticket_id, quantity, total_price, status,
		       created_at, updated_at, deleted_at
		FROM orders
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*model.Order

	for rows.Next() {
		var order model.Order
		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.TicketID,
			&order.Quantity,
			&order.TotalPrice,
			&order.Status,
			&order.CreatedAt,
			&order.UpdatedAt,
			&order.DeletedAt,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, &order)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func (r *OrderRepositoryImpl) FindByTicketID(ctx context.Context, ticketID int) ([]*model.Order, error) {
	query := `
		SELECT id, user_id, ticket_id, quantity, total_price, status,
		       created_at, updated_at, deleted_at
		FROM orders
		WHERE ticket_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*model.Order

	for rows.Next() {
		var order model.Order
		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.TicketID,
			&order.Quantity,
			&order.TotalPrice,
			&order.Status,
			&order.CreatedAt,
			&order.UpdatedAt,
			&order.DeletedAt,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, &order)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func (r *OrderRepositoryImpl) UpdateStatusWithLock(
	ctx context.Context,
	tx pgx.Tx,
	id int,
	status model.OrderStatus,
) (*model.Order, error) {
	query := `
		UPDATE orders
		SET status = $1, updated_at = $2
		WHERE id = $3
		RETURNING id, user_id, ticket_id, quantity, total_price, status, created_at, updated_at
	`

	var order model.Order

	err := tx.QueryRow(ctx, query, status, time.Now().UTC(), id).Scan(
		&order.ID,
		&order.UserID,
		&order.TicketID,
		&order.Quantity,
		&order.TotalPrice,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}

	return &order, nil
}

func (r *OrderRepositoryImpl) Delete(ctx context.Context, id int) error {
	query := `
		UPDATE orders
		SET deleted_at = $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL
	`
	time := time.Now().UTC()
	result, err := r.pool.Exec(ctx, query, time, time, id)
	if err != nil {
		return err
	}

	// check if order exists and not already deleted
	if result.RowsAffected() == 0 {
		return apperrors.ErrOrderNotFound
	}

	return nil
}

func (r *OrderRepositoryImpl) GetUserTicketOrderCount(ctx context.Context, tx pgx.Tx, userID int, ticketID int) (int, error) {
	query := `
		SELECT COALESCE(SUM(quantity), 0)
		FROM orders
		WHERE user_id = $1 
		  AND ticket_id = $2 
		  AND status != $3
		  AND deleted_at IS NULL
	`

	var totalQuantity int
	err := tx.QueryRow(ctx, query, userID, ticketID, model.OrderStatusCancelled).Scan(&totalQuantity)
	if err != nil {
		return 0, err
	}

	return totalQuantity, nil
}
