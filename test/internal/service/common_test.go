package service

import (
	"context"
	"go-gin-high-concurrency/config"
	"go-gin-high-concurrency/internal/database"
	"go-gin-high-concurrency/internal/model"
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

var testDB *pgxpool.Pool

func TestMain(m *testing.M) {
	cfg := config.LoadTestConfig()

	var err error
	testDB, err = database.InitDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize test database: %v", err)
	}

	if err := testDB.Ping(context.Background()); err != nil {
		log.Fatalf("Failed to ping test database: %v", err)
	}

	log.Println("Test database connected successfully")
	log.Println("Running service tests...")

	code := m.Run()

	testDB.Close()
	log.Println("Test database closed")

	os.Exit(code)
}

func setupTestWithTruncate(t *testing.T) func() {
	t.Helper()
	ctx := context.Background()

	_, err := testDB.Exec(ctx, "TRUNCATE tickets, orders, users RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("Failed to truncate tables: %v", err)
	}

	return func() {}
}

func getTestDB() *pgxpool.Pool {
	if testDB == nil {
		panic("testDB is not initialized. Make sure TestMain has run.")
	}
	return testDB
}

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

func createTestTicket(t *testing.T, eventID int, eventName string, stock int, maxPerUser int) int {
	t.Helper()
	return createTestTicketWithStock(t, eventID, eventName, stock, stock, maxPerUser)
}

func createTestTicketWithStock(t *testing.T, eventID int, eventName string, totalStock, remainingStock int, maxPerUser int) int {
	t.Helper()
	ctx := context.Background()

	query := `
		INSERT INTO tickets (event_id, event_name, price, total_stock, remaining_stock, max_per_user)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var id int
	err := testDB.QueryRow(ctx, query,
		eventID, eventName, 1000.0, totalStock, remainingStock, maxPerUser,
	).Scan(&id)

	if err != nil {
		t.Fatalf("Failed to create test ticket: %v", err)
	}

	return id
}

func createTestOrder(t *testing.T, userID, ticketID int, quantity int, totalPrice float64, status model.OrderStatus) int {
	t.Helper()
	ctx := context.Background()

	query := `
		INSERT INTO orders (user_id, ticket_id, quantity, total_price, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	var id int
	err := testDB.QueryRow(ctx, query, userID, ticketID, quantity, totalPrice, status).Scan(&id)
	if err != nil {
		t.Fatalf("Failed to create test order: %v", err)
	}

	return id
}
