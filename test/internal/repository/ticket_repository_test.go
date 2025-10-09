package repository

import (
	"context"
	"testing"

	"go-gin-high-concurrency/internal/model"
	"go-gin-high-concurrency/internal/repository"
	apperrors "go-gin-high-concurrency/pkg/app_errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTicketRepository_Create(t *testing.T) {
	cleanup := setupTestWithTruncate(t)
	defer cleanup()

	repo := repository.NewTicketRepository(getTestDB())
	ctx := context.Background()

	ticket := &model.Ticket{
		EventID:        1001,
		EventName:      "Test Concert 2025",
		Price:          1500.0,
		TotalStock:     100,
		RemainingStock: 100,
		MaxPerUser:     5,
	}

	created, err := repo.Create(ctx, ticket)

	require.NoError(t, err)
	assert.NotZero(t, created.ID)
	assert.Equal(t, 1001, created.EventID)
	assert.Equal(t, "Test Concert 2025", created.EventName)
	assert.Equal(t, 1500.0, created.Price)
	assert.Equal(t, 100, created.TotalStock)
	assert.Equal(t, 100, created.RemainingStock)
	assert.Equal(t, 5, created.MaxPerUser)
	assert.NotZero(t, created.CreatedAt)
	assert.NotZero(t, created.UpdatedAt)
}

func TestTicketRepository_FindByID(t *testing.T) {
	repo := repository.NewTicketRepository(getTestDB())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		ticketID := createTestTicket(t, 1002, "Test Event", 50)

		found, err := repo.FindByID(ctx, ticketID)

		require.NoError(t, err)
		assert.Equal(t, ticketID, found.ID)
		assert.Equal(t, 1002, found.EventID)
		assert.Equal(t, "Test Event", found.EventName)
		assert.Equal(t, 50, found.TotalStock)
	})

	t.Run("NotFound", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		_, err := repo.FindByID(ctx, 99999)

		require.Error(t, err)
		assert.Equal(t, apperrors.ErrTicketNotFound, err)
	})
}

func TestTicketRepository_List(t *testing.T) {
	repo := repository.NewTicketRepository(getTestDB())
	ctx := context.Background()

	t.Run("EmptyList", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		tickets, err := repo.List(ctx)

		require.NoError(t, err)
		assert.Empty(t, tickets)
	})

	t.Run("OrderByCreatedAtDesc", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		createTestTicket(t, 1001, "Concert A", 100)
		createTestTicket(t, 1002, "Concert B", 200)
		createTestTicket(t, 1003, "Concert C", 300)

		tickets, err := repo.List(ctx)

		require.NoError(t, err)
		assert.Len(t, tickets, 3)
		assert.Equal(t, 1003, tickets[0].EventID)
		assert.Equal(t, 1002, tickets[1].EventID)
		assert.Equal(t, 1001, tickets[2].EventID)
	})
}

func TestTicketRepository_Update(t *testing.T) {
	repo := repository.NewTicketRepository(getTestDB())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		ticketID := createTestTicket(t, 1001, "Original", 100)
		eventName := "Updated Concert"
		price := 3000.0
		maxPerUser := 8
		updates := repository.UpdateTicketParams{
			EventName:  &eventName,
			Price:      &price,
			MaxPerUser: &maxPerUser,
		}

		updated, err := repo.Update(ctx, ticketID, updates)

		require.NoError(t, err)
		assert.Equal(t, "Updated Concert", updated.EventName)
		assert.Equal(t, 3000.0, updated.Price)
		assert.Equal(t, 8, updated.MaxPerUser)
		assert.Equal(t, 100, updated.TotalStock) // 未更新的字段保持不变
	})

	t.Run("NotFound", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		eventName := "Won't Update"
		updates := repository.UpdateTicketParams{
			EventName: &eventName,
		}

		_, err := repo.Update(ctx, 99999, updates)

		require.Error(t, err)
		assert.Equal(t, apperrors.ErrTicketNotFound, err)
	})

	t.Run("EmptyMap", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		ticketID := createTestTicket(t, 1003, "Concert", 100)
		updates := repository.UpdateTicketParams{}

		_, err := repo.Update(ctx, ticketID, updates)

		require.Error(t, err)
		assert.Equal(t, apperrors.ErrInvalidInput, err)
	})
}

func TestTicketRepository_Delete(t *testing.T) {
	repo := repository.NewTicketRepository(getTestDB())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		ticketID := createTestTicket(t, 1001, "To Delete", 100)

		err := repo.Delete(ctx, ticketID)
		require.NoError(t, err)

		// 验证软删除后无法查到
		_, err = repo.FindByID(ctx, ticketID)
		require.Error(t, err)
		assert.Equal(t, apperrors.ErrTicketNotFound, err)
	})

	t.Run("NotFound", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		err := repo.Delete(ctx, 99999)

		require.Error(t, err)
		assert.Equal(t, apperrors.ErrTicketNotFound, err)
	})

	t.Run("AlreadyDeleted", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		ticketID := createTestTicket(t, 1002, "Already Deleted", 100)

		err := repo.Delete(ctx, ticketID)
		require.NoError(t, err)

		// 第二次删除应该失败
		err = repo.Delete(ctx, ticketID)
		require.Error(t, err)
		assert.Equal(t, apperrors.ErrTicketNotFound, err)
	})
}

func TestTicketRepository_FindByIDWithLock(t *testing.T) {
	repo := repository.NewTicketRepository(getTestDB())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		ticketID := createTestTicket(t, 1001, "Lock Test", 100)

		tx, txCleanup := setupTestWithTransaction(t)
		defer txCleanup()

		ticket, err := repo.FindByIDWithLock(ctx, tx, ticketID)

		require.NoError(t, err)
		assert.Equal(t, ticketID, ticket.ID)
		assert.Equal(t, 100, ticket.RemainingStock)
	})

	t.Run("NotFound", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		tx, txCleanup := setupTestWithTransaction(t)
		defer txCleanup()

		_, err := repo.FindByIDWithLock(ctx, tx, 99999)

		require.Error(t, err)
		assert.Equal(t, apperrors.ErrTicketNotFound, err)
	})
}

func TestTicketRepository_DecrementStock(t *testing.T) {
	repo := repository.NewTicketRepository(getTestDB())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		ticketID := createTestTicket(t, 1001, "Test Event", 100)

		tx, txCleanup := setupTestWithTransaction(t)
		defer txCleanup()

		err := repo.DecrementStock(ctx, tx, ticketID, 30)
		require.NoError(t, err)

		ticket, err := repo.FindByIDWithLock(ctx, tx, ticketID)
		require.NoError(t, err)
		assert.Equal(t, 70, ticket.RemainingStock)
	})

	// 庫存不足
	t.Run("InsufficientStock", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		ticketID := createTestTicket(t, 1002, "Test Event", 5)

		tx, txCleanup := setupTestWithTransaction(t)
		defer txCleanup()

		err := repo.DecrementStock(ctx, tx, ticketID, 10)

		require.Error(t, err)
		assert.Equal(t, apperrors.ErrInsufficientStock, err)
	})

	// 庫存正好是0
	t.Run("ExactStock", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		ticketID := createTestTicket(t, 1003, "Test Event", 50)

		tx, txCleanup := setupTestWithTransaction(t)
		defer txCleanup()

		err := repo.DecrementStock(ctx, tx, ticketID, 50)
		require.NoError(t, err)

		ticket, err := repo.FindByIDWithLock(ctx, tx, ticketID)
		require.NoError(t, err)
		assert.Equal(t, 0, ticket.RemainingStock)
	})

	t.Run("NotFound", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		tx, txCleanup := setupTestWithTransaction(t)
		defer txCleanup()

		err := repo.DecrementStock(ctx, tx, 99999, 10)

		require.Error(t, err)
		assert.Equal(t, apperrors.ErrInsufficientStock, err)
	})
}

func TestTicketRepository_IncrementStock(t *testing.T) {
	repo := repository.NewTicketRepository(getTestDB())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		// 创建一个已售出部分的票（100 张总库存，剩余 70 张）
		ticketID := createTestTicketWithStock(t, 1001, "Test Event", 100, 70)

		tx, txCleanup := setupTestWithTransaction(t)
		defer txCleanup()

		// 退票 10 张
		err := repo.IncrementStock(ctx, tx, ticketID, 10)
		require.NoError(t, err)

		ticket, err := repo.FindByIDWithLock(ctx, tx, ticketID)
		require.NoError(t, err)
		assert.Equal(t, 80, ticket.RemainingStock)
	})

	// 剛好等於總庫存
	t.Run("ExactToTotalStock", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		ticketID := createTestTicketWithStock(t, 1002, "Test Event", 100, 90)

		tx, txCleanup := setupTestWithTransaction(t)
		defer txCleanup()

		err := repo.IncrementStock(ctx, tx, ticketID, 10)
		require.NoError(t, err)

		ticket, err := repo.FindByIDWithLock(ctx, tx, ticketID)
		require.NoError(t, err)
		assert.Equal(t, 100, ticket.RemainingStock)
	})

	// 超過總庫存
	t.Run("CannotExceedTotalStock", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		ticketID := createTestTicketWithStock(t, 1003, "Test Event", 100, 90)

		tx, txCleanup := setupTestWithTransaction(t)
		defer txCleanup()

		err := repo.IncrementStock(ctx, tx, ticketID, 20)

		require.Error(t, err)
	})

	t.Run("NotFound", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		tx, txCleanup := setupTestWithTransaction(t)
		defer txCleanup()

		err := repo.IncrementStock(ctx, tx, 99999, 10)

		require.Error(t, err)
		assert.Equal(t, apperrors.ErrTicketNotFound, err)
	})
}

func TestTicketRepository_AddStock(t *testing.T) {
	repo := repository.NewTicketRepository(getTestDB())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		// total_stock = 100, remaining_stock = 80（已售出 20 张）
		ticketID := createTestTicketWithStock(t, 1001, "Concert", 100, 80)

		tx, txCleanup := setupTestWithTransaction(t)
		defer txCleanup()

		// 追加 50 张票
		err := repo.AddStock(ctx, tx, ticketID, 50)
		require.NoError(t, err)

		ticket, err := repo.FindByIDWithLock(ctx, tx, ticketID)
		require.NoError(t, err)
		assert.Equal(t, 150, ticket.TotalStock)     // 100 + 50
		assert.Equal(t, 130, ticket.RemainingStock) // 80 + 50
	})

	t.Run("AddToSoldOut", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		// total_stock = 100, remaining_stock = 0（已售罄）
		ticketID := createTestTicketWithStock(t, 1002, "Concert", 100, 0)

		tx, txCleanup := setupTestWithTransaction(t)
		defer txCleanup()

		// 追加 30 张票
		err := repo.AddStock(ctx, tx, ticketID, 30)
		require.NoError(t, err)

		ticket, err := repo.FindByIDWithLock(ctx, tx, ticketID)
		require.NoError(t, err)
		assert.Equal(t, 130, ticket.TotalStock)    // 100 + 30
		assert.Equal(t, 30, ticket.RemainingStock) // 0 + 30
	})

	t.Run("InvalidQuantity_Zero", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		ticketID := createTestTicket(t, 1003, "Concert", 100)

		tx, txCleanup := setupTestWithTransaction(t)
		defer txCleanup()

		err := repo.AddStock(ctx, tx, ticketID, 0)

		require.Error(t, err)
		assert.Equal(t, apperrors.ErrInvalidInput, err)
	})

	t.Run("InvalidQuantity_Negative", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		ticketID := createTestTicket(t, 1004, "Concert", 100)

		tx, txCleanup := setupTestWithTransaction(t)
		defer txCleanup()

		err := repo.AddStock(ctx, tx, ticketID, -10)

		require.Error(t, err)
		assert.Equal(t, apperrors.ErrInvalidInput, err)
	})

	t.Run("NotFound", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		tx, txCleanup := setupTestWithTransaction(t)
		defer txCleanup()

		err := repo.AddStock(ctx, tx, 99999, 50)

		require.Error(t, err)
		assert.Equal(t, apperrors.ErrTicketNotFound, err)
	})
}
