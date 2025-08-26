package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"order_service/internal/api"
	"order_service/internal/model"

	mockapi "order_service/internal/mocks/api"
)

func TestAPI_Order(t *testing.T) {
	s := mockapi.NewService(t)
	ctx := context.Background()
	a := api.New(s)

	testOrder := createTestOrder()

	s.EXPECT().Order(ctx, testOrder.ID).
		Return(testOrder, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/order/"+testOrder.ID.String(), nil)
	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp api.OrderResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, testOrder.ID.String(), resp.ID.String())
}

func TestAPI_Order_InvalidID(t *testing.T) {
	s := mockapi.NewService(t)
	a := api.New(s)

	req := httptest.NewRequest(http.MethodGet, "/order/invalid-uuid", nil)
	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)

	var resp echo.Map
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, echo.Map{"reason": "invalid request format or params"}, resp)
}

func TestAPI_Order_EmptyID(t *testing.T) {
	s := mockapi.NewService(t)
	a := api.New(s)

	req := httptest.NewRequest(http.MethodGet, "/order/", nil)
	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)

	var resp echo.Map
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Contains(t, echo.Map{"reason": "invalid request format or params"}, resp)
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
			Timestamp:     1637907727,
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
