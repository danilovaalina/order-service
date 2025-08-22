# Order Service

Микросервис для обработки заказов с использованием Kafka, PostgreSQL и Go.

## 🏗️ Архитектура

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Kafka     │───▶│ Order       │───▶│ PostgreSQL  │
│ (Producer)  │    │ Service     │    │ (Storage)   │
└─────────────┘    └─────────────┘    └─────────────┘
                            │
                            ▼
                    ┌─────────────┐
                    │   Cache     │
                    │ (In-Memory) │
                    └─────────────┘
                            │
                            ▼
                    ┌─────────────┐
                    │   HTTP API  │◀───┐
                    └─────────────┘    │
                            │          │
                            ▼          │
                    ┌─────────────┐    │
                    │ Web UI      │────┘
                    └─────────────┘
```

## 🛠️ Технологии

- **Go 1.24** - основной язык программирования
- **PostgreSQL 14** - реляционная база данных
- **Apache Kafka 4.0** - брокер сообщений
- **Docker & Docker Compose** - контейнеризация
- **Echo** - HTTP фреймворк
- **pgx** - PostgreSQL драйвер
- **Sarama** - Kafka клиент
- **Testify & Mockery** - тестирование

## 📁 Структура проекта

```
order-service/
├── cmd/                 # Точка входа
│   └── main.go
├── internal/           # Внутренняя логика
│   ├── api/            # HTTP API
│   ├── service/        # Бизнес-логика
│   ├── repository/     # Работа с БД
│   ├── model/          # Доменные модели
│   ├── cache/          # Кэширование
│   └── processor/      # Работа с Kafka
├── static/             # Статические файлы (Web UI)
│   ├── index.html
│   ├── style.css
│   └── script.js
├── configs/            # Конфигурационные файлы
│   └── config.yaml
└── docker-compose.yml  # Docker конфигурация
```

## 🚀 Развертывание

### 1. **Запуск с Docker**

docker-compose up

### 2. **Локальная разработка**
```bash
# Запуск зависимостей
docker-compose up -d postgres kafka1 kafka2 kafka3

# Запуск приложения
go run cmd/main.go
```

### 3. **Тестирование**
```bash
# Unit тесты
go test ./...

# Coverage
go test -cover ./...
```

## 🎯 Основные endpoints

- `GET /` - Web интерфейс
- `GET /order/{order_uid}` - Получение заказа

## 📊 Функциональность

- ✅ Прием заказов через Kafka
- ✅ Хранение в PostgreSQL
- ✅ Кэширование в памяти (LRU)
- ✅ Web интерфейс для просмотра заказов
- ✅ Валидация данных
- ✅ Обработка ошибок
- ✅ Graceful shutdown

## 📈 Мониторинг

- **Kafka UI**: http://localhost:8080
- **PostgreSQL**: localhost:5433
- **Order Service**: http://localhost:8081

## 📦 Конфигурация

### Пример Docker конфигурации: (`config.docker.yaml`):
cp config.docker.yaml config.yaml

### Пример локальной конфигурации: (`config.local.yaml`):
cp config.local.yaml config.yaml

## 🎯 Особенности

- **Идемпотентность** - повторные сообщения не создают дубликаты
- **Кэширование** - LRU кэш с TTL
- **Транзакционность** - все операции в транзакциях
- **Graceful shutdown** - корректное завершение работы
- **Логирование** - структурированные логи
