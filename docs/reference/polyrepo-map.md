# Карта репозиториев orbit-polytech

На этой странице собраны все репозитории организации `orbit-polytech` — и те, что уже работают, и запланированные на будущее.

## Все репозитории (15)

| Репозиторий | Назначение | Статус |
|---|---|---|
| [`contracts`](https://github.com/orbit-polytech/contracts) | Protobuf-схемы и сгенерированный код Go / Python (`v0.5.9`) | **Работает сейчас** |
| [`infra`](https://github.com/orbit-polytech/infra) | Docker Compose (`apps`, `ai`, `tunnel`, `monitoring`, `ops`, `all`), PostgreSQL, Redis, Prometheus/Loki/Grafana (`v0.4.0`) | **Работает сейчас** |
| [`gateway-service`](https://github.com/orbit-polytech/gateway-service) | Публичный HTTP API Gateway (Auth, Schedule, AI, HTTP RED-метрики `:9094`) | **Работает сейчас** |
| [`auth-service`](https://github.com/orbit-polytech/auth-service) | Email OTP, JWT Ed25519, refresh-сессии, Google OAuth, identities, настройки групп, `/metrics :9091` | **Работает сейчас** |
| [`schedule-service`](https://github.com/orbit-polytech/schedule-service) | Расписание занятий, PostgreSQL, Redis-кэш + Streams + синхронизация с `rasp.dmami.ru`, `/metrics :9092` | **Работает сейчас** |
| [`ai-service`](https://github.com/orbit-polytech/ai-service) | Мультипровайдерный LLM (Groq/OpenRouter/Gemini), FastEmbed, гибридный RAG в pgvector, история, оценки, `/metrics :9093` | **Работает сейчас** |
| [`knowledge-pipeline`](https://github.com/orbit-polytech/knowledge-pipeline) | ETL-конвейер (`orbit-etl`): сбор `mospolytech.ru`, PDF и `curated_manual/` в `output/manifest.json` | **Работает сейчас** |
| `ops-bot` *(приватный)* | Операционный Telegram-бот для алертов Alertmanager, триажа вопросов и дайджестов (`v0.1.0`) | **Работает сейчас** |
| [`mobile`](https://github.com/orbit-polytech/mobile) | Мобильное приложение на Flutter (`v0.1.16-beta+17`) | **Работает сейчас (бета)** |
| [`orbit-docs`](https://github.com/orbit-polytech/orbit-docs) | Инженерная документация на MkDocs и учебный `examples/tasks-service` | **Работает сейчас** |
| [`users-service`](https://github.com/orbit-polytech/users-service) | Профили студентов, роли и привязка групп | **В разработке** — базовый каркас в `main` |
| `tasks-service` | Личные и групповые задачи, дедлайны | **Учебный пример / запланировано** |
| `navigation-service` | Корпуса, аудитории и маршруты | **Запланировано** |
| `notification-service` | Push/Telegram-уведомления по событиям | **Запланировано** |
| `bot-service` | Пользовательский Telegram-бот для студентов через Gateway | **Запланировано** |

## Схема взаимодействия

```text
   [ mobile (Flutter бета) ]      [ bot-service (план) ]
               │                            │
               └─────── HTTP/REST/SSE ──────┘
                             │
                    (ngrok при --profile tunnel)
                             │
                             ▼
                     [ gateway-service ] (:8080)
                             │ gRPC
         ┌───────────────────┼───────────────────┬────────────┐
         ▼                   ▼                   ▼            ▼
  [ auth-service ]   [ schedule-service ]  [ ai-service ]  [ tasks/users (план) ]
         │                   │                   │            │
   [Postgres/Redis]    [Postgres/Redis]    [pgvector + LLM]  [RabbitMQ (план)]
                                                 ▲
                                                 │ --manifest
                                        [knowledge-pipeline]

  Контур наблюдаемости (--profile monitoring / ops):
  [auth/schedule/ai/gateway :9091..9094] + [exporters] ──▶ Prometheus ──▶ Alertmanager ──▶ ops-bot
  [Docker logs (JSON)] ──▶ Alloy ──▶ Loki ──▶ Grafana ◀── Prometheus
```

Сейчас вживую работают:
1. **Клиентский и шлюзовой контур:** `mobile` (или HTTP/Swagger) → `gateway-service` → `auth-service` (с Redis и `orbit_auth`), `schedule-service` (с Redis/PostgreSQL и фоновой синхронизацией `rasp.dmami.ru`) и `ai-service` (с pgvector и мультипровайдерным LLM-роутером).
2. **Конвейер знаний:** `knowledge-pipeline` собирает и чистит документы вуза в Knowledge Bundle (`manifest.json`), который индексируется в `ai-service`.
3. **Эксплуатация и мониторинг:** Prometheus собирает метрики с внутренних портов сервисов и экспортеров БД, Alloy складывает JSON-логи в Loki, Grafana рисует дашборды, а Alertmanager и `ops-bot` доносят инциденты и проблемные AI-ответы в Telegram.

Актуальный срез по каждому компоненту — на странице [«Текущее состояние»](../project/current-state.md).
