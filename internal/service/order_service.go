package service

import (
	"context"
	"go-gin-high-concurrency/internal/cache"
	"go-gin-high-concurrency/internal/model"
	"go-gin-high-concurrency/internal/queue"
	"go-gin-high-concurrency/internal/repository"
	apperrors "go-gin-high-concurrency/pkg/app_errors"
	"go-gin-high-concurrency/pkg/logger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type OrderService interface {
	// 創建訂單(Redis庫存管理)
	PrepareOrder(ctx context.Context, req model.CreateOrderRequest) (*model.Order, error)
	// 創建訂單(Queue持久化)
	DispatchOrder(ctx context.Context, order *model.Order) error
	OrderList(ctx context.Context) ([]*model.Order, error)
	GetOrderByOrderID(ctx context.Context, orderID uuid.UUID) (*model.Order, error)
	ConfirmOrderByOrderID(ctx context.Context, orderID uuid.UUID) error
	CancelOrderByOrderID(ctx context.Context, orderID uuid.UUID) error
	DeleteOrderByOrderID(ctx context.Context, orderID uuid.UUID) error
}

type OrderServiceImpl struct {
	pool             *pgxpool.Pool
	repository       repository.OrderRepository
	ticketRepository repository.TicketRepository
	inventoryManager cache.RedisTicketInventoryManager
	orderQueue       queue.OrderQueue
}

func NewOrderService(
	pool *pgxpool.Pool,
	orderRepository repository.OrderRepository,
	ticketRepository repository.TicketRepository,
	inventoryManager cache.RedisTicketInventoryManager,
	orderQueue queue.OrderQueue,
) OrderService {
	return &OrderServiceImpl{
		pool:             pool,
		repository:       orderRepository,
		ticketRepository: ticketRepository,
		inventoryManager: inventoryManager,
		orderQueue:       orderQueue,
	}
}

func (s *OrderServiceImpl) PrepareOrder(ctx context.Context, req model.CreateOrderRequest) (*model.Order, error) {
	// 1. 使用 Redis 庫存管理器檢查庫存
	result, price, err := s.inventoryManager.DecreStock(ctx, req.TicketID, req.Quantity, req.UserID)
	if err != nil {
		return nil, err
	}
	if !result {
		return nil, apperrors.ErrInsufficientStock
	}

	requestID := uuid.New().String()

	// 立即返回訂單資訊
	order := &model.Order{
		UserID:     req.UserID,
		RequestID:  requestID,
		TicketID:   req.TicketID,
		Quantity:   req.Quantity,
		TotalPrice: price * float64(req.Quantity),
		Status:     model.OrderStatusPending,
	}

	// 1. 嘗試發送 MQ：ctx跟隨請求的生命週期，用戶不等了就取消
	err = s.orderQueue.PublishOrder(ctx, order)
	if err != nil {
		logger.WithComponent("service").Error("failed to publish order", zap.Error(err))
		// MQ紀錄失敗，回滾庫存(絕對不能讓使用者搶到票, 所以不使用go routine)
		// 2. 回滾庫存：RollbackStock使用context.Background()傳遞, 確保RollbackStock一定會執行
		s.inventoryManager.RollbackStock(context.Background(), req.TicketID, req.Quantity, req.UserID)
		return nil, apperrors.ErrInternalServerError
	}

	return order, nil
}

func (s *OrderServiceImpl) DispatchOrder(ctx context.Context, order *model.Order) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 寫入訂單到資料庫
	createdOrder, err := s.repository.Create(ctx, tx, order)
	if err != nil {
		return err
	}

	// 更新票券庫存
	ticket, err := s.ticketRepository.FindByID(ctx, createdOrder.TicketID)
	if err != nil {
		return err
	}

	err = s.ticketRepository.DecrementStock(ctx, tx, ticket.ID, createdOrder.Quantity)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *OrderServiceImpl) OrderList(ctx context.Context) ([]*model.Order, error) {
	return s.repository.List(ctx)
}

func (s *OrderServiceImpl) GetOrderByOrderID(ctx context.Context, orderID uuid.UUID) (*model.Order, error) {
	return s.repository.FindByOrderID(ctx, orderID)
}

func (s *OrderServiceImpl) ConfirmOrderByOrderID(ctx context.Context, orderID uuid.UUID) error {
	order, err := s.repository.FindByOrderID(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status != model.OrderStatusPending {
		return apperrors.ErrInvalidOrderStatus
	}
	return s.confirmOrderByID(ctx, order.ID)
}

func (s *OrderServiceImpl) CancelOrderByOrderID(ctx context.Context, orderID uuid.UUID) error {
	order, err := s.repository.FindByOrderID(ctx, orderID)
	if err != nil {
		return err
	}
	if order.Status != model.OrderStatusPending {
		return apperrors.ErrInvalidOrderStatus
	}
	return s.cancelOrderByID(ctx, order.ID)
}

func (s *OrderServiceImpl) DeleteOrderByOrderID(ctx context.Context, orderID uuid.UUID) error {
	order, err := s.repository.FindByOrderID(ctx, orderID)
	if err != nil {
		return err
	}
	return s.repository.Delete(ctx, order.ID)
}

func (s *OrderServiceImpl) confirmOrderByID(ctx context.Context, id int) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = s.repository.UpdateStatusWithLock(ctx, tx, id, model.OrderStatusConfirmed)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *OrderServiceImpl) cancelOrderByID(ctx context.Context, id int) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	order, err := s.repository.UpdateStatusWithLock(ctx, tx, id, model.OrderStatusCancelled)
	if err != nil {
		return err
	}
	err = s.ticketRepository.IncrementStock(ctx, tx, order.TicketID, order.Quantity)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
