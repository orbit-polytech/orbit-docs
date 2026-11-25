# Текущее состояние Orbit AI

**Проверенный срез:** ветки `main` репозиториев экосистемы на 25 ноября 2026 года (`infra v0.6.0`, `contracts v0.5.12`, `mobile v0.1.27-beta+28`, `ops-bot v0.1.0`). Локальный стенд поднят через `infra` с профилем `all`.

**Назначение страницы:** честно отделить то, что уже работает в `main`, от сервисов в разработке, планов и учебных примеров. Черновик ролевой модели и RACI приведён в разделе [«Роли и зоны ответственности»](roles.md).

## Обозначения статуса

| Статус | Значение |
|---|---|
| **Работает сейчас** | Код в `main`, покрыт тестами/CI и проверен на локальном стенде |
| **Работает сейчас (бета)** | Выпускаются рабочие релизные сборки, базовые сценарии готовы, часть модулей ещё добавляется |
| **В разработке** | Есть каркас или незавершённые ветки, прикладные сценарии не готовы |
| **Запланировано** | Есть место в архитектуре и паспорте, реализация ещё не начата |
| **Учебный пример** | Эталонный код для обучения (`examples/tasks-service`), не production-сервис |

## Репозитории и компоненты

| Компонент | Статус | Что сейчас в `main` |
|---|---|---|
| `contracts` | **Работает сейчас** | `v0.5.12` — Protobuf-схемы `auth.v1`, `schedule.v1`, `ai.v1`, `users.v1`; генерация Go и Python (`.pyi` / `py.typed`); диалоги, фидбэк, профиль пользователя (`display_name`, `username`), смена email и настройки групп |
| `auth-service` | **Работает сейчас** | Email OTP, смена email (`BeginEmailChange` / `ConfirmEmailChange`), JWT Ed25519, refresh-сессии, Google OAuth, привязка/отвязка identities, предпочтения групп (`00005`), `/metrics :9091` и JSON `slog` (`contracts@v0.5.11`) |
| `users-service` | **Работает сейчас** | Go `1.26.10`, профили пользователей в PostgreSQL (`orbit_users`), миграции `00001`–`00002` (`display_name`, `username`), gRPC `:50052`, `/metrics :9096` и JSON `slog` (`contracts@v0.5.12`) |
| `schedule-service` | **Работает сейчас** | Реестр групп, PostgreSQL, Redis-кэш + Streams + фоновая синхронизация с `rasp.dmami.ru`, `/metrics :9092`, бизнес-метрики синка, JSON `slog` (`contracts@v0.5.0`) |
| `gateway-service` | **Работает сейчас** | HTTP Auth + `/oauth/exchange` + `PATCH /preferences` + `POST /api/v1/auth/email/change/begin\|confirm`, профиль (`GET/PUT /api/v1/users/profile`), публичное расписание, AI SSE/отмена/диалоги/оценки с Redis rate limiting (`429` + `retry_after_sec`, заголовок чата до 120 символов), обращения в поддержку (`POST /api/v1/support/issues` -> `ops-bot`), Swagger, HTTP RED-метрики на `:9094` (изолирован от `:8080`), JSON `slog` (`contracts@v0.5.12`) |
| `ai-service` | **Работает сейчас** | gRPC streaming, мультипровайдер `RouterLLMClient` (Groq / OpenRouter / Gemini), роутер запросов, FastEmbed + гибридный RAG (pgvector + FTS) + опциональный reranker, история/удаление диалогов, оценки, трекинг токенов и квот, миграции `001–006`, `/metrics :9093` (`contracts@v0.5.10`) |
| `knowledge-pipeline` | **Работает сейчас** | Отдельный ETL (`orbit-etl`) для сбора `mospolytech.ru`, PDF-регламентов и ручных справочников `curated_manual/` в `output/manifest.json` для загрузки в `ai-service` |
| `ops-bot` | **Работает сейчас** | Приватный операционный Telegram-бот (`v0.1.0`): вебхук Alertmanager (`:8081`), приём обращений из `POST /api/v1/support/issues`, разбор инцидентов и плохих отзывов в Forum Topics, ежедневный дайджест токенов |
| `infra` | **Работает сейчас** | `v0.6.0` — профили `apps` (PostgreSQL, Redis, миграторы, Auth, Users, Schedule, Gateway, `web` на `:8088`), `ai`, `tunnel` (`ngrok` проксирует `web:80`), `monitoring` (Prometheus, Alertmanager, Loki, Alloy, Grafana, exporters), `ops` (`ops-bot`), `all`; 9 образов `apps` и 11 всего в GHCR; роли БД `orbit_user`, `orbit_metrics`, `orbit_ops_bot` |
| `mobile` / `web` | **Работает сейчас (бета)** | Бета `v0.1.27-beta+28` на Flutter: Android APK и **Web / PWA** (`Dockerfile.web`, `web/nginx.conf`, контейнер `web` на `127.0.0.1:8088`); светлая дизайн-система, `DesktopShellSidebar`, Floating Dock, `IndexedStack`, офлайн-расписание в Hive, выбор/синхронизация групп, Email OTP, смена email, Google OAuth (deep link `ru.mospolytech.orbit://auth/callback`), экран профиля (`features/profile` -> `users-service`), форма поддержки (`features/support` -> `POST /api/v1/support/issues`), AI-ассистент (SSE, история, удаление, оценки, повтор, цитирование выделенного фрагмента, таймер cooldown при `429`); CI и подпись APK |
| `tasks-service` | **Учебный пример / запланировано** | Учебный эталон в `orbit-docs/examples/tasks-service`; отдельный сервис задач в проектировании |
| `navigation-service` | **Запланировано** | Целевая архитектура и паспорт сервиса; базовый разбор кодов аудиторий пока работает в `ai-service` |
| `notification-service` | **Запланировано** | Целевая архитектура под события RabbitMQ |
| `bot-service` | **Запланировано** | Пользовательский Telegram-бот для студентов (отдельно от админского `ops-bot`) |

## Что проверено на живом стенде

### Auth, Users, Gateway, Support и Schedule

- Регистрация и вход по Email OTP, смена email (`POST /api/v1/auth/email/change/begin` и `POST /api/v1/auth/email/change/confirm`), ротация refresh-сессий с отзывом семейства при повторном использовании.
- Вход и безопасная привязка Google OAuth (с re-auth окном, binding cookie, `POST /api/v1/auth/oauth/exchange` и мобильным редиректом на `ru.mospolytech.orbit://auth/callback`).
- Хранение и обновление предпочтений групп (`primary_group`, `favorite_groups`) через `GET /api/v1/auth/me` и `PATCH /api/v1/auth/preferences`.
- Профили пользователей через `users-service` (`Go 1.26.10`, gRPC `:50052`, БД `orbit_users`, миграции `00001`–`00002` с полями `display_name` и `username`) и маршруты Gateway `GET /api/v1/users/profile` и `PUT /api/v1/users/profile`.
- Отправка обращений в поддержку из клиента через `POST /api/v1/support/issues` в операционный бот `ops-bot`.
- Публичный поиск групп и расписание на неделю/период до 93 дней без JWT (`READY` / `PENDING_SYNC` / `is_stale`, Redis-кэш + PostgreSQL + Streams).
- Подробнее: [API расписания](../gateway/schedule-api.md), [паспорт `users-service`](../reference/services/users.md).

### AI-контур и база знаний

- `ai-service` подключён к Gateway и работает через `POST /api/v1/ai/chat/stream`, `/api/v1/ai/chat/cancel`, список/историю/удаление диалогов (`/api/v1/ai/conversations*`, ограничение заголовка диалога — 120 символов) и оценки ответов (`PUT .../feedback`).
- На Gateway включён Redis rate limiting для AI-запросов с возвратом `429` и `retry_after_sec`.
- `RouterLLMClient` маршрутизирует запросы по ролям (`router`, `casual`, `knowledge_rag`, `campus_navigation`, `general_study`) с цепочкой fallback между Groq, OpenRouter и Gemini.
- Поиск в `orbit_ai` гибридный (HNSW-векторы FastEmbed + полнотекстовый поиск PostgreSQL + опциональный cross-encoder reranker).
- Актуальный корпус Московского Политеха собирается отдельным пайплайном [`knowledge-pipeline`](../reference/services/knowledge-pipeline.md) и грузится через `--manifest`.
- Подробнее: [AI API](../gateway/ai-api.md), [загрузка базы знаний](../python-ai/knowledge-ingestion.md), [ограничения](../python-ai/current-limitations.md).

### Наблюдаемость и `ops-bot`

- В локальном `infra v0.6.0` подняты Prometheus (`:9090`), Alertmanager (`:9093`), Loki, Alloy, Grafana (`:3000` с 3 дашбордами в папке `Orbit Polytech`), `postgres-exporter`, `redis-exporter` и `ops-bot`.
- Подтверждено на стенде: все **8 таргетов Prometheus** (включая `users-service:9096/metrics`) в статусе `UP`, Alloy забирает JSON-логи контейнеров в Loki, а запрос `/metrics` на публичном порту `:8080` возвращает `404` (метрики отдаются только на внутренних портах `9091–9094` и `9096`).
- `ops-bot` принимает алерты Alertmanager и обращения из `POST /api/v1/support/issues`, а также постит уведомления, неразобранные вопросы AI и негативные отзывы студентов в 5 Forum Topics (`#alerts`, `#llm-quotas`, `#untriaged`, `#feedback`, `#digest`).
- Подробнее: [Мониторинг и логи](../contracts-infra/observability.md), [паспорт ops-bot](../reference/services/ops-bot.md).

### Мобильное приложение и Web/PWA (`mobile` / `web`)

- Клиент (`v0.1.27-beta+28`) выпускается как Android APK и **Web / PWA** (`Dockerfile.web`, `web/nginx.conf`, контейнер `web` на `127.0.0.1:8088`; через `--profile tunnel` `ngrok` проксирует `web:80`).
- В клиенте работают: адаптивная оболочка (`DesktopShellSidebar` на широких экранах и Floating Dock на мобильных, переключение вкладок через `IndexedStack`), расписание с офлайн-кэшем Hive, переключение недель, шторка деталей пары, гостевой режим, вход по Email OTP и Google (через браузер и `app_links`), смена email, синхронизация выбранных групп с разрешением конфликтов, экран профиля (`features/profile` -> `users-service`), форма поддержки (`features/support` -> `POST /api/v1/support/issues`), настройки с привязкой/отвязкой Google и ИИ-ассистент (SSE, история, удаление чатов, оценки, повтор ответа, цитирование выделенного фрагмента и таймер cooldown при `429`).
- Подробнее: [Обзор mobile](../mobile-flutter/overview.md), [паспорт mobile](../reference/services/mobile.md).

## Ближайшие шаги

1. Спроектировать и реализовать `tasks-service` (серверные личные и групповые задачи со сроками).
2. Постепенно расширять и проверять корпус базы знаний в `knowledge-pipeline` по отзывам и кейсам из `#untriaged` / `#feedback`.
3. Развивать дополнительные сценарии расписания (поиск по преподавателям и аудиториям, сессия).
4. Спроектировать и подготовить к запуску следующие доменные сервисы (`navigation-service`, `notification-service`, студентский `bot-service`).

!!! warning "Что всё ещё в планах"
    `users-service` и экран профиля уже работают в `main`, но RabbitMQ и Jaeger для трейсинга в стек пока не добавлены. Вход через Telegram для студентов, `tasks-service`, `navigation-service`, `notification-service`, студентский `bot-service`, а также вкладки «Задачи» и «Заметки» в мобильном приложении остаются на следующих этапах.
