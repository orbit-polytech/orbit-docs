# Сводная таблица портов и переменных окружения

В данном справочнике зафиксированы все выделенные порты и стандартные переменные окружения для локального запуска сервисов Orbit AI.

---

## 1. Реестр портов инфраструктуры и сервисов

| Сервис / Инфраструктура | Протокол | Внутренний порт (Docker) | Внешний порт (Host) | Назначение |
| :--- | :--- | :-: | :-: | :--- |
| **PostgreSQL 16** | TCP | 5432 | **5432** | Общий кластер баз данных (изолированные БД) |
| **Redis 7** | TCP | 6379 | **6379** | In-memory кэш и блэклист токенов |
| **RabbitMQ** (AMQP) | TCP | 5672 | **5672** | Брокер сообщений для сервисов |
| **RabbitMQ Management** | HTTP | 15672 | **15672** | Веб-панель управления очередями (guest / guest) |
| **gateway-service** | HTTP | 8080 | **8080** | Единая публичная точка входа (REST API / Swagger) |
| **auth-service** | gRPC | 50051 | 50051 | Сервис аутентификации |
| **users-service** | gRPC | 50052 | 50052 | Профили студентов и группы |
| **schedule-service** | gRPC | 50053 | 50053 | Расписание занятий |
| **navigation-service** | gRPC | 50054 | 50054 | Навигация по корпусам |
| **tasks-service** | gRPC | 50055 | 50055 | Задачи и дедлайны |
| **notification-service**| gRPC | 50056 | 50056 | Сервис уведомлений |
| **ai-service** | gRPC | 50057 | 50057 | AI RAG-поиск по регламентам вуза |
| **bot-service** | - | - | - | Telegram-бот (работает в режиме polling к Telegram) |
| **Jaeger UI** | HTTP | 16686 | **16686** | Веб-интерфейс распределенного трейсинга OpenTelemetry |
| **Prometheus** | HTTP | 9090 | **9090** | Сбор метрик сервисов |

---

## 2. Стандартные переменные окружения сервисов

### Базовые настройки среды

```ini
# Окружение: local | staging | production
APP_ENV=local

# Уровень логирования: debug | info | warn | error
LOG_LEVEL=debug
```

### Подключение к PostgreSQL

Каждый микросервис подключается к своей персональной БД согласно [ADR-002](../explanation/adr/002-database-per-service.md):

```ini
DB_HOST=localhost
DB_PORT=5432
DB_USER=orbit
DB_PASSWORD=orbit_dev_secret
DB_NAME=tasks_db    # tasks_db | schedule_db | auth_db | ai_knowledge_db
DB_SSLMODE=disable
```

### Подключение к Redis

```ini
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
```

### Подключение к RabbitMQ

```ini
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
```

### Настройки безопасности и ключей (JWT)

```ini
# Публичный ключ Ed25519 (для gateway-service)
JWT_PUBLIC_KEY_PEM="-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----"

# Приватный ключ Ed25519 (строго ТОЛЬКО для auth-service)
JWT_PRIVATE_KEY_PEM="-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----"

# Время жизни токенов
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=168h
```
