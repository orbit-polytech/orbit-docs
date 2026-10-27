# Соглашения об именовании в проекте

Единый стиль именования по всем технологиям проекта снижает когнитивную нагрузку и предотвращает хаос в кодовой базе.

---

## 1. Сводная таблица правил именования

| Сущность | Стиль | Пример | Запрещено |
|---|---|---|---|
| **Репозитории GitHub** | `kebab-case` в нижнем регистре | `auth-service`, `gateway-service`, `contracts` | `AuthService`, `orbit_auth` |
| **Ветки Git** | `<тип>/<номер>-<kebab-case>` | `feat/42-task-deadlines`, `fix/10-jwt-exp` | `feature/TaskDeadlines`, `Danil/fix` |
| **Коммиты Git** | Conventional Commits в нижнем регистре | `feat(tasks): add deadline calculation` | `Fix bug`, `Update` |
| **Файлы исходного кода Go** | `snake_case.go` | `task_repository.go`, `create_task.go` | `TaskRepository.go` |
| **Пакеты Go** | Одно короткое слово в нижнем регистре | `package domain`, `package config` | `package task_domain`, `package myUtils` |
| **Файлы Python** | `snake_case.py` | `pgvector_store.py`, `gemini_client.py` | `PgVectorStore.py` |
| **Файлы Dart/Flutter** | `snake_case.dart` | `tasks_screen.dart`, `task_tile.dart` | `TasksScreen.dart` |
| **Файлы Protobuf** | `snake_case.proto` | `auth_service.proto`, `task_events.proto` | `AuthService.proto` |
| **Сообщения Protobuf** | `PascalCase` | `CreateTaskRequest`, `TaskResponse` | `create_task_request` |
| **Поля Protobuf** | `snake_case` | `user_id`, `idempotency_key`, `due_at` | `userId`, `DueAt` |
| **Перечисления Protobuf** | `UPPER_SNAKE_CASE` с префиксом | `TASK_STATUS_OPEN`, `TASK_STATUS_DONE` | `OPEN`, `STATUS_OPEN` |
| **Таблицы PostgreSQL** | `snake_case` во множественном числе | `tasks`, `sessions`, `refresh_tokens` | `Task`, `TBL_TASKS` |
| **Колонки PostgreSQL** | `snake_case` | `id`, `owner_id`, `created_at` | `ID`, `createdAt` |
| **Ключи Redis** | Двоеточия `<сервис>:<сущность>:<id>` | `schedule:cache:v1:group:261-334:regular`, `auth:blacklist:<jti>`| `scheduleGroup261334` |
| **Очереди RabbitMQ** | Точки `<сервис>.<сущность>.<событие>` | `tasks.task.created`, `schedule.lesson.moved` | `TasksQueue` |
| **Переменные окружения** | `UPPER_SNAKE_CASE` | `DATABASE_URL`, `GRPC_ADDR`, `APP_ENV` | `database_url`, `GrpcAddr` |
