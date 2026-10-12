# Репозиторий infra: Docker Compose для разработки

Репозиторий [**`infra`**](https://github.com/orbit-polytech/infra) обеспечивает запуск полноценного микросервисного стенда Orbit AI на любом компьютере.

---

## 1. Структура `infra`

```text
infra/
├── compose.yaml                 # Основной файл запуска всех контейнеров
├── compose.debug.yaml           # Проброс gRPC-портов для отладки через grpcurl
├── .env.example                 # Шаблон локальных переменных окружения
├── secrets/                     # Ключи JWT (в .gitignore)
│   └── .gitkeep
├── postgres/
│   └── init/
│       └── 01-schemas.sql       # Инициализация отдельных схем и ролей сервисов
├── rabbitmq/
│   └── definitions.json         # Объявление exchanges и очередей
├── scripts/
│   └── gen-keys.sh              # Скрипт генерации ключей Ed25519
├── Makefile
└── README.md
```

---

## 2. Ключевые правила `compose.yaml`

1. **Контроль готовности (Healthchecks)**: каждая база данных и брокер снабжены проверкой здоровья `healthcheck`. Сервисы зависят от них через `depends_on: { condition: service_healthy }`, что исключает падения микросервисов при старте из-за не готового сокета PostgreSQL.
2. **Профили запуска (Compose Profiles)**:
   * `docker compose up -d postgres redis rabbitmq jaeger` — только базы данных (для разработчиков, запускающих сервисы на хосте);
   * `docker compose --profile core up -d` — базовые сервисы (`auth-service`, `tasks-service`, `gateway-service`);
   * `docker compose --profile ai up -d` — сервис нейросети `ai-service`;
   * `docker compose --profile all up -d` — запуск абсолютно всех микросервисов проекта.
3. **Безопасная привязка к `127.0.0.1`**: порты баз данных привязаны только к локальному loopback-интерфейсу (`127.0.0.1:5432`), защищая от доступа из публичных сетей Wi-Fi в общежитии или аудитории Политеха.

```yaml
name: orbit

services:
  postgres:
    image: pgvector/pgvector:pg16
    environment:
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: ${POSTGRES_DB}
    ports: ["127.0.0.1:5432:5432"]
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./postgres/init:/docker-entrypoint-initdb.d:ro
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U $${POSTGRES_USER} -d $${POSTGRES_DB}"]
      interval: 5s
      timeout: 3s
      retries: 10

  redis:
    image: redis:7-alpine
    ports: ["127.0.0.1:6379:6379"]
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      retries: 10

  rabbitmq:
    image: rabbitmq:4-management-alpine
    ports: ["127.0.0.1:5672:5672", "127.0.0.1:15672:15672"]
    volumes:
      - ./rabbitmq/definitions.json:/etc/rabbitmq/definitions.json:ro
    healthcheck:
      test: ["CMD", "rabbitmq-diagnostics", "-q", "ping"]
      interval: 10s
      retries: 10

  jaeger:
    image: jaegertracing/jaeger:2
    ports: ["127.0.0.1:16686:16686", "127.0.0.1:4317:4317"]

  gateway-service:
    restart: unless-stopped
    profiles: ["core", "all"]
    build: ../gateway-service
    env_file: .env
    ports: ["127.0.0.1:8080:8080"]
    volumes:
      - ./secrets/jwt_ed25519.pub:/run/secrets/jwt_ed25519.pub:ro
    depends_on:
      auth-service: { condition: service_started }
      tasks-service: { condition: service_started }
      redis: { condition: service_healthy }

volumes:
  pgdata:
```

---

## 3. Изоляция схем базы данных

В PostgreSQL используется подход **одна инстанция БД — изолированные схемы**:
* У каждого сервиса своя схема: `auth`, `tasks`, `schedule`, `navigation`, `ai`.
* У каждого сервиса своя отдельная учётная запись в СУБД с правами только на свою схему (`DATABASE_URL=postgres://tasks:pass@postgres:5432/orbit?search_path=tasks`).

Это физически запрещает одному сервису прочитать конфиденциальные таблицы другого сервиса.
