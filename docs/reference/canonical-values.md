# Канонические значения Orbit AI

!!! info "Как пользоваться справочником"
    Значения разделены на **текущую реализацию** и **целевую архитектуру**. Версия конкретного сервиса определяется его `go.mod`, Dockerfile и release-тегом, а не общей цифрой для всей организации.

## Текущий стенд

| Компонент | Внутренний адрес | Адрес на хосте | Статус |
|---|---|---|---|
| Gateway HTTP | `gateway-service:8080` | `127.0.0.1:8080` | Реализовано |
| Auth gRPC | `auth-service:50051` | `127.0.0.1:50051` | Реализовано |
| Schedule gRPC | `schedule-service:50054` | `127.0.0.1:50054` | Реализовано |
| AI gRPC | `ai-service:50057` | `127.0.0.1:50057` | Реализовано, профиль `ai`/`all` |
| PostgreSQL | `postgres:5432` | `127.0.0.1:5434` | Реализовано |
| Redis | `redis:6379` | `127.0.0.1:6380` | Реализовано |

Host-порты могут быть изменены в `.env`. Межконтейнерные адреса менять обычно не требуется.

## Основные переменные текущего стенда

| Переменная | Потребитель | Назначение |
|---|---|---|
| `POSTGRES_DSN` | Auth / Schedule | Отдельное значение DSN для базы каждого сервиса |
| `SCHEDULE_POSTGRES_DB` | Compose / postgres-init | Имя базы Schedule, по умолчанию `orbit_schedule` |
| `SCHEDULE_GRPC_ADDR` | Gateway | Адрес Schedule: `schedule-service:50054` |
| `SCHEDULE_RPC_TIMEOUT` | Gateway | Таймаут Schedule RPC, в Compose `5s` |
| `SCHEDULE_SYNC_ENABLED` | Schedule через Compose | Фоновая синхронизация; в Compose по умолчанию `true` |
| `REDIS_ADDR` | Auth, Schedule | Адрес Redis внутри Compose |
| `REDIS_PASSWORD` | Redis, Auth, Schedule | Пароль локального Redis |
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
| Поиск групп (без JWT) | `http://localhost:8080/api/v1/schedule/groups?query=261` |
| Расписание текущей недели (без JWT) | `http://localhost:8080/api/v1/schedule/groups/261-334` |
| Google callback | `http://localhost:8080/api/v1/auth/oauth/google/callback` |

## Целевая архитектура

Следующие компоненты зарезервированы архитектурой, но ещё не входят в текущий стенд:

| Компонент | Планируемый порт | Статус |
|---|---:|---|
| Users gRPC | 50052 | Запланировано |
| Navigation gRPC | Требует согласования | Запланировано; прежний `50054` занят Schedule |
| Tasks gRPC | 50055 | Учебный пример / запланировано |
| Notification gRPC | 50056 | Запланировано |
| RabbitMQ | 5672 / 15672 | Запланировано |
| Jaeger UI | 16686 | Запланировано |

Актуальный статус компонентов находится на странице [«Текущее состояние»](../project/current-state.md).

Для текущего Schedule закреплены `50054`, база `orbit_schedule`, контракт `v0.5.0`, максимум 93 календарных дня и зона `Europe/Moscow`. Старые схемы с `50053` не являются источником конфигурации. Назначать новый порт Navigation нужно отдельно, не копируя занятую цифру.


## Текущий AI-контур

Groq + FastEmbed, Python 3.13+, PostgreSQL/pgvector `orbit_ai`; тесты — `orbit_ai_test`. Контракт AI — `v0.5.2`. Порты контейнеров не совпадают с host mapping: PostgreSQL внутри `5432`, на хосте `5434`, не отдельный `5435`. `task up` = `apps` без AI; совместный стенд — профиль `all`. Полный [справочник env](ports-and-env.md), [AI API](../gateway/ai-api.md).
