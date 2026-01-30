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

func TestTicketRepository_Create(t *testing.T) {
	cleanup := setupTestWithTruncate(t)
	defer cleanup()

	eventID := createTestEvent(t, "Test Concert 2025")
	repo := repository.NewTicketRepository(getTestDB())
	ctx := context.Background()

	ticket := &model.Ticket{
		TicketID:       uuid.New(),
		EventID:        eventID,
		Name:           "Test Concert 2025",
		Price:          1500.0,
		TotalStock:     100,
		RemainingStock: 100,
		MaxPerUser:     5,
	}

	created, err := repo.Create(ctx, ticket)

	require.NoError(t, err)
	assert.NotZero(t, created.ID)
	assert.Equal(t, eventID, created.EventID)
	assert.Equal(t, "Test Concert 2025", created.Name)
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

		eventID := createTestEvent(t, "Test Event")
		ticketID := createTestTicket(t, eventID, "Test Event", 50)

		found, err := repo.FindByID(ctx, ticketID)

		require.NoError(t, err)
		assert.Equal(t, ticketID, found.ID)
		assert.Equal(t, eventID, found.EventID)
		assert.Equal(t, "Test Event", found.Name)
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

func TestTicketRepository_FindByTicketID(t *testing.T) {
	repo := repository.NewTicketRepository(getTestDB())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		eventID := createTestEvent(t, "Test Event")
		id := createTestTicket(t, eventID, "Test Event", 50)
		ticket, err := repo.FindByID(ctx, id)
		require.NoError(t, err)

		found, err := repo.FindByTicketID(ctx, ticket.TicketID)

		require.NoError(t, err)
		assert.Equal(t, id, found.ID)
		assert.Equal(t, ticket.TicketID, found.TicketID)
		assert.Equal(t, eventID, found.EventID)
		assert.Equal(t, "Test Event", found.Name)
		assert.Equal(t, 50, found.TotalStock)
	})

	t.Run("NotFound", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		_, err := repo.FindByTicketID(ctx, uuid.New())

		require.Error(t, err)
		assert.Equal(t, apperrors.ErrTicketNotFound, err)
	})

	t.Run("DeletedTicket_NotFound", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		eventID := createTestEvent(t, "To Delete")
		id := createTestTicket(t, eventID, "To Delete", 100)
		ticket, err := repo.FindByID(ctx, id)
		require.NoError(t, err)
		err = repo.Delete(ctx, ticket.TicketID)
		require.NoError(t, err)

		_, err = repo.FindByTicketID(ctx, ticket.TicketID)

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

		e1 := createTestEvent(t, "Concert A")
		e2 := createTestEvent(t, "Concert B")
		e3 := createTestEvent(t, "Concert C")
		createTestTicket(t, e1, "Concert A", 100)
		createTestTicket(t, e2, "Concert B", 200)
		createTestTicket(t, e3, "Concert C", 300)

		tickets, err := repo.List(ctx)

		require.NoError(t, err)
		assert.Len(t, tickets, 3)
		assert.Equal(t, e3, tickets[0].EventID)
		assert.Equal(t, e2, tickets[1].EventID)
		assert.Equal(t, e1, tickets[2].EventID)
	})
}

func TestTicketRepository_Update(t *testing.T) {
	repo := repository.NewTicketRepository(getTestDB())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		eventID := createTestEvent(t, "Original")
		id := createTestTicket(t, eventID, "Original", 100)
		ticket, err := repo.FindByID(ctx, id)
		require.NoError(t, err)
		eventName := "Updated Concert"
		price := 3000.0
		maxPerUser := 8
		updates := model.UpdateTicketParams{
			Name:       &eventName,
			Price:      &price,
			MaxPerUser: &maxPerUser,
		}

		updated, err := repo.Update(ctx, ticket.TicketID, updates)

		require.NoError(t, err)
		assert.Equal(t, "Updated Concert", updated.Name)
		assert.Equal(t, 3000.0, updated.Price)
		assert.Equal(t, 8, updated.MaxPerUser)
		assert.Equal(t, 100, updated.TotalStock) // 未更新的字段保持不变
	})

	t.Run("NotFound", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		eventName := "Won't Update"
		updates := model.UpdateTicketParams{
			Name: &eventName,
		}

		_, err := repo.Update(ctx, uuid.New(), updates)

		require.Error(t, err)
		assert.Equal(t, apperrors.ErrTicketNotFound, err)
	})

	t.Run("EmptyMap", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		eventID := createTestEvent(t, "Concert")
		id := createTestTicket(t, eventID, "Concert", 100)
		ticket, err := repo.FindByID(ctx, id)
		require.NoError(t, err)
		updates := model.UpdateTicketParams{}

		_, err = repo.Update(ctx, ticket.TicketID, updates)

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

		eventID := createTestEvent(t, "To Delete")
		id := createTestTicket(t, eventID, "To Delete", 100)
		ticket, err := repo.FindByID(ctx, id)
		require.NoError(t, err)

		err = repo.Delete(ctx, ticket.TicketID)
		require.NoError(t, err)

		// 验证软删除后无法查到
		_, err = repo.FindByID(ctx, id)
		require.Error(t, err)
		assert.Equal(t, apperrors.ErrTicketNotFound, err)
	})

	t.Run("NotFound", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		err := repo.Delete(ctx, uuid.New())

		require.Error(t, err)
		assert.Equal(t, apperrors.ErrTicketNotFound, err)
	})

	t.Run("AlreadyDeleted", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		eventID := createTestEvent(t, "Already Deleted")
		id := createTestTicket(t, eventID, "Already Deleted", 100)
		ticket, err := repo.FindByID(ctx, id)
		require.NoError(t, err)

		err = repo.Delete(ctx, ticket.TicketID)
		require.NoError(t, err)

		// 第二次删除应该失败
		err = repo.Delete(ctx, ticket.TicketID)
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

		eventID := createTestEvent(t, "Lock Test")
		ticketID := createTestTicket(t, eventID, "Lock Test", 100)

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

		eventID := createTestEvent(t, "Test Event")
		ticketID := createTestTicket(t, eventID, "Test Event", 100)

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

		eventID := createTestEvent(t, "Test Event")
		ticketID := createTestTicket(t, eventID, "Test Event", 5)

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

		eventID := createTestEvent(t, "Test Event")
		ticketID := createTestTicket(t, eventID, "Test Event", 50)

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
		eventID := createTestEvent(t, "Test Event")
		ticketID := createTestTicketWithStock(t, eventID, "Test Event", 100, 70)

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

		eventID := createTestEvent(t, "Test Event")
		ticketID := createTestTicketWithStock(t, eventID, "Test Event", 100, 90)

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

		eventID := createTestEvent(t, "Test Event")
		ticketID := createTestTicketWithStock(t, eventID, "Test Event", 100, 90)

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
		eventID := createTestEvent(t, "Concert")
		ticketID := createTestTicketWithStock(t, eventID, "Concert", 100, 80)

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
		eventID := createTestEvent(t, "Concert")
		ticketID := createTestTicketWithStock(t, eventID, "Concert", 100, 0)

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

		eventID := createTestEvent(t, "Concert")
		ticketID := createTestTicket(t, eventID, "Concert", 100)

		tx, txCleanup := setupTestWithTransaction(t)
		defer txCleanup()

		err := repo.AddStock(ctx, tx, ticketID, 0)

		require.Error(t, err)
		assert.Equal(t, apperrors.ErrInvalidInput, err)
	})

	t.Run("InvalidQuantity_Negative", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		eventID := createTestEvent(t, "Concert")
		ticketID := createTestTicket(t, eventID, "Concert", 100)

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

/* 輔助函數 */

// createTestTicket 創建測試用 ticket
func createTestTicket(t *testing.T, eventID int, eventName string, stock int) int {
	t.Helper()
	return createTestTicketWithStock(t, eventID, eventName, stock, stock)
}

// createTestTicketWithStock 創建測試用 ticket，可分別指定總庫存與剩餘庫存
func createTestTicketWithStock(t *testing.T, eventID int, eventName string, totalStock, remainingStock int) int {
	t.Helper()
	ctx := context.Background()
	query := `
		INSERT INTO tickets (event_id, name, price, total_stock, remaining_stock, max_per_user)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	var id int
	err := testDB.QueryRow(ctx, query, eventID, eventName, 1000.0, totalStock, remainingStock, 5).Scan(&id)
	require.NoError(t, err)
	return id
}
