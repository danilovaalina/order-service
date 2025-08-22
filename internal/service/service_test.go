package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	mockservice "order_service/internal/mocks/service"
	"order_service/internal/model"
	"order_service/internal/service"
)

func TestService_Order_FromCache(t *testing.T) {
	id := uuid.New()

	c := mockservice.NewCache(t)
	r := mockservice.NewRepository(t)

	expected := model.Order{
		ID: id,
	}

	c.EXPECT().Get(id.String()).Return(expected).Once()

	s := service.New(r, c, 100)

	order, err := s.Order(context.Background(), id)

	require.NoError(t, err)
	require.Equal(t, expected, order)

	r.AssertNotCalled(t, "Orders")
	c.AssertNotCalled(t, "Set")
}

func TestService_Order_FromRepository(t *testing.T) {
	id := uuid.New()

	c := mockservice.NewCache(t)
	r := mockservice.NewRepository(t)

	c.EXPECT().Get(id.String()).Return(nil).Once()

	expected := model.Order{
		ID: id,
	}

	r.EXPECT().Orders(context.Background(), model.OrderFilter{
		OrderID: id,
	}).Return([]model.Order{expected}, nil).Once()

	c.EXPECT().Set(id.String(), expected).Return().Once()

	s := service.New(r, c, 100)
	order, err := s.Order(context.Background(), id)

	require.NoError(t, err)
	require.Equal(t, expected, order)
}

func TestService_Order_CacheMiss_RepositoryError(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()

	c := mockservice.NewCache(t)
	r := mockservice.NewRepository(t)

	c.EXPECT().Get(id.String()).Return(nil).Once()

	testErr := model.ErrOrderNotFound
	r.EXPECT().Orders(ctx, model.OrderFilter{
		OrderID: id,
	}).Return(nil, testErr).Once()

	c.AssertNotCalled(t, "Set")

	s := service.New(r, c, 100)
	order, err := s.Order(ctx, id)

	require.Error(t, err)
	require.Equal(t, testErr, err)
	require.Equal(t, model.Order{}, order)
}
