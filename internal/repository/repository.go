package repository

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
	_ "github.com/redis/go-redis/v9"
	"order_service/internal/model"
)

type Repository struct {
	pool    *pgxpool.Pool
	builder sq.StatementBuilderType
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool:    pool,
		builder: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

func (r *Repository) CreateOrder(ctx context.Context, order model.Order) (model.Order, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.Order{}, errors.WithStack(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	customer, err := r.createCustomer(ctx, tx, order.Customer)
	if err != nil {
		return model.Order{}, err
	}

	address, err := r.createAddress(ctx, tx, order.Address)
	if err != nil {
		return model.Order{}, err
	}

	newOrder, err := r.createOrder(ctx, tx, order)
	if err != nil {
		return model.Order{}, err
	}
	newOrder.Address = address

	payment, err := r.createPayment(ctx, tx, order.Payment)
	if err != nil {
		return model.Order{}, err
	}
	newOrder.Payment = payment

	items := make([]model.Item, len(order.Items))
	for i, orderItem := range order.Items {
		items[i] = orderItem.Item
	}

	newItems, err := r.createItems(ctx, tx, items)
	if err != nil {
		return model.Order{}, err
	}

	sizes, err := r.createSizes(ctx, tx, order.Items)
	if err != nil {
		return model.Order{}, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return model.Order{}, errors.WithStack(err)
	}

	newOrderItems, err := r.createOrderItems(ctx, order.Items)
	if err != nil {
		return model.Order{}, err
	}
	for i := range newOrderItems {
		for _, newItem := range newItems {
			if newOrderItems[i].Item.ID == newItem.ID {
				newOrderItems[i].Item = newItem
				break
			}
		}
	}

	for i := range newOrderItems {
		for _, size := range sizes {
			if newOrderItems[i].Item.ID == size.ItemID {
				newOrderItems[i].Size = size.Size
				break
			}
		}
	}
	newOrder.Customer = customer
	newOrder.Items = newOrderItems

	return newOrder, nil
}

func (r *Repository) createCustomer(ctx context.Context, tx pgx.Tx, customer model.Customer) (model.Customer, error) {
	query := `
        insert into customer (id, name, email, phone)
        values ($1, $2, $3, $4)
        on conflict (id) 
        do update set
            name = excluded.name,
            email = excluded.email,
            phone = excluded.phone
        returning id, name, email, phone
    `
	rows, err := tx.Query(ctx, query, customer.ID, customer.Name, customer.Email, customer.Phone)
	if err != nil {
		return model.Customer{}, errors.WithStack(err)
	}

	row, err := pgx.CollectExactlyOneRow[customerRow](rows, pgx.RowToStructByNameLax[customerRow])
	if err != nil {
		return model.Customer{}, errors.WithStack(err)
	}

	return r.customerModel(row), nil
}

func (r *Repository) customerModel(row customerRow) model.Customer {
	return model.Customer{
		ID:    row.ID,
		Name:  row.Name,
		Email: row.Email,
		Phone: row.Phone,
	}
}

type customerRow struct {
	ID    uuid.UUID `db:"id"`
	Name  string    `db:"name"`
	Email string    `db:"email"`
	Phone string    `db:"phone"`
}

func (r *Repository) createAddress(ctx context.Context, tx pgx.Tx, address model.Address) (model.Address, error) {
	query := `
        with a as (
            insert into address (customer_id, zip, city, address, region)
            values ($1, $2, $3, $4, $5)
            on conflict (customer_id, zip, city, address, region) do nothing
            returning id, customer_id, zip, city, address, region
        )
        select id, customer_id, zip, city, address, region from a
        union all
        select id, customer_id, zip, city, address, region from address 
        where customer_id = $1 and zip = $2 and city = $3 and address = $4 and region = $5
        and not exists (select a from a)
    `

	rows, err := tx.Query(ctx, query,
		address.CustomerID,
		address.Zip,
		address.City,
		address.Address,
		address.Region,
	)
	if err != nil {
		return model.Address{}, errors.WithStack(err)
	}

	row, err := pgx.CollectExactlyOneRow[addressRow](rows, pgx.RowToStructByNameLax[addressRow])
	if err != nil {
		return model.Address{}, errors.WithStack(err)
	}

	return r.addressModel(row), nil
}

func (r *Repository) addressModel(row addressRow) model.Address {
	return model.Address{
		ID:         row.ID,
		CustomerID: row.CustomerID,
		Zip:        row.Zip,
		City:       row.City,
		Address:    row.Address,
		Region:     row.Region,
	}
}

type addressRow struct {
	ID         uuid.UUID `db:"id"`
	CustomerID uuid.UUID `db:"customer_id"`
	Zip        string    `db:"zip"`
	City       string    `db:"city"`
	Address    string    `db:"address"`
	Region     string    `db:"region"`
}

func (r *Repository) createOrder(ctx context.Context, tx pgx.Tx, order model.Order) (model.Order, error) {
	query := `
        insert into "order" (id, customer_id, address_id, track_number, entry,
                            locale, internal_signature, delivery_service, sm_id, created)
        values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
        on conflict (id) 
        do update set
            track_number = coalesce(nullif(excluded.track_number, ''), "order".track_number),
            delivery_service = coalesce(nullif(excluded.delivery_service, ''), "order".delivery_service),
            internal_signature = coalesce(nullif(excluded.internal_signature, ''), "order".internal_signature),
            sm_id = excluded.sm_id
        returning id, customer_id, track_number, entry, locale, internal_signature, delivery_service, sm_id, created
    `

	rows, err := tx.Query(ctx, query,
		order.ID,
		order.Customer.ID,
		order.Address.ID,
		order.TrackNumber,
		order.Entry,
		order.Locale,
		order.InternalSignature,
		order.DeliveryService,
		order.SmID,
		order.Created,
	)
	if err != nil {
		return model.Order{}, errors.WithStack(err)
	}

	row, err := pgx.CollectExactlyOneRow[orderRow](rows, pgx.RowToStructByNameLax[orderRow])
	if err != nil {
		return model.Order{}, errors.WithStack(err)
	}

	return r.orderModel(row), nil
}

func (r *Repository) createPayment(ctx context.Context, tx pgx.Tx, payment model.Payment) (model.Payment, error) {
	query := `
        insert into payment (order_id, transaction_id, request_id, currency, provider,
                            amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
        values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
        on conflict (order_id) 
        do update set
            transaction_id = excluded.transaction_id,
            request_id = excluded.request_id,
            currency = excluded.currency,
            provider = excluded.provider,
            payment_dt = excluded.payment_dt,
            bank = excluded.bank
        returning id, order_id, transaction_id, request_id, currency, provider, amount, payment_dt, bank, delivery_cost,
            goods_total, custom_fee
    `

	rows, err := tx.Query(ctx, query,
		payment.OrderID,
		payment.TransactionID,
		payment.RequestID,
		payment.Currency,
		payment.Provider,
		payment.Amount,
		payment.Timestamp,
		payment.Bank,
		payment.DeliveryCost,
		payment.GoodsTotal,
		payment.CustomFee,
	)
	if err != nil {
		return model.Payment{}, errors.WithStack(err)
	}

	row, err := pgx.CollectExactlyOneRow[paymentRow](rows, pgx.RowToStructByNameLax[paymentRow])
	if err != nil {
		return model.Payment{}, errors.WithStack(err)
	}

	return paymentModel(row), nil
}

func paymentModel(row paymentRow) model.Payment {
	return model.Payment{
		OrderID:       row.ID,
		TransactionID: row.TransactionID,
		RequestID:     row.RequestID,
		Currency:      row.Currency,
		Provider:      row.Provider,
		Amount:        row.Amount,
		Timestamp:     row.PaymentDt,
		Bank:          row.Bank,
		GoodsTotal:    row.GoodsTotal,
		CustomFee:     row.CustomFee,
	}
}

type paymentRow struct {
	ID            uuid.UUID `db:"id"`
	OrderID       uuid.UUID `db:"order_id"`
	TransactionID uuid.UUID `db:"transaction_id"`
	RequestID     uuid.UUID `db:"request_id"`
	Currency      string    `db:"currency"`
	Provider      string    `db:"provider"`
	Amount        int64     `db:"amount"`
	PaymentDt     int64     `db:"payment_dt"`
	Bank          string    `db:"bank"`
	DeliveryCost  int64     `db:"delivery_cost"`
	GoodsTotal    int64     `db:"goods_total"`
	CustomFee     int64     `db:"custom_fee"`
}

func (r *Repository) createOrderItems(ctx context.Context, orderItems []model.OrderItem) ([]model.OrderItem, error) {
	b := &pgx.Batch{}

	query := `
        insert into order_item (order_id, nm_id, chrt_id, rid, price, sale, quantity,
                               total_price, status)
        values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        on conflict (rid) 
        do update set status = case 
									when excluded.status in ('pending', 'processing', 'assembling', 'in_transit', 
															 'delivered', 'cancelled', 'returned')
										then excluded.status 
									else order_item.status 
        end
        returning rid, order_id, nm_id, chrt_id, price, sale, quantity, total_price, status, created
    `

	for _, orderItem := range orderItems {
		b.Queue(query,
			orderItem.OrderID,
			orderItem.Item.ID,
			orderItem.ChrtID,
			orderItem.ID,
			orderItem.Price,
			orderItem.Sale,
			orderItem.Quantity,
			orderItem.TotalPrice,
			string(orderItem.Status),
		)
	}
	br := r.pool.SendBatch(ctx, b)
	defer func() { _ = br.Close() }()

	newOrderItems := make([]model.OrderItem, 0, len(orderItems))
	for i := 0; i < b.Len(); i++ {
		rows, err := br.Query()
		if err != nil {
			return nil, errors.WithStack(err)
		}

		row, err := pgx.CollectExactlyOneRow[orderItemRow](rows, pgx.RowToStructByNameLax[orderItemRow])
		if err != nil {
			return nil, errors.WithStack(err)
		}
		newOrderItems = append(newOrderItems, r.orderItemModel(row))
	}

	return newOrderItems, nil
}

func (r *Repository) createItems(ctx context.Context, tx pgx.Tx, items []model.Item) ([]model.Item, error) {
	b := &pgx.Batch{}

	query := `
        insert into item (nm_id, name, brand, price)
        values ($1, $2, $3, $4)
        on conflict (nm_id) 
        do update set
            name = excluded.name,
            brand = excluded.brand,
            price = excluded.price
        returning nm_id, name, brand, price
    `

	for _, item := range items {
		b.Queue(query, item.ID, item.Name, item.Brand, item.Price)
	}
	br := tx.SendBatch(ctx, b)
	defer func() { _ = br.Close() }()

	newItems := make([]model.Item, 0, len(items))

	for i := 0; i < b.Len(); i++ {
		rows, err := br.Query()
		if err != nil {
			return nil, errors.WithStack(err)
		}
		row, err := pgx.CollectExactlyOneRow[itemRow](rows, pgx.RowToStructByNameLax[itemRow])
		if err != nil {
			return nil, errors.WithStack(err)
		}
		newItems = append(newItems, modelItem(row))
	}

	return newItems, nil
}

type itemRow struct {
	ID    uuid.UUID `db:"nm_id"`
	Name  string    `db:"name"`
	Brand string    `db:"brand"`
	Price int64     `db:"price"`
}

func modelItem(row itemRow) model.Item {
	return model.Item{
		ID:    row.ID,
		Name:  row.Name,
		Price: row.Price,
		Brand: row.Brand,
	}
}

func (r *Repository) createSizes(ctx context.Context, tx pgx.Tx, orderItems []model.OrderItem) ([]model.Size, error) {
	b := &pgx.Batch{}
	query := `
        insert into size (id, nm_id, tech_size, price)
        values ($1, $2, $3, $4)
        on conflict (id) do update set
            nm_id = excluded.nm_id,
            tech_size = excluded.tech_size,
            price = excluded.price
        returning id, nm_id, tech_size
    `

	for _, orderItem := range orderItems {
		b.Queue(query,
			orderItem.ChrtID,
			orderItem.Item.ID,
			orderItem.Size,
			orderItem.Price,
		)
	}

	br := tx.SendBatch(ctx, b)
	defer func() { _ = br.Close() }()

	sizes := make([]model.Size, 0, len(orderItems))

	for i := 0; i < b.Len(); i++ {
		rows, err := br.Query()
		if err != nil {
			return nil, errors.WithStack(err)
		}
		row, err := pgx.CollectExactlyOneRow[sizeRow](rows, pgx.RowToStructByNameLax[sizeRow])
		if err != nil {
			return nil, errors.WithStack(err)
		}
		sizes = append(sizes, r.sizeModel(row))
	}

	return sizes, nil
}

type sizeRow struct {
	ID       int64     `db:"id"`
	NmID     uuid.UUID `db:"nm_id"`
	TechSize string    `db:"tech_size"`
	SKU      string    `db:"sku"`
	Price    int64     `db:"price"`
	Name     string    `db:"name"`
}

func (r *Repository) sizeModel(row sizeRow) model.Size {
	return model.Size{
		ID:     row.ID,
		ItemID: row.NmID,
		Size:   row.TechSize,
	}
}

func (r *Repository) Orders(ctx context.Context, opts model.OrderFilter) ([]model.Order, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	orders, err := r.orders(ctx, tx, opts)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(orders))
	for i, order := range orders {
		ids[i] = order.ID
	}

	items, err := r.getOrderItems(ctx, tx, ids...)
	if err != nil {
		return nil, err
	}

	for i := range orders {
		for _, orderItem := range items {
			if orders[i].ID == orderItem.OrderID {
				orders[i].Items = append(orders[i].Items, orderItem)
			}
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return orders, nil
}

func (r *Repository) orders(ctx context.Context, tx pgx.Tx, opts model.OrderFilter) ([]model.Order, error) {
	b := r.builder.
		Select("o.id as id," +
			"o.customer_id as customer_id, " +
			"o.track_number, " +
			"o.entry, " +
			"o.locale," +
			"o.internal_signature," +
			"o.delivery_service, " +
			"o.sm_id, " +
			"o.created," +
			"c.name as customer_name, " +
			"c.email as customer_email, " +
			"c.phone as customer_phone," +
			"a.zip, " +
			"a.city, " +
			"a.address, " +
			"a.region," +
			"p.transaction_id, " +
			"p.request_id, " +
			"p.currency," +
			"p.provider, " +
			"p.amount, " +
			"p.payment_dt, " +
			"p.bank, " +
			"p.delivery_cost," +
			"p.goods_total, " +
			"p.custom_fee",
		).From("\"order\" o").
		LeftJoin("customer c on o.customer_id = c.id").
		LeftJoin("address a on o.address_id = a.id").
		LeftJoin("payment p on o.id = p.order_id")

	if opts.OrderID != uuid.Nil {
		b = b.Where(sq.Eq{"o.id": opts.OrderID})
	}

	if opts.IsRecent {
		b = b.OrderBy("created desc")
	}

	if opts.Limit > 0 {
		b = b.Limit(opts.Limit)
	}

	query, args, err := b.ToSql()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	orderRows, err := pgx.CollectRows[orderRow](rows, pgx.RowToStructByNameLax[orderRow])
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if len(orderRows) == 0 {
		return nil, model.ErrOrderNotFound
	}

	orders := make([]model.Order, 0, len(orderRows))
	for _, row := range orderRows {
		orders = append(orders, r.orderModel(row))
	}

	return orders, nil
}

func (r *Repository) orderModel(row orderRow) model.Order {
	return model.Order{
		ID:                row.ID,
		TrackNumber:       row.TrackNumber,
		Entry:             row.Entry,
		Locale:            row.Locale,
		InternalSignature: row.InternalSignature,
		DeliveryService:   row.DeliveryService,
		SmID:              row.SmID,
		Created:           row.Created,
		Customer: model.Customer{
			ID:    row.CustomerID,
			Name:  row.CustomerName,
			Email: row.CustomerEmail,
			Phone: row.CustomerPhone,
		},
		Address: model.Address{
			CustomerID: row.CustomerID,
			Zip:        row.Zip,
			City:       row.City,
			Address:    row.DeliveryAddress,
			Region:     row.Region,
		},
		Payment: model.Payment{
			TransactionID: row.TransactionID,
			RequestID:     row.RequestID,
			Currency:      row.Currency,
			Provider:      row.Provider,
			Amount:        row.Amount,
			Timestamp:     row.PaymentDt,
			Bank:          row.Bank,
			DeliveryCost:  row.DeliveryCost,
			GoodsTotal:    row.GoodsTotal,
			CustomFee:     row.CustomFee,
		},
	}
}

type orderRow struct {
	ID                uuid.UUID `db:"id"`
	CustomerID        uuid.UUID `db:"customer_id"`
	TrackNumber       string    `db:"track_number"`
	Entry             string    `db:"entry"`
	Locale            string    `db:"locale"`
	InternalSignature string    `db:"internal_signature"`
	DeliveryService   string    `db:"delivery_service"`
	SmID              int64     `db:"sm_id"`
	Created           time.Time `db:"created"`
	CustomerName      string    `db:"customer_name"`
	CustomerEmail     string    `db:"customer_email"`
	CustomerPhone     string    `db:"customer_phone"`
	Zip               string    `db:"zip"`
	City              string    `db:"city"`
	DeliveryAddress   string    `db:"address"`
	Region            string    `db:"region"`
	TransactionID     uuid.UUID `db:"transaction_id"`
	RequestID         uuid.UUID `db:"request_id"`
	Currency          string    `db:"currency"`
	Provider          string    `db:"provider"`
	Amount            int64     `db:"amount"`
	PaymentDt         int64     `db:"payment_dt"`
	Bank              string    `db:"bank"`
	DeliveryCost      int64     `db:"delivery_cost"`
	GoodsTotal        int64     `db:"goods_total"`
	CustomFee         int64     `db:"custom_fee"`
	//ChrtID            string    `db:"chrt_id"`
	//ItemPrice         int64     `db:"item_price"`
	//ItemSale          int64     `db:"sale"`
	//Quantity          int64     `db:"quantity"`
}

func (r *Repository) getOrderItems(ctx context.Context, tx pgx.Tx, id ...uuid.UUID) ([]model.OrderItem, error) {
	query := `
        SELECT 
            oi.rid,
            oi.order_id,
            oi.chrt_id,
            oi.price,
            oi.sale,
            oi.quantity,
            oi.total_price,
            oi.status,
            s.tech_size as size,
            i.nm_id,
            i.brand,
            i.name
        FROM order_item oi
        JOIN size s ON oi.chrt_id = s.id
        JOIN item i ON s.nm_id = i.nm_id
        WHERE oi.order_id = ANY($1)
    `
	rows, err := tx.Query(ctx, query, pq.Array(id))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	itemRows, err := pgx.CollectRows[orderItemRow](rows, pgx.RowToStructByNameLax[orderItemRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.WithStack(model.ErrItemNotFound)
		}
		return nil, errors.WithStack(err)
	}

	items := make([]model.OrderItem, 0, len(itemRows))
	for _, row := range itemRows {
		items = append(items, r.orderItemModel(row))
	}

	return items, nil
}

type orderItemRow struct {
	ID         uuid.UUID `db:"rid"`
	OrderID    uuid.UUID `db:"order_id"`
	NmID       uuid.UUID `db:"nm_id"`
	ChrtID     int64     `db:"chrt_id"`
	Price      int64     `db:"price"`
	Sale       int64     `db:"sale"`
	Quantity   int64     `db:"quantity"`
	TotalPrice int64     `db:"total_price"`
	Status     string    `db:"status"`
	Size       string    `db:"size"`
	Brand      string    `db:"brand"`
	Name       string    `db:"name"`
	Created    time.Time `db:"created"`
}

func (r *Repository) orderItemModel(row orderItemRow) model.OrderItem {
	return model.OrderItem{
		ID:      row.ID,
		ChrtID:  row.ChrtID,
		OrderID: row.OrderID,
		Item: model.Item{
			ID:    row.NmID,
			Name:  row.Name,
			Brand: row.Brand,
		},
		Price:      row.Price,
		Sale:       row.Sale,
		Size:       row.Size,
		Quantity:   row.Quantity,
		TotalPrice: row.TotalPrice,
		Status:     model.ItemStatus(row.Status),
	}
}
