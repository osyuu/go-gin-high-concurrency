package service

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"go-gin-high-concurrency/internal/model"
	"go-gin-high-concurrency/internal/repository"
	"go-gin-high-concurrency/internal/service"

	"github.com/stretchr/testify/assert"
)

// Simulates real scenario: 100 users simultaneously purchasing 10 tickets
func TestConcurrentOrderCreate_NoOversell(t *testing.T) {
	cleanup := setupTestWithTruncate(t)
	defer cleanup()

	ctx := context.Background()
	orderRepo := repository.NewOrderRepository(getTestDB())
	ticketRepo := repository.NewTicketRepository(getTestDB())
	orderService := service.NewOrderService(getTestDB(), orderRepo, ticketRepo)

	// Concurrency parameters
	concurrentUsers := 100 // 100 different users
	quantityPerUser := 1   // 1 ticket per user
	totalStock := 10       // Only 10 tickets available

	// Create 100 different users
	userIDs := make([]int, concurrentUsers)
	for i := 0; i < concurrentUsers; i++ {
		userIDs[i] = createTestUser(t, fmt.Sprintf("User%d", i), fmt.Sprintf("user%d@test.com", i))
	}

	// Create ticket (10 tickets)
	ticketID := createTestTicketWithStock(t, 2001, "Popular Concert", 1000, totalStock, 1)

	// Collect results
	var wg sync.WaitGroup
	successCount := 0
	failCount := 0
	var mu sync.Mutex

	// Simulate 100 different users purchasing 10 tickets concurrently
	for i := 0; i < concurrentUsers; i++ {
		wg.Add(1)
		go func(userIndex int) {
			defer wg.Done()

			req := model.CreateOrderRequest{
				UserID:   userIDs[userIndex],
				TicketID: ticketID,
				Quantity: quantityPerUser,
			}

			_, err := orderService.Create(ctx, req)

			mu.Lock()
			if err == nil {
				successCount++
			} else {
				failCount++
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Verify results
	t.Logf("100 users competing for 10 tickets - Success: %d, Failed: %d", successCount, failCount)

	// Critical assertions: exactly 10 tickets sold, no overselling
	ticket, _ := ticketRepo.FindByID(ctx, ticketID)
	assert.Equal(t, totalStock, successCount, "Successful orders should equal total stock")
	assert.Equal(t, 0, ticket.RemainingStock, "Remaining stock should be 0")
	assert.Equal(t, concurrentUsers-totalStock, failCount, "90 users should fail")
}

// TestConcurrentRaceCondition tests for race conditions
func TestConcurrentRaceCondition(t *testing.T) {
	cleanup := setupTestWithTruncate(t)
	defer cleanup()

	ctx := context.Background()
	orderRepo := repository.NewOrderRepository(getTestDB())
	ticketRepo := repository.NewTicketRepository(getTestDB())
	orderService := service.NewOrderService(getTestDB(), orderRepo, ticketRepo)

	// Create 50 users
	userIDs := make([]int, 50)
	for i := 0; i < 50; i++ {
		userIDs[i] = createTestUser(t, fmt.Sprintf("RaceUser%d", i), fmt.Sprintf("race%d@test.com", i))
	}

	ticketID := createTestTicketWithStock(t, 2002, "Race Test Concert", 1000, 50, 1)

	var wg sync.WaitGroup

	// 50 concurrent requests
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			req := model.CreateOrderRequest{
				UserID:   userIDs[index],
				TicketID: ticketID,
				Quantity: 1,
			}

			orderService.Create(ctx, req)
		}(i)
	}

	wg.Wait()

	// If there are race conditions, go test -race will detect them
	t.Log("Race condition detection completed (use go test -race for detailed report)")
}
