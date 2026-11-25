# Карта репозиториев orbit-polytech

На этой странице собраны все компоненты экосистемы `orbit-polytech`: **12 существующих репозиториев на GitHub** (ключевые рабочие репозитории перечислены в первой таблице) и **4 запланированных сервиса**, для которых отдельные репозитории ещё не созданы.

## Существующие репозитории (12 на GitHub)

| Репозиторий | Назначение | Статус |
|---|---|---|
| [`contracts`](https://github.com/orbit-polytech/contracts) | Protobuf-схемы и сгенерированный код Go / Python (`v0.5.12`) | **Работает сейчас** |
| [`infra`](https://github.com/orbit-polytech/infra) | Docker Compose (`apps`, `ai`, `tunnel`, `monitoring`, `ops`, `all`), PostgreSQL, Redis, Nginx Web, Prometheus/Loki/Grafana (`v0.6.0`) | **Работает сейчас** |
| [`gateway-service`](https://github.com/orbit-polytech/gateway-service) | Публичный HTTP API Gateway (Auth, Users, Schedule, AI, Support, rate limiting в Redis, HTTP RED-метрики `:9094`) | **Работает сейчас** |
| [`auth-service`](https://github.com/orbit-polytech/auth-service) | Email OTP, смена email, JWT Ed25519, refresh-сессии, Google OAuth, identities, настройки групп, `/metrics :9091` | **Работает сейчас** |
| [`users-service`](https://github.com/orbit-polytech/users-service) | Профили пользователей (`display_name`, `@username`, группа, курс, аватар, роль), PostgreSQL (`orbit_users`), `/metrics :9096` | **Работает сейчас** |
| [`schedule-service`](https://github.com/orbit-polytech/schedule-service) | Расписание занятий, PostgreSQL, Redis-кэш + Streams + синхронизация с `rasp.dmami.ru`, `/metrics :9092` | **Работает сейчас** |
| [`ai-service`](https://github.com/orbit-polytech/ai-service) | Мультипровайдерный LLM (Groq/OpenRouter/Gemini), FastEmbed, гибридный RAG в pgvector, история, оценки, `/metrics :9093` | **Работает сейчас** |
| [`knowledge-pipeline`](https://github.com/orbit-polytech/knowledge-pipeline) | ETL-конвейер (`orbit-etl`): сбор `mospolytech.ru`, PDF и `curated_manual/` в `output/manifest.json` | **Работает сейчас** |
| `ops-bot` *(приватный)* | Операционный Telegram-бот для алертов Alertmanager, обращений `/issues/webhook`, триажа и дайджестов (`v0.1.0`) | **Работает сейчас** |
| [`mobile`](https://github.com/orbit-polytech/mobile) | Клиент на Flutter (`v0.1.27-beta+28`): Android APK и Web/PWA-контейнер `web` (`:8088` → `:80`) | **Работает сейчас (бета)** |
| [`orbit-docs`](https://github.com/orbit-polytech/orbit-docs) | Инженерная документация на MkDocs и учебный `examples/tasks-service` | **Работает сейчас** |

*(Ещё один репозиторий в организации на GitHub — служебный профильный репозиторий организации; отдельные репозитории для будущих сервисов ниже пока не созданы.)*

## Запланированные сервисы (4, репозитории не созданы)

| Сервис | Назначение | Статус |
|---|---|---|
| `tasks-service` | Личные и групповые задачи, дедлайны | **Учебный пример / запланировано** |
| `navigation-service` | Корпуса, аудитории и маршруты | **Запланировано** |
| `notification-service` | Push/Telegram-уведомления по событиям | **Запланировано** |
| `bot-service` | Пользовательский Telegram-бот для студентов через Gateway | **Запланировано** |

## Схема взаимодействия

```text
   [ mobile (Android бета) ]   [ web (Flutter Web/PWA :8088 -> :80) ]   [ bot-service (план) ]
               │                                 ▲                                │
               │                      (ngrok при --profile tunnel)                │
               │                                 │                                │
               │                        Nginx proxy (/api/, /swagger/)            │
               └────────────── HTTP/REST/SSE ────┴────────────────────────────────┘
                                                 │
                                                 ▼
                                         [ gateway-service ] (:8080)
                                                 │
       ┌─────────────────┬───────────────────┬───┴──────────┬──────────────────┐
       ▼ gRPC            ▼ gRPC              ▼ gRPC         ▼ gRPC             ▼ HTTP POST
[ auth-service ]  [ users-service ]  [ schedule-service ] [ ai-service ]  [ ops-bot :8081 ]
  (:50051)          (:50052)           (:50054)             (:50057)      (/issues/webhook)
       │                 │                   │                  │
 [Postgres/Redis]   [Postgres]       [Postgres/Redis]    [pgvector + LLM]
                                                                ▲
                                                                │ --manifest
                                                       [knowledge-pipeline]

  Будущий контур (план): [ tasks / navigation / notification ] + [ RabbitMQ ]

  Контур наблюдаемости (--profile monitoring / ops):
  [auth/schedule/ai/gateway/users :9091..9094, :9096] + [exporters] ──▶ Prometheus ──▶ Alertmanager ──▶ ops-bot (/alerts/webhook)
  [Docker logs (JSON)] ──▶ Alloy ──▶ Loki ──▶ Grafana ◀── Prometheus
```

Сейчас вживую работают:
1. **Клиентский и шлюзовой контур:** `mobile` (Android APK) и `web` (Flutter Web/PWA через Nginx-прокси на `:8088`, куда также смотрит `ngrok`) → `gateway-service` (`:8080`, с rate limiting в Redis) → `auth-service` (`orbit_auth`), `users-service` (`orbit_users`), `schedule-service` (`orbit_schedule` и синк `rasp.dmami.ru`), `ai-service` (`orbit_ai` + pgvector + мультипровайдерный LLM-роутер) и `ops-bot` (`/issues/webhook`).
2. **Конвейер знаний:** `knowledge-pipeline` собирает и чистит документы вуза в Knowledge Bundle (`manifest.json`), который индексируется в `ai-service`.
3. **Эксплуатация и мониторинг:** Prometheus собирает метрики с 8 таргетов (включая `users-service:9096` и экспортеры БД), Alloy складывает JSON-логи в Loki, Grafana рисует дашборды, а Alertmanager и `ops-bot` доставляют алерты, проблемные AI-ответы, дизлайки и обращения студентов из приложения в Telegram.

Актуальный срез по каждому компоненту — на странице [«Текущее состояние»](../project/current-state.md).
