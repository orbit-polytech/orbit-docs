# Репозиторий infra: Docker Compose и БД

`infra` (`v0.4.0`) — единая точка запуска локального стенда Orbit из готовых OCI-образов в GHCR. Для запуска базовый `compose.yaml` не требует клонировать соседние репозитории, достаточно настроить [доступ к приватным пакетам GHCR](../quickstart/ghcr-access.md).

## Состав стенда

| Сервис | Профиль | Роль |
|---|---|---|
| `postgres` | всегда | PostgreSQL 16 (`pgvector`); базы Auth, Schedule, AI и AI test |
| `redis` | всегда | OTP, OAuth state и rate limits Auth; кэш, Streams и координация Schedule |
| `jwt-keygen` | `apps`, `all` | One-shot генерация Ed25519-ключей без доступа в сеть |
| `postgres-init` | `apps`, `ai`, `monitoring`, `ops`, `all` | One-shot создание баз, расширения `vector` и ролей БД |
| `auth-migrate` | `apps`, `all` | One-shot миграции Goose базы `orbit_auth` |
| `schedule-migrate` | `apps`, `all` | One-shot миграции Goose базы `orbit_schedule` |
| `auth-service` | `apps`, `all` | gRPC `:50051`, метрики `:9091` |
| `schedule-service` | `apps`, `all` | gRPC `:50054`, метрики `:9092`, фоновая синхронизация |
| `gateway-service` | `apps`, `all` | HTTP `:8080`, Swagger, метрики `:9094` (только внутренняя сеть) |
| `ai-migrate` | `ai`, `all` | One-shot Alembic-миграции `orbit_ai` (тот же образ, что у `ai-service`) |
| `ai-service` | `ai`, `all` | gRPC `:50057`, метрики `:9093`, мультипровайдерный LLM и RAG |
| `ngrok` (`orbit-ngrok`) | `tunnel`, `all` | Внешний HTTPS-туннель к `gateway-service:8080`, инспектор `:4040` |
| `postgres-exporter`, `redis-exporter` | `monitoring`, `all` | Экспорт метрик PostgreSQL и Redis в Prometheus |
| `prometheus` | `monitoring`, `all` | Сбор метрик (`:9090`) и проверка alert rules |
| `alertmanager` | `monitoring`, `all` | Маршрутизация алертов в `ops-bot` (`:9093`) |
| `loki`, `alloy` | `monitoring`, `all` | Сбор и хранение JSON-логов Docker-контейнеров |
| `grafana` | `monitoring`, `all` | Дашборды `Orbit Polytech` (`:3000`) |
| `ops-bot` | `ops`, `all` | Операционный Telegram-бот, вебхук Alertmanager (`:8081`) |

**`task up` поднимает только профиль `apps` (без AI, мониторинга, `ops-bot` и Ngrok).** Для полного стенда запускайте `docker compose --profile all up -d --wait --wait-timeout 180`. Обычный `docker compose up -d` без `--profile` поднимет только `postgres` и `redis`.

RabbitMQ и Jaeger в текущий Compose пока не включены.

## Порядок запуска и готовность

1. `postgres` и `redis` проходят healthcheck.
2. `postgres-init` создаёт роли и базы; параллельно `jwt-keygen` готовит Ed25519-ключи (если тома ещё пустые).
3. `auth-migrate`, `schedule-migrate` и `ai-migrate` ждут завершения `postgres-init`.
4. Микросервисы стартуют после своих миграций и зависимостей; `gateway-service` ждёт `auth-service`, `schedule-service` и `jwt-keygen` (от `ai-service` в `depends_on` не зависит).
5. Контейнеры мониторинга и `ops-bot` поднимаются после `postgres-init` и своих БД/источников.

У всех one-shot сервисов нормальный статус после старта — `Exited (0)`. Healthcheck `schedule-service` говорит о запуске сервиса, а не о том, что расписание всех групп уже выкачано.

## Основные команды

```bash
cp .env.example .env
# Заполните секреты в .env и авторизуйтесь в ghcr.io
task up
task status
```

| Команда | Назначение |
|---|---|
| `task up` | Профиль `apps` (PostgreSQL, Redis, Auth, Schedule, Gateway) |
| `task up:deps` | Только PostgreSQL и Redis |
| `task db:init` / `task init:db` | Повторный запуск `postgres-init` без удаления томов |
| `task build` | Локальная сборка сервисов из `compose.override.yaml` |
| `task status` | Статус всех контейнеров профиля `apps` |
| `task logs` | Логи профиля `apps` |
| `task down` | Остановка с сохранением данных и ключей |
| `task clean` | Полное удаление контейнеров и томов (PostgreSQL, Redis, JWT, Prometheus, Loki, Grafana) |
| `task tunnel:up` / `task tunnel:down` | Запуск и остановка Ngrok-туннеля для `gateway-service` |
| `task test` | Прогон Go-тестов инфраструктурных скриптов |

Команды для полного стека выполняются напрямую через Docker Compose с `--profile all` (или `--profile monitoring` / `--profile ops`):

```bash
docker compose --profile all pull
docker compose --profile all up -d --wait --wait-timeout 180
docker compose --profile all ps -a
```

## Роли БД и старый volume PostgreSQL

Entrypoint официального PostgreSQL выполняет `/docker-entrypoint-initdb.d` только на пустом каталоге данных. Если в `infra` появился новый сервис или роль, а volume уже существует, не удаляйте данные — выполните `postgres-init`:

```bash
task up:deps
task db:init
```

Скрипт `scripts/initdb/01-init-user.sh` идемпотентно создаёт:
- `APP_DB_USER` (`orbit_user`) — рабочую роль без superuser с правами на базы `orbit_auth`, `orbit_schedule`, `orbit_ai` и `orbit_ai_test` (пароль обновляется из `APP_DB_PASSWORD`);
- `METRICS_DB_USER` (`orbit_metrics`) — роль мониторинга с `pg_monitor` (создаётся при непустом `METRICS_DB_PASSWORD`);
- `OPS_BOT_DB_USER` (`orbit_ops_bot`) — ограниченную роль для `orbit_ai`: `SELECT` на все таблицы схемы `public` и `UPDATE (triaged_at, triaged_by, ops_notified_at)` в `generation_attempts` (создаётся при непустом `OPS_BOT_DB_PASSWORD`).

Перед удалением тома помните, что `task clean` стирает в том числе `jwt_private`/`jwt_public` и тома мониторинга (`prometheus_data`, `loki_data`, `grafana_data`).

## Ключи Ed25519

Приватный ключ монтируется read-only только в `auth-service` (`jwt_private`), публичный — только в `gateway-service` (`jwt_public`). Обычные `down` / `up` ключи не меняют.

## Локальная разработка

```bash
cp compose.override.yaml.example compose.override.yaml
# Оставьте build-блоки только для нужных сервисов, остальные удалите или закомментируйте
task build
docker compose --profile apps up -d --pull never --wait --wait-timeout 120
```

`compose.override.yaml` не должен попадать в Git. При изменении миграций Schedule или Auth собирайте также `schedule-migrate` / `auth-migrate`.

Подробнее о мониторинге: [Мониторинг и логи](observability.md), о запуске AI: [локальный стенд с AI](../how-to/ai-local-stack.md).
