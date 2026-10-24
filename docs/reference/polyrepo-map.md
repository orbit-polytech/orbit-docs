# Карта репозиториев orbit-polytech

Эта страница описывает одновременно существующие репозитории и целевую структуру организации.

## Текущее и целевое состояние

| Репозиторий | Назначение | Статус |
|---|---|---|
| [`contracts`](https://github.com/orbit-polytech/contracts) | Protobuf-контракты и сгенерированный Go-код | **Работает сейчас** |
| [`infra`](https://github.com/orbit-polytech/infra) | Docker Compose, PostgreSQL, Redis, keygen и миграции | **Работает сейчас** — Auth, Schedule, Gateway и one-shot сервисы в `main` |
| [`gateway-service`](https://github.com/orbit-polytech/gateway-service) | Публичный HTTP API Gateway | **Работает сейчас** |
| [`auth-service`](https://github.com/orbit-polytech/auth-service) | Email OTP, JWT, refresh-сессии, Google OAuth и identities | **Работает сейчас** |
| [`users-service`](https://github.com/orbit-polytech/users-service) | Профили, группы и роли | **Запланировано** — репозиторий пуст |
| [`schedule-service`](https://github.com/orbit-polytech/schedule-service) | Регулярное расписание, фоновая синхронизация и PostgreSQL-снимки | **Работает сейчас** |
| `navigation-service` | Корпуса, аудитории и маршруты | **Запланировано** |
| `tasks-service` | Задачи и дедлайны | **Учебный пример / запланировано** |
| `notification-service` | Push/Telegram-уведомления и события | **Запланировано** |
| `ai-service` | RAG, Gemini и pgvector | **Запланировано** |
| `bot-service` | Telegram-клиент системы через Gateway | **Запланировано** |
| `mobile` | Flutter-клиент | **Запланировано** |
| [`orbit-docs`](https://github.com/orbit-polytech/orbit-docs) | Инженерная база знаний и учебный пример | **Работает сейчас** |

Целевая экосистема содержит 13 репозиториев. Репозиторий считается реализованным только после появления рабочего кода, CI и соответствующей отметки на странице [текущего состояния](../project/current-state.md).

## Целевая схема взаимодействия

```text
[ mobile ]             [ bot-service ]
     │                         │
     └──────── HTTP/JSON ──────┘
                 │
                 ▼
         [ gateway-service ]
                 │ gRPC
     ┌───────────┼───────────┬────────────┐
     ▼           ▼           ▼            ▼
 [ auth ]    [ schedule ] [ tasks ]    [ ai ]
     │                       │            │
 [Postgres/Redis]       [ RabbitMQ ]  [pgvector]
                            │
                            ▼
                    [ notification ]
```

Текущие рабочие пути: Gateway → Auth → PostgreSQL/Redis и публичный Gateway → Schedule → PostgreSQL. Schedule обновляет снимки через фоновую синхронизацию с `rasp.dmami.ru`, не обращаясь к Users/Auth за профилем посетителя.
