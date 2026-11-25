# Соглашения об именовании в проекте

Единый стиль именования по всем технологиям проекта снижает когнитивную нагрузку и предотвращает хаос в кодовой базе.

---

## 1. Сводная таблица правил именования

| Сущность | Стиль | Пример | Запрещено |
|---|---|---|---|
| **Репозитории GitHub** | `kebab-case` в нижнем регистре | `auth-service`, `gateway-service`, `contracts` | `AuthService`, `orbit_auth` |
| **Ветки Git** | `<тип>/<номер>-<кратко-на-английском>` (как в [GitHub Flow](../quickstart/git-workflow.md)) | `feat/42-task-deadlines`, `fix/57-refresh-token-reuse`, `docs/12-local-setup` | `feature/TaskDeadlines`, `Danil/fix` |
| **Коммиты Git** | Conventional Commits в нижнем регистре | `feat(tasks): add deadline calculation` | `Fix bug`, `Update` |
| **Файлы исходного кода Go** | `snake_case.go` | `task_repository.go`, `create_task.go` | `TaskRepository.go` |
| **Пакеты Go** | Одно короткое слово в нижнем регистре | `package domain`, `package config` | `package task_domain`, `package myUtils` |
| **Файлы Python** | `snake_case.py` | `pgvector_store.py`, `gemini_client.py` | `PgVectorStore.py` |
| **Файлы Dart/Flutter** | `snake_case.dart` | `tasks_screen.dart`, `task_tile.dart` | `TasksScreen.dart` |
| **Файлы Protobuf** | `snake_case.proto` | `auth_service.proto`, `task_events.proto` | `AuthService.proto` |
| **Сообщения Protobuf** | `PascalCase` | `CreateTaskRequest`, `TaskResponse` | `create_task_request` |
| **Поля Protobuf** | `snake_case` | `user_id`, `idempotency_key`, `due_at` | `userId`, `DueAt` |
| **Перечисления Protobuf** | `UPPER_SNAKE_CASE` с префиксом | `TASK_STATUS_OPEN`, `TASK_STATUS_DONE` | `OPEN`, `STATUS_OPEN` |
| **Таблицы PostgreSQL** | `snake_case` во множественном числе | `accounts`, `refresh_sessions`, `profiles` | `Account`, `TBL_TASKS` |
| **Колонки PostgreSQL** | `snake_case` | `id`, `account_id`, `created_at` | `ID`, `createdAt` |
| **Ключи Redis** | Двоеточия `<сервис>:<сущность>:<id>` | `schedule:cache:v1:group:261-334:regular`, `auth:otp:data:<scope>`, `auth:otp:cooldown:<scope>`, `auth:oauth:state:<state>`, `orbit:ai:rl:win:<user_id>`, `orbit:support:rl:win:<key>` | `scheduleGroup261334` |
| **Очереди RabbitMQ *(план)*** | Точки `<сервис>.<сущность>.<событие>` | `tasks.task.created`, `schedule.lesson.moved` | `TasksQueue` |
| **Переменные окружения** | `UPPER_SNAKE_CASE` | `DATABASE_URL`, `GRPC_ADDR`, `APP_ENV` | `database_url`, `GrpcAddr` |
