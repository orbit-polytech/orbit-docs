# 02. Запуск локального окружения (infra)

!!! info "Статус: актуальный сценарий контейнеризации"
    Руководство относится к стенду PostgreSQL + Redis + Auth + Schedule + Gateway. Это профиль `apps` без AI. AI/pgvector уже добавлены отдельно через `ai`/`all`: [инструкция](../../how-to/ai-local-stack.md). RabbitMQ и Jaeger остаются будущими этапами.

## 1. Зачем нужен `infra`

Репозиторий `infra` позволяет участнику команды поднять backend без установки Go и без клонирования всех сервисов. Базовый Compose запускает готовые образы; исходники подключаются только через локальный override.

## 2. Первый запуск

```bash
git clone https://github.com/orbit-polytech/infra.git
cd infra
cp .env.example .env
# Заполните локальные секреты и выполните вход в GHCR (см. ссылку ниже)
task up
```

Для private-пакетов сначала выполните [авторизацию Docker и проверку шести образов](../../quickstart/ghcr-access.md).

Проверьте состояние:

```bash
task status
```

Основные сервисы должны быть `healthy`, а `jwt-keygen`, `postgres-init`, `auth-migrate` и `schedule-migrate` — `Exited (0)`.

## 3. Проверка Gateway

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

Swagger: [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html).

## 4. Сети и порты

Внутри Compose сервисы обращаются друг к другу по DNS-именам:

```text
gateway-service -> auth-service:50051
gateway-service -> schedule-service:50054
schedule-service -> postgres:5432 (orbit_schedule)
auth-service    -> postgres:5432
auth-service    -> redis:6379
```

На хост по умолчанию публикуются:

- Gateway: `127.0.0.1:8080`;
- Auth gRPC: `127.0.0.1:50051` для локальной диагностики;
- Schedule gRPC: `127.0.0.1:50054` для локальной диагностики;
- PostgreSQL: `127.0.0.1:5434`;
- Redis: `127.0.0.1:6380`.

## 5. Разработка одного сервиса

```bash
cp compose.override.yaml.example compose.override.yaml
```

В текущем шаблоне build-блоки активны у всех сервисов. Удалите/закомментируйте лишние и оставьте только нужный:

```yaml
services:
  auth-service:
    build:
      context: ../auth-service
      dockerfile: Dockerfile
```

Затем:

```bash
task build
docker compose --profile apps up -d --pull never --wait
```

## 6. Управление данными

- `task down` сохраняет volumes;
- `task clean` удаляет PostgreSQL, Redis и JWT-ключи;
- после удаления JWT-томов ранее выданные токены становятся недействительными.

Повторная подготовка существующего volume: `task up:deps` → `task db:init` → `task up`. Не удаляйте данные ради появления базы Schedule.

При обновлении готовых образов сначала выполните `docker compose --profile apps pull`, затем `task up`. Локальный `latest` сам по себе не гарантирует актуальную версию.

Публичная проверка: `curl "http://localhost:8080/api/v1/schedule/groups/261-334"`. Первичная синхронизация может ещё идти, даже если Schedule уже `healthy`. См. [API расписания](../../gateway/schedule-api.md).

## 7. Продолжение

- [Запуск актуального стенда](../../quickstart/local-setup.md)
- [Текущее состояние проекта](../../project/current-state.md)
- [Справочник портов и env](../../reference/ports-and-env.md)
