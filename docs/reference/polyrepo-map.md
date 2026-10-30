# Карта репозиториев orbit-polytech

Эта страница описывает одновременно существующие репозитории и целевую структуру организации.

## Текущее и целевое состояние

| Репозиторий | Назначение | Статус |
|---|---|---|
| [`contracts`](https://github.com/orbit-polytech/contracts) | Protobuf-контракты и сгенерированный Go/Python-код | **Работает сейчас** |
| [`infra`](https://github.com/orbit-polytech/infra) | Docker Compose, PostgreSQL, Redis, keygen и миграции | **Работает сейчас** — Auth, Schedule, Gateway, AI/pgvector и one-shot сервисы в `main` |
| [`gateway-service`](https://github.com/orbit-polytech/gateway-service) | Публичный HTTP API Gateway | **Работает сейчас** |
| [`auth-service`](https://github.com/orbit-polytech/auth-service) | Email OTP, JWT, refresh-сессии, Google OAuth и identities | **Работает сейчас** |
| [`users-service`](https://github.com/orbit-polytech/users-service) | Профили, группы и роли | **Запланировано** — репозиторий пуст |
| [`schedule-service`](https://github.com/orbit-polytech/schedule-service) | Регулярное расписание, PostgreSQL, Redis-кэш и распределённая синхронизация | **Работает сейчас** |
| `navigation-service` | Корпуса, аудитории и маршруты | **Запланировано** |
| `tasks-service` | Задачи и дедлайны | **Учебный пример / запланировано** |
| `notification-service` | Push/Telegram-уведомления и события | **Запланировано** |
| [`ai-service`](https://github.com/orbit-polytech/ai-service) | Groq, FastEmbed, RAG и pgvector | **Работает сейчас** — gRPC, JWT через Gateway, профили `ai`/`all` |
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

Текущие рабочие пути: Gateway → Auth → PostgreSQL/Redis и публичный Gateway → Schedule → Redis/PostgreSQL. Schedule обновляет расписание через Redis Streams и фоновую синхронизацию с `rasp.dmami.ru`, не обращаясь к Users/Auth за профилем посетителя.


Третий текущий путь — защищённый Gateway → AI → PostgreSQL/pgvector и Groq; эмбеддинги вычисляет FastEmbed. В организации проверено 10 существующих репозиториев, из них Users, Tasks и Mobile пока пустые. Число 13 выше относится к целевой карте, а не к количеству реализованных сервисов.
