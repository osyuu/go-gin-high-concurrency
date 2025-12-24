package cache

import (
	"context"
	"go-gin-high-concurrency/test/internal/testutil"
	"log"
	"os"
	"testing"

	"github.com/redis/go-redis/v9"
)

var testRdb *redis.Client

func TestMain(m *testing.M) {
	_, rdb, cleanup, err := testutil.Setup()
	if err != nil {
		log.Fatalf("Failed to setup test environment: %v", err)
	}
	defer cleanup()
	testRdb = rdb

	code := m.Run()
	os.Exit(code)
}

func getTestRdb() *redis.Client {
	if testRdb == nil {
		panic("testRdb is not initialized. Make sure TestMain has run.")
	}
	return testRdb
}

func clearRedis(ctx context.Context) {
	err := testRdb.FlushDB(ctx).Err()
	if err != nil {
		panic(err)
	}
}
