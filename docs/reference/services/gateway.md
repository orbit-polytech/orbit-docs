# gateway-service

**Репозиторий:** [`orbit-polytech/gateway-service`](https://github.com/orbit-polytech/gateway-service)

**Статус:** работает сейчас в `main` (`contracts@v0.5.12`).

## Назначение

Единственная публичная HTTP-точка входа в систему. Принимает HTTP/JSON-запросы, проверяет Ed25519 access JWT на защищённых маршрутах, ограничивает частоту запросов в Redis и маршрутизирует вызовы в Auth, Users, Schedule, AI (gRPC) и `ops-bot` (HTTP webhook).

## Что реализовано

- `/healthz` и `/readyz`, Swagger UI (`/swagger/index.html`);
- публичный поиск групп и чтение расписания: `GET /api/v1/schedule/groups` и `GET /api/v1/schedule/groups/{group}`;
- Email OTP (`send-code`, `verify-code`), `refresh`, `logout`;
- аккаунт, предпочтения групп и смена email: `GET /api/v1/auth/me`, `PATCH /api/v1/auth/preferences`, `POST /api/v1/auth/email/change/begin`, `POST /api/v1/auth/email/change/confirm` (требуют JWT);
- Google OAuth (`login`, `callback`, мобильный `POST /api/v1/auth/oauth/exchange`), а также привязка/список/отвязка внешних identities (привязка с телефона редиректит на deep link `ru.mospolytech.orbit://auth/callback`);
- профиль пользователя: `GET /api/v1/users/profile` и `PUT /api/v1/users/profile` (требуют JWT, проксируются в `users-service:50052`);
- AI: `POST /api/v1/ai/chat/stream` (SSE + Redis rate limiter), `/chat/cancel`, список/история/удаление `/api/v1/ai/conversations*`, оценки `PUT .../feedback`;
- поддержка и баг-репорты: `POST /api/v1/support/issues` (`OptionalAuth` + Redis rate limiter, отправка в `ops-bot:8081/issues/webhook`);
- middleware: `RequestID`, JSON `Logger` (`log/slog`, `service="gateway-service"`), `Metrics` (HTTP RED-метрики Prometheus), `Recovery`, `CORS`, `Auth`, `OptionalAuth`, `AIRateLimit`, `SupportIssueRateLimit`;
- изолированный внутренний HTTP-сервер метрик `:9094` (`/metrics`, `/livez`).

## Не отвечает за

- выпуск токенов и хранение сессий;
- прямой доступ к PostgreSQL сервисов;
- бизнес-логику расписания, профилей, RAG и будущих задач.

## Порты и зависимости

| Категория | Значение |
|---|---|
| HTTP внутри / на хосте | `:8080` / `127.0.0.1:8080` (также проксируется через `web:80` / `127.0.0.1:8088`) |
| Metrics HTTP (только внутри сети) | `:9094` (`/metrics`, `/livez`; на `:8080` `/metrics` отдаёт `404`) |
| Auth gRPC | `auth-service:50051` |
| Users gRPC | `users-service:50052` |
| Schedule gRPC | `schedule-service:50054` |
| AI gRPC | `ai-service:50057` |
| Ops Bot Webhook | `http://ops-bot:8081` (`POST /issues/webhook`) |
| Redis | `redis:6379` (rate limiters `orbit:ai:rl:*`, `orbit:support:rl:*`; при недоступности Redis — fallback на in-memory) |
| JWT | Только публичный Ed25519-ключ |

## Основные env

```ini
GATEWAY_HOST=0.0.0.0
GATEWAY_PORT=8080
METRICS_ADDR=0.0.0.0:9094
LOG_LEVEL=info
AUTH_GRPC_ADDR=auth-service:50051
USERS_GRPC_ADDR=users-service:50052
SCHEDULE_GRPC_ADDR=schedule-service:50054
AI_GRPC_ADDR=ai-service:50057
AI_RPC_TIMEOUT=120s
SCHEDULE_RPC_TIMEOUT=5s
JWT_PUBLIC_KEY_PATH=/run/secrets/jwt_public/jwt_ed25519.pub
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000,http://localhost:8080,http://127.0.0.1:8080
SWAGGER_ENABLED=true
COOKIE_SECURE=false
MOBILE_AUTH_REDIRECT_URI=ru.mospolytech.orbit://auth/callback
REDIS_ADDR=redis:6379
REDIS_PASSWORD=...
REDIS_DB=0
AI_RATE_LIMIT_ENABLED=true
AI_RATE_LIMIT_MAX_REQUESTS=15
AI_RATE_LIMIT_WINDOW=1h
AI_RATE_LIMIT_COOLDOWN=5s
OPS_BOT_WEBHOOK_URL=http://ops-bot:8081
OPS_BOT_WEBHOOK_TOKEN=...
```

`COOKIE_SECURE=false` подходит только для локального стенда без TLS; за HTTPS-прокси включайте `true`.

Подробнее: [обзор Gateway](../../gateway/overview.md), [публичное API расписания](../../gateway/schedule-api.md), [AI API](../../gateway/ai-api.md), [Observability](../../contracts-infra/observability.md).
