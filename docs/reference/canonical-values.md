# Канонические значения проекта Orbit AI

Если где-то в старых обсуждениях или черновиках встречаются другие значения, **эта таблица является единственным источником правды**.

---

## 1. Базовые версии и технологии

| Параметр | Каноническое значение | Примечание |
|---|---|---|
| **Язык Go** | **`1.25`** | Минимальная версия компилятора Go в `go.mod` |
| **Язык Python** | **`3.12`** | Python в `pyproject.toml` |
| **Размерность эмбеддинга** | **`vector(768)`** | Размер вектора Google Gemini `text-embedding-004` в pgvector |
| **База данных** | **`PostgreSQL 16`** | Образ `pgvector/pgvector:pg16` |
| **Кэш данных** | **`Redis 7`** | Alpine |
| **Брокер очередей** | **`RabbitMQ 4`** | `rabbitmq:4-management-alpine` |
| **Трейсинг** | **`Jaeger 2`** | `jaegertracing/jaeger:2` |

> [!NOTE]
> Названия моделей Gemini **не зашиваются в коде**. Они читаются из переменных окружения (`AI_CHAT_MODEL`, `AI_EMBEDDING_MODEL`). Актуальные идентификаторы моделей проверяются на [ai.google.dev/gemini-api/docs/models](https://ai.google.dev/gemini-api/docs/models).

---

## 2. Канонические сетевые порты

| Компонент / Сервис | Порт | Протокол | Доступность снаружи |
|---|---|---|---|
| **`gateway-service`** | **`8080`** | HTTP / REST | Да (`127.0.0.1:8080`) |
| **`auth-service`** | **`50051`** | gRPC | Только во внутренней сети (`compose.debug.yaml`) |
| **`users-service`** | **`50052`** | gRPC | Только во внутренней сети |
| **`schedule-service`** | **`50053`** | gRPC | Только во внутренней сети |
| **`navigation-service`** | **`50054`** | gRPC | Только во внутренней сети |
| **`tasks-service`** | **`50055`** | gRPC | Только во внутренней сети |
| **`notification-service`** | **`50056`** | gRPC | Только во внутренней сети |
| **`ai-service`** | **`50057`** | gRPC | Только во внутренней сети |
| **`bot-service`** | — | Long Polling / Webhook | Не слушает входящих портов в dev-режиме |
| **`PostgreSQL`** | **`5432`** | TCP | Да (`127.0.0.1:5432`) |
| **`Redis`** | **`6379`** | TCP | Да (`127.0.0.1:6379`) |
| **`RabbitMQ AMQP`** | **`5672`** | AMQP | Да (`127.0.0.1:5672`) |
| **`RabbitMQ UI`** | **`15672`** | HTTP | Да (`127.0.0.1:15672`) |
| **`Jaeger UI`** | **`16686`** | HTTP | Да (`127.0.0.1:16686`) |
| **`Jaeger OTLP`** | **`4317`** | gRPC | Да (`127.0.0.1:4317`) |

---

## 3. Таблица переменных окружения

### Общие переменные микросервисов:
* `APP_ENV`: режим окружения (`local`, `staging`, `production`). От него зависят gRPC reflection и dev-логин.
* `LOG_LEVEL`: уровень детализации логов (`debug`, `info`, `warn`, `error`).
* `DATABASE_URL`: строка подключения с изоляцией схемы (`postgres://user:pass@host:5432/orbit?search_path=<service>`).
* `REDIS_ADDR`: адрес инстанции Redis (`localhost:6379` или `redis:6379`).
* `RABBITMQ_URL`: строка подключения AMQP (`amqp://guest:guest@localhost:5672/`).
* `OTEL_EXPORTER_OTLP_ENDPOINT`: адрес агента Jaeger (`http://localhost:4317`).
* `SHUTDOWN_TIMEOUT`: таймаут Graceful Shutdown (`5s`–`10s`).
