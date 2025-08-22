package api

import (
	"context"
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"order_service/internal/model"
)

type Service interface {
	Order(ctx context.Context, orderID uuid.UUID) (model.Order, error)
}

type API struct {
	*echo.Echo
	service Service
}

func New(service Service) *API {
	a := &API{
		Echo:    echo.New(),
		service: service,
	}

	a.Static("/static", "/static")

	a.GET("/order/:id", a.order)
	a.GET("/", a.serveIndex)

	return a
}

func (a *API) serveIndex(c echo.Context) error {
	return c.File("/static/index.html")
}

func (a *API) order(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"reason": "invalid request format or params"})
	}

	orderID, err := uuid.Parse(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"reason": "invalid request format or params"})
	}

	order, err := a.service.Order(c.Request().Context(), orderID)
	if err != nil {
		if errors.Is(err, model.ErrOrderNotFound) {
			return c.JSON(http.StatusNotFound, echo.Map{"reason": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"reason": err.Error()})
	}

	return c.JSON(http.StatusOK, a.orderFromModel(order))
}

type deliveryResponse struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Zip     string `json:"zip"`
	City    string `json:"city"`
	Address string `json:"address"`
	Region  string `json:"region"`
	Email   string `json:"email"`
}

type paymentResponse struct {
	Transaction  uuid.UUID `json:"transaction"`
	RequestID    uuid.UUID `json:"request_id"`
	Currency     string    `json:"currency"`
	Provider     string    `json:"provider"`
	Amount       int64     `json:"amount"`
	PaymentDT    int64     `json:"payment_dt"`
	Bank         string    `json:"bank"`
	DeliveryCost int64     `json:"delivery_cost"`
	GoodsTotal   int64     `json:"goods_total"`
	CustomFee    int64     `json:"custom_fee"`
}

type itemResponse struct {
	ChrtID      int64     `json:"chrt_id"`
	TrackNumber string    `json:"track_number"`
	Price       int64     `json:"price"`
	Rid         uuid.UUID `json:"rid"`
	Name        string    `json:"name"`
	Sale        int64     `json:"sale"`
	Size        string    `json:"size"`
	TotalPrice  int64     `json:"total_price"`
	NmID        uuid.UUID `json:"nm_id"`
	Brand       string    `json:"brand"`
	Status      string    `json:"status"`
}

type orderResponse struct {
	ID                uuid.UUID        `json:"id"`
	TrackNumber       string           `json:"track_number"`
	Entry             string           `json:"entry"`
	Delivery          deliveryResponse `json:"delivery"`
	Payment           paymentResponse  `json:"payment"`
	Items             []itemResponse   `json:"items"`
	Locale            string           `json:"locale"`
	InternalSignature string           `json:"internal_signature"`
	CustomerID        uuid.UUID        `json:"customer_id"`
	DeliveryService   string           `json:"delivery_service"`
	SmID              int64            `json:"sm_id"`
	DateCreated       time.Time        `json:"date_created"`
}

func (a *API) orderFromModel(order model.Order) orderResponse {
	return orderResponse{
		ID:                order.ID,
		TrackNumber:       order.TrackNumber,
		Entry:             order.Entry,
		Delivery:          a.deliveryFromModels(order.Customer, order.Address),
		Payment:           a.paymentFromModel(order.Payment),
		Items:             a.itemsFromModels(order.Items),
		Locale:            order.Locale,
		InternalSignature: order.InternalSignature,
		CustomerID:        order.Customer.ID,
		DeliveryService:   order.DeliveryService,
		SmID:              order.SmID,
		DateCreated:       order.Created,
	}
}

func (a *API) deliveryFromModels(customer model.Customer, address model.Address) deliveryResponse {
	return deliveryResponse{
		Name:    customer.Name,
		Phone:   customer.Phone,
		Zip:     address.Zip,
		City:    address.City,
		Address: address.Address,
		Region:  address.Region,
		Email:   customer.Email,
	}
}

func (a *API) paymentFromModel(payment model.Payment) paymentResponse {
	return paymentResponse{
		Transaction:  payment.TransactionID,
		RequestID:    payment.RequestID,
		Currency:     payment.Currency,
		Provider:     payment.Provider,
		Amount:       payment.Amount,
		PaymentDT:    payment.Timestamp,
		Bank:         payment.Bank,
		DeliveryCost: payment.DeliveryCost,
		GoodsTotal:   payment.GoodsTotal,
		CustomFee:    payment.CustomFee,
	}
}

func (a *API) itemsFromModels(items []model.OrderItem) []itemResponse {
	var r = make([]itemResponse, 0, len(items))
	for _, item := range items {
		r = append(r, a.itemFromModel(item))
	}

	return r
}

func (a *API) itemFromModel(orderItem model.OrderItem) itemResponse {
	return itemResponse{
		ChrtID:     orderItem.ChrtID,
		Price:      orderItem.Price,
		Rid:        orderItem.ID,
		Name:       orderItem.Item.Name,
		Sale:       orderItem.Sale,
		Size:       orderItem.Size,
		TotalPrice: orderItem.TotalPrice,
		NmID:       orderItem.Item.ID,
		Brand:      orderItem.Item.Brand,
		Status:     string(orderItem.Status),
	}
}
