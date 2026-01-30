package service

import (
	"context"

	"go-gin-high-concurrency/internal/model"
	"go-gin-high-concurrency/internal/repository"

	"github.com/google/uuid"
)

type TicketService interface {
	List(ctx context.Context) ([]*model.Ticket, error)
	GetByTicketID(ctx context.Context, ticketID uuid.UUID) (*model.Ticket, error)
	Create(ctx context.Context, ticket *model.Ticket) (*model.Ticket, error)
	UpdateByTicketID(ctx context.Context, ticketID uuid.UUID, params model.UpdateTicketParams) (*model.Ticket, error)
	DeleteByTicketID(ctx context.Context, ticketID uuid.UUID) error
}

type TicketServiceImpl struct {
	repo repository.TicketRepository
}

func NewTicketService(repo repository.TicketRepository) TicketService {
	return &TicketServiceImpl{repo: repo}
}

func (s *TicketServiceImpl) List(ctx context.Context) ([]*model.Ticket, error) {
	return s.repo.List(ctx)
}

func (s *TicketServiceImpl) GetByTicketID(ctx context.Context, ticketID uuid.UUID) (*model.Ticket, error) {
	return s.repo.FindByTicketID(ctx, ticketID)
}

func (s *TicketServiceImpl) Create(ctx context.Context, ticket *model.Ticket) (*model.Ticket, error) {
	ticket.TicketID = uuid.New()
	return s.repo.Create(ctx, ticket)
}

func (s *TicketServiceImpl) UpdateByTicketID(ctx context.Context, ticketID uuid.UUID, params model.UpdateTicketParams) (*model.Ticket, error) {
	return s.repo.Update(ctx, ticketID, params)
}

func (s *TicketServiceImpl) DeleteByTicketID(ctx context.Context, ticketID uuid.UUID) error {
	return s.repo.Delete(ctx, ticketID)
}
