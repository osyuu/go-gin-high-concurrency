package service

import (
	"context"
	"testing"

	"go-gin-high-concurrency/internal/model"
	"go-gin-high-concurrency/internal/repository"
	"go-gin-high-concurrency/internal/service"
	apperrors "go-gin-high-concurrency/pkg/app_errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderService_Create(t *testing.T) {
	ctx := context.Background()

	orderRepo := repository.NewOrderRepository(getTestDB())
	ticketRepo := repository.NewTicketRepository(getTestDB())
	orderService := service.NewOrderService(getTestDB(), orderRepo, ticketRepo)

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		userID := createTestUser(t, "Test User", "test@example.com")
		ticketID := createTestTicket(t, 1001, "Concert", 100, 5)

		req := model.CreateOrderRequest{
			UserID:   userID,
			TicketID: ticketID,
			Quantity: 5,
		}

		order, err := orderService.Create(ctx, req)

		require.NoError(t, err)
		assert.NotZero(t, order.ID)
		assert.Equal(t, userID, order.UserID)
		assert.Equal(t, ticketID, order.TicketID)
		assert.Equal(t, 5, order.Quantity)
		assert.Equal(t, 5000.0, order.TotalPrice) // 1000 * 5
		assert.Equal(t, model.OrderStatusPending, order.Status)

		// 驗證庫存已減少
		ticket, _ := ticketRepo.FindByID(ctx, ticketID)
		assert.Equal(t, 95, ticket.RemainingStock) // 100 - 5
	})

	t.Run("InsufficientStock", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		userID := createTestUser(t, "Test User", "test@example.com")
		ticketID := createTestTicketWithStock(t, 1002, "Concert", 100, 10, 15)

		req := model.CreateOrderRequest{
			UserID:   userID,
			TicketID: ticketID,
			Quantity: 15, // 超過庫存
		}

		_, err := orderService.Create(ctx, req)

		require.Error(t, err)
		assert.Equal(t, apperrors.ErrInsufficientStock, err)

		// 驗證庫存未變（事務回滾）
		ticket, _ := ticketRepo.FindByID(ctx, ticketID)
		assert.Equal(t, 10, ticket.RemainingStock)
	})

	t.Run("TicketNotFound", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		userID := createTestUser(t, "Test User", "test@example.com")

		req := model.CreateOrderRequest{
			UserID:   userID,
			TicketID: 99999,
			Quantity: 1,
		}

		_, err := orderService.Create(ctx, req)

		require.Error(t, err)
		assert.Equal(t, apperrors.ErrTicketNotFound, err)
	})

	t.Run("ExceedsMaxPerUser", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		userID := createTestUser(t, "Test User", "test@example.com")
		ticketID := createTestTicket(t, 1003, "Concert", 100, 5) // max_per_user = 5

		// 第一次購買 3 張
		req1 := model.CreateOrderRequest{
			UserID:   userID,
			TicketID: ticketID,
			Quantity: 3,
		}
		_, err := orderService.Create(ctx, req1)
		require.NoError(t, err)

		// 第二次嘗試購買 3 張（總共 6 張，超過 max_per_user=5）
		req2 := model.CreateOrderRequest{
			UserID:   userID,
			TicketID: ticketID,
			Quantity: 3,
		}
		_, err = orderService.Create(ctx, req2)

		require.Error(t, err)
		assert.Equal(t, apperrors.ErrExceedsMaxPerUser, err)

		// 驗證庫存未變（第二次購買失敗，事務回滾）
		ticket, _ := ticketRepo.FindByID(ctx, ticketID)
		assert.Equal(t, 97, ticket.RemainingStock) // 100 - 3（只有第一次成功）
	})

	t.Run("MaxPerUserExact", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		userID := createTestUser(t, "Test User", "test@example.com")
		ticketID := createTestTicket(t, 1004, "Concert", 100, 5) // max_per_user = 5

		// 購買 5 張，剛好等於 max_per_user
		req := model.CreateOrderRequest{
			UserID:   userID,
			TicketID: ticketID,
			Quantity: 5,
		}
		order, err := orderService.Create(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, 5, order.Quantity)

		// 驗證庫存減少
		ticket, _ := ticketRepo.FindByID(ctx, ticketID)
		assert.Equal(t, 95, ticket.RemainingStock)
	})
}

func TestOrderService_Cancel(t *testing.T) {
	ctx := context.Background()

	orderRepo := repository.NewOrderRepository(getTestDB())
	ticketRepo := repository.NewTicketRepository(getTestDB())
	orderService := service.NewOrderService(getTestDB(), orderRepo, ticketRepo)

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		userID := createTestUser(t, "Test User", "test@example.com")
		ticketID := createTestTicket(t, 1001, "Concert", 100, 5)

		// 先創建訂單
		req := model.CreateOrderRequest{
			UserID:   userID,
			TicketID: ticketID,
			Quantity: 5,
		}
		order, err := orderService.Create(ctx, req)
		require.NoError(t, err)

		// 取消訂單
		err = orderService.Cancel(ctx, order.ID)
		require.NoError(t, err)

		// 驗證訂單狀態已改變
		cancelledOrder, err := orderRepo.FindByID(ctx, order.ID)
		require.NoError(t, err)
		assert.Equal(t, model.OrderStatusCancelled, cancelledOrder.Status)

		// 驗證庫存已退回
		ticket, err := ticketRepo.FindByID(ctx, ticketID)
		require.NoError(t, err)
		assert.Equal(t, 100, ticket.RemainingStock) // 95 + 5 = 100
	})

	t.Run("OrderNotFound", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		err := orderService.Cancel(ctx, 99999)

		require.Error(t, err)
	})
}

func TestOrderService_Confirm(t *testing.T) {
	ctx := context.Background()

	orderRepo := repository.NewOrderRepository(getTestDB())
	ticketRepo := repository.NewTicketRepository(getTestDB())
	orderService := service.NewOrderService(getTestDB(), orderRepo, ticketRepo)

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		userID := createTestUser(t, "Test User", "test@example.com")
		ticketID := createTestTicket(t, 1001, "Concert", 100, 5)

		// 先創建訂單
		req := model.CreateOrderRequest{
			UserID:   userID,
			TicketID: ticketID,
			Quantity: 5,
		}
		order, err := orderService.Create(ctx, req)
		require.NoError(t, err)

		// 確認訂單
		err = orderService.Confirm(ctx, order.ID)
		require.NoError(t, err)

		// 驗證訂單狀態已改變
		confirmedOrder, err := orderRepo.FindByID(ctx, order.ID)
		require.NoError(t, err)
		assert.Equal(t, model.OrderStatusConfirmed, confirmedOrder.Status)
	})

	t.Run("OrderNotFound", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		err := orderService.Confirm(ctx, 99999)

		require.Error(t, err)
	})
}
