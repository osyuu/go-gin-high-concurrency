package service

import (
	"go-gin-high-concurrency/test/internal/testutil"
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var testDB *pgxpool.Pool
var testRdb *redis.Client

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

func getTestDB() *pgxpool.Pool {
	if testDB == nil {
		panic("testDB is not initialized. Make sure TestMain has run.")
	}
	return testDB
}
