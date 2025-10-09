package service

import (
	"context"
	"go-gin-high-concurrency/internal/model"
	"go-gin-high-concurrency/internal/repository"
	apperrors "go-gin-high-concurrency/pkg/app_errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderService interface {
	Create(ctx context.Context, req model.CreateOrderRequest) (*model.Order, error)
	List(ctx context.Context) ([]*model.Order, error)
	GetByID(ctx context.Context, id int) (*model.Order, error)
	Confirm(ctx context.Context, id int) error
	Cancel(ctx context.Context, id int) error
	Delete(ctx context.Context, id int) error
}

type OrderServiceImpl struct {
	pool             *pgxpool.Pool
	repository       repository.OrderRepository
	ticketRepository repository.TicketRepository
}

func NewOrderService(
	pool *pgxpool.Pool,
	orderRepository repository.OrderRepository,
	ticketRepository repository.TicketRepository,
) OrderService {
	return &OrderServiceImpl{
		pool:             pool,
		repository:       orderRepository,
		ticketRepository: ticketRepository,
	}
}

func (s *OrderServiceImpl) Create(ctx context.Context, req model.CreateOrderRequest) (*model.Order, error) {

	// 1. begin transaction
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// 2. check if ticket exists
	ticket, err := s.ticketRepository.FindByIDWithLock(ctx, tx, req.TicketID)
	if err != nil {
		return nil, err
	}

	// 3. check if ticket has enough stock
	if ticket.RemainingStock < req.Quantity {
		return nil, apperrors.ErrInsufficientStock
	}

	// 3. create order
	order := &model.Order{
		UserID:     req.UserID,
		TicketID:   req.TicketID,
		Quantity:   req.Quantity,
		TotalPrice: ticket.Price * float64(req.Quantity),
		Status:     model.OrderStatusPending,
	}

	order, err = s.repository.Create(ctx, tx, order)
	if err != nil {
		return nil, err
	}

	// 4. decrement ticket remaining stock
	err = s.ticketRepository.DecrementStock(ctx, tx, ticket.ID, req.Quantity)
	if err != nil {
		return nil, err
	}

	// 5. commit transaction
	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return order, nil
}

func (s *OrderServiceImpl) List(ctx context.Context) ([]*model.Order, error) {
	return s.repository.List(ctx)
}

func (s *OrderServiceImpl) GetByID(ctx context.Context, id int) (*model.Order, error) {
	return s.repository.FindByID(ctx, id)
}

func (s *OrderServiceImpl) Confirm(ctx context.Context, id int) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = s.repository.UpdateStatusWithLock(ctx, tx, id, model.OrderStatusConfirmed)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (s *OrderServiceImpl) Cancel(ctx context.Context, id int) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 2. update order status
	order, err := s.repository.UpdateStatusWithLock(ctx, tx, id, model.OrderStatusCancelled)
	if err != nil {
		return err
	}

	// 3. increment ticket remaining stock
	err = s.ticketRepository.IncrementStock(ctx, tx, order.TicketID, order.Quantity)
	if err != nil {
		return err
	}

	// 4. commit transaction
	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (s *OrderServiceImpl) Delete(ctx context.Context, id int) error {
	return s.repository.Delete(ctx, id)
}
