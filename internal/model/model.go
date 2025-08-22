package model

import (
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
)

var (
	ErrOrderNotFound = errors.New("order not found")
	ErrItemNotFound  = errors.New("item not found")
)

type ItemStatus string

const (
	Pending    ItemStatus = "pending"
	Processing ItemStatus = "processing"
	Assembling ItemStatus = "assembling"
	InTransit  ItemStatus = "in_transit"
	Delivered  ItemStatus = "delivered"
	Cancelled  ItemStatus = "cancelled"
	Returned   ItemStatus = "returned"
)

var StatusCode = map[int64]ItemStatus{
	100: Pending,
	200: Processing,
	202: Delivered,
	300: Assembling,
	400: InTransit,
	500: Cancelled,
	600: Returned,
}

type Customer struct {
	ID    uuid.UUID
	Name  string
	Email string
	Phone string
}

type Address struct {
	ID         uuid.UUID
	CustomerID uuid.UUID
	Zip        string
	City       string
	Address    string
	Region     string
}

type Payment struct {
	ID            uuid.UUID
	OrderID       uuid.UUID
	TransactionID uuid.UUID
	RequestID     uuid.UUID
	Currency      string
	Provider      string
	Amount        int64
	Timestamp     int64
	Bank          string
	DeliveryCost  int64
	GoodsTotal    int64
	CustomFee     int64
}

//type Item struct {
//	NmID             uuid.UUID
//	Name             string
//	Category         string
//	CommonAttributes []Attribute
//}

type Item struct {
	ID    uuid.UUID
	Name  string
	Price int64
	Size  string
	Brand string
	//SpecificAttributes []Attribute
}

type Order struct {
	ID                uuid.UUID
	TrackNumber       string
	Entry             string
	Items             []OrderItem
	Locale            string
	InternalSignature string
	DeliveryService   string
	SmID              int64
	Created           time.Time

	Customer Customer
	Address  Address
	Payment  Payment
}

type OrderFilter struct {
	OrderID  uuid.UUID
	IsRecent bool
	Limit    uint64
}

type OrderItem struct {
	ID         uuid.UUID
	OrderID    uuid.UUID
	Item       Item
	ChrtID     int64
	Price      int64
	Sale       int64
	Size       string
	Quantity   int64
	TotalPrice int64
	Status     ItemStatus
	Created    time.Time
}

type Size struct {
	ID     int64
	ItemID uuid.UUID
	Size   string
}

//type Attribute struct {
//	ID       int64
//	Name     string
//	Value    string
//	IsCommon bool
//}
