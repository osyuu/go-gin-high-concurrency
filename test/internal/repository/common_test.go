package repository

import (
	"context"
	"fmt"
	"go-gin-high-concurrency/config"
	"go-gin-high-concurrency/internal/database"
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// testDB 是測試用的資料庫連接池
// 通過 InitDatabase 獲得，不依賴 GetPool()
var testDB *pgxpool.Pool

func TestMain(m *testing.M) {
	cfg := config.LoadTestConfig()

	var err error
	testDB, err = database.InitDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize test database: %v", err)
	}

	// 確保資料庫連接正常
	if err := testDB.Ping(context.Background()); err != nil {
		log.Fatalf("Failed to ping test database: %v", err)
	}

	log.Println("Test database connected successfully")
	log.Println("Running repository tests...")

	code := m.Run()
	testDB.Close()
	log.Println("Test database closed")

	os.Exit(code)
}

func setupTestWithTruncate(t *testing.T) func() {
	t.Helper()
	ctx := context.Background()

	// 清空所有測試資料，保留 schema
	_, err := testDB.Exec(ctx, "TRUNCATE tickets, orders, users RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("Failed to truncate tables: %v", err)
	}

	return func() {
	}
}

// setupTestWithTransaction 使用 Transaction Rollback 方式
// 適合測試 transaction 相關的邏輯
func setupTestWithTransaction(t *testing.T) (pgx.Tx, func()) {
	t.Helper()
	ctx := context.Background()

	// 開始一個測試用的 transaction
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	cleanup := func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Logf("Warning: failed to rollback transaction: %v", err)
		}
	}

	return tx, cleanup
}

// getTestDB 返回測試用的資料庫連接池
// 用於創建 repository 實例
func getTestDB() *pgxpool.Pool {
	if testDB == nil {
		panic("testDB is not initialized. Make sure TestMain has run.")
	}
	return testDB
}

// createTestTicket 輔助函數：創建測試用的 ticket
// total_stock 和 remaining_stock 都设置为 stock
func createTestTicket(t *testing.T, eventID int, eventName string, stock int) int {
	t.Helper()
	return createTestTicketWithStock(t, eventID, eventName, stock, stock)
}

// createTestTicketWithStock 輔助函數：創建測試用的 ticket，可以分別指定總庫存和剩餘庫存
func createTestTicketWithStock(t *testing.T, eventID int, eventName string, totalStock, remainingStock int) int {
	t.Helper()
	ctx := context.Background()

	query := `
		INSERT INTO tickets (event_id, event_name, price, total_stock, remaining_stock, max_per_user)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var id int
	err := testDB.QueryRow(ctx, query,
		eventID, eventName, 1000.0, totalStock, remainingStock, 5,
	).Scan(&id)

	if err != nil {
		t.Fatalf("Failed to create test ticket: %v", err)
	}

	return id
}

// createTestUser 輔助函數：創建測試用的 user
func createTestUser(t *testing.T, name, email string) int {
	t.Helper()
	ctx := context.Background()

	query := `
		INSERT INTO users (name, email)
		VALUES ($1, $2)
		RETURNING id
	`

	var id int
	err := testDB.QueryRow(ctx, query, name, email).Scan(&id)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return id
}

// assertRowCount 輔助函數：檢查資料表的行數
func assertRowCount(t *testing.T, table string, expected int) {
	t.Helper()
	ctx := context.Background()

	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE deleted_at IS NULL", table)
	err := testDB.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count rows in %s: %v", table, err)
	}

	if count != expected {
		t.Errorf("Expected %d rows in %s, got %d", expected, table, count)
	}
}
