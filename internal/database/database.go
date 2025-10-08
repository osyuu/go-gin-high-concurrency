package database

import (
	"context"
	"fmt"
	"go-gin-high-concurrency/config"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func InitDatabase(config *config.DatabaseConfig) (*pgxpool.Pool, error) {

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s timezone=%s",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.DBName,
		config.SSLMode,
		"UTC",
	)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	// 設置連接池參數
	poolConfig.MaxConns = 25        // 最大連接數
	poolConfig.MinConns = 5         // 最小連接數
	poolConfig.MaxConnLifetime = time.Hour  // 連接最大生命週期
	poolConfig.MaxConnIdleTime = time.Minute * 30  // 最大閒置時間

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, err
	}

	err = pool.Ping(context.Background())
	if err != nil {
		return nil, err
	}

	return pool, nil
}