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
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()

	pool, err := database.InitDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer pool.Close()

	rdb, err := database.InitRedis(&cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to initialize redis: %v", err)
	}
	defer rdb.Close()

	// 初始化 Repository
	orderRepository := repository.NewOrderRepository(pool)
	ticketRepository := repository.NewTicketRepository(pool)
	userRepository := repository.NewUserRepository(pool)
	_ = userRepository // 保留以備將來使用

	// 初始化 Cache
	inventoryManager := cache.NewRedisTicketInventoryManager(rdb)

	// 初始化 Redis Stream	 Queue
	orderQueue, err := queue.NewRedisStreamOrderQueue(rdb, "order-queue", nil)
	if err != nil {
		log.Fatalf("Failed to create Redis stream order queue: %v", err)
	}

	// 初始化 Service
	orderService := service.NewOrderService(pool, orderRepository, ticketRepository, inventoryManager, orderQueue)

	// Worker 使用 Background context（長期運行的後台任務，獨立於 HTTP Server）
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	orderWorker := worker.NewOrderWorker(orderService, orderQueue)
	if err := orderWorker.Start(workerCtx); err != nil {
		log.Fatalf("Failed to start order worker: %v", err)
	}
	log.Println("Order worker started successfully")

	// 初始化 Handler 和 Router
	orderHandler := handler.NewOrderHandler(orderService)
	router := gin.Default()

	// Health check
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	// 註冊訂單相關路由
	orderHandler.RegisterRoutes(router)

	// 創建 HTTP Server（使用 http.Server 以支持優雅關閉）
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// 在 goroutine 中啟動服務器
	go func() {
		log.Println("Server starting on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 使用 signal.NotifyContext 來監聽終止信號
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 等待終止信號
	<-ctx.Done()

	log.Println("Shutting down server...")

	// 設置 shutdown timeout（給正在處理的請求時間完成）
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// 1. 先停止接收新請求（關閉 HTTP Server）
	// 注意：Gin 會自動等待正在處理的 HTTP 請求完成
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	} else {
		log.Println("Server gracefully stopped")
	}

	// 2. 停止 Worker（讓它完成正在處理的訂單）
	log.Println("Stopping worker...")
	workerCancel()

	// 等待 Worker 完成（給一點時間讓正在處理的訂單完成）
	// 注意：在實際生產環境中，你可能需要更精確的等待機制（例如使用 sync.WaitGroup）
	workerShutdownCtx, workerShutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer workerShutdownCancel()

	// 等待 Worker 完成或超時
	select {
	case <-workerShutdownCtx.Done():
		if workerShutdownCtx.Err() == context.DeadlineExceeded {
			log.Println("Worker shutdown timeout exceeded")
		} else {
			log.Println("Worker stopped successfully")
		}
	case <-time.After(2 * time.Second):
		log.Println("Waiting for worker to finish processing...")
	}

	// 檢查 HTTP Server shutdown 是否超時
	select {
	case <-shutdownCtx.Done():
		if shutdownCtx.Err() == context.DeadlineExceeded {
			log.Println("Shutdown timeout exceeded, forcing shutdown")
		}
	default:
	}

	// 關閉資料庫和 Redis 連接（defer 會自動執行）
	log.Println("Server shutdown complete")
}
