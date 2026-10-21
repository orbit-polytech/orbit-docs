# Канонические значения Orbit AI

!!! info "Как пользоваться справочником"
    Значения разделены на **текущую реализацию** и **целевую архитектуру**. Версия конкретного сервиса определяется его `go.mod`, Dockerfile и release-тегом, а не общей цифрой для всей организации.

## Текущий стенд

| Компонент | Внутренний адрес | Адрес на хосте | Статус |
|---|---|---|---|
| Gateway HTTP | `gateway-service:8080` | `127.0.0.1:8080` | Реализовано |
| Auth gRPC | `auth-service:50051` | `127.0.0.1:50051` | Реализовано |
| PostgreSQL | `postgres:5432` | `127.0.0.1:5434` | Реализовано |
| Redis | `redis:6379` | `127.0.0.1:6380` | Реализовано |

Host-порты могут быть изменены в `.env`. Межконтейнерные адреса менять обычно не требуется.

## Основные переменные текущего стенда

| Переменная | Потребитель | Назначение |
|---|---|---|
| `POSTGRES_DSN` | Auth | Строка подключения приложения к PostgreSQL |
| `REDIS_ADDR` | Auth | Адрес Redis внутри Compose |
| `REDIS_PASSWORD` | Redis, Auth | Пароль локального Redis |
| `OTP_HMAC_SECRET` | Auth | HMAC-секрет для OTP, минимум 16 символов |
| `RESEND_API_KEY` | Auth | Ключ Resend для реальной отправки OTP |
| `RESEND_FROM_EMAIL` | Auth | Разрешённый отправитель Resend |
| `JWT_PRIVATE_KEY_PATH` | Auth | Путь к приватному Ed25519-ключу |
| `JWT_PUBLIC_KEY_PATH` | Gateway | Путь к публичному Ed25519-ключу |
| `AUTH_GRPC_ADDR` | Gateway | Адрес Auth: `auth-service:50051` |
| `GOOGLE_OAUTH_ENABLED` | Auth | Включение Google OAuth |
| `AUTH_GOOGLE_CLIENT_ID` | Auth | Google OAuth Client ID |
| `AUTH_GOOGLE_CLIENT_SECRET` | Auth | Google OAuth Client Secret |
| `AUTH_GOOGLE_REDIRECT_URI` | Auth | Callback через Gateway |

Секреты задаются только в локальном `.env` или секрет-хранилище. Реальные значения не добавляются в `.env.example` и Git.

## Текущие URL

| Назначение | URL |
|---|---|
| Liveness | `http://localhost:8080/healthz` |
| Readiness | `http://localhost:8080/readyz` |
| Swagger UI | `http://localhost:8080/swagger/index.html` |
| Auth API | `http://localhost:8080/api/v1/auth/...` |
| Google callback | `http://localhost:8080/api/v1/auth/oauth/google/callback` |

## Целевая архитектура

Следующие компоненты зарезервированы архитектурой, но ещё не входят в текущий стенд:

| Компонент | Планируемый порт | Статус |
|---|---:|---|
| Users gRPC | 50052 | Запланировано |
| Schedule gRPC | 50053 | Запланировано |
| Navigation gRPC | 50054 | Запланировано |
| Tasks gRPC | 50055 | Учебный пример / запланировано |
| Notification gRPC | 50056 | Запланировано |
| AI gRPC | 50057 | Запланировано |
| RabbitMQ | 5672 / 15672 | Запланировано |
| Jaeger UI | 16686 | Запланировано |

Актуальный статус компонентов находится на странице [«Текущее состояние»](../project/current-state.md).
