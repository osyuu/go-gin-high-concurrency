package config

import (
	"os"
	"strconv"
)

type Config struct {
	Database DatabaseConfig
	Redis    RedisConfig
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

var AppConfig *Config

func LoadConfig() *Config {
	dbConfig := GetDatabaseConfig()
	redisConfig := GetRedisConfig()

	AppConfig = &Config{
		Database: dbConfig,
		Redis:    redisConfig,
	}

	return AppConfig
}

func LoadTestConfig() *Config {
	testConfig := &DatabaseConfig{
		Host:     "localhost",
		Port:     "5433", // 測試 DB 用 5433 port
		User:     "postgres",
		Password: "postgres",
		DBName:   "test_db",
		SSLMode:  "disable",
	}

	testRedisConfig := RedisConfig{
		Host:     "localhost",
		Port:     "6380", // 測試 Redis 用 6380 port
		Password: "",
		DB:       1,
	}

	return &Config{
		Database: *testConfig,
		Redis:    testRedisConfig,
	}
}

func GetDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		DBName:   getEnv("DB_NAME", "postgres"),
		SSLMode:  getEnv("DB_SSL_MODE", "disable"),
	}
}

func GetRedisConfig() RedisConfig {
	db, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		panic(err)
	}

	return RedisConfig{
		Host:     getEnv("REDIS_HOST", "localhost"),
		Port:     getEnv("REDIS_PORT", "6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       db,
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
