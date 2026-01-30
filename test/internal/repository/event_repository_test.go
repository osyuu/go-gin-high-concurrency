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

func TestEventRepository_Create(t *testing.T) {
	repo := repository.NewEventRepository(getTestDB())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		desc := "Outdoor live show"
		event := &model.Event{
			EventID:     uuid.New(),
			Name:        "Summer Concert 2025",
			Description: &desc,
		}

		created, err := repo.Create(ctx, event)

		require.NoError(t, err)
		assert.NotZero(t, created.ID)
		assert.Equal(t, event.EventID, created.EventID)
		assert.Equal(t, "Summer Concert 2025", created.Name)
		require.NotNil(t, created.Description)
		assert.Equal(t, "Outdoor live show", *created.Description)
		assert.NotZero(t, created.CreatedAt)
		assert.NotZero(t, created.UpdatedAt)
	})
}

func TestEventRepository_List(t *testing.T) {
	repo := repository.NewEventRepository(getTestDB())
	ctx := context.Background()

	t.Run("EmptyList", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		events, err := repo.List(ctx)

		require.NoError(t, err)
		assert.Empty(t, events)
	})

	t.Run("OrderByCreatedAtDesc", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		id1 := createTestEvent(t, "Event A")
		id2 := createTestEvent(t, "Event B")
		id3 := createTestEvent(t, "Event C")

		events, err := repo.List(ctx)

		require.NoError(t, err)
		require.Len(t, events, 3)
		// 後建立的在前（created_at DESC）
		assert.Equal(t, id3, events[0].ID)
		assert.Equal(t, id2, events[1].ID)
		assert.Equal(t, id1, events[2].ID)
		assert.Equal(t, "Event C", events[0].Name)
		assert.Equal(t, "Event B", events[1].Name)
		assert.Equal(t, "Event A", events[2].Name)
	})
}

func TestEventRepository_FindByID(t *testing.T) {
	repo := repository.NewEventRepository(getTestDB())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		eventID := createTestEvent(t, "Find Me")

		found, err := repo.FindByID(ctx, eventID)

		require.NoError(t, err)
		assert.Equal(t, eventID, found.ID)
		assert.Equal(t, "Find Me", found.Name)
	})

	t.Run("NotFound", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		_, err := repo.FindByID(ctx, 99999)

		require.Error(t, err)
		assert.ErrorIs(t, err, apperrors.ErrEventNotFound)
	})
}

func TestEventRepository_FindByEventID(t *testing.T) {
	repo := repository.NewEventRepository(getTestDB())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		desc := "Find by event_id"
		event := &model.Event{
			EventID:     uuid.New(),
			Name:        "UUID Lookup",
			Description: &desc,
		}
		created, err := repo.Create(ctx, event)
		require.NoError(t, err)

		found, err := repo.FindByEventID(ctx, created.EventID)

		require.NoError(t, err)
		assert.Equal(t, created.ID, found.ID)
		assert.Equal(t, created.EventID, found.EventID)
		assert.Equal(t, "UUID Lookup", found.Name)
	})

	t.Run("NotFound", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		_, err := repo.FindByEventID(ctx, uuid.New())

		require.Error(t, err)
		assert.ErrorIs(t, err, apperrors.ErrEventNotFound)
	})
}

func TestEventRepository_Update(t *testing.T) {
	repo := repository.NewEventRepository(getTestDB())
	ctx := context.Background()

	t.Run("Success_UpdateName", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		eventID := createTestEvent(t, "Original Name")
		newName := "Updated Name"
		params := model.UpdateEventParams{Name: &newName}

		updated, err := repo.Update(ctx, eventID, params)

		require.NoError(t, err)
		assert.Equal(t, eventID, updated.ID)
		assert.Equal(t, "Updated Name", updated.Name)
	})

	t.Run("Success_UpdateDescription", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		eventID := createTestEvent(t, "Event")
		desc := "New description"
		params := model.UpdateEventParams{Description: &desc}

		updated, err := repo.Update(ctx, eventID, params)

		require.NoError(t, err)
		assert.Equal(t, eventID, updated.ID)
		require.NotNil(t, updated.Description)
		assert.Equal(t, "New description", *updated.Description)
	})

	t.Run("Success_UpdateBoth", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		eventID := createTestEvent(t, "Old")
		name := "New Name"
		desc := "New Desc"
		params := model.UpdateEventParams{Name: &name, Description: &desc}

		updated, err := repo.Update(ctx, eventID, params)

		require.NoError(t, err)
		assert.Equal(t, "New Name", updated.Name)
		require.NotNil(t, updated.Description)
		assert.Equal(t, "New Desc", *updated.Description)
	})

	t.Run("NotFound", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		name := "Any"
		params := model.UpdateEventParams{Name: &name}

		_, err := repo.Update(ctx, 99999, params)

		require.Error(t, err)
		assert.ErrorIs(t, err, apperrors.ErrEventNotFound)
	})

	t.Run("InvalidInput_EmptyParams", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		eventID := createTestEvent(t, "Event")
		params := model.UpdateEventParams{}

		_, err := repo.Update(ctx, eventID, params)

		require.Error(t, err)
		assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
	})
}

/* 輔助函數：供 ticket_repository_test、order_repository_test 等引用 */

// createTestEvent 創建測試用 event，回傳 events.id（ticket 的 FK 需要先有 event）
func createTestEvent(t *testing.T, name string) int {
	t.Helper()
	ctx := context.Background()
	query := `INSERT INTO events (name) VALUES ($1) RETURNING id`
	var id int
	err := testDB.QueryRow(ctx, query, name).Scan(&id)
	require.NoError(t, err)
	return id
}
