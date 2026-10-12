# Карта репозиториев организации orbit-polytech

Вся экосистема Orbit AI размещается в единой организации на GitHub: [**github.com/orbit-polytech**](https://github.com/orbit-polytech).

---

## 1. Репозитории микросервисов и модулей

Префикс `orbit-` в названиях репозиториев **не используется**, так как они уже живут внутри пространства имён организации `orbit-polytech`:

| Репозиторий | Назначение | Технологический стек |
|---|---|---|
| [`contracts`](https://github.com/orbit-polytech/contracts) | Централизованные Protobuf v3 контракты и сгенерированный код | Buf CLI, Protobuf v3, Go, Python, Dart |
| [`infra`](https://github.com/orbit-polytech/infra) | Docker Compose для локальной разработки, схемы БД | Docker Compose, PostgreSQL 16 + pgvector, Redis 7, RabbitMQ, Jaeger |
| [`gateway-service`](https://github.com/orbit-polytech/gateway-service) | Единый публичный HTTP API шлюз для внешних клиентов | Go 1.25, HTTP/REST, gRPC клиенты, Ed25519 JWT Verifier |
| [`auth-service`](https://github.com/orbit-polytech/auth-service) | Аутентификация через Telegram, сессии, выпуск JWT | Go 1.25, gRPC, PostgreSQL, Redis |
| [`users-service`](https://github.com/orbit-polytech/users-service) | Профили студентов, группы, права доступа и роли | Go 1.25, gRPC, PostgreSQL |
| [`schedule-service`](https://github.com/orbit-polytech/schedule-service) | Расписание пар, аудиторий, преподавателей, парсинг | Go 1.25, gRPC, Redis Cache, PostgreSQL |
| [`navigation-service`](https://github.com/orbit-polytech/navigation-service) | Корпуса Политеха (БС, АВ, ПР), аудитории, построение маршрутов | Go 1.25, gRPC, PostgreSQL |
| [`tasks-service`](https://github.com/orbit-polytech/tasks-service) | Задачи, дедлайны лабораторных (эталонный Go-сервис) | Go 1.25, gRPC, Clean Architecture, PostgreSQL |
| [`notification-service`](https://github.com/orbit-polytech/notification-service) | Очередь отправки Push и Telegram-уведомлений | Go 1.25, gRPC, RabbitMQ Consumer |
| [`ai-service`](https://github.com/orbit-polytech/ai-service) | RAG поиск по базе знаний, Gemini API, векторный поиск | Python 3.12, FastAPI, gRPC, pgvector, uv |
| [`bot-service`](https://github.com/orbit-polytech/bot-service) | Официальный Telegram-бот для студентов Политеха | Python 3.12, aiogram 3 |
| [`mobile`](https://github.com/orbit-polytech/mobile) | Кроссплатформенное мобильное приложение | Flutter 3.x, Dart, MVVM / BLoC |
| [`orbit-docs`](https://github.com/orbit-polytech/orbit-docs) | Центральная инженерная база знаний и документация | MkDocs Material, Markdown |

---

## 2. Схема взаимодействия репозиториев

```text
[ mobile (Flutter) ]       [ bot-service (Python) ]
         │                            │
         └─────────────┬──────────────┘
                       │ HTTP/JSON (:8080)
                       ▼
             [ gateway-service ]
                       │
       ┌───────────────┼───────────────┬───────────────┐
       │ gRPC (:50051) │ gRPC (:50053) │ gRPC (:50055) │ gRPC (:50057)
       ▼               ▼               ▼               ▼
 [ auth-service ] [ schedule-svc ] [ tasks-service ] [ ai-service ]
       │                               │               │
       │                               ▼               ▼
       │                          [ RabbitMQ ]   [ pgvector ]
       │                               │
       │                               ▼
       └────────────────────> [ notification-service ]
```

Все микросервисы зависят от единого репозитория контрактов `contracts` и запускаются через репозиторий `infra`.
