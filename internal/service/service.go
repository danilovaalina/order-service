package service

import (
	"context"

	"github.com/google/uuid"
	"order_service/internal/model"
)

type Repository interface {
	Orders(ctx context.Context, opts model.OrderFilter) ([]model.Order, error)
	CreateOrder(ctx context.Context, order model.Order) (model.Order, error)
}

type Cache interface {
	Set(key string, value interface{})
	Get(key string) interface{}
}

type Service struct {
	repository Repository
	cache      Cache
	limit      uint64
}

func New(repository Repository, cache Cache, limit uint64) *Service {
	return &Service{
		repository: repository,
		cache:      cache,
		limit:      limit,
	}
}

func (s *Service) Order(ctx context.Context, orderID uuid.UUID) (model.Order, error) {
	order := s.cache.Get(orderID.String())
	if order != nil {
		return order.(model.Order), nil
	}

	orders, err := s.repository.Orders(ctx, model.OrderFilter{
		OrderID: orderID,
	})
	if err != nil {
		return model.Order{}, err
	}

	s.cache.Set(orderID.String(), orders[0])

	return orders[0], nil
}

func (s *Service) ProcessOrder(ctx context.Context, order model.Order) error {
	newOrder, err := s.repository.CreateOrder(ctx, order)
	if err != nil {
		return err
	}

	s.cache.Set(newOrder.ID.String(), newOrder)
	return nil
}

func (s *Service) WarmUpCache(ctx context.Context) error {
	orders, err := s.repository.Orders(ctx, model.OrderFilter{
		IsRecent: true,
		Limit:    s.limit,
	})
	if err != nil {
		return err
	}

	for _, order := range orders {
		s.cache.Set(order.ID.String(), order)
	}

	return nil
}
