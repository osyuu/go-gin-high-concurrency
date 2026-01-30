package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"go-gin-high-concurrency/internal/cache"
	"go-gin-high-concurrency/internal/handler"
	"go-gin-high-concurrency/internal/model"
	"go-gin-high-concurrency/internal/queue"
	"go-gin-high-concurrency/internal/repository"
	"go-gin-high-concurrency/internal/service"
	"go-gin-high-concurrency/internal/worker"
	"go-gin-high-concurrency/test/internal/testutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testDB  *pgxpool.Pool
	testRdb *redis.Client
)

func TestMain(m *testing.M) {
	db, rdb, cleanup, err := testutil.Setup()
	if err != nil {
		log.Fatalf("Failed to setup test environment: %v", err)
	}
	defer cleanup()
	testDB = db
	testRdb = rdb

	code := m.Run()
	os.Exit(code)
}

type failingQueue struct{}

func (f *failingQueue) PublishOrder(ctx context.Context, order *model.Order) error {
	return errors.New("queue publish failed") // 總是返回錯誤
}

func (f *failingQueue) SubscribeOrders(ctx context.Context) (<-chan queue.Delivery, error) {
	out := make(chan queue.Delivery)
	close(out) // 返回一個已關閉的 channel
	return out, nil
}

func setupIntegrationTest(t *testing.T, useFailingQueue bool) (*gin.Engine, func()) {
	t.Helper()
	ctx := context.Background()

	// 清空資料庫和 Redis
	cleanupDB(ctx, t)
	cleanupRedis(ctx, t)

	// 初始化所有真實組件
	orderRepo := repository.NewOrderRepository(testDB)
	ticketRepo := repository.NewTicketRepository(testDB)
	inventoryManager := cache.NewRedisTicketInventoryManager(testRdb)

	// 初始化
	var orderService service.OrderService
	var orderQueue queue.OrderQueue
	var workerCancel context.CancelFunc

	if useFailingQueue {
		orderQueue = &failingQueue{}
		orderService = service.NewOrderService(testDB, orderRepo, ticketRepo, inventoryManager, orderQueue)
	} else {
		// 使用 Redis Stream 版 Queue
		cfg := &queue.RedisStreamOrderQueueConfig{
			ClaimMinIdleTime:   1 * time.Second,
			MaxRetryCount:      3,
			ReadGroupBlockTime: 500 * time.Millisecond,
		}
		var err error
		orderQueue, err = queue.NewRedisStreamOrderQueue(testRdb, "order-queue-test", cfg)
		if err != nil {
			t.Fatalf("Failed to create Redis stream order queue: %v", err)
		}
		orderService = service.NewOrderService(testDB, orderRepo, ticketRepo, inventoryManager, orderQueue)

		// 初始化 Worker
		workerCtx, cancel := context.WithCancel(context.Background())
		workerCancel = cancel
		orderWorker := worker.NewOrderWorker(orderService, orderQueue)
		if err := orderWorker.Start(workerCtx); err != nil {
			t.Fatalf("Failed to start worker: %v", err)
		}
	}

	// 初始化 Handler 和 Router（含 Event / Ticket API，供 createTestEventViaAPI / createTestTicketViaAPI 使用）
	eventRepo := repository.NewEventRepository(testDB)
	eventService := service.NewEventService(eventRepo, ticketRepo, inventoryManager)
	eventHandler := handler.NewEventHandler(eventService)
	ticketService := service.NewTicketService(ticketRepo)
	ticketHandler := handler.NewTicketHandler(ticketService)

	orderHandler := handler.NewOrderHandler(orderService)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	orderHandler.RegisterRoutes(router)
	eventHandler.RegisterRoutes(router)
	ticketHandler.RegisterRoutes(router)

	cleanup := func() {
		if workerCancel != nil {
			workerCancel()
			time.Sleep(100 * time.Millisecond) // 等待 worker 停止
		}
		cleanupDB(ctx, t)
		cleanupRedis(ctx, t)
	}

	return router, cleanup
}

func cleanupDB(ctx context.Context, t *testing.T) {
	t.Helper()
	_, err := testDB.Exec(ctx, "TRUNCATE tickets, orders, users, events RESTART IDENTITY CASCADE")
	if err != nil {
		t.Logf("Warning: failed to truncate tables: %v", err)
	}
}

func cleanupRedis(ctx context.Context, t *testing.T) {
	t.Helper()
	err := testRdb.FlushDB(ctx).Err()
	if err != nil {
		t.Logf("Warning: failed to flush redis: %v", err)
	}
}

func createTestUser(t *testing.T, name, email string) int {
	t.Helper()
	ctx := context.Background()
	userRepo := repository.NewUserRepository(testDB)

	user := &model.User{
		Name:  name,
		Email: email,
	}
	created, err := userRepo.Create(ctx, user)
	require.NoError(t, err)
	return created.ID
}

// createTestEventViaAPI 透過 POST /api/v1/events 建立活動，回傳 events.id（供 createTestTicket 關聯）
func createTestEventViaAPI(t *testing.T, router *gin.Engine, name string) int {
	t.Helper()
	body := map[string]interface{}{"name": name}
	req := createHTTPRequest("POST", "/api/v1/events", body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, "create event via API: %s", w.Body.String())
	var event model.Event
	err := json.NewDecoder(w.Body).Decode(&event)
	require.NoError(t, err)
	return event.ID
}

// createTestTicketViaAPI 透過 POST /api/v1/events 與 POST /api/v1/tickets 建立活動與票券，回傳 tickets.id（供訂單與庫存預熱使用）
func createTestTicket(t *testing.T, router *gin.Engine, eventName string, price float64, totalStock, maxPerUser int) int {
	t.Helper()
	eventID := createTestEventViaAPI(t, router, eventName)
	body := map[string]interface{}{
		"event_id":     eventID,
		"name":         eventName,
		"price":        price,
		"total_stock":  totalStock,
		"max_per_user": maxPerUser,
	}
	req := createHTTPRequest("POST", "/api/v1/tickets", body)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, "create ticket via API: %s", w.Body.String())
	var ticket model.Ticket
	err := json.NewDecoder(w.Body).Decode(&ticket)
	require.NoError(t, err)
	return ticket.ID
}

func warmUpInventory(t *testing.T, inventoryManager cache.RedisTicketInventoryManager, ticketID int, stock int, price float64, limit int) {
	t.Helper()
	ctx := context.Background()
	err := inventoryManager.WarmUpInventory(ctx, ticketID, stock, price, limit)
	require.NoError(t, err)
}

func createJSONRequest(data interface{}) *bytes.Buffer {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return bytes.NewBuffer([]byte(""))
	}
	return bytes.NewBuffer(jsonData)
}

func createHTTPRequest(method, url string, body interface{}) *http.Request {
	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequest(method, url, createJSONRequest(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return nil
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

// postCreateOrder 發送 POST /api/v1/orders 請求，回傳 ResponseRecorder 供斷言
func postCreateOrder(t *testing.T, router *gin.Engine, req model.CreateOrderRequest) *httptest.ResponseRecorder {
	t.Helper()
	httpReq := createHTTPRequest("POST", "/api/v1/orders", req)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)
	return w
}

// TestOrderHandler_Integration_EndToEnd 測試完整流程：HTTP → Handler → Service → Queue → Worker → Database
func TestOrderHandler_Integration_EndToEnd(t *testing.T) {
	router, cleanup := setupIntegrationTest(t, false)
	defer cleanup()

	ctx := context.Background()

	// 1. 準備測試資料
	userID := createTestUser(t, "Test User", "test@example.com")
	ticketID := createTestTicket(t, router, "Test Event", 100.0, 100, 2)

	// 2. 預熱 Redis 庫存
	inventoryManager := cache.NewRedisTicketInventoryManager(testRdb)
	warmUpInventory(t, inventoryManager, ticketID, 100, 100.0, 2)

	// 3. 等待 Worker 處理（給一點時間讓 Worker 啟動）
	time.Sleep(200 * time.Millisecond)

	// 4. 發送 HTTP 請求創建訂單
	w := postCreateOrder(t, router, model.CreateOrderRequest{
		UserID:   userID,
		TicketID: ticketID,
		Quantity: 2,
	})

	// 5. 驗證 HTTP 回應
	assert.Equal(t, http.StatusCreated, w.Code)

	var orderResponse model.Order
	err := json.Unmarshal(w.Body.Bytes(), &orderResponse)
	require.NoError(t, err)
	assert.Equal(t, userID, orderResponse.UserID)
	assert.Equal(t, ticketID, orderResponse.TicketID)
	assert.Equal(t, 2, orderResponse.Quantity)
	assert.Equal(t, model.OrderStatusPending, orderResponse.Status)

	// 6. 等待 Worker 處理訂單（最多等待 2 秒）
	orderRepo := repository.NewOrderRepository(testDB)
	var createdOrder *model.Order
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		orders, err := orderRepo.List(ctx)
		if err != nil {
			t.Logf("Error listing orders: %v", err)
			continue
		}

		// 根據 RequestID 找到對應的訂單
		for _, order := range orders {
			if order.RequestID == orderResponse.RequestID {
				createdOrder = order
				break
			}
		}
		if createdOrder != nil {
			t.Logf("Order found in database after %d retries", i+1)
			break
		}
	}
	require.NotNil(t, createdOrder)
	require.NoError(t, err)

	// 7. 驗證資料庫中的訂單
	assert.Equal(t, userID, createdOrder.UserID)
	assert.Equal(t, ticketID, createdOrder.TicketID)
	assert.Equal(t, 2, createdOrder.Quantity)
	assert.Equal(t, 200.0, createdOrder.TotalPrice)

	// 8. 驗證資料庫中的票券庫存已扣減
	ticketRepo := repository.NewTicketRepository(testDB)
	ticket, err := ticketRepo.FindByID(ctx, ticketID)
	require.NoError(t, err)
	assert.Equal(t, 98, ticket.RemainingStock) // 100 - 2 = 98

	// 9. 驗證 Redis 庫存已扣減
	redisStock, err := inventoryManager.GetStock(ctx, ticketID)
	require.NoError(t, err)
	assert.Equal(t, 98, redisStock) // 100 - 2 = 98
}

// TestOrderHandler_Integration_RollbackOnPublishFailure 測試 PublishOrder 失敗時的回滾機制
func TestOrderHandler_Integration_RollbackOnPublishFailure(t *testing.T) {
	// 使用會失敗的 Queue
	router, cleanup := setupIntegrationTest(t, true)
	defer cleanup()

	ctx := context.Background()

	// 1. 準備測試資料
	userID := createTestUser(t, "Test User", "test@example.com")
	ticketID := createTestTicket(t, router, "Test Event", 100.0, 100, 2)

	// 2. 預熱 Redis 庫存
	inventoryManager := cache.NewRedisTicketInventoryManager(testRdb)
	warmUpInventory(t, inventoryManager, ticketID, 100, 100.0, 2)

	// 3. 驗證初始庫存
	initialStock, err := inventoryManager.GetStock(ctx, ticketID)
	require.NoError(t, err)
	assert.Equal(t, 100, initialStock)

	// 5. 發送 HTTP 請求（正常情況下應該成功）
	w := postCreateOrder(t, router, model.CreateOrderRequest{
		UserID:   userID,
		TicketID: ticketID,
		Quantity: 1,
	})

	// 5. 驗證 HTTP 回應是 500（因為 PublishOrder 失敗）
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 6. 驗證 Redis 庫存被回滾
	afterStock, err := inventoryManager.GetStock(ctx, ticketID)
	require.NoError(t, err)
	assert.Equal(t, 100, afterStock)

	// 7. 驗證資料庫中的訂單被回滾
	orderRepo := repository.NewOrderRepository(testDB)
	orders, err := orderRepo.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, len(orders))
}

// TestOrderHandler_Integration_InsufficientStock 測試庫存不足的情況
func TestOrderHandler_Integration_InsufficientStock(t *testing.T) {
	router, cleanup := setupIntegrationTest(t, false)
	defer cleanup()

	ctx := context.Background()

	// 1. 準備測試資料（庫存只有 1 張）
	userID := createTestUser(t, "Test User", "test@example.com")
	ticketID := createTestTicket(t, router, "Test Event", 100.0, 1, 2)

	// 2. 預熱 Redis 庫存（只有 1 張）
	inventoryManager := cache.NewRedisTicketInventoryManager(testRdb)
	warmUpInventory(t, inventoryManager, ticketID, 1, 100.0, 2)

	// 3. 發送 HTTP 請求（嘗試購買 2 張）
	w := postCreateOrder(t, router, model.CreateOrderRequest{
		UserID:   userID,
		TicketID: ticketID,
		Quantity: 2,
	})

	// 4. 驗證 HTTP 回應是 409 Conflict
	assert.Equal(t, http.StatusConflict, w.Code)

	// 5. 驗證 Redis 庫存沒有被扣減
	stock, err := inventoryManager.GetStock(ctx, ticketID)
	require.NoError(t, err)
	assert.Equal(t, 1, stock) // 庫存應該還是 1
}

// TestOrderHandler_Integration_ExceedsMaxPerUser 測試超過個人購買限制的情況
func TestOrderHandler_Integration_ExceedsMaxPerUser(t *testing.T) {
	router, cleanup := setupIntegrationTest(t, false)
	defer cleanup()

	// 1. 準備測試資料（個人限制 2 張）
	userID := createTestUser(t, "Test User", "test@example.com")
	ticketID := createTestTicket(t, router, "Test Event", 100.0, 100, 2)

	// 2. 預熱 Redis 庫存
	inventoryManager := cache.NewRedisTicketInventoryManager(testRdb)
	warmUpInventory(t, inventoryManager, ticketID, 100, 100.0, 2)

	// 3. 第一次購買 2 張（應該成功）
	w1 := postCreateOrder(t, router, model.CreateOrderRequest{
		UserID:   userID,
		TicketID: ticketID,
		Quantity: 2,
	})
	assert.Equal(t, http.StatusCreated, w1.Code)

	// 4. 等待 Worker 處理
	time.Sleep(500 * time.Millisecond)

	// 5. 第二次嘗試購買 1 張（應該失敗，因為已經買了 2 張）
	w2 := postCreateOrder(t, router, model.CreateOrderRequest{
		UserID:   userID,
		TicketID: ticketID,
		Quantity: 1,
	})

	// 6. 驗證 HTTP 回應是 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, w2.Code)
}

// TestOrderHandler_Integration_ConcurrentOrders 測試高併發場景
func TestOrderHandler_Integration_ConcurrentOrders(t *testing.T) {
	router, cleanup := setupIntegrationTest(t, false)
	defer cleanup()

	ctx := context.Background()

	// 1. 準備測試資料（庫存 10 張）
	userID := createTestUser(t, "Test User", "test@example.com")
	ticketID := createTestTicket(t, router, "Test Event", 100.0, 10, 10)

	// 2. 預熱 Redis 庫存
	inventoryManager := cache.NewRedisTicketInventoryManager(testRdb)
	warmUpInventory(t, inventoryManager, ticketID, 10, 100.0, 10)

	// 3. 等待 Worker 啟動
	time.Sleep(200 * time.Millisecond)

	// 4. 併發發送 20 個請求（超過庫存）
	var wg sync.WaitGroup
	successCount := 0
	conflictCount := 0
	var mu sync.Mutex

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			w := postCreateOrder(t, router, model.CreateOrderRequest{
				UserID:   userID,
				TicketID: ticketID,
				Quantity: 1,
			})

			mu.Lock()
			if w.Code == http.StatusCreated {
				successCount++
			}
			if w.Code == http.StatusConflict {
				conflictCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// 5. 等待 Worker 處理所有訂單
	time.Sleep(2 * time.Second)

	// 6. 驗證只有 10 個請求成功（庫存只有 10 張）
	assert.Equal(t, 10, successCount, "應該只有 10 個請求成功")
	// 應該有 10 個請求失敗，因為庫存只有 10 張
	assert.Equal(t, 10, conflictCount, "應該有 10 個請求失敗")

	// 7. 驗證資料庫中的訂單數量
	orderRepo := repository.NewOrderRepository(testDB)
	orders, err := orderRepo.List(ctx)
	require.NoError(t, err)
	assert.Equal(t, 10, len(orders), "資料庫中應該有 10 筆訂單")

	// 8. 驗證 Redis 庫存為 0
	stock, err := inventoryManager.GetStock(ctx, ticketID)
	require.NoError(t, err)
	assert.Equal(t, 0, stock, "Redis 庫存應該為 0")

	// 9. 驗證資料庫中的票券庫存為 0
	ticketRepo := repository.NewTicketRepository(testDB)
	ticket, err := ticketRepo.FindByID(ctx, ticketID)
	require.NoError(t, err)
	assert.Equal(t, 0, ticket.RemainingStock, "資料庫庫存應該為 0")
}
