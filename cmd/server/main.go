package main

import (
	"context"
	"go-gin-high-concurrency/config"
	"go-gin-high-concurrency/internal/cache"
	"go-gin-high-concurrency/internal/database"
	"go-gin-high-concurrency/internal/handler"
	"go-gin-high-concurrency/internal/queue"
	"go-gin-high-concurrency/internal/repository"
	"go-gin-high-concurrency/internal/service"
	"go-gin-high-concurrency/internal/worker"
	"go-gin-high-concurrency/pkg/logger"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	cfg := config.LoadConfig()

	pool, err := database.InitDatabase(&cfg.Database)
	if err != nil {
		logger.L.Fatal("Failed to initialize database", zap.Error(err))
	}
	defer pool.Close()

	rdb, err := database.InitRedis(&cfg.Redis)
	if err != nil {
		logger.L.Fatal("Failed to initialize redis", zap.Error(err))
	}
	defer rdb.Close()

	// 初始化 Repository
	orderRepository := repository.NewOrderRepository(pool)
	ticketRepository := repository.NewTicketRepository(pool)
	userRepository := repository.NewUserRepository(pool)
	eventRepository := repository.NewEventRepository(pool)
	_ = userRepository // 保留以備將來使用

	// 初始化 Cache
	inventoryManager := cache.NewRedisTicketInventoryManager(rdb)

	// 初始化 Redis Stream	 Queue
	orderQueue, err := queue.NewRedisStreamOrderQueue(rdb, "order-queue", nil)
	if err != nil {
		logger.L.Fatal("Failed to create Redis stream order queue", zap.Error(err))
	}

	// 初始化 Service
	orderService := service.NewOrderService(pool, orderRepository, ticketRepository, inventoryManager, orderQueue)
	eventService := service.NewEventService(eventRepository)
	ticketService := service.NewTicketService(ticketRepository)

	// Worker 使用 Background context（長期運行的後台任務，獨立於 HTTP Server）
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	orderWorker := worker.NewOrderWorker(orderService, orderQueue)
	if err := orderWorker.Start(workerCtx); err != nil {
		logger.L.Fatal("Failed to start order worker", zap.Error(err))
	}
	logger.L.Info("Order worker started successfully")

	// 初始化 Handler 和 Router
	orderHandler := handler.NewOrderHandler(orderService)
	eventHandler := handler.NewEventHandler(eventService)
	ticketHandler := handler.NewTicketHandler(ticketService)
	router := gin.Default()

	// Health check
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	// 註冊路由
	orderHandler.RegisterRoutes(router)
	eventHandler.RegisterRoutes(router)
	ticketHandler.RegisterRoutes(router)

	// 創建 HTTP Server（使用 http.Server 以支持優雅關閉）
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// 在 goroutine 中啟動服務器
	go func() {
		logger.L.Info("Server starting on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.L.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// 使用 signal.NotifyContext 來監聽終止信號
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 等待終止信號
	<-ctx.Done()

	logger.L.Info("Shutting down server...")

	// 設置 shutdown timeout（給正在處理的請求時間完成）
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// 1. 先停止接收新請求（關閉 HTTP Server）
	// 注意：Gin 會自動等待正在處理的 HTTP 請求完成
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.L.Warn("Server forced to shutdown", zap.Error(err))
	} else {
		logger.L.Info("Server gracefully stopped")
	}

	// 2. 停止 Worker（讓它完成正在處理的訂單）
	logger.L.Info("Stopping worker...")
	workerCancel()

	// 等待 Worker 完成（給一點時間讓正在處理的訂單完成）
	// 注意：在實際生產環境中，你可能需要更精確的等待機制（例如使用 sync.WaitGroup）
	workerShutdownCtx, workerShutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer workerShutdownCancel()

	// 等待 Worker 完成或超時
	select {
	case <-workerShutdownCtx.Done():
		if workerShutdownCtx.Err() == context.DeadlineExceeded {
			logger.L.Warn("Worker shutdown timeout exceeded")
		} else {
			logger.L.Info("Worker stopped successfully")
		}
	case <-time.After(2 * time.Second):
		logger.L.Info("Waiting for worker to finish processing...")
	}

	// 檢查 HTTP Server shutdown 是否超時
	select {
	case <-shutdownCtx.Done():
		if shutdownCtx.Err() == context.DeadlineExceeded {
			logger.L.Warn("Shutdown timeout exceeded, forcing shutdown")
		}
	default:
	}

	// 關閉資料庫和 Redis 連接（defer 會自動執行）
	logger.L.Info("Server shutdown complete")
	_ = logger.L.Sync()
}
