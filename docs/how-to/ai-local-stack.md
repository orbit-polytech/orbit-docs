# Запуск локального стенда с AI

## 1. Предварительные условия

Пройдите [основной Quickstart](../quickstart/local-setup.md): Docker Compose v2, Task, клон `infra`, локальный `.env`, доступ к GHCR. Для AI дополнительно нужен Read-доступ к `ghcr.io/orbit-polytech/ai-service`. Образ используется и сервером, и `ai-migrate`; отдельного `ai-migrate` пакета нет.

В `infra/.env` заполните `GROQ_API_KEY` и выберите `AI_MODEL`, доступную вашему аккаунту Groq. Шаблон содержит модель по умолчанию, но её доступность и квоты нужно проверять у провайдера. В Compose `APP_ENV` по умолчанию `production`: пустой ключ не является offline-режимом и останавливает AI.

Не помещайте ключи в Git и не отправляйте `.env`/полный вывод `docker compose config` в переписку.

## 2. Выберите профиль

| Запуск | Состав |
|---|---|
| `task up` (`apps`) | Auth, Schedule, Gateway и зависимости; **без AI** |
| `docker compose --profile ai up -d` | AI, его миграции и необходимые зависимости; без Gateway/Auth |
| `docker compose --profile all up -d` | Совместный backend-стенд с AI |

Для работы HTTP AI API нужен последний вариант:

```bash
git pull --ff-only
docker compose --profile all pull
docker compose --profile all up -d --wait --wait-timeout 180
docker compose --profile all ps -a
```

Увеличенный wait-timeout позволяет дождаться запуска, но не меняет параметры healthcheck контейнера. Если выбран локальный override, убедитесь, что он содержит только реально имеющиеся build-context. Потребителю готовых образов override не нужен.

`postgres-init`, keygen и миграторы должны завершиться с кодом `0`; серверы — стать `healthy`. AI зависит от успешного `ai-migrate`. Gateway не ждёт AI в `depends_on`: готовность Gateway не доказывает готовность AI.

## 3. Обновление существующей базы

Текущий PostgreSQL образ — `pgvector/pgvector:pg16`. Для существующего volume сделайте резервную копию и проверьте совместимость перед сменой образа; Compose не заменяет major-upgrade PostgreSQL.

Provisioner создаёт базы `orbit_ai` и `orbit_ai_test`, расширение vector и права приложения. initdb-скрипты сами не выполняются повторно на непустом volume; для явного повторного provision:

```bash
task up:deps
task db:init
docker compose --profile all up -d --wait --wait-timeout 180
```

Не используйте `task clean` / `down -v` для добавления AI: это удаляет данные и JWT-ключи. `ai-migrate` применяет Alembic к `orbit_ai`, **не** к `orbit_ai_test`. Для live-тестов миграции тестовой базы применяются отдельно с тестовым DSN; не направляйте тесты на рабочую БД.

## 4. Проверка

```bash
docker compose --profile all logs --tail 100 postgres-init ai-migrate ai-service
docker compose --profile all exec ai-service python -m orbit_ai.presentation.grpc.health_probe
```

После запуска получите JWT через Auth и выполните [пример AI API](../gateway/ai-api.md). Пустая база знаний допустима для старта, но не подтверждает готовность RAG. Для локальной проверки можно отдельно загрузить [demo-документы](../python-ai/knowledge-ingestion.md).

БД внутри сети: `postgres:5432/orbit_ai`; с хоста: `127.0.0.1:5434`. gRPC AI: внутри `ai-service:50057`, на хосте только `127.0.0.1:50057`. Пользовательский API — Gateway `:8080`.

## 5. Локальная сборка и управление

Шаблон `compose.override.yaml.example` уже содержит **активные** build-блоки для всех сервисов. После копирования удалите/закомментируйте все ненужные блоки; иначе Compose потребует соседние репозитории.

```bash
# После настройки override для ai-service и ai-migrate:
docker compose --profile all build ai-service ai-migrate
docker compose --profile all up -d --pull never --wait --wait-timeout 180
```

Оба AI-сервиса должны собираться из одного Dockerfile/исходного кода. Контракты устанавливаются как Git-зависимость, соседний `contracts` для контейнера не нужен. Модель FastEmbed включена в образ; смена `EMBEDDING_MODEL_NAME` требует отдельной проверки кэша, размерности и повторной индексации, а не только изменения env.

Для AI используйте Compose явно: `docker compose --profile all logs -f ai-service`, `docker compose --profile all ps -a`, `docker compose --profile all down`. Текущие `task logs/status/down/build` ориентированы на `apps`, отдельных `task up:ai` / `up:all` нет. `down` без `-v` сохраняет данные.

Источники: [Compose](https://github.com/orbit-polytech/infra/blob/main/compose.yaml), [Taskfile](https://github.com/orbit-polytech/infra/blob/main/Taskfile.yml), [override](https://github.com/orbit-polytech/infra/blob/main/compose.override.yaml.example).
