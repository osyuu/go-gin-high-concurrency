package service

import (
	"context"
	"errors"
	"testing"

	cacheMocks "go-gin-high-concurrency/internal/cache/mocks"
	"go-gin-high-concurrency/internal/model"
	queueMocks "go-gin-high-concurrency/internal/queue/mocks"
	repoMocks "go-gin-high-concurrency/internal/repository/mocks"
	"go-gin-high-concurrency/internal/service"
	"go-gin-high-concurrency/pkg/app_errors"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupMock(t *testing.T) (*cacheMocks.MockRedisTicketInventoryManager, *queueMocks.MockOrderQueue, *repoMocks.MockOrderRepository, *repoMocks.MockTicketRepository) {
	mockInventory := cacheMocks.NewMockRedisTicketInventoryManager(t)
	mockQueue := queueMocks.NewMockOrderQueue(t)
	orderRepo := repoMocks.NewMockOrderRepository(t)
	ticketRepo := repoMocks.NewMockTicketRepository(t)
	return mockInventory, mockQueue, orderRepo, ticketRepo
}

func TestOrderService_PrepareOrder(t *testing.T) {
	ctx := context.Background()
	db := getTestDB()

	t.Run("Success", func(t *testing.T) {
		mockInventory, mockQueue, orderRepo, ticketRepo := setupMock(t)
		orderService := service.NewOrderService(db, orderRepo, ticketRepo, mockInventory, mockQueue)

		mockQueue.EXPECT().PublishOrder(ctx, mock.Anything).Return(nil).Once()
		mockInventory.EXPECT().DecreStock(ctx, 10, 2, 1).Return(true, 100.0, nil).Once()

		// 執行
		req := model.CreateOrderRequest{UserID: 1, TicketID: 10, Quantity: 2}
		order, err := orderService.PrepareOrder(ctx, req)

		// 驗證結果
		require.NoError(t, err)
		assert.NotNil(t, order)
		assert.Equal(t, model.OrderStatusPending, order.Status)

		// 驗證 Mock 是否按照預期運作
		mockInventory.AssertExpectations(t)
		mockQueue.AssertExpectations(t)
	})

	t.Run("Failed - ErrInsufficientStock", func(t *testing.T) {
		mockInventory, mockQueue, orderRepo, ticketRepo := setupMock(t)
		orderService := service.NewOrderService(db, orderRepo, ticketRepo, mockInventory, mockQueue)

		mockInventory.EXPECT().DecreStock(ctx, 10, 2, 1).Return(false, 0.0, app_errors.ErrInsufficientStock).Once()

		// 執行
		req := model.CreateOrderRequest{UserID: 1, TicketID: 10, Quantity: 2}
		_, err := orderService.PrepareOrder(ctx, req)

		// 驗證結果
		require.Error(t, err)
		assert.ErrorIs(t, err, app_errors.ErrInsufficientStock)

		// 驗證 Mock 是否按照預期運作
		mockInventory.AssertExpectations(t)
		mockQueue.AssertExpectations(t)
	})

	t.Run("Failed - RollbackStock", func(t *testing.T) {
		mockInventory, mockQueue, orderRepo, ticketRepo := setupMock(t)
		orderService := service.NewOrderService(db, orderRepo, ticketRepo, mockInventory, mockQueue)

		mockInventory.EXPECT().DecreStock(ctx, 10, 2, 1).Return(true, 100.0, nil).Once()
		mockInventory.EXPECT().RollbackStock(mock.Anything, 10, 2, 1).Return(nil).Once()
		mockQueue.EXPECT().PublishOrder(ctx, mock.Anything).Return(errors.New("failed to publish order")).Once()

		// 執行
		req := model.CreateOrderRequest{UserID: 1, TicketID: 10, Quantity: 2}
		_, err := orderService.PrepareOrder(ctx, req)

		// 驗證結果
		require.Error(t, err)
		assert.ErrorIs(t, err, app_errors.ErrInternalServerError)

		// 驗證 Mock 是否按照預期運作
		mockInventory.AssertExpectations(t)
		mockQueue.AssertExpectations(t)
	})

	t.Run("Failed - RollbackStock(Failed to rollback stock)", func(t *testing.T) {
		mockInventory, mockQueue, orderRepo, ticketRepo := setupMock(t)
		orderService := service.NewOrderService(db, orderRepo, ticketRepo, mockInventory, mockQueue)

		mockInventory.EXPECT().DecreStock(ctx, 10, 2, 1).Return(true, 100.0, nil).Once()
		mockInventory.EXPECT().RollbackStock(mock.Anything, 10, 2, 1).Return(errors.New("failed to rollback stock")).Once()
		mockQueue.EXPECT().PublishOrder(ctx, mock.Anything).Return(errors.New("failed to publish order")).Once()

		// 執行
		req := model.CreateOrderRequest{UserID: 1, TicketID: 10, Quantity: 2}
		_, err := orderService.PrepareOrder(ctx, req)

		// 驗證結果
		require.Error(t, err)
		assert.ErrorIs(t, err, app_errors.ErrInternalServerError)

		// 驗證 Mock 是否按照預期運作
		mockInventory.AssertExpectations(t)
		mockQueue.AssertExpectations(t)
	})
}

func TestOrderService_DispatchOrder(t *testing.T) {
	ctx := context.Background()
	db := getTestDB()

	t.Run("Success", func(t *testing.T) {
		mockInventory, mockQueue, orderRepo, ticketRepo := setupMock(t)
		orderService := service.NewOrderService(db, orderRepo, ticketRepo, mockInventory, mockQueue)

		expectedOrder := &model.Order{ID: 1, RequestID: "123", UserID: 1, TicketID: 10, Quantity: 2, TotalPrice: 100.0, Status: model.OrderStatusPending}
		// Mock
		orderRepo.EXPECT().Create(ctx, mock.Anything, mock.Anything).Return(expectedOrder, nil)
		ticketRepo.EXPECT().FindByID(ctx, 10).Return(&model.Ticket{ID: 10}, nil).Once()
		ticketRepo.EXPECT().DecrementStock(ctx, mock.Anything, 10, 2).Return(nil).Once()

		// 執行
		order := expectedOrder
		err := orderService.DispatchOrder(ctx, order)

		// 驗證結果
		require.NoError(t, err)
		assert.NotNil(t, order)

		// 驗證 Mock 是否按照預期運作
		orderRepo.AssertExpectations(t)
		ticketRepo.AssertExpectations(t)
	})

	t.Run("Failed - DecrementStock", func(t *testing.T) {
		mockInventory, mockQueue, orderRepo, ticketRepo := setupMock(t)
		orderService := service.NewOrderService(db, orderRepo, ticketRepo, mockInventory, mockQueue)

		// Mock
		orderRepo.EXPECT().Create(ctx, mock.Anything, mock.Anything).Return(&model.Order{ID: 1, UserID: 1, TicketID: 10, Quantity: 2, TotalPrice: 100.0, Status: model.OrderStatusPending}, nil).Once()
		ticketRepo.EXPECT().FindByID(ctx, 10).Return(&model.Ticket{ID: 10}, nil).Once()
		ticketRepo.EXPECT().DecrementStock(ctx, mock.Anything, 10, 2).Return(errors.New("db error")).Once()

		// 執行
		order := &model.Order{ID: 1, UserID: 1, TicketID: 10, Quantity: 2, TotalPrice: 100.0, Status: model.OrderStatusPending}
		err := orderService.DispatchOrder(ctx, order)

		// 驗證結果
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")

		// 驗證 Mock 是否按照預期運作
		orderRepo.AssertExpectations(t)
		ticketRepo.AssertExpectations(t)
	})
}

func TestOrderService_RemainingMethods(t *testing.T) {
	ctx := context.Background()
	db := getTestDB()

	// --- 1. OrderList ---
	t.Run("OrderList - Success", func(t *testing.T) {
		mockInventory, mockQueue, orderRepo, ticketRepo := setupMock(t)
		orderService := service.NewOrderService(db, orderRepo, ticketRepo, mockInventory, mockQueue)

		expectedOrders := []*model.Order{{ID: 1}, {ID: 2}}
		orderRepo.EXPECT().List(ctx).Return(expectedOrders, nil).Once()

		orders, err := orderService.OrderList(ctx)
		assert.NoError(t, err)
		assert.Len(t, orders, 2)
	})

	// --- 2. GetOrderByOrderID ---
	t.Run("GetOrderByOrderID - Success", func(t *testing.T) {
		mockInventory, mockQueue, orderRepo, ticketRepo := setupMock(t)
		orderService := service.NewOrderService(db, orderRepo, ticketRepo, mockInventory, mockQueue)

		orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		expectedOrder := &model.Order{ID: 1, OrderID: orderID}
		orderRepo.EXPECT().FindByOrderID(ctx, orderID).Return(expectedOrder, nil).Once()

		order, err := orderService.GetOrderByOrderID(ctx, orderID)
		assert.NoError(t, err)
		assert.Equal(t, orderID, order.OrderID)
	})

	// --- 3. ConfirmOrderByOrderID ---
	t.Run("ConfirmOrderByOrderID - Success", func(t *testing.T) {
		mockInventory, mockQueue, orderRepo, ticketRepo := setupMock(t)
		orderService := service.NewOrderService(db, orderRepo, ticketRepo, mockInventory, mockQueue)

		orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
		orderRepo.EXPECT().FindByOrderID(ctx, orderID).Return(&model.Order{ID: 1}, nil).Once()
		orderRepo.EXPECT().UpdateStatusWithLock(ctx, mock.Anything, 1, model.OrderStatusConfirmed).
			Return(&model.Order{ID: 1}, nil).Once()

		err := orderService.ConfirmOrderByOrderID(ctx, orderID)
		assert.NoError(t, err)
	})

	t.Run("ConfirmOrderByOrderID - Failed On Update", func(t *testing.T) {
		mockInventory, mockQueue, orderRepo, ticketRepo := setupMock(t)
		orderService := service.NewOrderService(db, orderRepo, ticketRepo, mockInventory, mockQueue)

		orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440002")
		orderRepo.EXPECT().FindByOrderID(ctx, orderID).Return(&model.Order{ID: 1}, nil).Once()
		orderRepo.EXPECT().UpdateStatusWithLock(ctx, mock.Anything, 1, model.OrderStatusConfirmed).
			Return(nil, errors.New("update error")).Once()

		err := orderService.ConfirmOrderByOrderID(ctx, orderID)
		assert.Error(t, err)
	})

	// --- 4. CancelOrderByOrderID ---
	t.Run("CancelOrderByOrderID - Success", func(t *testing.T) {
		mockInventory, mockQueue, orderRepo, ticketRepo := setupMock(t)
		orderService := service.NewOrderService(db, orderRepo, ticketRepo, mockInventory, mockQueue)

		orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440003")
		cancelledOrder := &model.Order{ID: 1, TicketID: 10, Quantity: 2}
		orderRepo.EXPECT().FindByOrderID(ctx, orderID).Return(&model.Order{ID: 1}, nil).Once()
		orderRepo.EXPECT().UpdateStatusWithLock(ctx, mock.Anything, 1, model.OrderStatusCancelled).
			Return(cancelledOrder, nil).Once()
		ticketRepo.EXPECT().IncrementStock(ctx, mock.Anything, 10, 2).
			Return(nil).Once()

		err := orderService.CancelOrderByOrderID(ctx, orderID)
		assert.NoError(t, err)
	})

	t.Run("CancelOrderByOrderID - Failed On IncrementStock", func(t *testing.T) {
		mockInventory, mockQueue, orderRepo, ticketRepo := setupMock(t)
		orderService := service.NewOrderService(db, orderRepo, ticketRepo, mockInventory, mockQueue)

		orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440004")
		cancelledOrder := &model.Order{ID: 1, TicketID: 10, Quantity: 2}
		orderRepo.EXPECT().FindByOrderID(ctx, orderID).Return(&model.Order{ID: 1}, nil).Once()
		orderRepo.EXPECT().UpdateStatusWithLock(ctx, mock.Anything, 1, model.OrderStatusCancelled).
			Return(cancelledOrder, nil).Once()
		ticketRepo.EXPECT().IncrementStock(ctx, mock.Anything, 10, 2).
			Return(errors.New("db error")).Once()

		err := orderService.CancelOrderByOrderID(ctx, orderID)
		assert.Error(t, err)
	})

	// --- 5. DeleteOrderByOrderID ---
	t.Run("DeleteOrderByOrderID - Success", func(t *testing.T) {
		mockInventory, mockQueue, orderRepo, ticketRepo := setupMock(t)
		orderService := service.NewOrderService(db, orderRepo, ticketRepo, mockInventory, mockQueue)

		orderID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440005")
		orderRepo.EXPECT().FindByOrderID(ctx, orderID).Return(&model.Order{ID: 1}, nil).Once()
		orderRepo.EXPECT().Delete(ctx, 1).Return(nil).Once()

		err := orderService.DeleteOrderByOrderID(ctx, orderID)
		assert.NoError(t, err)
	})
}
