package testutil

import (
	"context"
	"fmt"
	"go-gin-high-concurrency/config"
	"go-gin-high-concurrency/internal/database"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func Setup() (*pgxpool.Pool, *redis.Client, func(), error) {
	cfg := config.LoadTestConfig()

	var err error
	testDB, err := database.InitDatabase(&cfg.Database)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to initialize test database: %v", err)
	}

	if err := testDB.Ping(context.Background()); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to ping test database: %v", err)
	}

	log.Println("Test database connected successfully")

	testRdb, err := database.InitRedis(&cfg.Redis)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to initialize redis: %v", err)
	}
	log.Println("Test redis connected successfully")

	log.Println("Running service tests...")

	cleanup := func() {
		testDB.Close()
		log.Println("Test database closed")

		testRdb.Close()
		log.Println("Test redis closed")
	}

	return testDB, testRdb, cleanup, nil
}

// SetupRedisOnly 僅初始化 Redis，用於只依賴 Redis 的測試（如 queue 整合測試）
func SetupRedisOnly() (*redis.Client, func(), error) {
	cfg := config.LoadTestConfig()
	rdb, err := database.InitRedis(&cfg.Redis)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize redis: %v", err)
	}
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, nil, fmt.Errorf("failed to ping redis: %v", err)
	}
	cleanup := func() { rdb.Close() }
	return rdb, cleanup, nil
}
