package queue_test

import (
	"log"
	"os"
	"testing"

	"go-gin-high-concurrency/test/internal/testutil"

	"github.com/redis/go-redis/v9"
)

var testRdb *redis.Client

func TestMain(m *testing.M) {
	rdb, cleanup, err := testutil.SetupRedisOnly()
	if err != nil {
		log.Fatalf("setup redis: %v", err)
	}
	defer cleanup()
	testRdb = rdb
	code := m.Run()
	os.Exit(code)
}
