# Сетевые порты и переменные окружения

## Актуальный локальный стенд

| Сервис | Протокол | В контейнере | На хосте по умолчанию | Доступ |
|---|---|---:|---:|---|
| `gateway-service` (API) | HTTP | 8080 | `127.0.0.1:8080` | Публичный периметр (и через Ngrok) |
| `gateway-service` (Metrics) | HTTP | 9094 | Не публикуется | Только внутренняя сеть Docker |
| `auth-service` (gRPC) | gRPC | 50051 | `127.0.0.1:50051` | Внутренний + диагностика с хоста |
| `auth-service` (Metrics) | HTTP | 9091 | Не публикуется | Только внутренняя сеть Docker |
| `schedule-service` (gRPC) | gRPC | 50054 | `127.0.0.1:50054` | Внутренний + диагностика с хоста |
| `schedule-service` (Metrics) | HTTP | 9092 | Не публикуется | Только внутренняя сеть Docker |
| `ai-service` (gRPC) | gRPC | 50057 | `127.0.0.1:50057` | Внутренний + диагностика с хоста |
| `ai-service` (Metrics) | HTTP | 9093 | Не публикуется | Только внутренняя сеть Docker |
| `postgres` | TCP | 5432 | `127.0.0.1:5434` | Только локально для хоста |
| `redis` | TCP | 6379 | `127.0.0.1:6380` | Только локально для хоста |
| `ngrok` (инспектор) | HTTP | 4040 | `127.0.0.1:4040` | Диагностика туннеля (`tunnel` / `all`) |
| `prometheus` | HTTP | 9090 | `127.0.0.1:9090` | Метрики и alert rules (`monitoring` / `all`) |
| `alertmanager` | HTTP | 9093 | `127.0.0.1:9093` | Управление алертами (`monitoring` / `all`) |
| `grafana` | HTTP | 3000 | `127.0.0.1:3000` | Дашборды и логи (`monitoring` / `all`) |
| `loki` / `alloy` | HTTP | 3100 / 12345 | Не публикуются | Внутренняя сеть (`monitoring` / `all`) |
| `postgres-exporter` / `redis-exporter` | HTTP | 9187 / 9121 | Не публикуются | Внутренняя сеть (`monitoring` / `all`) |
| `ops-bot` (Webhook) | HTTP | 8081 | Не публикуется | Внутренняя сеть (`ops` / `all`) |

Внутри Docker-сети сервисы обращаются друг к другу по имени контейнера и внутреннему порту (например, `postgres:5432`), а с хоста — на `127.0.0.1:<host_port>` (например, `localhost:5434`).

## Переменные `infra/.env`

Актуальный шаблон лежит в [`infra/.env.example`](https://github.com/orbit-polytech/infra/blob/main/.env.example):

```ini
# App
APP_ENV=production
LOG_LEVEL=info

# Gateway & Host Ports
GATEWAY_HOST_PORT=8080
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000,http://localhost:8080,http://127.0.0.1:8080
SWAGGER_ENABLED=true
COOKIE_SECURE=false

AUTH_GRPC_HOST_PORT=50051
SCHEDULE_GRPC_HOST_PORT=50054
AI_GRPC_HOST_PORT=50057
AUTH_GRPC_ADDR=auth-service:50051
SCHEDULE_GRPC_ADDR=schedule-service:50054
AI_GRPC_ADDR=ai-service:50057

# Postgres
POSTGRES_HOST_PORT=5434
POSTGRES_IMAGE=pgvector/pgvector:pg16
POSTGRES_ADMIN_USER=postgres
POSTGRES_ADMIN_PASSWORD=change_this_admin_password
APP_DB_USER=orbit_user
APP_DB_PASSWORD=change_this_app_password
POSTGRES_DB=orbit_auth
SCHEDULE_POSTGRES_DB=orbit_schedule
AI_POSTGRES_DB=orbit_ai
AI_TEST_POSTGRES_DB=orbit_ai_test

# Redis
REDIS_HOST_PORT=6380
REDIS_PASSWORD=change_this_redis_password

# Auth
OTP_HMAC_SECRET=your_secret_minimum_16_characters_long
RESEND_API_KEY=re_your_api_key_here
RESEND_FROM_EMAIL=auth@orbit.local
GOOGLE_OAUTH_ENABLED=false
AUTH_GOOGLE_CLIENT_ID=
AUTH_GOOGLE_CLIENT_SECRET=
AUTH_GOOGLE_REDIRECT_URI=http://localhost:8080/api/v1/auth/oauth/google/callback

# Schedule
SCHEDULE_SYNC_ENABLED=true
SCHEDULE_CACHE_TTL=24h
SCHEDULE_CACHE_JITTER=2h
SCHEDULE_FRESHNESS_TTL=24h
SCHEDULE_SYNC_INTERVAL=6h
SCHEDULE_SYNC_RPS=2
SCHEDULE_SHUTDOWN_TIMEOUT=10s
SCHEDULE_SOURCE_BASE_URL=https://rasp.dmami.ru

# AI (мультипровайдер)
AI_PROVIDER=groq
GROQ_API_KEY=
GEMINI_API_KEY=
OPENROUTER_API_KEY=
AI_MODEL=qwen/qwen3.8-27b
RAG_ENABLED=true
EMBEDDING_MODEL_NAME=sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2
AI_MAX_OUTPUT_TOKENS=768
AI_MAX_CONCURRENT_GENERATIONS=50

# Tunnel (Ngrok)
NGROK_AUTHTOKEN=
NGROK_URL=
NGROK_INSPECT_PORT=4040

# Monitoring (Prometheus / Grafana / Alertmanager)
GRAFANA_HOST_PORT=3000
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASSWORD=change_this_grafana_password
PROMETHEUS_HOST_PORT=9090
ALERTMANAGER_HOST_PORT=9093
METRICS_DB_USER=orbit_metrics
METRICS_DB_PASSWORD=change_this_metrics_db_password

# Ops Bot
OPS_BOT_IMAGE=ghcr.io/orbit-polytech/ops-bot:latest
OPS_BOT_DB_USER=orbit_ops_bot
OPS_BOT_DB_PASSWORD=change_this_ops_bot_password
OPS_BOT_WEBHOOK_TOKEN=change_this_webhook_bearer_token
OPS_BOT_TELEGRAM_TOKEN=
OPS_BOT_GROUP_ID=0
OPS_BOT_ADMIN_IDS=
OPS_BOT_TOPIC_ALERTS_ID=0
OPS_BOT_TOPIC_LLM_QUOTAS_ID=0
OPS_BOT_TOPIC_UNTRIAGED_ID=0
OPS_BOT_TOPIC_FEEDBACK_ID=0
OPS_BOT_TOPIC_DIGEST_ID=0
```

## Готовые образы

В `.env` переопределяются:

- `JWT_KEYGEN_IMAGE` (`ghcr.io/orbit-polytech/auth-keygen:latest`)
- `AUTH_MIGRATE_IMAGE` (`ghcr.io/orbit-polytech/auth-migrate:latest`)
- `AUTH_SERVICE_IMAGE` (`ghcr.io/orbit-polytech/auth-service:latest`)
- `GATEWAY_SERVICE_IMAGE` (`ghcr.io/orbit-polytech/gateway-service:latest`)
- `SCHEDULE_SERVICE_IMAGE` (`ghcr.io/orbit-polytech/schedule-service:latest`)
- `SCHEDULE_MIGRATE_IMAGE` (`ghcr.io/orbit-polytech/schedule-migrate:latest`)
- `AI_SERVICE_IMAGE` (`ghcr.io/orbit-polytech/ai-service:latest` — и для `ai-service`, и для `ai-migrate`)
- `OPS_BOT_IMAGE` (`ghcr.io/orbit-polytech/ops-bot:latest`)

## Env приложения и Compose — не одно и то же

Файл `.env` подставляет значения `${...}` в `compose.yaml`, но контейнер получает только те переменные, которые явно перечислены в его секции `environment`.

Для Schedule в Compose фиксированы `SCHEDULE_GRPC_PORT=50054`, `METRICS_ADDR=0.0.0.0:9092` и `REDIS_ADDR=redis:6379`. Для Go-сервисов `METRICS_ADDR` по умолчанию задан как `0.0.0.0:9091` (Auth), `0.0.0.0:9092` (Schedule) и `0.0.0.0:9094` (Gateway). При изменении `.env` контейнеры нужно пересоздать (`docker compose ... up -d`).

## Правила

1. `.env.example` хранит только безопасные плейсхолдеры.
2. Локальный `.env` никогда не коммитится.
3. Gateway не получает БД DSN, Resend key, OTP secret или ключи LLM-провайдеров.
4. `ops-bot` и `postgres-exporter` работают под отдельными ограниченными ролями (`orbit_ops_bot` и `orbit_metrics`), а не под `orbit_user` или `postgres`.
5. Внешний трафик идёт только в `:8080` Gateway; порты метрик `9091–9094` наружу не публикуются.

## AI: параметры приложения

| Переменная | Назначение и значение по умолчанию в коде |
|---|---|
| `AI_PROVIDER` | Основной провайдер: `groq` (по умолчанию), `openrouter` или `gemini` |
| `GROQ_API_KEY`, `OPENROUTER_API_KEY`, `GEMINI_API_KEY` | Ключи провайдеров; в `production` обязателен ключ для выбранного `AI_PROVIDER` |
| `AI_MODEL` | Модель по умолчанию (в коде `openai/gpt-oss-120b`; в `infra/.env.example` может стоять `qwen/qwen3.8-27b`) |
| `AI_API_BASE_URL` | Base URL для Groq/OpenAI-совместимого провайдера (`https://api.groq.com/openai/v1`) |
| `AI_ALLOWED_MODELS` | Список разрешённых моделей для явного выбора в запросе |
| `METRICS_HOST` / `METRICS_PORT` | Внутренний сервер метрик Prometheus (`0.0.0.0` / `9093`) |
| `ORBIT_INTERNAL_SERVICE_SECRET` | Опциональный внутренний shared secret для проверки gRPC-вызовов (алиас `INTERNAL_SERVICE_SECRET`) |
| `RAG_ENABLED` | Включает поиск по базе знаний (`true`) |
| `RAG_ROUTER_ENABLED` | Включает LLM/семантическую маршрутизацию запросов по доменам (`true`) |
| `RAG_RERANKER_ENABLED` / `RAG_RERANKER_MODEL_NAME` | Опциональный cross-encoder reranker (`false` / `BAAI/bge-reranker-v2-m3`) |
| `RAG_MIN_SIMILARITY` / `RAG_TOP_K` | Порог релевантности и число фрагментов в коде: `0.45` / `8` (в локальном `.env.example` AI: `0.65` / `4`) |
| `EMBEDDING_MODEL_NAME` | `sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2` (384 измерения) |
| `CHAT_HISTORY_LIMIT` | Глубина подгружаемой истории диалога (`10`) |
| `AI_MAX_OUTPUT_TOKENS` | Лимит токенов ответа в коде `1536` (в `infra/.env.example` `768`; для Groq валидатор ограничивает до `<= 2048`) |
| `AI_TTFT_TIMEOUT_SEC`, `AI_STREAM_TOTAL_TIMEOUT_SEC`, `AI_CHUNK_IDLE_TIMEOUT_SEC` | Таймауты генерации: `15.0` / `60.0` / `10.0` |
| `AI_MAX_CONCURRENT_GENERATIONS` | Лимит одновременных генераций (`50`) |
| `GROQ_VERIFY_SSL` / `GROQ_CA_BUNDLE` | Проверка TLS (в `production` отключать `GROQ_VERIFY_SSL` запрещено) |
| `POSTGRES_TEST_DSN` | Отдельная тестовая база `orbit_ai_test` с хоста (`localhost:5434/orbit_ai_test`) |

Параметры вроде `RAG_ROUTER_ENABLED`, `RAG_RERANKER_ENABLED`, `ORBIT_INTERNAL_SERVICE_SECRET`, `RAG_MIN_SIMILARITY` или `RAG_TOP_K` в базовом `compose.yaml` не прокинуты — при необходимости передавайте их через `compose.override.yaml`. Подробнее о настройках AI: [`.env.example`](https://github.com/orbit-polytech/ai-service/blob/main/.env.example).

## Запланированные порты

`50054` занят Schedule, `50057` — AI, контур мониторинга (`9090`, `9093`, `3000`, `9091–9094`) и `ops-bot` (`8081`) уже работают в `infra`. К будущим этапам относятся `users-service` (`50052`), `tasks-service` (`50055`), `notification-service` (`50056`), `navigation-service`, RabbitMQ (`5672`/`15672`) и Jaeger (`16686`). См. [канонические значения](canonical-values.md).
