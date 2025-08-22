package processor

import (
	"context"
	"encoding/json"
	"time"

	"github.com/IBM/sarama"
	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"order_service/internal/model"
)

var (
	errValidationOrderUIDEmpty = errors.New("validation failed: order_uid is emptyr")
	errValidationNoItems       = errors.New("validation failed: order contains no items")
	errValidationNoValidItems  = errors.New("validation failed: order contains no valid items")
)

type Service interface {
	ProcessOrder(ctx context.Context, order model.Order) error
}

type OrderProcessor struct {
	group   sarama.ConsumerGroup
	service Service
	topics  []string
}

type consumerGroupHandler struct {
	service Service
}

func New(brokers []string, topics []string, groupID string, service Service) *OrderProcessor {
	config := sarama.NewConfig()

	group, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		log.Error().Stack().Err(err).Send()
	}

	return &OrderProcessor{
		group:   group,
		topics:  topics,
		service: service,
	}
}

func (p *OrderProcessor) Start(ctx context.Context) error {

	log.Info().Msg("Starting Kafka consumer...")

	handler := &consumerGroupHandler{
		service: p.service,
	}

	for {
		if err := ctx.Err(); err != nil {
			return errors.WithStack(err)
		}

		err := p.group.Consume(ctx, p.topics, handler)
		if err != nil {
			return errors.WithStack(err)
		}
	}
}

func (p *OrderProcessor) Stop() error {
	log.Info().Msg("Closing Kafka consumer...")

	return p.group.Close()
}

func (h consumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (h consumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (h consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		log.Info().Msgf("Received message from topic %s, partition %d, offset %d",
			message.Topic, message.Partition, message.Offset)

		if err := h.processOrderMessage(session.Context(), message.Value); err != nil {
			log.Error().Stack().Err(err).Send()
			continue
		}

		session.MarkMessage(message, "")
	}

	return nil
}

func (h consumerGroupHandler) processOrderMessage(ctx context.Context, message []byte) error {
	var msg orderMessage
	err := json.Unmarshal(message, &msg)
	if err != nil {
		return errors.WithStack(err)
	}

	err = validateMessage(msg)
	if err != nil {
		return err
	}

	err = h.service.ProcessOrder(ctx, orderToModel(msg))
	if err != nil {
		return err
	}

	return nil
}

func validateMessage(msg orderMessage) error {
	if msg.OrderUID == uuid.Nil {
		return errValidationOrderUIDEmpty
	}

	if msg.CustomerID == uuid.Nil {
		return errValidationNoItems
	}

	if len(msg.Items) == 0 {
		return errValidationNoItems
	}

	isValid := false
	for _, i := range msg.Items {
		if i.ChrtID > 0 && i.Rid != uuid.Nil {
			isValid = true
			break
		}
	}

	if !isValid {
		return errValidationNoValidItems
	}

	return nil
}

type orderMessage struct {
	OrderUID          uuid.UUID `json:"order_uid"`
	TrackNumber       string    `json:"track_number"`
	Entry             string    `json:"entry"`
	Delivery          delivery  `json:"delivery"`
	Payment           payment   `json:"payment"`
	Items             []item    `json:"items"`
	Locale            string    `json:"locale"`
	InternalSignature string    `json:"internal_signature"`
	CustomerID        uuid.UUID `json:"customer_id"`
	DeliveryService   string    `json:"delivery_service"`
	ShardKey          string    `json:"shardkey"`
	SmID              int64     `json:"sm_id"`
	DateCreated       time.Time `json:"date_created"`
	OofShard          string    `json:"oof_shard"`
}

type delivery struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Zip     string `json:"zip"`
	City    string `json:"city"`
	Address string `json:"address"`
	Region  string `json:"region"`
	Email   string `json:"email"`
}

type payment struct {
	Transaction  uuid.UUID `json:"transaction"`
	RequestID    uuid.UUID `json:"request_id"`
	Currency     string    `json:"currency"`
	Provider     string    `json:"provider"`
	Amount       int64     `json:"amount"`
	PaymentDt    int64     `json:"payment_dt"`
	Bank         string    `json:"bank"`
	DeliveryCost int64     `json:"delivery_cost"`
	GoodsTotal   int64     `json:"goods_total"`
	CustomFee    int64     `json:"custom_fee"`
}

type item struct {
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
	Status      int64     `json:"status"`
}

func orderToModel(msg orderMessage) model.Order {
	return model.Order{
		ID:                msg.OrderUID,
		TrackNumber:       msg.TrackNumber,
		Entry:             msg.Entry,
		Items:             itemsToModels(msg.Items),
		Locale:            msg.Locale,
		InternalSignature: msg.InternalSignature,
		DeliveryService:   msg.DeliveryService,
		SmID:              msg.SmID,
		Created:           msg.DateCreated,
		Customer:          deliveryToCustomer(msg.CustomerID, msg.Delivery),
		Address:           deliveryToAddress(msg.CustomerID, msg.Delivery),
		Payment:           paymentToModel(msg.OrderUID, msg.Payment),
	}
}

func deliveryToAddress(customerID uuid.UUID, delivery delivery) model.Address {
	return model.Address{
		CustomerID: customerID,
		Zip:        delivery.Zip,
		City:       delivery.City,
		Address:    delivery.Address,
		Region:     delivery.Region,
	}
}

func deliveryToCustomer(customerID uuid.UUID, delivery delivery) model.Customer {
	return model.Customer{
		ID:    customerID,
		Name:  delivery.Name,
		Email: delivery.Email,
		Phone: delivery.Phone,
	}
}

func itemsToModels(items []item) []model.OrderItem {
	orderItems := make([]model.OrderItem, 0, len(items))
	for _, i := range items {
		orderItems = append(orderItems, itemToModel(i))
	}
	return orderItems
}

func itemToModel(i item) model.OrderItem {
	return model.OrderItem{
		ID:     i.Rid,
		ChrtID: i.ChrtID,
		Item: model.Item{
			ID:    i.NmID,
			Name:  i.Name,
			Brand: i.Brand,
			Price: i.Price,
		},
		Price:      i.Price,
		Sale:       i.Sale,
		Size:       i.Size,
		TotalPrice: i.TotalPrice,
		Status:     model.StatusCode[i.Status],
	}
}

func paymentToModel(orderID uuid.UUID, payment payment) model.Payment {
	return model.Payment{
		OrderID:       orderID,
		TransactionID: payment.Transaction,
		RequestID:     payment.RequestID,
		Currency:      payment.Currency,
		Provider:      payment.Provider,
		Amount:        payment.Amount,
		Timestamp:     payment.PaymentDt,
		Bank:          payment.Bank,
		DeliveryCost:  payment.DeliveryCost,
		GoodsTotal:    payment.GoodsTotal,
		CustomFee:     payment.CustomFee,
	}
}
