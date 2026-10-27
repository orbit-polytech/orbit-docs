# Диагностика Docker-стенда

Текущий `infra` запускает PostgreSQL, Redis, Auth, Schedule, Gateway и четыре one-shot сервиса. RabbitMQ и Jaeger пока не развёрнуты.

## Сначала статус и логи

Из каталога `infra`:

```bash
task status
docker compose --profile apps logs --tail 100 postgres-init auth-migrate schedule-migrate
docker compose --profile apps logs --tail 100 auth-service schedule-service gateway-service
```

`jwt-keygen`, `postgres-init`, `auth-migrate` и `schedule-migrate` в `Exited (0)` — нормальный результат, не CrashLoop. Ненулевой код требует изучения логов. Сохраните имя сервиса, тег образа и сообщение ошибки; не отправляйте секреты `.env` или Docker config.

## Не скачивается private-образ

Проверьте [вход в GHCR и Read-доступ ко всем шести пакетам](../quickstart/ghcr-access.md). `Login Succeeded` не гарантирует права чтения конкретного пакета. `manifest unknown` также может означать отсутствие опубликованного тега.

Если падает **CI на `setup-buildx-action` при скачивании `moby/buildkit`**, это ещё не сборка Dockerfile и не доступ к private-образу Orbit в GHCR. Изучите полный лог с именем реестра, повторите временно неудавшуюся job и проверьте доступ к реестру builder-образа. Предупреждение Node `punycode` само по себе не причина ошибки Docker daemon. Не добавляйте PAT команды в workflow вслепую.

## Занят порт

Дефолты на хосте: PostgreSQL `5434`, Redis `6380`, Gateway `8080`, Auth `50051`, Schedule `50054`. Меняйте только нужный `*_HOST_PORT` в `.env`, а не межконтейнерный адрес.

```powershell
# PowerShell: пример для PostgreSQL
Get-NetTCPConnection -LocalPort 5434 | Select-Object LocalPort, OwningProcess
```

```bash
# Linux/macOS
lsof -i :5434
docker ps --filter "publish=5434"
```

Сначала определите владельца порта. Не завершайте чужие процессы и не останавливайте все PostgreSQL-службы без необходимости.

## Старая база и ошибка мигратора

Скрипты `docker-entrypoint-initdb.d` не запускаются повторно на существующем PostgreSQL volume. Для добавления Schedule в текущем стеке предусмотрен `postgres-init`:

```bash
task up:deps
task db:init
task up
```

Это не замена Goose-миграциям и не способ сбросить административный пароль. Проверьте `POSTGRES_DB`, `SCHEDULE_POSTGRES_DB`, credentials и фактическое наличие базы Auth. Повторная подготовка обновляет пароль существующей app-роли из `APP_DB_PASSWORD`. См. [подробности infra](../contracts-infra/infra.md).

## Контейнеры healthy, но расписание ещё не готово

`healthy` не подтверждает загрузку расписания всех групп. При `200 PENDING_SYNC` задание подтверждено: проверьте работу воркеров, доступность источника и `SCHEDULE_SYNC_ENABLED`. Повторяйте запрос с задержкой, не в бесконечном тесном цикле.

`404` — неизвестная группа; `200 READY` с пустым `days` — готовый результат без занятий за выбранные даты. `503` — недоступен upstream либо нет сохранённого расписания и невозможно подтвердить постановку задания. Ошибка Redis-кэша при доступном PostgreSQL не должна сама по себе скрывать уже загруженные данные.

Проверьте `SCHEDULE_GRPC_ADDR`, RPC timeout, логи, Redis и миграции. При начальной недоступности Redis чтение может работать из PostgreSQL, а синхронизация ожидает восстановления Redis. В Compose запуск контейнера также зависит от healthcheck Redis.

Пошаговые проверки — [диагностика синхронизации](schedule-sync-diagnostics.md), семантика ответов — [API](../gateway/schedule-api.md).

## Изменения конфигурации или latest не применились

```bash
# После публикации новых образов
docker compose --profile apps pull
task up
```

`restart` не перечитывает env из Compose и не меняет образ контейнера. Для локально собранных сервисов не перетирайте изменения pull: используйте сценарий override из [Quickstart](../quickstart/local-setup.md).

## Контейнеры не видят друг друга

Сравните Compose-проект и сети:

```bash
docker compose --profile apps ps -a
docker network ls
docker network inspect orbit-infra_default
```

`orbit-infra_default` — обычное имя сети для текущего `name: orbit-infra`; при переопределении имени проекта оно меняется. Отдельную сеть `orbit-net` текущий Compose не объявляет. Внутри контейнеров нужны DNS-адреса `postgres:5432`, `redis:6379`, `auth-service:50051`, `schedule-service:50054`, а не `localhost` и не host-порты.

## Освобождение места и полный сброс

Сначала оцените использование:

```bash
docker system df
```

Очистка build cache — отдельное действие, например `docker builder prune`. Не удаляйте volumes для исправления скачивания образа, неизвестного порта или незагруженного расписания.

!!! danger "Только намеренный полный сброс локального стенда"
    `task clean` удаляет данные PostgreSQL, Redis и оба JWT-тома. После запуска будут новые ключи, старые JWT перестанут приниматься. Делайте резервную копию нужных данных. Обычный `task down` сохраняет volumes.
