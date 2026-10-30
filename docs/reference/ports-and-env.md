# Сетевые порты и переменные окружения

## Актуальный локальный стенд

| Сервис | Протокол | В контейнере | На хосте по умолчанию | Публичный |
|---|---|---:|---:|---|
| Gateway | HTTP | 8080 | `127.0.0.1:8080` | Да, локально |
| Auth | gRPC | 50051 | `127.0.0.1:50051` | Только для диагностики |
| Schedule | gRPC | 50054 | `127.0.0.1:50054` | Только для диагностики |
| AI | gRPC | 50057 | `127.0.0.1:50057` | Только для диагностики |
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
SCHEDULE_GRPC_HOST_PORT=50054

# PostgreSQL
POSTGRES_DB=orbit_auth
SCHEDULE_POSTGRES_DB=orbit_schedule
POSTGRES_ADMIN_USER=postgres
POSTGRES_ADMIN_PASSWORD=<local-admin-password>
APP_DB_USER=orbit_user
APP_DB_PASSWORD=<local-app-password>

# Redis
REDIS_PASSWORD=<local-redis-password>

# Schedule: параметр Compose, включает синхронизацию
SCHEDULE_SYNC_ENABLED=true
SCHEDULE_CACHE_TTL=24h
SCHEDULE_CACHE_JITTER=2h
SCHEDULE_FRESHNESS_TTL=24h
SCHEDULE_SYNC_INTERVAL=6h
SCHEDULE_SYNC_RPS=2
SCHEDULE_SOURCE_BASE_URL=https://rasp.dmami.ru
SCHEDULE_SHUTDOWN_TIMEOUT=10s

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

## Готовые образы

В `.env` можно переопределить `JWT_KEYGEN_IMAGE`, `AUTH_MIGRATE_IMAGE`, `AUTH_SERVICE_IMAGE`, `GATEWAY_SERVICE_IMAGE`, `SCHEDULE_SERVICE_IMAGE`, `SCHEDULE_MIGRATE_IMAGE`. Для AI дополнительно доступен `AI_SERVICE_IMAGE` (сервер и мигратор). Без переопределения используются соответствующие `ghcr.io/orbit-polytech/*:latest`. Для фиксированного среза указывайте реально опубликованные теги/digest.

## Env приложения и env Compose — не одно и то же

`.env` служит источником подстановки `${...}` в Compose, а не автоматически передаётся целиком каждому контейнеру. Приложения получают только значения, явно перечисленные в `environment`.

Для Schedule в Compose фиксированы адрес прослушивания `0.0.0.0:50054` и подключение Redis `redis:6379`, база `0`; пароль берётся из `REDIS_PASSWORD`. Gateway получает `SCHEDULE_GRPC_ADDR=schedule-service:50054` и `SCHEDULE_RPC_TIMEOUT=5s`.

| Параметр из `.env` | Дефолт Compose | Назначение |
|---|---|---|
| `SCHEDULE_CACHE_TTL` | `24h` | Срок хранения кэша |
| `SCHEDULE_CACHE_JITTER` | `2h` | Случайная положительная добавка `0..2h` |
| `SCHEDULE_FRESHNESS_TTL` | `24h` | Порог устаревания расписания; не TTL Redis |
| `SCHEDULE_SYNC_ENABLED` | `true` | Включение синхронизации |
| `SCHEDULE_SYNC_INTERVAL` | `6h` | Интервал планового обхода |
| `SCHEDULE_SYNC_RPS` | `2` | Скорость пополнения общего token bucket; ёмкость 5 допускает всплески |
| `SCHEDULE_SOURCE_BASE_URL` | `https://rasp.dmami.ru` | Источник расписания |
| `SCHEDULE_SHUTDOWN_TIMEOUT` | `10s` | Время корректного завершения |

Эти параметры уже связаны с `${...}` в Compose и переопределяются через `.env`. Для остальных настроек сначала проверьте `environment` и локальный override. Для Go-сервисов при запуске из IDE передайте env самому процессу; AI Settings отдельно настроен читать `.env`. После изменения env пересоздайте контейнеры.

## Правила

1. `.env.example` содержит только placeholders и безопасные локальные значения.
2. `.env` не коммитится.
3. Gateway не получает DSN, Resend key и OTP secret.
4. Auth не получает публичные административные credentials, если они не нужны приложению.
5. Внешние клиенты обращаются только к Gateway; прямые порты сервисов предназначены для локальной диагностики.

## Зарезервированные порты целевой архитектуры

`50054` уже занят Schedule, `50057` — реализованным AI. Остальные планируемые порты, RabbitMQ и observability относятся к будущим этапам. См. [канонические значения](canonical-values.md) и [текущее состояние](../project/current-state.md).


## AI: env приложения и Compose

| Переменная | Значение / назначение |
|---|---|
| `AI_SERVICE_IMAGE` | OCI-образ AI; используется также `ai-migrate` |
| `AI_GRPC_HOST_PORT` | Host gRPC port, `50057`, loopback |
| `AI_GRPC_ADDR` | Gateway → `ai-service:50057` |
| `AI_RPC_TIMEOUT` | Gateway, по умолчанию `60s` в Compose |
| `AI_POSTGRES_DB` / `AI_TEST_POSTGRES_DB` | `orbit_ai` / `orbit_ai_test` |
| `DATABASE_URL` | AI DSN `postgresql+asyncpg://...@postgres:5432/orbit_ai` |
| `APP_ENV` | В Compose по умолчанию `production`; без Groq key AI не стартует |
| `GROQ_API_KEY` | Секрет провайдера, только локально |
| `AI_MODEL` | ID модели, доступной в Groq; сверить с провайдером |
| `RAG_ENABLED` | По умолчанию `true` |
| `AI_MAX_OUTPUT_TOKENS` | По умолчанию `2048` |
| `AI_MAX_CONCURRENT_GENERATIONS` | По умолчанию `50`; не пользовательская квота |

Процесс AI дополнительно поддерживает `EMBEDDING_MODEL_NAME`, `RAG_MIN_SIMILARITY` (`0.65`), `RAG_TOP_K` (`4`), `AI_TTFT_TIMEOUT_SEC` (`15`), `AI_STREAM_TOTAL_TIMEOUT_SEC` (`60`), `AI_CHUNK_IDLE_TIMEOUT_SEC` (`10`), `DB_POOL_SIZE`, `DB_MAX_SIZE`, `DB_POOL_TIMEOUT`. В текущем Compose эти параметры не все прокинуты через `environment`: для изменения добавьте явный локальный override, не только строку `.env`. `DB_MAX_SIZE` в Settings задаёт **max_overflow** поверх pool size, а не общий максимум.

`POSTGRES_TEST_DSN` используется тестами с хоста, по умолчанию пример указывает `localhost:5434/orbit_ai_test`; не передавайте рабочую БД. Шаблон приложения: [AI .env.example](https://github.com/orbit-polytech/ai-service/blob/main/.env.example). В отличие от Go-приложений, AI Settings настроен читать локальный `.env`; это не меняет правил передачи env в контейнеры.
