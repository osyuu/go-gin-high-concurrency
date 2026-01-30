package service

import (
	"context"
	"errors"
	"testing"

	cacheMocks "go-gin-high-concurrency/internal/cache/mocks"
	"go-gin-high-concurrency/internal/model"
	repoMocks "go-gin-high-concurrency/internal/repository/mocks"
	"go-gin-high-concurrency/internal/service"
	"go-gin-high-concurrency/pkg/app_errors"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupEventServiceMocks(t *testing.T) (
	*repoMocks.MockEventRepository,
	*repoMocks.MockTicketRepository,
	*cacheMocks.MockRedisTicketInventoryManager,
) {
	eventRepo := repoMocks.NewMockEventRepository(t)
	ticketRepo := repoMocks.NewMockTicketRepository(t)
	inventoryManager := cacheMocks.NewMockRedisTicketInventoryManager(t)
	return eventRepo, ticketRepo, inventoryManager
}

func TestEventService_OpenForSale(t *testing.T) {
	ctx := context.Background()
	eventID := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	event := &model.Event{ID: 1, EventID: eventID, Name: "Test Event"}

	t.Run("Success - warms all tickets under event", func(t *testing.T) {
		eventRepo, ticketRepo, inventoryManager := setupEventServiceMocks(t)
		eventService := service.NewEventService(eventRepo, ticketRepo, inventoryManager)

		tickets := []*model.Ticket{
			{ID: 10, EventID: 1, Name: "A", TotalStock: 100, Price: 50, MaxPerUser: 2},
			{ID: 11, EventID: 1, Name: "B", TotalStock: 200, Price: 80, MaxPerUser: 5},
		}

		eventRepo.EXPECT().FindByEventID(ctx, eventID).Return(event, nil).Once()
		ticketRepo.EXPECT().ListByEventID(ctx, 1).Return(tickets, nil).Once()
		inventoryManager.EXPECT().WarmUpInventory(ctx, 10, 100, 50.0, 2).Return(nil).Once()
		inventoryManager.EXPECT().WarmUpInventory(ctx, 11, 200, 80.0, 5).Return(nil).Once()

		err := eventService.OpenForSale(ctx, eventID)

		require.NoError(t, err)
		eventRepo.AssertExpectations(t)
		ticketRepo.AssertExpectations(t)
		inventoryManager.AssertExpectations(t)
	})

	t.Run("Success - no tickets under event", func(t *testing.T) {
		eventRepo, ticketRepo, inventoryManager := setupEventServiceMocks(t)
		eventService := service.NewEventService(eventRepo, ticketRepo, inventoryManager)

		eventRepo.EXPECT().FindByEventID(ctx, eventID).Return(event, nil).Once()
		ticketRepo.EXPECT().ListByEventID(ctx, 1).Return([]*model.Ticket{}, nil).Once()

		err := eventService.OpenForSale(ctx, eventID)

		require.NoError(t, err)
		eventRepo.AssertExpectations(t)
		ticketRepo.AssertExpectations(t)
		inventoryManager.AssertNotCalled(t, "WarmUpInventory")
	})

	t.Run("Failed - event not found", func(t *testing.T) {
		eventRepo, ticketRepo, inventoryManager := setupEventServiceMocks(t)
		eventService := service.NewEventService(eventRepo, ticketRepo, inventoryManager)

		eventRepo.EXPECT().FindByEventID(ctx, eventID).Return(nil, app_errors.ErrEventNotFound).Once()

		err := eventService.OpenForSale(ctx, eventID)

		require.Error(t, err)
		assert.ErrorIs(t, err, app_errors.ErrEventNotFound)
		eventRepo.AssertExpectations(t)
		ticketRepo.AssertNotCalled(t, "ListByEventID")
		inventoryManager.AssertNotCalled(t, "WarmUpInventory")
	})

	t.Run("Failed - ListByEventID error", func(t *testing.T) {
		eventRepo, ticketRepo, inventoryManager := setupEventServiceMocks(t)
		eventService := service.NewEventService(eventRepo, ticketRepo, inventoryManager)

		eventRepo.EXPECT().FindByEventID(ctx, eventID).Return(event, nil).Once()
		ticketRepo.EXPECT().ListByEventID(ctx, 1).Return(nil, errors.New("db error")).Once()

		err := eventService.OpenForSale(ctx, eventID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
		eventRepo.AssertExpectations(t)
		ticketRepo.AssertExpectations(t)
		inventoryManager.AssertNotCalled(t, "WarmUpInventory")
	})

	t.Run("Failed - WarmUpInventory error", func(t *testing.T) {
		eventRepo, ticketRepo, inventoryManager := setupEventServiceMocks(t)
		eventService := service.NewEventService(eventRepo, ticketRepo, inventoryManager)

		tickets := []*model.Ticket{
			{ID: 10, EventID: 1, TotalStock: 100, Price: 50, MaxPerUser: 2},
		}

		eventRepo.EXPECT().FindByEventID(ctx, eventID).Return(event, nil).Once()
		ticketRepo.EXPECT().ListByEventID(ctx, 1).Return(tickets, nil).Once()
		inventoryManager.EXPECT().WarmUpInventory(ctx, 10, 100, 50.0, 2).Return(errors.New("redis error")).Once()

		err := eventService.OpenForSale(ctx, eventID)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "redis error")
		eventRepo.AssertExpectations(t)
		ticketRepo.AssertExpectations(t)
		inventoryManager.AssertExpectations(t)
	})
}
