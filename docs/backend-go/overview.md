# Go-бэкенд: Архитектура и структура микросервисов

Все активные Go-сервисы команды Orbit (`auth-service`, `users-service`, `schedule-service`, `gateway-service`) строятся по единым принципам **Clean Architecture** (порты и адаптеры). Как устроен каждый слой подробно разобрано в [Чистой архитектуре и DDD](clean-architecture.md).

---

## 1. Официальный стандарт размещения кода Go

Архитектура сервисов опирается на официальное руководство команды Go [«Organizing a Go module»](https://go.dev/doc/modules/layout).

!!! warning "Без `golang-standards/project-layout`"
    Мы **не используем** неофициальный шаблон `golang-standards/project-layout` с избыточными каталогами (`pkg/`, `scripts/` и т.д.). Структура модулей остаётся компактной, а изоляцию слоёв обеспечивает `internal/`.

---

## 2. Каноническая структура действующих Go-сервисов

Ниже — фактическая раскладка каталогов на базе живых сервисов (`auth-service`, `users-service`, `schedule-service`, `gateway-service`):

```text
<service-name>/
├── cmd/
│   └── <service>/
│       └── main.go                  # Минимальная точка входа (запуск internal/app)
├── internal/
│   ├── app/                         # Composition Root: сборка зависимостей, жизненный цикл, метрики
│   ├── config/                      # Типизированный Config, загрузка и валидация env
│   ├── domain/                      # Чистое бизнес-ядро: сущности, Value Objects, доменные ошибки
│   ├── usecase/                     # Бизнес-сценарии (use cases) и интерфейсы портов (ports.go)
│   ├── infrastructure/              # Реализации портов:
│   │   ├── postgres/                # SQL-репозитории на pgx/v5
│   │   ├── redisstore/              # Кэш, сессии, rate-limit на go-redis (при наличии)
│   │   └── ...                      # Внешние клиенты (email, oauth, gRPC-клиенты в gateway)
│   └── transport/                   # Входящий транспорт:
│       ├── grpc/                    # gRPC-сервер, хендлеры, маппинг ошибок (в доменных сервисах)
│       ├── http/                    # HTTP-роутер, middleware, REST/SSE хендлеры (в gateway-service)
│       └── metrics/                 # Prometheus /metrics endpoint (при выделении в transport)
├── migrations/
│   ├── 00001_init_schema.sql        # Единый файл Goose (-- +goose Up / -- +goose Down)
│   └── 00002_...sql
├── .env.example
├── .gitignore
├── .golangci.yml                    # Конфигурация golangci-lint v2 (version: "2")
├── Dockerfile
├── Dockerfile.migrate               # Образ миграций с Goose (в сервисах с БД)
├── Taskfile.yml                     # Единые команды: task run / test / lint / build / migrate-*
├── go.mod
├── go.sum
└── README.md
```

---

## 3. Зоны ответственности директорий

* **`cmd/<service>/main.go`**: тонкая точка запуска, которая передаёт управление в `internal/app`.
* **`internal/app/` (Composition Root)**: чтение конфигурации (`internal/config`), инициализация подключений к PostgreSQL/Redis, сборка репозиториев, юзкейсов, gRPC/HTTP-сервера и сервера метрик, регистрация healthcheck и корректное завершение (`SIGINT`, `SIGTERM`).
* **`internal/config/`**: структура `Config`, чтение переменных окружения и их валидация.
* **`internal/domain/`**: чистое бизнес-ядро. Содержит сущности, Value Objects с проверкой инвариантов и доменные ошибки (`errors.go`). Без зависимостей от БД, сетей и транспорта.
* **`internal/usecase/`**: сценарии использования. Сервис обращается к внешнему миру только через интерфейсы портов (`ports.go`), не зная деталей реализации PostgreSQL или Redis.
* **`internal/infrastructure/<технология>/`**: вторичные адаптеры (Driven Adapters): `postgres`, `redisstore`, `email`, `oauth`, а в `gateway-service` — gRPC-клиенты (`authclient`, `usersclient`, `scheduleclient`, `aiclient`) и `ratelimit`.
* **`internal/transport/`**: первичный адаптер (Driving Adapter) — `grpc/` (в `auth-service`, `users-service`, `schedule-service`) или `http/` (в `gateway-service`). Принимает запросы, валидирует вход, вызывает `usecase` (или gRPC-клиент в gateway) и маппит доменные ошибки в коды gRPC / HTTP.

---

## 4. Направление зависимостей (Dependency Rule)

```text
transport/* ──> usecase ──> domain
                   ^
infrastructure/* ──┘ (реализуют интерфейсы из usecase/ports.go)

internal/app знает обо всех слоях и связывает их в Composition Root
```

Стрелки зависимостей идут **строго внутрь**:
1. `domain` не зависит ни от одного внутреннего инфраструктурного или транспортного слоя.
2. `usecase` зависит от `domain` и абстрактных интерфейсов портов, но не импортирует `pgx`, `go-redis` или `transport/*`.
3. `infrastructure` реализует порты `usecase` и оперирует моделями `domain`.
4. `transport` вызывает `usecase` и не содержит доменной логики.

Изоляция слоёв проверяется на code review, а базовый статический анализ и форматирование в CI обеспечивает **`golangci-lint` v2** (`.golangci.yml`).
