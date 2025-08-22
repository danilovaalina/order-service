create table customer
(
    id    uuid primary key default gen_random_uuid(),
    name  text not null,
    email text not null,
    phone text not null
);

create table address
(
    id          uuid primary key default gen_random_uuid(),
    customer_id uuid references customer (id),
    zip         text not null,
    city        text not null,
    address     text not null,
    region      text not null,
    unique (customer_id, zip, city, address, region)
);

create table "order"
(
    id                 uuid primary key default gen_random_uuid(),
    customer_id        uuid references customer (id),
    address_id         uuid                                       not null references address (id),
    track_number       text,
    entry              text,
    locale             text,
    internal_signature text             default ''                not null,
    delivery_service   text,
    sm_id              bigint,
    created            timestamp        default current_timestamp not null
);

create table payment
(
    id             uuid primary key default gen_random_uuid(),
    order_id       uuid references "order" (id),
    transaction_id uuid,
    request_id     uuid, --идентификатор запроса, который генерируется клиентом при отправке данных в api
    currency       text,
    provider       text,
    amount         bigint,
    payment_dt     bigint,
    bank           text,
    delivery_cost  bigint,
    goods_total    bigint,
    custom_fee     bigint,
    unique (order_id)
);

-- базовые модели товаров
-- create table item
-- (
--     id       uuid primary key default gen_random_uuid(),
--     name     text not null,
--     category text
-- );

create type item_status as enum (
    'pending',
    'processing',
    'assembling',
    'in_transit',
    'delivered',
    'cancelled',
    'returned'
    );

create table item
(
    nm_id uuid primary key default gen_random_uuid(), --конкретная sku: mascaras vivienne sabo, размер 0, черная (определенный размер/комплектация)
    price bigint,
    name  text,
    brand text

);

create table size
(
    chrt_id   bigint primary key, -- числовой id размера для данного артикула
    nm_id     uuid references item (nm_id),
    tech_size text,               -- технический размер ("0", "m", "l")
    sku       text,               -- баркод
    price     bigint,
    name      text                -- название варианта
);


create table order_item
(
    rid         uuid primary key default gen_random_uuid(),
    order_id    uuid    not null references "order" (id),
    item_id     uuid    not null references item (nm_id),
    chrt_id     bigint  not null references size (id),
    price       integer not null,           -- цена за единицу на момент заказа
    sale        integer          default 0, -- скидка % на эту позицию
    quantity    integer          default 1, -- количество
    total_price integer not null,           -- итоговая стоимость позиции
    status      item_status,
    created     timestamp        default now()
);

-- типы характеристик
-- create table attribute
-- (
--     id        uuid primary key default gen_random_uuid(),
--     name      text not null unique,          -- "цвет", "размер", "материал"
--     type      text not null,                 -- "string", "number", "boolean", "datetime"
--     is_common boolean          default false -- общая для всех вариантов?
-- );

-- значения характеристик
-- create table attribute_value
-- (
--     id           uuid primary key default gen_random_uuid(),
--     attribute_id uuid not null references attribute (id),
--     value        text not null, -- "красный", "m", "хлопок"
--     unique (attribute_id, value)
-- );

-- связь характеристик с базовой моделью (item)
-- create table item_attribute
-- (
--     item_id      uuid not null references item (id),
--     attribute_id uuid not null references attribute_value (id),
--     unique (item_id, attribute_id)
-- );

-- связь характеристик с конкретным вариантом (product)
-- create table product_attribute
-- (
--     nm_id        uuid not null references product (nm_id),
--     attribute_id uuid not null references attribute_value (id),
--     unique (nm_id, attribute_id)
-- );
