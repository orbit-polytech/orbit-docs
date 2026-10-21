# Репозиторий infra: текущий Docker Compose

!!! info "Статус: в разработке"
    Полный стек проверен в feature-ветке контейнеризации. До публикации образов в GHCR чистому клону может потребоваться локальный override.

`infra` — потребитель готовых OCI-образов и единая точка запуска локального стенда. Базовый `compose.yaml` не зависит от соседних исходников.

## Состав актуального стенда

| Сервис | Роль |
|---|---|
| `postgres` | PostgreSQL 16, данные Auth |
| `redis` | OTP, OAuth state, rate limits и временные данные |
| `jwt-keygen` | One-shot генерация/проверка пары Ed25519 |
| `auth-migrate` | One-shot миграции Goose |
| `auth-service` | gRPC Auth на внутреннем порту `50051` |
| `gateway-service` | HTTP API, health endpoints и Swagger на `8080` |

RabbitMQ, Jaeger, AI и остальные бизнес-сервисы относятся к следующим этапам целевой архитектуры и пока не входят в актуальный Compose.

## Основные команды

```bash
cp .env.example .env
task up
task status
```

```bash
task up:deps  # только PostgreSQL и Redis
task logs
task down     # сохранить данные и ключи
task clean    # удалить данные и ключи
```

## Ключи Ed25519

Ключи хранятся в раздельных Docker volumes:

- приватный том монтируется только в Auth;
- публичный том монтируется только в Gateway;
- keygen работает без сети;
- повторный `down` → `up` не ротирует существующую пару;
- `clean` удаляет оба тома и приводит к генерации новой пары.

## Локальный override

Файл `compose.override.yaml.example` позволяет собирать один выбранный сервис из соседнего репозитория. Рабочий `compose.override.yaml` добавлен в `.gitignore` и автоматически подхватывается Docker Compose.

```bash
cp compose.override.yaml.example compose.override.yaml
task build
```

Базовый Compose при этом остаётся автономным и использует `image:` для всех приложений.
