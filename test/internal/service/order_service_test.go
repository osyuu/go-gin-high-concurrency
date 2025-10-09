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
		ticketID := createTestTicket(t, 1001, "Concert", 100)

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
		ticketID := createTestTicket(t, 1002, "Concert", 10)

		req := model.CreateOrderRequest{
			UserID:   userID,
			TicketID: ticketID,
			Quantity: 20, // 超過庫存
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
		ticketID := createTestTicket(t, 1001, "Concert", 100)

		// 先創建訂單
		req := model.CreateOrderRequest{
			UserID:   userID,
			TicketID: ticketID,
			Quantity: 10,
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
		assert.Equal(t, 100, ticket.RemainingStock) // 90 + 10 = 100
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
		ticketID := createTestTicket(t, 1001, "Concert", 100)

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
