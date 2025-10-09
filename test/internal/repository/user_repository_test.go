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

func TestUserRepository_Create(t *testing.T) {
	repo := repository.NewUserRepository(getTestDB())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		user := &model.User{
			Name:  "Test User",
			Email: "test@example.com",
		}

		created, err := repo.Create(ctx, user)

		require.NoError(t, err)
		assert.NotZero(t, created.ID)
		assert.Equal(t, "Test User", created.Name)
		assert.Equal(t, "test@example.com", created.Email)
		assert.NotZero(t, created.CreatedAt)
		assert.NotZero(t, created.UpdatedAt)
	})
}

func TestUserRepository_FindByID(t *testing.T) {
	repo := repository.NewUserRepository(getTestDB())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		userID := createTestUser(t, "Test User", "test@example.com")

		found, err := repo.FindByID(ctx, userID)

		require.NoError(t, err)
		assert.Equal(t, userID, found.ID)
		assert.Equal(t, "Test User", found.Name)
		assert.Equal(t, "test@example.com", found.Email)
	})

	t.Run("NotFound", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		_, err := repo.FindByID(ctx, 99999)

		require.Error(t, err)
	})
}

func TestUserRepository_FindByEmail(t *testing.T) {
	repo := repository.NewUserRepository(getTestDB())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		userID := createTestUser(t, "Test User", "test@example.com")

		found, err := repo.FindByEmail(ctx, "test@example.com")

		require.NoError(t, err)
		assert.Equal(t, userID, found.ID)
		assert.Equal(t, "Test User", found.Name)
		assert.Equal(t, "test@example.com", found.Email)
	})

	t.Run("NotFound", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		_, err := repo.FindByEmail(ctx, "notexist@example.com")

		require.Error(t, err)
	})
}

func TestUserRepository_List(t *testing.T) {
	repo := repository.NewUserRepository(getTestDB())
	ctx := context.Background()

	t.Run("EmptyList", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		users, err := repo.List(ctx)

		require.NoError(t, err)
		assert.Empty(t, users)
	})

	t.Run("OrderByCreatedAtDesc", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		userID1 := createTestUser(t, "User A", "userA@example.com")
		userID2 := createTestUser(t, "User B", "userB@example.com")
		userID3 := createTestUser(t, "User C", "userC@example.com")

		users, err := repo.List(ctx)

		require.NoError(t, err)
		assert.Len(t, users, 3)
		assert.Equal(t, userID3, users[0].ID)
		assert.Equal(t, userID2, users[1].ID)
		assert.Equal(t, userID1, users[2].ID)
	})
}

func TestUserRepository_Update(t *testing.T) {
	repo := repository.NewUserRepository(getTestDB())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		userID := createTestUser(t, "Original Name", "test@example.com")

		newName := "Updated Name"
		params := repository.UpdateUserParams{
			Name: &newName,
		}

		updated, err := repo.Update(ctx, userID, params)

		require.NoError(t, err)
		assert.Equal(t, "Updated Name", updated.Name)
		assert.Equal(t, "test@example.com", updated.Email) // Email 不變
		assert.NotEqual(t, updated.CreatedAt, updated.UpdatedAt)
	})

	t.Run("NotFound", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		newName := "Won't Update"
		params := repository.UpdateUserParams{Name: &newName}

		_, err := repo.Update(ctx, 99999, params)

		require.Error(t, err)
	})

	t.Run("EmptyParams", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		userID := createTestUser(t, "Test User", "test@example.com")
		params := repository.UpdateUserParams{}

		_, err := repo.Update(ctx, userID, params)

		require.Error(t, err)
		assert.Equal(t, apperrors.ErrInvalidInput, err)
	})
}

func TestUserRepository_Delete(t *testing.T) {
	repo := repository.NewUserRepository(getTestDB())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		userID := createTestUser(t, "To Delete", "delete@example.com")

		err := repo.Delete(ctx, userID)
		require.NoError(t, err)

		// 驗證軟刪除後無法查到
		_, err = repo.FindByID(ctx, userID)
		require.Error(t, err)
	})

	t.Run("NotFound", func(t *testing.T) {
		cleanup := setupTestWithTruncate(t)
		defer cleanup()

		err := repo.Delete(ctx, 99999)

		require.Error(t, err)
		assert.Equal(t, apperrors.ErrUserNotFound, err)
	})
}
