# Канонические значения Orbit AI

!!! info "Как читать справочник"
    Значения разделены на **текущий стенд** и **запланированные компоненты**. Точную версию зависимостей каждого сервиса смотрите в его `go.mod` / `pyproject.toml` / `pubspec.yaml`.

## Текущий стенд (`infra v0.4.0`)

| Компонент | Внутренний адрес | Адрес на хосте | Профиль / статус |
|---|---|---|---|
| Gateway HTTP | `gateway-service:8080` | `127.0.0.1:8080` | `apps`, `all` |
| Gateway Metrics | `gateway-service:9094` | Не публикуется | Только внутри сети Compose |
| Auth gRPC | `auth-service:50051` | `127.0.0.1:50051` | `apps`, `all` |
| Auth Metrics | `auth-service:9091` | Не публикуется | Только внутри сети Compose |
| Schedule gRPC | `schedule-service:50054` | `127.0.0.1:50054` | `apps`, `all` |
| Schedule Metrics | `schedule-service:9092` | Не публикуется | Только внутри сети Compose |
| AI gRPC | `ai-service:50057` | `127.0.0.1:50057` | `ai`, `all` |
| AI Metrics | `ai-service:9093` | Не публикуется | Только внутри сети Compose |
| PostgreSQL 16 (pgvector) | `postgres:5432` | `127.0.0.1:5434` | Базовый сервис |
| Redis 7 | `redis:6379` | `127.0.0.1:6380` | Базовый сервис |
| Ngrok инспектор | `ngrok:4040` | `127.0.0.1:4040` | `tunnel`, `all` |
| Prometheus | `prometheus:9090` | `127.0.0.1:9090` | `monitoring`, `all` |
| Alertmanager | `alertmanager:9093` | `127.0.0.1:9093` | `monitoring`, `all` |
| Grafana | `grafana:3000` | `127.0.0.1:3000` | `monitoring`, `all` |
| Loki | `loki:3100` | Не публикуется | `monitoring`, `all` |
| Grafana Alloy | `alloy:12345` | Не публикуется | `monitoring`, `all` |
| Postgres / Redis Exporters | `postgres-exporter:9187`, `redis-exporter:9121` | Не публикуется | `monitoring`, `all` |
| Ops Bot | `ops-bot:8081` | Не публикуется | `ops`, `all` |

Host-порты можно переопределить в `infra/.env`. Внутри контейнеров всегда используйте внутренние имена сервисов и внутренние порты (`postgres:5432`, `redis:6379`).

## Ключевые переменные стенда

| Переменная | Потребитель | Назначение |
|---|---|---|
| `POSTGRES_DSN` | Auth / Schedule | Строка подключения к `orbit_auth` или `orbit_schedule` |
| `DATABASE_URL` | AI | AsyncPG DSN для `orbit_ai` (`postgresql+asyncpg://...`) |
| `SCHEDULE_POSTGRES_DB`, `AI_POSTGRES_DB`, `AI_TEST_POSTGRES_DB` | Compose / `postgres-init` | Имена баз (`orbit_schedule`, `orbit_ai`, `orbit_ai_test`) |
| `METRICS_DB_USER`, `OPS_BOT_DB_USER` | `postgres-init` / Exporters / `ops-bot` | Отдельные роли `orbit_metrics` (`pg_monitor`) и `orbit_ops_bot` |
| `AUTH_GRPC_ADDR`, `SCHEDULE_GRPC_ADDR`, `AI_GRPC_ADDR` | Gateway | Внутренние адреса gRPC (`...:50051`, `...:50054`, `...:50057`) |
| `SCHEDULE_SYNC_ENABLED` | Schedule (через Compose) | Фоновая синхронизация с `rasp.dmami.ru`; в Compose включена |
| `REDIS_ADDR`, `REDIS_PASSWORD` | Auth, Schedule, `redis-exporter` | Адрес и пароль Redis |
| `OTP_HMAC_SECRET`, `RESEND_API_KEY`, `RESEND_FROM_EMAIL` | Auth | Email OTP |
| `JWT_PRIVATE_KEY_PATH` / `JWT_PUBLIC_KEY_PATH` | Auth / Gateway | Ed25519-ключи в раздельных томах `jwt_private` и `jwt_public` |
| `GOOGLE_OAUTH_ENABLED`, `AUTH_GOOGLE_*` | Auth | Google OAuth/OIDC и привязка аккаунта |
| `AI_PROVIDER`, `GROQ_API_KEY`, `OPENROUTER_API_KEY`, `GEMINI_API_KEY`, `AI_MODEL` | AI | Выбор основного провайдера и ключи моделей |
| `GRAFANA_ADMIN_*`, `OPS_BOT_*`, `NGROK_*` | Мониторинг, `ops-bot`, Ngrok | Настройка дашбордов, Telegram-топиков, вебхука и туннеля |

Секреты заполняются только в локальном `.env` и не попадают в Git.

## Текущие URL стенда

| Назначение | URL |
|---|---|
| Liveness / Readiness Gateway | `http://localhost:8080/healthz`, `http://localhost:8080/readyz` |
| Swagger UI | `http://localhost:8080/swagger/index.html` |
| Auth API и настройки | `http://localhost:8080/api/v1/auth/...` (`/me`, `/preferences`, `/oauth/...`, `/identities/...`) |
| Поиск групп (без JWT) | `http://localhost:8080/api/v1/schedule/groups?query=261` |
| Расписание текущей недели (без JWT) | `http://localhost:8080/api/v1/schedule/groups/261-334` |
| AI API (с JWT) | `http://localhost:8080/api/v1/ai/chat/stream`, `/conversations`, `.../feedback` |
| Grafana / Prometheus / Alertmanager | `http://localhost:3000`, `http://localhost:9090`, `http://localhost:9093` |
| Ngrok Inspector (при `--profile tunnel`) | `http://localhost:4040` |

## Запланированная архитектура

Следующие компоненты зарезервированы на будущее и ещё не входят в рабочий стенд:

| Компонент | Планируемый порт | Статус |
|---|---:|---|
| Users gRPC | `50052` | В разработке (каркас без интеграции) |
| Navigation gRPC | Требует выбора (старый `50054` занят Schedule) | Запланировано |
| Tasks gRPC | `50055` | Учебный пример / запланировано |
| Notification gRPC | `50056` | Запланировано |
| RabbitMQ | `5672` / `15672` | Запланировано |
| Jaeger UI | `16686` | Запланировано |

Для Schedule каноническими остаются порт `50054`, база `orbit_schedule`, контракт `v0.5.0`, лимит `93` дня и зона `Europe/Moscow`.

## Текущие контракты и AI-контур

- `contracts`: последний тег **`v0.5.9`** (используется `gateway-service` и `ai-service`; `auth-service` использует `v0.5.5`, `schedule-service` — `v0.5.0`).
- `ai-service`: Python 3.13+, мультипровайдерный `RouterLLMClient` (Groq / OpenRouter / Gemini), FastEmbed (`384` измерения), гибридный поиск (pgvector + FTS), база знаний из `knowledge-pipeline` (`--manifest`), метрики на `:9093`.
- Полный список env — в [Портах и env](ports-and-env.md), про мониторинг — в [Observability](../contracts-infra/observability.md).
