# Сетевые порты и переменные окружения

## Актуальный локальный стенд

| Сервис | Протокол | В контейнере | На хосте по умолчанию | Публичный |
|---|---|---:|---:|---|
| Gateway | HTTP | 8080 | `127.0.0.1:8080` | Да, локально |
| Auth | gRPC | 50051 | `127.0.0.1:50051` | Только для диагностики |
| PostgreSQL | TCP | 5432 | `127.0.0.1:5434` | Нет |
| Redis | TCP | 6379 | `127.0.0.1:6380` | Нет |

В контейнере нельзя использовать host-порты. Например, Auth подключается к `postgres:5432`, а инструмент на машине разработчика — к `localhost:5434`.

## Переменные `infra/.env`

```ini
# Host ports
GATEWAY_HOST_PORT=8080
POSTGRES_HOST_PORT=5434
REDIS_HOST_PORT=6380
AUTH_GRPC_HOST_PORT=50051

# PostgreSQL
POSTGRES_DB=orbit_auth
POSTGRES_ADMIN_USER=postgres
POSTGRES_ADMIN_PASSWORD=<local-admin-password>
APP_DB_USER=orbit_user
APP_DB_PASSWORD=<local-app-password>

# Redis
REDIS_PASSWORD=<local-redis-password>

# Auth
OTP_HMAC_SECRET=<random-secret-at-least-16-characters>
RESEND_API_KEY=<resend-key>
RESEND_FROM_EMAIL=<verified-sender>

# Optional Google OAuth
GOOGLE_OAUTH_ENABLED=false
AUTH_GOOGLE_CLIENT_ID=
AUTH_GOOGLE_CLIENT_SECRET=
AUTH_GOOGLE_REDIRECT_URI=http://localhost:8080/api/v1/auth/oauth/google/callback
```

## Правила

1. `.env.example` содержит только placeholders и безопасные локальные значения.
2. `.env` не коммитится.
3. Gateway не получает DSN, Resend key и OTP secret.
4. Auth не получает публичные административные credentials, если они не нужны приложению.
5. Внешние клиенты обращаются только к Gateway; прямые порты сервисов предназначены для локальной диагностики.

## Зарезервированные порты целевой архитектуры

Порты `50052`–`50057`, RabbitMQ и observability относятся к будущим этапам. См. [канонические значения](canonical-values.md) и [текущее состояние](../project/current-state.md).
