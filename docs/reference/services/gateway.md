# gateway-service

**Репозиторий:** [`orbit-polytech/gateway-service`](https://github.com/orbit-polytech/gateway-service)

**Статус:** работает сейчас в `main` (`contracts@v0.5.9`).

## Назначение

Единственная публичная HTTP-точка входа в систему. Принимает HTTP/JSON запросы, проверяет Ed25519 access JWT на защищённых маршрутах и маршрутизирует вызовы в Auth, Schedule или AI по gRPC.

## Что реализовано

- `/healthz` и `/readyz`, Swagger UI (`/swagger/index.html`);
- публичный поиск групп и чтение расписания: `GET /api/v1/schedule/groups` и `GET /api/v1/schedule/groups/{group}`;
- Email OTP (`send-code`, `verify-code`), `refresh`, `logout`;
- аккаунт и предпочтения групп: `GET /api/v1/auth/me` и `PATCH /api/v1/auth/preferences` (требуют JWT);
- Google OAuth (`login`, `callback`, мобильный `POST /api/v1/auth/oauth/exchange`), а также привязка/список/отвязка внешних identities (привязка с телефона редиректит на deep link `ru.mospolytech.orbit://auth/callback`);
- AI: `POST /api/v1/ai/chat/stream` (SSE), `/chat/cancel`, список/история/удаление `/api/v1/ai/conversations*`, оценки `PUT .../feedback`;
- middleware: `RequestID`, JSON `Logger` (`log/slog`, `service="gateway-service"`), `Metrics` (HTTP RED-метрики Prometheus), `Recovery`, `CORS`, `Auth` (JWT);
- изолированный внутренний HTTP-сервер метрик `:9094` (`/metrics`, `/livez`).

## Не отвечает за

- выпуск токенов и хранение сессий;
- прямой доступ к PostgreSQL или Redis сервисов;
- бизнес-логику расписания, RAG и будущих задач.

## Порты и зависимости

| Категория | Значение |
|---|---|
| HTTP внутри / на хосте | `:8080` / `127.0.0.1:8080` |
| Metrics HTTP (только внутри сети) | `:9094` (`/metrics`, `/livez`; на `:8080` `/metrics` отдаёт `404`) |
| Auth gRPC | `auth-service:50051` |
| Schedule gRPC | `schedule-service:50054` |
| AI gRPC | `ai-service:50057` |
| JWT | Только публичный Ed25519-ключ |

## Основные env

```ini
GATEWAY_HOST=0.0.0.0
GATEWAY_PORT=8080
METRICS_ADDR=0.0.0.0:9094
LOG_LEVEL=info
AUTH_GRPC_ADDR=auth-service:50051
SCHEDULE_GRPC_ADDR=schedule-service:50054
AI_GRPC_ADDR=ai-service:50057
AI_RPC_TIMEOUT=120s
SCHEDULE_RPC_TIMEOUT=5s
JWT_PUBLIC_KEY_PATH=/run/secrets/jwt_public/jwt_ed25519.pub
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000
SWAGGER_ENABLED=true
COOKIE_SECURE=false
MOBILE_AUTH_REDIRECT_URI=ru.mospolytech.orbit://auth/callback
```

`COOKIE_SECURE=false` подходит только для локального стенда без TLS; за HTTPS-прокси включайте `true`.

Подробнее: [обзор Gateway](../../gateway/overview.md), [публичное API расписания](../../gateway/schedule-api.md), [AI API](../../gateway/ai-api.md), [Observability](../../contracts-infra/observability.md).
