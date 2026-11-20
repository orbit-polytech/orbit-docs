# Текущее состояние Orbit AI

**Проверенный срез:** ветки `main` репозиториев экосистемы на 20 ноября 2026 года (`infra v0.4.0`, `contracts v0.5.9`, `mobile 0.1.16-beta+17`, `ops-bot v0.1.0`). Локальный стенд поднят через `infra` с профилем `all`.

**Назначение страницы:** честно отделить то, что уже работает в `main`, от сервисов в разработке и учебных примеров.

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
| `contracts` | **Работает сейчас** | `v0.5.9` — Protobuf-схемы `auth.v1`, `schedule.v1`, `ai.v1`, `users.v1`; генерация Go и Python (`.pyi` / `py.typed`); диалоги, фидбэк и настройки групп |
| `auth-service` | **Работает сейчас** | Email OTP, JWT Ed25519, refresh-сессии, Google OAuth, привязка/отвязка identities, предпочтения групп (`00005`), `/metrics :9091` и JSON `slog` (`contracts@v0.5.5`) |
| `schedule-service` | **Работает сейчас** | Реестр групп, PostgreSQL, Redis-кэш + Streams + фоновая синхронизация с `rasp.dmami.ru`, `/metrics :9092`, бизнес-метрики синка, JSON `slog` (`contracts@v0.5.0`) |
| `gateway-service` | **Работает сейчас** | HTTP Auth + `/oauth/exchange` + `PATCH /preferences` + мобильный redirect привязки Google, публичное расписание, AI SSE/отмена/диалоги/оценки, Swagger, HTTP RED-метрики на `:9094` (изолирован от `:8080`), JSON `slog` (`contracts@v0.5.9`) |
| `ai-service` | **Работает сейчас** | gRPC streaming, мультипровайдер `RouterLLMClient` (Groq / OpenRouter / Gemini), роутер запросов, FastEmbed + гибридный RAG (pgvector + FTS) + опциональный reranker, история/удаление диалогов, оценки, трекинг токенов и квот, миграции `001–006`, `/metrics :9093` |
| `knowledge-pipeline` | **Работает сейчас** | Отдельный ETL (`orbit-etl`) для сбора `mospolytech.ru`, PDF-регламентов и ручных справочников `curated_manual/` в `output/manifest.json` для загрузки в `ai-service` |
| `ops-bot` | **Работает сейчас** | Приватный операционный Telegram-бот (`v0.1.0`): вебхук Alertmanager (`:8081`), разбор инцидентов и плохих отзывов в Forum Topics, ежедневный дайджест токенов |
| `infra` | **Работает сейчас** | `v0.4.0` — профили `apps`, `ai`, `tunnel` (`ngrok`), `monitoring` (Prometheus, Alertmanager, Loki, Alloy, Grafana, exporters), `ops` (`ops-bot`), `all`; 8 образов Orbit в GHCR и роли БД `orbit_user`, `orbit_metrics`, `orbit_ops_bot` |
| `mobile` | **Работает сейчас (бета)** | Бета `v0.1.16-beta+17` на Flutter: светлая дизайн-система, Floating Dock, офлайн-расписание в Hive, выбор/синхронизация групп, Email OTP и Google OAuth (deep link `ru.mospolytech.orbit://auth/callback`), настройки с привязкой Google, AI-ассистент (SSE, история, удаление, оценки, повтор); CI и подпись APK |
| `users-service` | **В разработке** | Начальный каркас в `main`; прикладные профили ещё не подключены к стенду |
| `tasks-service` | **Учебный пример / запланировано** | Учебный эталон в `orbit-docs/examples/tasks-service`; отдельный сервис задач в проектировании |
| `navigation-service` | **Запланировано** | Целевая архитектура и паспорт сервиса; базовый разбор кодов аудиторий пока работает в `ai-service` |
| `notification-service` | **Запланировано** | Целевая архитектура под события RabbitMQ |
| `bot-service` | **Запланировано** | Пользовательский Telegram-бот для студентов (отдельно от админского `ops-bot`) |

## Что проверено на живом стенде

### Auth, Gateway и Schedule

- Регистрация и вход по Email OTP, ротация refresh-сессий с отзывом семейства при повторном использовании.
- Вход и безопасная привязка Google OAuth (с re-auth окном, binding cookie, `POST /api/v1/auth/oauth/exchange` и мобильным редиректом на `ru.mospolytech.orbit://auth/callback`).
- Хранение и обновление предпочтений групп (`primary_group`, `favorite_groups`) через `GET /api/v1/auth/me` и `PATCH /api/v1/auth/preferences`.
- Публичный поиск групп и расписание на неделю/период до 93 дней без JWT (`READY` / `PENDING_SYNC` / `is_stale`, Redis-кэш + PostgreSQL + Streams).
- Подробнее: [API расписания](../gateway/schedule-api.md).

### AI-контур и база знаний

- `ai-service` подключён к Gateway и работает через `POST /api/v1/ai/chat/stream`, `/api/v1/ai/chat/cancel`, список/историю/удаление диалогов (`/api/v1/ai/conversations*`) и оценки ответов (`PUT .../feedback`).
- `RouterLLMClient` маршрутизирует запросы по ролям (`router`, `casual`, `knowledge_rag`, `campus_navigation`, `general_study`) с цепочкой fallback между Groq, OpenRouter и Gemini.
- Поиск в `orbit_ai` гибридный (HNSW-векторы FastEmbed + полнотекстовый поиск PostgreSQL + опциональный cross-encoder reranker).
- Актуальный корпус Московского Политеха собирается отдельным пайплайном [`knowledge-pipeline`](../reference/services/knowledge-pipeline.md) и грузится через `--manifest`.
- Подробнее: [AI API](../gateway/ai-api.md), [загрузка базы знаний](../python-ai/knowledge-ingestion.md), [ограничения](../python-ai/current-limitations.md).

### Наблюдаемость и `ops-bot`

- В локальном `infra v0.4.0` подняты Prometheus (`:9090`), Alertmanager (`:9093`), Loki, Alloy, Grafana (`:3000` с 3 дашбордами в папке `Orbit Polytech`), `postgres-exporter`, `redis-exporter` и `ops-bot`.
- Подтверждено на стенде: все 7 таргетов Prometheus в статусе `UP`, Alloy забирает JSON-логи контейнеров в Loki, а запрос `/metrics` на публичном порту `:8080` возвращает `404` (метрики отдаются только на внутренних портах `9091–9094`).
- `ops-bot` принимает алерты Alertmanager и постит уведомления, неразобранные вопросы AI и негативные отзывы студентов в 5 Forum Topics (`#alerts`, `#llm-quotas`, `#untriaged`, `#feedback`, `#digest`).
- Подробнее: [Мониторинг и логи](../contracts-infra/observability.md), [паспорт ops-bot](../reference/services/ops-bot.md).

### Мобильное приложение (`mobile`)

- В мобильном клиенте (`v0.1.16-beta+17`) работают: расписание с офлайн-кэшем Hive, переключение недель, шторка деталей пары, гостевой режим, вход по Email OTP и Google (через браузер и `app_links`), синхронизация выбранных групп с разрешением конфликтов, экран настроек с привязкой/отвязкой Google и ИИ-ассистент со стримингом SSE, историей, удалением чатов, оценками и повтором ответа.
- Подробнее: [Обзор mobile](../mobile-flutter/overview.md), [паспорт mobile](../reference/services/mobile.md).

## Ближайшие шаги

1. Спроектировать и реализовать `tasks-service` (серверные личные и групповые задачи со сроками).
2. Довести до запуска `users-service` и перенести расширенные данные профиля/ролей из временных полей.
3. Постепенно расширять и проверять корпус базы знаний в `knowledge-pipeline` по отзывам и кейсам из `#untriaged` / `#feedback`.
4. Развивать дополнительные сценарии расписания (поиск по преподавателям и аудиториям, сессия).

!!! warning "Что всё ещё в планах"
    RabbitMQ и Jaeger для трейсинга в стек пока не добавлены. Вход через Telegram для студентов, `tasks-service`, `users-service`, `navigation-service`, `notification-service`, студентский `bot-service`, а также вкладки «Задачи» и «Заметки» в мобильном приложении остаются на следующих этапах.
