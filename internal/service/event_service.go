package service

import (
	"context"

	"go-gin-high-concurrency/internal/cache"
	"go-gin-high-concurrency/internal/model"
	"go-gin-high-concurrency/internal/repository"

	"github.com/google/uuid"
)

type EventService interface {
	List(ctx context.Context) ([]*model.Event, error)
	GetByEventID(ctx context.Context, eventID uuid.UUID) (*model.Event, error)
	Create(ctx context.Context, event *model.Event) (*model.Event, error)
	UpdateByEventID(ctx context.Context, eventID uuid.UUID, params model.UpdateEventParams) (*model.Event, error)
	// OpenForSale 活動開賣：預熱該活動底下所有票種的 Redis 庫存
	OpenForSale(ctx context.Context, eventID uuid.UUID) error
}

type EventServiceImpl struct {
	repo             repository.EventRepository
	ticketRepo       repository.TicketRepository
	inventoryManager cache.RedisTicketInventoryManager
}

func NewEventService(repo repository.EventRepository, ticketRepo repository.TicketRepository, inventoryManager cache.RedisTicketInventoryManager) EventService {
	return &EventServiceImpl{repo: repo, ticketRepo: ticketRepo, inventoryManager: inventoryManager}
}

func (s *EventServiceImpl) List(ctx context.Context) ([]*model.Event, error) {
	return s.repo.List(ctx)
}

func (s *EventServiceImpl) GetByEventID(ctx context.Context, eventID uuid.UUID) (*model.Event, error) {
	return s.repo.FindByEventID(ctx, eventID)
}

func (s *EventServiceImpl) Create(ctx context.Context, event *model.Event) (*model.Event, error) {
	if event.EventID == uuid.Nil {
		event.EventID = uuid.New()
	}
	return s.repo.Create(ctx, event)
}

func (s *EventServiceImpl) UpdateByEventID(ctx context.Context, eventID uuid.UUID, params model.UpdateEventParams) (*model.Event, error) {
	event, err := s.repo.FindByEventID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, event.ID, params)
}

func (s *EventServiceImpl) OpenForSale(ctx context.Context, eventID uuid.UUID) error {
	event, err := s.repo.FindByEventID(ctx, eventID)
	if err != nil {
		return err
	}
	tickets, err := s.ticketRepo.ListByEventID(ctx, event.ID)
	if err != nil {
		return err
	}
	for _, t := range tickets {
		if err := s.inventoryManager.WarmUpInventory(ctx, t.ID, t.TotalStock, t.Price, t.MaxPerUser); err != nil {
			return err
		}
	}
	return nil
}
