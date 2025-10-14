package config

import (
	"os"
)

type Config struct {
	Database DatabaseConfig
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

var AppConfig *Config

func LoadConfig() *Config {
	dbConfig := GetDatabaseConfig()

	AppConfig = &Config{
		Database: dbConfig,
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

	return &Config{
		Database: *testConfig,
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

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
