package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	mockservice "order_service/internal/mocks/service"
	"order_service/internal/model"
	"order_service/internal/service"
)

func TestService_Order_FromCache(t *testing.T) {
	id := uuid.New()

	c := mockservice.NewCache(t)
	r := mockservice.NewRepository(t)

	s := service.New(r, c, 100)

	expected := model.Order{
		ID: id,
	}

	c.EXPECT().Get(id.String()).Return(expected).Once()

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

	s := service.New(r, c, 100)

	c.EXPECT().Get(id.String()).Return(nil).Once()

	expected := model.Order{
		ID: id,
	}

	r.EXPECT().Orders(context.Background(), model.OrderFilter{
		OrderID: id,
	}).Return([]model.Order{expected}, nil).Once()

	c.EXPECT().Set(id.String(), expected).Return().Once()

	order, err := s.Order(context.Background(), id)

	require.NoError(t, err)
	require.Equal(t, expected, order)
}

func TestService_Order_RepositoryError(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()

	c := mockservice.NewCache(t)
	r := mockservice.NewRepository(t)

	s := service.New(r, c, 100)

	c.EXPECT().Get(id.String()).Return(nil).Once()

	testErr := model.ErrOrderNotFound
	r.EXPECT().Orders(ctx, model.OrderFilter{
		OrderID: id,
	}).Return(nil, testErr).Once()

	c.AssertNotCalled(t, "Set")

	order, err := s.Order(ctx, id)

	require.Error(t, err)
	require.Equal(t, testErr, err)
	require.Equal(t, model.Order{}, order)
}

func TestService_Order_Empty(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()

	c := mockservice.NewCache(t)
	r := mockservice.NewRepository(t)

	s := service.New(r, c, 100)

	c.EXPECT().Get(id.String()).Return(nil).Once()

	r.EXPECT().Orders(ctx, model.OrderFilter{
		OrderID: id,
	}).Return([]model.Order{}, model.ErrOrderNotFound).Once()

	c.AssertNotCalled(t, "Set")

	order, err := s.Order(ctx, id)

	require.Error(t, err)
	require.Equal(t, model.ErrOrderNotFound, err)
	require.Equal(t, model.Order{}, order)
}

func TestService_ProcessOrder(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()

	c := mockservice.NewCache(t)
	r := mockservice.NewRepository(t)

	s := service.New(r, c, 100)

	testOrder := model.Order{
		ID: id,
	}

	r.EXPECT().CreateOrder(ctx, testOrder).Return(testOrder, nil).Once()

	c.EXPECT().Set(id.String(), testOrder).Return().Once()

	err := s.ProcessOrder(ctx, testOrder)

	require.NoError(t, err)
	require.Equal(t, testOrder, model.Order{
		ID: id,
	})
}

func TestService_ProcessOrder_Error(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()

	c := mockservice.NewCache(t)
	r := mockservice.NewRepository(t)

	s := service.New(r, c, 100)

	r.EXPECT().CreateOrder(ctx, model.Order{
		ID: id,
	}).Return(model.Order{}, pgx.ErrNoRows).Once()

	c.AssertNotCalled(t, "Set")

	err := s.ProcessOrder(ctx, model.Order{
		ID: id,
	})

	require.Error(t, err)
	require.Equal(t, pgx.ErrNoRows, err)

	r.AssertExpectations(t)
	c.AssertExpectations(t)
}

func TestService_WarmUpCache(t *testing.T) {
	ctx := context.Background()

	r := mockservice.NewRepository(t)
	c := mockservice.NewCache(t)

	limit := uint64(50)
	s := service.New(r, c, limit)

	testOrders := []model.Order{
		createTestOrder(),
		createTestOrder(),
		createTestOrder(),
	}

	r.EXPECT().Orders(ctx, model.OrderFilter{
		IsRecent: true,
		Limit:    limit,
	}).Return(testOrders, nil).Once()

	for _, order := range testOrders {
		c.EXPECT().Set(order.ID.String(), order).Return().Once()
	}

	err := s.WarmUpCache(ctx)

	require.NoError(t, err)

	r.AssertExpectations(t)
	c.AssertExpectations(t)
}

func TestService_WarmUpCache_Error(t *testing.T) {
	ctx := context.Background()

	r := mockservice.NewRepository(t)
	c := mockservice.NewCache(t)

	s := service.New(r, c, 100)

	r.EXPECT().Orders(ctx, model.OrderFilter{
		IsRecent: true,
		Limit:    uint64(100),
	}).Return(nil, pgx.ErrNoRows).Once()

	c.AssertNotCalled(t, "Set")

	err := s.WarmUpCache(ctx)

	require.Error(t, err)
	require.Equal(t, pgx.ErrNoRows, err)

	r.AssertExpectations(t)
	c.AssertExpectations(t)
}

func createTestOrder() model.Order {
	return model.Order{
		ID:                uuid.New(),
		TrackNumber:       "WBILMTESTTRACK",
		Entry:             "WBIL",
		Locale:            "en",
		InternalSignature: "",
		DeliveryService:   "meest",
		SmID:              99,

		Customer: model.Customer{
			ID:    uuid.New(),
			Name:  "Test Testov",
			Email: "test@example.com",
			Phone: "+79990000000",
		},

		Address: model.Address{
			ID:         uuid.New(),
			CustomerID: uuid.New(),
			Zip:        "2639809",
			City:       "Kiryat Mozkin",
			Address:    "Ploshad Mira 15",
			Region:     "Kraiot",
		},

		Payment: model.Payment{
			ID:            uuid.New(),
			OrderID:       uuid.New(),
			TransactionID: uuid.New(),
			RequestID:     uuid.New(),
			Currency:      "USD",
			Provider:      "wbpay",
			Amount:        1817,
			Timestamp:     time.Now().Unix(),
			Bank:          "alpha",
			DeliveryCost:  1500,
			GoodsTotal:    317,
			CustomFee:     0,
		},

		Items: []model.OrderItem{
			{
				ID:      uuid.New(),
				OrderID: uuid.New(),
				Item: model.Item{
					ID:    uuid.New(),
					Name:  "Mascaras",
					Brand: "Vivienne Sabo",
				},
				ChrtID:     9934930,
				Price:      453,
				Sale:       30,
				Quantity:   1,
				TotalPrice: 317,
				Status:     "pending",
				Size:       "0",
			},
		},
	}
}
