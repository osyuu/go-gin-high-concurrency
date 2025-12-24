package cache

import (
	"context"
	"fmt"
	"go-gin-high-concurrency/internal/cache"
	"go-gin-high-concurrency/pkg/app_errors"
	"strconv"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func verifyStock(t *testing.T, ctx context.Context, inventory cache.RedisTicketInventoryManager, ticketID int, expectedStock int) {
	t.Helper()
	stock, err := inventory.GetStock(ctx, ticketID)
	assert.NoError(t, err)
	assert.Equal(t, expectedStock, stock)
}

func verifyUserBought(t *testing.T, ctx context.Context, client *redis.Client, ticketID int, userID int, expectedBought int) {
	t.Helper()
	usersKey := fmt.Sprintf("ticket:%d:users", ticketID)
	bought, err := client.HGet(ctx, usersKey, strconv.Itoa(userID)).Int()
	if err == redis.Nil {
		bought = 0
		assert.Equal(t, expectedBought, bought)
		return
	}
	assert.NoError(t, err)
	assert.Equal(t, expectedBought, bought)
}

func TestTicketInventory_WarmUpInventory(t *testing.T) {
	ctx := context.Background()
	redis := getTestRdb()
	inventory := cache.NewRedisTicketInventoryManager(redis)
	clearRedis(ctx)
	t.Cleanup(func() {
		clearRedis(ctx)
	})

	t.Run("Success", func(t *testing.T) {
		defer clearRedis(ctx)
		err := inventory.WarmUpInventory(ctx, 1, 100, 100.5, 2)
		assert.NoError(t, err)
		info, err := inventory.GetInfo(ctx, 1)
		assert.NoError(t, err)
		assert.Equal(t, 100, info.Stock)
		assert.Equal(t, 100.5, info.Price)
		assert.Equal(t, 2, info.Limit)
	})
}

func TestTicketInventory_GetStock(t *testing.T) {
	ctx := context.Background()
	redis := getTestRdb()
	inventory := cache.NewRedisTicketInventoryManager(redis)
	clearRedis(ctx)
	t.Cleanup(func() {
		clearRedis(ctx)
	})

	t.Run("Success", func(t *testing.T) {
		defer clearRedis(ctx)
		err := inventory.WarmUpInventory(ctx, 1, 100, 100.5, 2)
		assert.NoError(t, err)
		stock, err := inventory.GetStock(ctx, 1)
		assert.NoError(t, err)
		assert.Equal(t, 100, stock)
	})

	t.Run("Failed - NotFound", func(t *testing.T) {
		defer clearRedis(ctx)
		stock, err := inventory.GetStock(ctx, 1)
		assert.Equal(t, app_errors.ErrTicketNotFound, err)
		assert.Equal(t, -1, stock)
	})
}

func TestTicketInventory_GetInfo(t *testing.T) {
	ctx := context.Background()
	redis := getTestRdb()
	inventory := cache.NewRedisTicketInventoryManager(redis)
	clearRedis(ctx)
	t.Cleanup(func() {
		clearRedis(ctx)
	})

	t.Run("Success", func(t *testing.T) {
		defer clearRedis(ctx)
		err := inventory.WarmUpInventory(ctx, 1, 100, 100.5, 2)
		assert.NoError(t, err)
		info, err := inventory.GetInfo(ctx, 1)
		assert.NoError(t, err)
		assert.Equal(t, 100, info.Stock)
		assert.Equal(t, 100.5, info.Price)
		assert.Equal(t, 2, info.Limit)
	})

	t.Run("Failed - NotFound", func(t *testing.T) {
		defer clearRedis(ctx)
		info, err := inventory.GetInfo(ctx, 1)
		assert.Equal(t, app_errors.ErrTicketNotFound, err)
		assert.Equal(t, cache.RedisTicketInfo{}, info)
	})
}

func TestTicketInventory_DecreStock(t *testing.T) {
	ctx := context.Background()
	redis := getTestRdb()
	inventory := cache.NewRedisTicketInventoryManager(redis)
	clearRedis(ctx)
	t.Cleanup(func() {
		clearRedis(ctx)
	})

	t.Run("Success", func(t *testing.T) {
		defer clearRedis(ctx)
		err := inventory.WarmUpInventory(ctx, 1, 100, 100.5, 2)
		assert.NoError(t, err)
		result, price, err := inventory.DecreStock(ctx, 1, 2, 1)
		assert.NoError(t, err)
		assert.True(t, result)
		assert.Equal(t, 100.5, price)

		// 驗證庫存
		verifyStock(t, ctx, inventory, 1, 98)

		// 驗證使用者購買紀錄
		verifyUserBought(t, ctx, redis, 1, 1, 2)
	})

	t.Run("Failed - InsufficientStock", func(t *testing.T) {
		defer clearRedis(ctx)
		err := inventory.WarmUpInventory(ctx, 1, 1, 100.5, 2)
		assert.NoError(t, err)
		result, price, err := inventory.DecreStock(ctx, 1, 2, 1)
		assert.Equal(t, app_errors.ErrInsufficientStock, err)
		assert.False(t, result)
		assert.Equal(t, 0.0, price)

		// 驗證庫存
		verifyStock(t, ctx, inventory, 1, 1)

		// 驗證使用者購買紀錄
		verifyUserBought(t, ctx, redis, 1, 1, 0)
	})

	t.Run("Failed - ExceedsMaxPerUser", func(t *testing.T) {
		defer clearRedis(ctx)
		err := inventory.WarmUpInventory(ctx, 1, 100, 100.5, 2)
		assert.NoError(t, err)
		result, price, err := inventory.DecreStock(ctx, 1, 3, 1)
		assert.Equal(t, app_errors.ErrExceedsMaxPerUser, err)
		assert.False(t, result)
		assert.Equal(t, 0.0, price)

		// 驗證庫存
		verifyStock(t, ctx, inventory, 1, 100)

		// 驗證使用者購買紀錄
		verifyUserBought(t, ctx, redis, 1, 1, 0)
	})

	t.Run("Failed - ExceedsMaxPerUser - AlreadyBought", func(t *testing.T) {
		defer clearRedis(ctx)
		err := inventory.WarmUpInventory(ctx, 1, 100, 100.5, 2)
		assert.NoError(t, err)

		// 第一次購買 1 張
		result, price, err := inventory.DecreStock(ctx, 1, 1, 1)
		assert.NoError(t, err)
		assert.True(t, result)
		assert.Equal(t, 100.5, price)

		// 驗證購買
		verifyStock(t, ctx, inventory, 1, 99)
		verifyUserBought(t, ctx, redis, 1, 1, 1)

		// 第二次購買 2 張，超過個人購買限制
		result, price, err = inventory.DecreStock(ctx, 1, 2, 1)
		assert.Equal(t, app_errors.ErrExceedsMaxPerUser, err)
		assert.False(t, result)
		assert.Equal(t, 0.0, price)

		// 驗證第二次購買失敗
		verifyStock(t, ctx, inventory, 1, 99)
		verifyUserBought(t, ctx, redis, 1, 1, 1)
	})

	t.Run("Failed - TicketNotFound", func(t *testing.T) {
		defer clearRedis(ctx)
		result, price, err := inventory.DecreStock(ctx, 1, 1, 1)
		assert.Equal(t, app_errors.ErrTicketNotFound, err)
		assert.False(t, result)
		assert.Equal(t, 0.0, price)

		// 驗證使用者購買紀錄
		verifyUserBought(t, ctx, redis, 1, 1, 0)
	})
}

func TestTicketInventory_RollbackStock(t *testing.T) {
	ctx := context.Background()
	redis := getTestRdb()
	inventory := cache.NewRedisTicketInventoryManager(redis)
	clearRedis(ctx)
	t.Cleanup(func() {
		clearRedis(ctx)
	})

	t.Run("Success", func(t *testing.T) {
		defer clearRedis(ctx)
		err := inventory.WarmUpInventory(ctx, 1, 100, 100.5, 2)
		assert.NoError(t, err)

		// 購買 2 張
		result, _, err := inventory.DecreStock(ctx, 1, 2, 1)
		assert.NoError(t, err)
		assert.True(t, result)

		// 驗證購買後
		verifyStock(t, ctx, inventory, 1, 98)
		verifyUserBought(t, ctx, redis, 1, 1, 2)

		// 回滾 2 張
		err = inventory.RollbackStock(ctx, 1, 2, 1)
		assert.NoError(t, err)

		// 驗證回滾後
		verifyStock(t, ctx, inventory, 1, 100)
		verifyUserBought(t, ctx, redis, 1, 1, 0)
	})
}
