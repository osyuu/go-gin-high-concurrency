package main

import (
	"go-gin-high-concurrency/config"
	"go-gin-high-concurrency/internal/database"
	"log"

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

	router := gin.Default()
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	router.Run() // デフォルトで0.0.0.0:8080で待機します
}
