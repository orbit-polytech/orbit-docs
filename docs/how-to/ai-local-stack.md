# Запуск локального стенда с AI

## 1. Подготовка

Перед началом пройдите [базовый Quickstart](../quickstart/local-setup.md): установите Docker Compose v2, Task, склонируйте `infra`, создайте `.env` и настройте [доступ к GHCR](../quickstart/ghcr-access.md) для `ai-service` (сервер и `ai-migrate` берут один и тот же образ).

В `infra/.env` выберите провайдера и заполните соответствующий ключ:

```ini
AI_PROVIDER=groq         # или openrouter / gemini
GROQ_API_KEY=...
OPENROUTER_API_KEY=...
GEMINI_API_KEY=...
AI_MODEL=qwen/qwen3.8-27b
```

При `APP_ENV=production` отсутствие API-ключа для выбранного `AI_PROVIDER` останавливает запуск `ai-service`. Реальные ключи в Git добавлять нельзя.

## 2. Запуск с нужным профилем

| Команда | Что поднимется |
|---|---|
| `task up` | Профиль `apps`: `postgres`, `redis`, `jwt-keygen`, `postgres-init`, миграторы (`auth-migrate`, `schedule-migrate`, `users-migrate`), `auth-service`, `users-service`, `schedule-service`, `gateway-service`, `web` — без AI |
| `docker compose --profile ai up -d` | Только `ai-service`, `ai-migrate` и `postgres` + `postgres-init` |
| `docker compose --profile all up -d --wait --wait-timeout 180` | Полный стенд: `apps` + `ai` + `monitoring` + `ops` + `tunnel` |

Для работы HTTP API через Gateway с AI (и при желании с Grafana/`ops-bot`) запускайте `--profile all`:

```bash
git pull --ff-only
docker compose --profile all pull
docker compose --profile all up -d --wait --wait-timeout 180
docker compose --profile all ps -a
```

`postgres-init`, `jwt-keygen` и миграторы (`auth-migrate`, `schedule-migrate`, `users-migrate`, `ai-migrate`) должны завершиться с `Exited (0)`, а остальные контейнеры перейти в `healthy` / `running`.

## 3. Если том PostgreSQL уже был создан раньше

`infra` использует `pgvector/pgvector:pg16`. Если база уже поднималась раньше до появления AI, удалять volume (`task clean`) не нужно — достаточно переинициализировать роли и базы через `postgres-init`:

```bash
task up:deps
task db:init
docker compose --profile all up -d --wait --wait-timeout 180
```

`ai-migrate` накатит Alembic только на рабочую `orbit_ai`. Для интеграционных тестов с `POSTGRES_TEST_DSN` применяйте миграции на `orbit_ai_test` отдельно.

## 4. Проверка и загрузка базы знаний

Проверить здоровье AI-сервиса:

```bash
docker compose --profile all logs --tail 100 postgres-init ai-migrate ai-service
docker compose --profile all exec ai-service python -m orbit_ai.presentation.grpc.health_probe
```

Без загруженных документов `ai-service` запустится, но отвечать по регламентам вуза не сможет. Как собрать Knowledge Bundle в `knowledge-pipeline` и залить `manifest.json`, описано в [инструкции по загрузке базы знаний](../python-ai/knowledge-ingestion.md).

Расход токенов, задержки LLM и остатки квот после тестовых запросов удобно смотреть в Grafana (`http://127.0.0.1:3000`, дашборд **Orbit — AI & LLM Tokens / Quotas**) или через команду `/tokens` в [`ops-bot`](../reference/services/ops-bot.md).

## 5. Локальная сборка AI из соседнего клона

Шаблон `compose.override.yaml.example` содержит активные `build:`-секции для всех сервисов. Скопируйте его в `compose.override.yaml` и оставьте только `ai-service` и `ai-migrate`:

```bash
docker compose --profile all build ai-service ai-migrate
docker compose --profile all up -d --pull never --wait --wait-timeout 180
```

Зависимость `orbit-contracts` подтягивается при сборке прямо из Git (`v0.5.10`), а модель FastEmbed кэшируется в Docker-слое.

Источники: [compose.yaml](https://github.com/orbit-polytech/infra/blob/main/compose.yaml), [Taskfile.yml](https://github.com/orbit-polytech/infra/blob/main/Taskfile.yml), [observability](../contracts-infra/observability.md).
