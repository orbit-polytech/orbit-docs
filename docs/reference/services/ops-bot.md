# ops-bot (Операционный Telegram-бот)

**Репозиторий:** `orbit-polytech/ops-bot` (приватный), релиз `v0.1.0`.

**Статус:** работает сейчас; запускается в профилях Compose `ops` и `all`.

## Назначение и границы

`ops-bot` — это внутренний служебный бот для команды разработки и дежурных. Он принимает алерты от Alertmanager, принимает обращения и баг-репорты из клиентского приложения через Gateway (`POST /issues/webhook`), отслеживает проблемные вопросы к ИИ и дизлайки студентов, а также присылает ежедневные сводки по расходу токенов.

!!! note "Не путать с `bot-service`"
    Пользовательский Telegram-бот для студентов (`bot-service`) по-прежнему находится в планах. `ops-bot` предназначен только для администраторов и эксплуатации.

## Стек

| Слой | Реализация |
|---|---|
| Runtime | Python 3.13+, `uv`, `uv.lock` |
| Telegram | `aiogram` 3.x (роутеры команд, inline-клавиатуры разбора, фильтр `ADMIN_IDS`) |
| Вебхук | `aiohttp` (`0.0.0.0:8081`, маршруты `GET /healthz`, `POST /alerts/webhook` и `POST /issues/webhook` с Bearer-авторизацией) |
| Метрики | `httpx`-клиент к Prometheus (`http://prometheus:9090`) |
| База данных | `SQLAlchemy asyncio` + `asyncpg`, роль `orbit_ops_bot` к `orbit_ai` |

Роль `orbit_ops_bot` настраивается скриптом `postgres-init` с минимальными правами: `SELECT` по таблицам `orbit_ai` и `UPDATE` только на три поля триажа (`triaged_at`, `triaged_by`, `ops_notified_at`) в `generation_attempts`.

## Топики супергруппы (Forum Topics)

Бот раскладывает уведомления по пяти отдельным веткам в админской супергруппе:

| Топик | Переменная env | Откуда приходят события |
|---|---|---|
| `#alerts` | `OPS_BOT_TOPIC_ALERTS_ID` | Вебхук Alertmanager (`POST /alerts/webhook`): падение сервисов, 5xx, высокая задержка, БД (включая `resolved`) |
| `#llm-quotas` | `OPS_BOT_TOPIC_LLM_QUOTAS_ID` | Вебхук Alertmanager: алерты с `category=llm-quota` (остаток квоты < 15% / < 5%, частые `429`) |
| `#untriaged` | `OPS_BOT_TOPIC_UNTRIAGED_ID` | Фоновый опрос `orbit_ai` каждые `TRIAGE_POLL_INTERVAL_SEC` сек (по умолчанию `30`): вопросы без достаточного контекста или со сбоем |
| `#feedback` | `OPS_BOT_TOPIC_FEEDBACK_ID` | Негативные оценки студентов (`feedback_rating = down`) и обращения/баг-репорты из приложения (`POST /issues/webhook` от Gateway) |
| `#digest` | `OPS_BOT_TOPIC_DIGEST_ID` | Ежедневная сводка по расписанию (`DIGEST_HOUR_UTC`, по умолчанию `6`): очередь неразобранного, расход токенов за 24ч и остатки квот |

В карточках в `#untriaged` и `#feedback` есть inline-кнопки: **«✅ В базу знаний / Разобрано»** и **«🚫 Игнор»** — они сразу отмечают запись в БД (`triaged_at`, `triaged_by`), чтобы кейс не всплывал повторно.

## Команды администраторов

Команды проверяются фильтром по списку `ADMIN_IDS` (кроме `/topics_id`, которая тоже доступна только ID из `ADMIN_IDS`, но работает ещё до заполнения `OPS_GROUP_ID`):

| Команда | Что делает |
|---|---|
| `/topics_id` | Возвращает `OPS_BOT_GROUP_ID` и `MESSAGE_THREAD_ID` текущей ветки для первичной настройки `.env` |
| `/status` | Запрашивает `up` по всем таргетам в Prometheus и выводит список сервисов |
| `/tokens` | Считает расход токенов LLM (вход/выход/запросы) за последние 24 часа из `orbit_ai` |
| `/quotas` | Показывает текущие остатки квот и процент по моделям из Prometheus |
| `/untriaged` | Показывает до 5 неразобранных вопросов или дизлайков с кнопками разбора |

## Конфигурация и запуск

Образ `ghcr.io/orbit-polytech/ops-bot:latest` приватный — нужен `docker login ghcr.io` с правом `read:packages` (см. [доступ к GHCR](../../quickstart/ghcr-access.md)). Переменные задаются в `infra/.env` (примеры из `.env.example`):

```ini
OPS_BOT_IMAGE=ghcr.io/orbit-polytech/ops-bot:latest
OPS_BOT_DB_USER=orbit_ops_bot
OPS_BOT_DB_PASSWORD=change_this_ops_bot_password
OPS_BOT_WEBHOOK_TOKEN=change_this_webhook_bearer_token
OPS_BOT_TELEGRAM_TOKEN=
OPS_BOT_GROUP_ID=-1001234567890
OPS_BOT_ADMIN_IDS=123456789
OPS_BOT_TOPIC_ALERTS_ID=0
OPS_BOT_TOPIC_LLM_QUOTAS_ID=0
OPS_BOT_TOPIC_UNTRIAGED_ID=0
OPS_BOT_TOPIC_FEEDBACK_ID=0
OPS_BOT_TOPIC_DIGEST_ID=0
```

После `docker compose --profile all up -d` или `--profile ops up -d` вебхук слушает внутри Docker-сети на `http://ops-bot:8081` (`POST /alerts/webhook` для Alertmanager и `POST /issues/webhook` для обращений от Gateway) и на хост наружу не публикуется.

Подробнее про метрики и алерты, которые питают бота: [Мониторинг и логи](../../contracts-infra/observability.md).
