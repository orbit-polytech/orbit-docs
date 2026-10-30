# Справочник контейнеров и инфраструктуры

## Текущий стенд

| Компонент | Внутри Compose | На хосте по умолчанию | Конфигурация |
|---|---|---|---|
| PostgreSQL 16 | `postgres:5432` | `127.0.0.1:5434` | Auth: `POSTGRES_DB`, Schedule: `SCHEDULE_POSTGRES_DB`, AI: `AI_POSTGRES_DB`, test: `AI_TEST_POSTGRES_DB` |
| Redis 7 | `redis:6379` | `127.0.0.1:6380` | Пароль обязателен; Auth и Schedule, раздельные пространства ключей |
| Auth gRPC | `auth-service:50051` | `127.0.0.1:50051` | Приватный JWT-том |
| Schedule gRPC | `schedule-service:50054` | `127.0.0.1:50054` | PostgreSQL, Redis-кэш и Streams; источник вызывается фоновыми воркерами |
| AI gRPC | `ai-service:50057` | `127.0.0.1:50057` | Groq, FastEmbed, orbit_ai, профиль ai/all |
| Gateway HTTP | `gateway-service:8080` | `127.0.0.1:8080` | Auth/Schedule/AI gRPC, публичный JWT-том |

Локальные пароли берутся из собственного `.env`; общие рабочие credentials не распространяются через документацию. Полный список env — в [справочнике](../reference/ports-and-env.md).

`jwt-keygen`, `postgres-init`, `auth-migrate`, `schedule-migrate`, `ai-migrate` — one-shot сервисы, ожидаемый итог `Exited (0)`. Подробнее: [infra и порядок запуска](infra.md).

## Целевой событийный контур (не развёрнут)

RabbitMQ и Jaeger пока не входят в текущий Compose. pgvector уже используется AI в текущем PostgreSQL образе. Ниже сохранены **проектные примеры** будущих exchange/routing keys; это не опубликованный контракт событий.

| Топик / Exchange | Routing Key | Очередь потребителя | Описание |
|---|---|---|---|
| `orbit.events` | `tasks.task.created` | `notification.tasks` | Создана задача |
| `orbit.events` | `tasks.task.completed` | `notification.tasks` | Задача выполнена |
| `orbit.events` | `schedule.lesson.changed` | `notification.schedule` | Изменение занятия |
| `orbit.events` | `auth.user.registered` | `users.sync` | Регистрация аккаунта |

Планируемый exchange — `topic`, `durable: true`. Имена, payload и гарантии доставки нужно согласовать в `contracts` до реализации. Schedule сейчас не публикует события и не зависит от RabbitMQ.
