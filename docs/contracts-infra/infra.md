# Репозиторий infra: текущий Docker Compose

`infra` — потребитель готовых OCI-образов и единая точка запуска локального стенда. Базовый `compose.yaml` в `main` не содержит `build:` и не зависит от соседних исходников. Для private-образов сначала настройте [Read-доступ и вход в GHCR](../quickstart/ghcr-access.md).

## Состав актуального стенда

| Сервис | Роль |
|---|---|
| `postgres` | PostgreSQL 16; отдельные базы Auth и Schedule |
| `redis` | OTP, OAuth state и rate limits Auth; кэш расписания, Streams и координация Schedule |
| `jwt-keygen` | One-shot подготовка пары Ed25519, без сети |
| `postgres-init` | One-shot создание роли/базы Schedule и настройка прав |
| `auth-migrate` | One-shot миграции Goose базы Auth |
| `schedule-migrate` | One-shot миграции Goose базы Schedule |
| `auth-service` | gRPC Auth, `50051` |
| `schedule-service` | gRPC Schedule, `50054`, фоновая синхронизация |
| `gateway-service` | HTTP API Auth и публичного расписания, Swagger, `8080` |

PostgreSQL и Redis запускаются без профиля. Приложения и one-shot сервисы включены в профили `apps` и `all`; **`task up` выбирает `apps`**. `docker compose up -d` без профиля не эквивалентен полному стенду.

RabbitMQ, Jaeger, pgvector, AI и остальные бизнес-сервисы пока не входят в актуальный Compose.

## Порядок запуска и готовность

1. PostgreSQL и Redis проходят healthcheck.
2. `postgres-init` подготавливает роль и базу Schedule; keygen независимо готовит ключи.
3. Оба мигратора ждут успешного `postgres-init`.
4. Auth ждёт свои миграции, Redis и keygen; Schedule — свои миграции, PostgreSQL и Redis.
5. Gateway стартует после healthcheck Auth и Schedule и завершения keygen.

Завершившиеся one-shot контейнеры должны иметь `Exited (0)`. Healthcheck Schedule не подтверждает готовность всех групп: первичная синхронизация продолжается в фоне.

## Основные команды

```bash
cp .env.example .env
# Заполните локальные значения и настройте доступ к GHCR
task up
task status
```

| Команда | Назначение |
|---|---|
| `task up` | Полный стек с ожиданием готовности контейнеров |
| `task up:deps` | Только PostgreSQL и Redis |
| `task db:init` / `task init:db` | Повторная подготовка роли, базы Schedule и прав |
| `task build` | Локальная сборка подключённых через override сервисов |
| `task logs` | Просмотр логов |
| `task down` | Остановка с сохранением данных и ключей |
| `task clean` | Полное удаление данных и ключей с подтверждением |

## Существующий том PostgreSQL

Официальный entrypoint PostgreSQL выполняет `docker-entrypoint-initdb.d` только на пустом каталоге данных. Поэтому добавление SQL/скрипта само по себе не обновляет старый volume.

В текущем стеке эту задачу решает `postgres-init`, вызывающий `scripts/initdb/01-init-user.sh` до миграций. Для ручного повторного запуска:

```bash
task up:deps
task db:init
task up
```

Скрипт создаёт роль приложения при отсутствии, задаёт ей пароль из `APP_DB_PASSWORD`, создаёт базу `${SCHEDULE_POSTGRES_DB:-orbit_schedule}` при отсутствии и выдаёт права. **Пароль существующей роли тоже обновляется.** Это не миграция содержимого базы и не ротация административного пароля PostgreSQL.

На старом volume `POSTGRES_DB` должен указывать на уже существующую базу Auth; простая смена этого имени или `POSTGRES_ADMIN_PASSWORD` в `.env` не переименовывает базу и не меняет пароль администратора. Не удаляйте volume ради добавления Schedule. Приложения работают без superuser, но текущая локальная конфигурация использует одну app-роль для обеих баз; отдельные runtime/migration-роли — дальнейшее усиление изоляции.

## Обновление готовых образов

`latest` — изменяемый тег, локальный кэш не гарантирует свежую версию. После обновления `infra`:

```bash
git pull --ff-only
docker compose --profile apps pull
task up
```

Публикация образов в репозиториях сервисов должна завершиться **до** обновления потребительского стенда. Для фиксированного среза укажите опубликованные SHA-теги или digest в переменных `*_IMAGE` из `.env.example`; версии Schedule и его мигратора выбирайте согласованно.

## Ключи Ed25519

Ключи хранятся в раздельных named volumes: приватный монтируется только в Auth, публичный — только в Gateway, оба read-only для приложений. Повторный `down` → `up` не ротирует пару. `task clean` удаляет оба тома: старые JWT перестанут проходить проверку после генерации новой пары.

## Локальная разработка

```bash
cp compose.override.yaml.example compose.override.yaml
# Раскомментируйте build только у изменяемых сервисов
task build
docker compose --profile apps up -d --pull never --wait --wait-timeout 120
```

`compose.override.yaml` автоматически объединяется с базовым файлом и не коммитится. Для локальных миграций Schedule подключите также сборку `schedule-migrate`; иначе схема может не соответствовать изменённому коду. Полная инструкция — [Quickstart](../quickstart/local-setup.md).
