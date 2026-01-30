package service

import (
	"context"

	"go-gin-high-concurrency/internal/model"
	"go-gin-high-concurrency/internal/repository"

	"github.com/google/uuid"
)

type EventService interface {
	List(ctx context.Context) ([]*model.Event, error)
	GetByEventID(ctx context.Context, eventID uuid.UUID) (*model.Event, error)
	Create(ctx context.Context, event *model.Event) (*model.Event, error)
	UpdateByEventID(ctx context.Context, eventID uuid.UUID, params model.UpdateEventParams) (*model.Event, error)
}

type EventServiceImpl struct {
	repo repository.EventRepository
}

func NewEventService(repo repository.EventRepository) EventService {
	return &EventServiceImpl{repo: repo}
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
