package cache

import (
	"context"
	"errors"
	"fmt"
	"go-gin-high-concurrency/pkg/app_errors"
	"strconv"

	"github.com/redis/go-redis/v9"
)

type RedisTicketInfo struct {
	Stock int
	Price float64
	Limit int
}

type RedisTicketInventoryManager interface {
	// 預熱：預先加載票的庫存到 Redis
	WarmUpInventory(ctx context.Context, tickelID int, stock int, price float64, limit int) error
	// 獲取：獲取票的庫存
	GetStock(ctx context.Context, ticketID int) (int, error)
	// 獲取：獲取票的資訊
	GetInfo(ctx context.Context, ticketID int) (RedisTicketInfo, error)
	// 減少：減少票的庫存 (使用Lua腳本確保原子性)
	DecreStock(ctx context.Context, ticketID int, quantity int, userID int) (bool, float64, error)
	// 回滾：回滾票的庫存及使用者購買紀錄 (使用Lua腳本確保原子性)
	RollbackStock(ctx context.Context, ticketID int, quantity int, userID int) error
}

type RedisTicketInventoryManagerImpl struct {
	client *redis.Client
}

func NewRedisTicketInventoryManager(client *redis.Client) RedisTicketInventoryManager {
	return &RedisTicketInventoryManagerImpl{
		client: client,
	}
}

// 庫存 key
func (m *RedisTicketInventoryManagerImpl) getInfoKey(ticketID int) string {
	return fmt.Sprintf("ticket:%d:info", ticketID)
}

// 用戶購買紀錄的 key
func (m *RedisTicketInventoryManagerImpl) getUsersKey(ticketID int) string {
	return fmt.Sprintf("ticket:%d:users", ticketID)
}

func (m *RedisTicketInventoryManagerImpl) WarmUpInventory(ctx context.Context, tickelID int, stock int, price float64, limit int) error {
	key := m.getInfoKey(tickelID)
	return m.client.HSet(ctx, key, map[string]interface{}{
		"stock": stock,
		"price": price,
		"limit": limit,
	}).Err()
}

func (m *RedisTicketInventoryManagerImpl) GetStock(ctx context.Context, ticketID int) (int, error) {
	key := m.getInfoKey(ticketID)
	// HMGet 回傳 slice，若只要一個欄位，建議用 HGet
	val, err := m.client.HGet(ctx, key, "stock").Int()
	if err == redis.Nil {
		return -1, app_errors.ErrTicketNotFound
	}
	return val, err
}

func (m *RedisTicketInventoryManagerImpl) GetInfo(ctx context.Context, ticketID int) (RedisTicketInfo, error) {
	key := m.getInfoKey(ticketID)
	result, err := m.client.HGetAll(ctx, key).Result()
	if err != nil {
		return RedisTicketInfo{}, err
	}

	// 檢查 key 是否存在
	if len(result) == 0 {
		return RedisTicketInfo{}, app_errors.ErrTicketNotFound
	}

	stock, err := strconv.Atoi(result["stock"])
	if err != nil {
		return RedisTicketInfo{}, fmt.Errorf("invalid stock: %v", err)
	}

	price, err := strconv.ParseFloat(result["price"], 64)
	if err != nil {
		return RedisTicketInfo{}, fmt.Errorf("invalid price: %v", err)
	}

	limit, err := strconv.Atoi(result["limit"])
	if err != nil {
		return RedisTicketInfo{}, fmt.Errorf("invalid limit: %v", err)
	}

	return RedisTicketInfo{
		Stock: stock,
		Price: price,
		Limit: limit,
	}, nil
}

/*
*

	減少票的庫存 (使用Lua腳本確保原子性)
	1. 檢查總庫存
	2. 檢查個人已購數量
	3. 執行扣減與紀錄
	4.
*/
func (m *RedisTicketInventoryManagerImpl) DecreStock(ctx context.Context, ticketID int, quantity int, userID int) (bool, float64, error) {
	key := m.getInfoKey(ticketID)
	usersKey := m.getUsersKey(ticketID)

	//
	script := `
		-- 1. 取得參數
		local ticket_key = KEYS[1]
		local users_key = KEYS[2]

		local user_id = tonumber(ARGV[1])
		local request_qty = tonumber(ARGV[2])

		-- 2. 取得票的資訊(總庫存、價格、個人購買限制)
		local ticket_info = redis.call('HMGET', ticket_key, 'stock', 'price', 'limit')
		local stock = ticket_info[1]
		local price = ticket_info[2]
		local limit = ticket_info[3]

		-- 3. 檢查數據是否存在
		if not stock or not price or not limit then
			return {-3, '0.0'} -- 錯誤：票券資訊未預熱
		end

		-- 4. 檢查總庫存
		if tonumber(stock) < request_qty then
			return {-1, '0.0'} -- 錯誤：庫存不足
		end

		-- 3. 檢查個人已購數量
		local user_bought = redis.call('HGET', users_key, user_id) or '0'
		if tonumber(user_bought) + request_qty > tonumber(limit) then
			return {-2, '0.0'} -- 錯誤：超過個人購買限制
		end

		-- 4. 執行扣減與紀錄
		-- 扣減庫存
		redis.call('HINCRBY', ticket_key, 'stock', -request_qty)
		-- 增加個人購買紀錄
		redis.call('HINCRBY', users_key, user_id, request_qty)

		return {1, tostring(price)} -- 搶票成功
	`

	result, err := m.client.Eval(ctx, script, []string{key, usersKey}, userID, quantity).Result()
	if err != nil {
		return false, 0, err
	}

	resSlice := result.([]interface{})
	code := resSlice[0].(int64) // Redis 數字通常回傳 int64
	priceStr := resSlice[1].(string)

	// 轉換價格為 float64
	price, _ := strconv.ParseFloat(priceStr, 64)

	switch code {
	case 1:
		return true, price, nil
	case -1:
		return false, 0.0, app_errors.ErrInsufficientStock
	case -2:
		return false, 0.0, app_errors.ErrExceedsMaxPerUser
	case -3:
		return false, 0.0, app_errors.ErrTicketNotFound
	default:
		return false, 0.0, errors.New("unexpected result")
	}
}

func (m *RedisTicketInventoryManagerImpl) RollbackStock(ctx context.Context, ticketID int, quantity int, userID int) error {
	key := m.getInfoKey(ticketID)
	usersKey := m.getUsersKey(ticketID)

	script := `
		-- 1. 取得參數
		local ticket_key = KEYS[1]
		local users_key = KEYS[2]
		local user_id = tonumber(ARGV[1])
		local rollback_qty = tonumber(ARGV[2])

		-- 2. 執行回滾庫存及使用者購買紀錄
		-- 回滾庫存
		redis.call('HINCRBY', ticket_key, 'stock', rollback_qty)
		-- 回滾個人購買紀錄
		redis.call('HINCRBY', users_key, user_id, -rollback_qty)

		return "OK"
	`

	_, err := m.client.Eval(ctx, script, []string{key, usersKey}, userID, quantity).Result()
	if err != nil {
		return err
	}

	return nil
}
