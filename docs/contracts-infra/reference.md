# Справочник контейнеров и инфраструктуры

## Текущий стенд

| Компонент | Внутри Compose | На хосте по умолчанию | Конфигурация |
|---|---|---|---|
| PostgreSQL 16 | `postgres:5432` | `127.0.0.1:5434` | Auth: `POSTGRES_DB`, Schedule: `SCHEDULE_POSTGRES_DB` |
| Redis 7 | `redis:6379` | `127.0.0.1:6380` | Пароль обязателен, используется Auth |
| Auth gRPC | `auth-service:50051` | `127.0.0.1:50051` | Приватный JWT-том |
| Schedule gRPC | `schedule-service:50054` | `127.0.0.1:50054` | PostgreSQL и внешний источник, без Redis |
| Gateway HTTP | `gateway-service:8080` | `127.0.0.1:8080` | Auth/Schedule gRPC, публичный JWT-том |

Локальные пароли берутся из собственного `.env`; общие рабочие credentials не распространяются через документацию. Полный список env — в [справочнике](../reference/ports-and-env.md).

`jwt-keygen`, `postgres-init`, `auth-migrate`, `schedule-migrate` — one-shot сервисы, ожидаемый итог `Exited (0)`. Подробнее: [infra и порядок запуска](infra.md).

## Целевой событийный контур (не развёрнут)

RabbitMQ, Jaeger и pgvector пока не входят в текущий Compose. Ниже сохранены **проектные примеры** будущих exchange/routing keys; это не опубликованный контракт событий.

| Топик / Exchange | Routing Key | Очередь потребителя | Описание |
|---|---|---|---|
| `orbit.events` | `tasks.task.created` | `notification.tasks` | Создана задача |
| `orbit.events` | `tasks.task.completed` | `notification.tasks` | Задача выполнена |
| `orbit.events` | `schedule.lesson.changed` | `notification.schedule` | Изменение занятия |
| `orbit.events` | `auth.user.registered` | `users.sync` | Регистрация аккаунта |

Планируемый exchange — `topic`, `durable: true`. Имена, payload и гарантии доставки нужно согласовать в `contracts` до реализации. Schedule сейчас не публикует события и не зависит от RabbitMQ.
