# Соглашения об именовании и кодстайл

Единообразие именования во всех сервисах Orbit AI позволяет любому разработчику команды мгновенно ориентироваться в чужих репозиториях.

---

## 1. Git: Ветки и коммиты

### Именование веток (Branch Naming)
Формат: `<type>/<short-description>` (строго латиница в нижнем регистре через дефис).

- `feature/add-complete-task` — новая функциональность;
- `fix/deadline-utc-timezone` — исправление бага;
- `refactor/inmemory-ring-buffer` — рефакторинг без изменения внешнего поведения;
- `docs/howto-vscode-debug` — обновление документации;
- `chore/bump-buf-version` — рутинные задачи, обновление зависимостей.

### Сообщения коммитов (Conventional Commits)
Формат: `<type>(<scope>): <краткое описание в повелительном наклонении>`

Примеры:
- `feat(tasks): implement CompleteTask gRPC handler`
- `fix(auth): validate x-user-id length and control characters`
- `test(config): add unit tests for TASK_MAX_LIMIT bounds`
- `docs(adr): add ADR-006 regarding Redis and RabbitMQ roles`

---

## 2. Архитектура и кодовая база

### Именование Go пакетов и файлов
- Пакеты — короткие существительные в нижнем регистре без подчеркиваний и `mixedCaps`: `taskgrpc`, `inmemory`, `domain`, `usecase`.
- Файлы — `snake_case`: `create_task.go`, `task_repository.go`, `handler_test.go`.

### Protobuf и gRPC
- **Пакеты:** `orbit.<service>.v<version>` (например, `orbit.tasks.v1`).
- **Сервисы:** `PascalCase` с суффиксом `Service`: `TaskService`, `ScheduleService`.
- **Методы (RPC):** `PascalCase`: `CreateTask`, `ListTasks`.
- **Запросы / Ответы:** `<Method>Request` и `<Method>Response`.
- **Поля сообщений:** `snake_case`: `task_id`, `idempotency_key`, `created_at`.
- **Перечисления (Enums):** `PascalCase`, первое значение строго `0 = <ENUM_NAME>_UNSPECIFIED`:
  ```protobuf
  enum TaskStatus {
    TASK_STATUS_UNSPECIFIED = 0;
    TASK_STATUS_OPEN = 1;
    TASK_STATUS_COMPLETED = 2;
  }
  ```

---

## 3. Базы данных и очереди

### PostgreSQL
- Имена таблиц: во множественном числе, `snake_case`: `tasks`, `outbox_events`, `user_profiles`.
- Имена колонок: `snake_case`: `owner_id`, `created_at`, `version`.
- Первичные ключи: строго `id UUID PRIMARY KEY`.
- Индексы: `idx_<table>_<columns>` (например, `idx_tasks_owner_status`).
- Ограничения (Constraints): `chk_<table>_<field>` (например, `chk_tasks_status`).

### Redis
Формат ключей: `<service>:<entity>:<id>` (строго через двоеточие):
- `schedule:group:261-334`
- `auth:blacklist:018f2345-6789`
- `gateway:ratelimit:ip:192.168.1.5`

### RabbitMQ
- Exchanges (Topic): `<domain>.events` (например, `tasks.events`, `schedule.events`).
- Очереди: `<consumer_service>.<domain>_events` (например, `notification.task_events`).
- Routing Keys: `<entity>.<action>` в настоящем или прошедшем времени (например, `task.created`, `task.completed`, `schedule.updated`).

---

## 4. Переменные окружения (.env)

Все переменные окружения пишутся строго заглавными буквами в `SCREAMING_SNAKE_CASE`:
- Общие настройки: `APP_ENV`, `LOG_LEVEL`, `GRPC_PORT`, `HTTP_PORT`.
- Настройки БД: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`.
- Брокер и кэш: `REDIS_ADDR`, `REDIS_PASSWORD`, `RABBITMQ_URL`.
