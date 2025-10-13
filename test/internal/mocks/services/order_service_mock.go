package services

import (
	"context"
	"go-gin-high-concurrency/internal/model"

	"github.com/stretchr/testify/mock"
)

type OrderServiceMock struct {
	mock.Mock
}

func NewOrderServiceMock() *OrderServiceMock {
	return &OrderServiceMock{}
}

func (m *OrderServiceMock) Create(ctx context.Context, req model.CreateOrderRequest) (*model.Order, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Order), args.Error(1)
}

func (m *OrderServiceMock) List(ctx context.Context) ([]*model.Order, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Order), args.Error(1)
}

func (m *OrderServiceMock) GetByID(ctx context.Context, id int) (*model.Order, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Order), args.Error(1)
}

func (m *OrderServiceMock) Confirm(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *OrderServiceMock) Cancel(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *OrderServiceMock) Delete(ctx context.Context, id int) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}
