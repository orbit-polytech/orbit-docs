# Справочник контейнеров и инфраструктуры

## Актуальный стенд (`infra v0.4.0`)

| Компонент | Внутри Compose | На хосте по умолчанию | Примечания и конфигурация |
|---|---|---|---|
| `postgres` | `postgres:5432` | `127.0.0.1:5434` | Auth (`POSTGRES_DB`), Schedule (`SCHEDULE_POSTGRES_DB`), AI (`AI_POSTGRES_DB`), test (`AI_TEST_POSTGRES_DB`) |
| `redis` | `redis:6379` | `127.0.0.1:6380` | Auth и Schedule в раздельных префиксах ключей |
| `auth-service` | `auth-service:50051` | `127.0.0.1:50051` | Метрики на `:9091` (внутренние), приватный JWT-том |
| `schedule-service` | `schedule-service:50054` | `127.0.0.1:50054` | Метрики на `:9092` (внутренние), кэш и Redis Streams |
| `ai-service` | `ai-service:50057` | `127.0.0.1:50057` | Метрики на `:9093` (внутренние), профиль `ai` / `all` |
| `gateway-service` | `gateway-service:8080` | `127.0.0.1:8080` | Метрики на `:9094` (внутренние), публичный JWT-том |
| `ngrok` (`orbit-ngrok`) | `ngrok:4040` | `127.0.0.1:4040` | HTTPS-туннель к `gateway-service:8080`, профиль `tunnel` / `all` |
| `prometheus` | `prometheus:9090` | `127.0.0.1:9090` | Опрос 7 таргетов, alert rules, профиль `monitoring` / `all` |
| `alertmanager` | `alertmanager:9093` | `127.0.0.1:9093` | Группировка алертов и вебхук в `ops-bot`, профиль `monitoring` / `all` |
| `grafana` | `grafana:3000` | `127.0.0.1:3000` | 3 дашборда в папке `Orbit Polytech`, профиль `monitoring` / `all` |
| `loki` / `alloy` | `loki:3100`, `alloy:12345` | Не публикуются | Сбор JSON-логов из Docker-контейнеров, профиль `monitoring` / `all` |
| `postgres-exporter` / `redis-exporter` | `:9187` / `:9121` | Не публикуются | Роль БД `orbit_metrics`, профиль `monitoring` / `all` |
| `ops-bot` | `ops-bot:8081` | Не публикуется | Вебхук Alertmanager и Telegram-бот, профиль `ops` / `all` |

One-shot контейнеры (`jwt-keygen`, `postgres-init`, `auth-migrate`, `schedule-migrate`, `ai-migrate`) после завершения работы имеют статус `Exited (0)`.

Все переменные окружения и порты описаны в [справочнике портов и env](../reference/ports-and-env.md), а устройство мониторинга — на странице [Observability](observability.md).

## Целевой событийный контур (в планах)

RabbitMQ и Jaeger в текущий Compose пока не входят. Таблица ниже — предварительный эскиз будущих событий, когда появятся `tasks-service` и `notification-service`:

| Exchange | Routing Key | Очередь потребителя | Описание |
|---|---|---|---|
| `orbit.events` | `tasks.task.created` | `notification.tasks` | Создана задача |
| `orbit.events` | `tasks.task.completed` | `notification.tasks` | Задача выполнена |
| `orbit.events` | `schedule.lesson.changed` | `notification.schedule` | Изменение занятия |
| `orbit.events` | `auth.user.registered` | `users.sync` | Регистрация аккаунта |

До появления реальной шины контракты событий должны быть отдельно согласованы в `contracts`.
