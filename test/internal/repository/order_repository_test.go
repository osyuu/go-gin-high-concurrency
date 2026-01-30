package repository

import (
	"context"
	"testing"

	"go-gin-high-concurrency/internal/model"
	"go-gin-high-concurrency/internal/repository"
	apperrors "go-gin-high-concurrency/pkg/app_errors"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderRepository_Create(t *testing.T) {
	repo := repository.NewOrderRepository(getTestDB())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		userID := createTestUser(t, "Test User", "test@example.com")
		eventID := createTestEvent(t, "Test Event")
		ticketID := createTestTicket(t, eventID, "Test Event", 100)

		order := &model.Order{
			UserID:     userID,
			TicketID:   ticketID,
			Quantity:   1,
			TotalPrice: 100.0,
			Status:     model.OrderStatusPending,
		}

		tx, txCleanup := setupTestWithTransaction(t)
		defer txCleanup()

		createdOrder, err := repo.Create(ctx, tx, order)

		require.NoError(t, err)
		assert.NotZero(t, createdOrder.ID)
		assert.Equal(t, userID, createdOrder.UserID)
		assert.Equal(t, ticketID, createdOrder.TicketID)
		assert.Equal(t, 1, createdOrder.Quantity)
		assert.Equal(t, 100.0, createdOrder.TotalPrice)
		assert.Equal(t, model.OrderStatusPending, createdOrder.Status)
		assert.NotZero(t, createdOrder.CreatedAt)
		assert.NotZero(t, createdOrder.UpdatedAt)
	})
}

func TestOrderRepository_FindByID(t *testing.T) {
	repo := repository.NewOrderRepository(getTestDB())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		userID := createTestUser(t, "Test User", "test@example.com")
		eventID := createTestEvent(t, "Test Event")
		ticketID := createTestTicket(t, eventID, "Test Event", 50)
		orderID := createTestOrder(t, userID, ticketID, 1, 100.0, model.OrderStatusPending)

		found, err := repo.FindByID(ctx, orderID)

		require.NoError(t, err)
		assert.Equal(t, orderID, found.ID)
		assert.Equal(t, userID, found.UserID)
		assert.Equal(t, ticketID, found.TicketID)
	})

	t.Run("NotFound", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		_, err := repo.FindByID(ctx, 99999)

		require.Error(t, err)
		assert.Equal(t, apperrors.ErrOrderNotFound, err)
	})
}

func TestOrderRepository_FindByUserID(t *testing.T) {
	repo := repository.NewOrderRepository(getTestDB())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		user1 := createTestUser(t, "User 1", "user1@example.com")
		user2 := createTestUser(t, "User 2", "user2@example.com")
		eventID := createTestEvent(t, "Concert")
		ticketID := createTestTicket(t, eventID, "Concert", 100)

		orderID1 := createTestOrder(t, user1, ticketID, 1, 100.0, model.OrderStatusPending)
		createTestOrder(t, user2, ticketID, 1, 100.0, model.OrderStatusPending)

		orders, err := repo.FindByUserID(ctx, user1)

		require.NoError(t, err)
		assert.Len(t, orders, 1)
		assert.Equal(t, orderID1, orders[0].ID)
	})

	t.Run("EmptyList", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		userID := createTestUser(t, "Test User", "test@example.com")
		orders, err := repo.FindByUserID(ctx, userID)

		require.NoError(t, err)
		assert.Empty(t, orders)
	})
}

func TestOrderRepository_FindByTicketID(t *testing.T) {
	repo := repository.NewOrderRepository(getTestDB())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		userID := createTestUser(t, "User", "user@example.com")
		e1 := createTestEvent(t, "Concert A")
		e2 := createTestEvent(t, "Concert B")
		ticket1 := createTestTicket(t, e1, "Concert A", 100)
		ticket2 := createTestTicket(t, e2, "Concert B", 100)

		orderID1 := createTestOrder(t, userID, ticket1, 1, 100.0, model.OrderStatusPending)
		createTestOrder(t, userID, ticket2, 1, 100.0, model.OrderStatusPending)

		orders, err := repo.FindByTicketID(ctx, ticket1)

		require.NoError(t, err)
		assert.Len(t, orders, 1)
		assert.Equal(t, orderID1, orders[0].ID)
	})

	t.Run("EmptyList", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		eventID := createTestEvent(t, "Concert")
		ticketID := createTestTicket(t, eventID, "Concert", 100)
		orders, err := repo.FindByTicketID(ctx, ticketID)

		require.NoError(t, err)
		assert.Empty(t, orders)
	})
}

func TestOrderRepository_List(t *testing.T) {
	repo := repository.NewOrderRepository(getTestDB())
	ctx := context.Background()

	t.Run("EmptyList", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		orders, err := repo.List(ctx)

		require.NoError(t, err)
		assert.Empty(t, orders)
	})

	t.Run("OrderByCreatedAtDesc", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		userID := createTestUser(t, "Test User", "test@example.com")
		eventID := createTestEvent(t, "Concert A")
		ticketID := createTestTicket(t, eventID, "Concert A", 100)

		orderID1 := createTestOrder(t, userID, ticketID, 1, 100.0, model.OrderStatusPending)
		orderID2 := createTestOrder(t, userID, ticketID, 1, 100.0, model.OrderStatusConfirmed)
		orderID3 := createTestOrder(t, userID, ticketID, 1, 100.0, model.OrderStatusCancelled)

		orders, err := repo.List(ctx)

		require.NoError(t, err)
		assert.Len(t, orders, 3)
		assert.Equal(t, orderID3, orders[0].ID)
		assert.Equal(t, orderID2, orders[1].ID)
		assert.Equal(t, orderID1, orders[2].ID)
	})

	t.Run("ExcludeDeleted", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		userID := createTestUser(t, "Test User", "test@example.com")
		eventID := createTestEvent(t, "Concert")
		ticketID := createTestTicket(t, eventID, "Concert", 100)

		orderID1 := createTestOrder(t, userID, ticketID, 1, 100.0, model.OrderStatusPending)
		orderID2 := createTestOrder(t, userID, ticketID, 1, 100.0, model.OrderStatusPending)

		// 删除第二个订单
		err := repo.Delete(ctx, orderID2)
		require.NoError(t, err)

		orders, err := repo.List(ctx)
		require.NoError(t, err)
		assert.Len(t, orders, 1)
		assert.Equal(t, orderID1, orders[0].ID)
	})
}

func TestOrderRepository_Delete(t *testing.T) {
	repo := repository.NewOrderRepository(getTestDB())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		userID := createTestUser(t, "Test User", "test@example.com")
		eventID := createTestEvent(t, "Concert A")
		ticketID := createTestTicket(t, eventID, "Concert A", 100)
		orderID := createTestOrder(t, userID, ticketID, 1, 100.0, model.OrderStatusPending)

		err := repo.Delete(ctx, orderID)
		require.NoError(t, err)

		// 驗證軟刪除後無法查到
		_, err = repo.FindByID(ctx, orderID)
		require.Error(t, err)
		assert.Equal(t, apperrors.ErrOrderNotFound, err)
	})

	t.Run("NotFound", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		err := repo.Delete(ctx, 99999)

		require.Error(t, err)
		assert.Equal(t, apperrors.ErrOrderNotFound, err)
	})
}

func TestOrderRepository_UpdateStatusWithLock(t *testing.T) {
	repo := repository.NewOrderRepository(getTestDB())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		userID := createTestUser(t, "Test User", "test@example.com")
		eventID := createTestEvent(t, "Concert A")
		ticketID := createTestTicket(t, eventID, "Concert A", 100)
		orderID := createTestOrder(t, userID, ticketID, 1, 100.0, model.OrderStatusPending)

		tx, txCleanup := setupTestWithTransaction(t)
		defer txCleanup()

		updated, err := repo.UpdateStatusWithLock(ctx, tx, orderID, model.OrderStatusConfirmed)

		require.NoError(t, err)
		assert.Equal(t, orderID, updated.ID)
		assert.Equal(t, model.OrderStatusConfirmed, updated.Status)
	})

	t.Run("NotFound", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		tx, txCleanup := setupTestWithTransaction(t)
		defer txCleanup()

		_, err := repo.UpdateStatusWithLock(ctx, tx, 99999, model.OrderStatusConfirmed)

		require.Error(t, err)
	})
}

func TestOrderRepository_GetUserTicketOrderCount(t *testing.T) {
	repo := repository.NewOrderRepository(getTestDB())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		userID := createTestUser(t, "Test User", "test@example.com")
		eventID := createTestEvent(t, "Concert")
		ticketID := createTestTicket(t, eventID, "Concert", 100)
		createTestOrder(t, userID, ticketID, 3, 3000.0, model.OrderStatusPending)

		tx, txCleanup := setupTestWithTransaction(t)
		defer txCleanup()

		count, err := repo.GetUserTicketOrderCount(ctx, tx, userID, ticketID)

		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	t.Run("NoOrders", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		userID := createTestUser(t, "Test User", "test@example.com")
		eventID := createTestEvent(t, "Concert")
		ticketID := createTestTicket(t, eventID, "Concert", 100)

		tx, txCleanup := setupTestWithTransaction(t)
		defer txCleanup()

		count, err := repo.GetUserTicketOrderCount(ctx, tx, userID, ticketID)

		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("ExcludeCancelled", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		userID := createTestUser(t, "Test User", "test@example.com")
		eventID := createTestEvent(t, "Concert")
		ticketID := createTestTicket(t, eventID, "Concert", 100)

		createTestOrder(t, userID, ticketID, 2, 2000.0, model.OrderStatusPending)
		createTestOrder(t, userID, ticketID, 3, 3000.0, model.OrderStatusCancelled)

		tx, txCleanup := setupTestWithTransaction(t)
		defer txCleanup()

		count, err := repo.GetUserTicketOrderCount(ctx, tx, userID, ticketID)

		require.NoError(t, err)
		assert.Equal(t, 2, count) // 不包括已取消的
	})
}

func TestOrderRepository_UniqueRequestID(t *testing.T) {
	repo := repository.NewOrderRepository(getTestDB())
	ctx := context.Background()

	t.Run("DuplicateRequestID_ShouldFail", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		userID := createTestUser(t, "User A", "a@example.com")
		eventID := createTestEvent(t, "Event")
		ticketID := createTestTicket(t, eventID, "Event", 100)

		// 共用的 request_id
		sharedRequestID := "same-request-id-123"

		// 1. 建立第一筆訂單，應該成功
		order1 := &model.Order{
			RequestID:  sharedRequestID,
			UserID:     userID,
			TicketID:   ticketID,
			Quantity:   1,
			TotalPrice: 100.0,
			Status:     model.OrderStatusPending,
		}

		tx1, txCleanup1 := setupTestWithTransaction(t)
		_, err := repo.Create(ctx, tx1, order1)
		require.NoError(t, err)
		tx1.Commit(ctx) // 必須 Commit 才會正式寫入索引
		txCleanup1()

		// 2. 嘗試建立第二筆「相同 RequestID」的訂單，應該失敗
		order2 := &model.Order{
			RequestID:  sharedRequestID, // 使用重複的 ID
			UserID:     userID,
			TicketID:   ticketID,
			Quantity:   2,
			TotalPrice: 200.0,
			Status:     model.OrderStatusPending,
		}

		tx2, txCleanup2 := setupTestWithTransaction(t)
		defer txCleanup2() // 這裡失敗後會自動 Rollback

		_, err = repo.Create(ctx, tx2, order2)

		// 斷言：這裡必須噴出 error
		assert.Error(t, err)
		// 甚至可以檢查錯誤訊息是否包含 unique constraint 的關鍵字
		assert.Contains(t, err.Error(), "unique")
		assert.Contains(t, err.Error(), "request_id")
	})
}

/* 輔助函數 */

// createTestOrder 創建測試用 order，回傳 orders.id
func createTestOrder(t *testing.T, userID, ticketID int, quantity int, totalPrice float64, status model.OrderStatus) int {
	t.Helper()
	ctx := context.Background()
	query := `
		INSERT INTO orders (request_id, user_id, ticket_id, quantity, total_price, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	var id int
	err := testDB.QueryRow(ctx, query, uuid.New().String(), userID, ticketID, quantity, totalPrice, status).Scan(&id)
	require.NoError(t, err)
	return id
}
