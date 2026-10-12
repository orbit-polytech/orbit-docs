# tasks-service — Эталонный Go-микросервис Orbit AI

Эталонный микросервис управления задачами и дедлайнами студентов, демонстрирующий применение принципов **Clean Architecture (Чистая архитектура)** и **Domain-Driven Design (DDD)** на Go 1.24+.

Сервис служит кодовой основой для практических глав обучающего курса [Orbit AI Docs](https://orbit-polytech.github.io/orbit-docs/).

---

## 1. Архитектурные слои

- **`internal/domain/`** — Чистый домен без внешних зависимостей (только стандартная библиотека Go). Содержит агрегат `Task`, Value Objects (`TaskID`, `OwnerID`, `Title`, `Deadline`), перечисления `TaskPriority` и `TaskStatus`, бизнес-инварианты и проверку версий оптимистичной блокировки.
- **`internal/usecase/`** — Сценарии приложения (`CreateTask`, `GetTask`, `ListTasks`, `CompleteTask`, `RescheduleTask`). Порты разделены по принципу ISP (`TaskSaver`, `TaskFinder`, `TaskLister`, `EventPublisher`, `IdempotencyStore`).
- **`internal/infrastructure/inmemory/`** — Потокобезопасные in-memory реализации репозитория, кольцевого буфера событий и хранилища ключей идемпотентности.
- **`internal/infrastructure/idgen/`** — Генераторы идентификаторов (`UUIDv4`) без паник.
- **`internal/transport/taskgrpc/`** — gRPC обработчик и перехватчики (`AuthInterceptor` с пропуском healthcheck и валидацией `x-user-id`, `LoggingInterceptor`, `RecoveryInterceptor`).
- **`cmd/tasks-service/`** — Точка сборки (Composition Root) с безопасным Graceful Shutdown через `signal.NotifyContext`.

---

## 2. Команды для проверки и запуска

```bash
# 1. Запуск всех юнит-тестов
go test -v ./...

# 2. Запуск статического анализатора (с проверкой depguard)
golangci-lint run ./...

# 3. Проверка схем Protobuf через Buf CLI
buf lint
buf generate

# 4. Локальный запуск сервиса
go run ./cmd/tasks-service/main.go
```
