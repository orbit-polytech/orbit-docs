# users-service (Профили пользователей)

**Репозиторий:** [`orbit-polytech/users-service`](https://github.com/orbit-polytech/users-service)

**Статус:** работает сейчас в `main` (`contracts@v0.5.12`), входит в профиль Compose `apps` / `all`.

---

## 1. Назначение
Хранит и управляет профилями пользователей Orbit AI: отображаемое имя (`display_name`), уникальный тег (`@username`), учебная группа (`academic_group`, например `261-334`), курс (`1–6`), аватар (`preset:<slug>` или HTTPS URL) и системная роль (`unspecified`, `student`, `teacher`, `admin`).

На HTTP-периметре в `gateway-service` подключены маршруты `GET /api/v1/users/profile` и `PUT /api/v1/users/profile` (требуют Bearer JWT). Метод `GetBatchProfiles` является внутренним gRPC-методом и наружу по HTTP не публикуется.

---

## 2. Технический стек
- **Язык:** Go `1.26.10` (Clean Architecture)
- **База данных:** PostgreSQL 16 (`orbit_users`), драйвер `pgx/v5` (`pgxpool`)
- **Миграции:** Goose CLI (`migrations/`, one-shot контейнер `users-migrate`):
  - `00001_create_profiles.sql` — таблица `profiles` (`account_id`, `avatar_url`, `academic_group`, `course`, `role`, устаревшие `first_name`/`last_name`/`middle_name`);
  - `00002_add_display_name_and_username.sql` — поля `display_name` (обязательное, до 64 символов) и `username` (`^[a-z0-9_]{3,32}$`, уникальный регистронезависимый индекс `idx_profiles_username_lower`, список зарезервированных имён).
- **Транспорт и контракт:** gRPC `orbit.users.v1.UserService` (`contracts@v0.5.12`), порт `50052`
- **Наблюдаемость:** Prometheus-метрики на внутреннем HTTP-сервере `:9096` (`/metrics`, `/livez`), `grpcprom`, структурированное логирование `log/slog`

---

## 3. Порты, healthcheck и база данных
- **gRPC порт:** `50052` (`127.0.0.1:50052` на хосте в Compose)
- **Metrics HTTP (внутренний):** `:9096` (`/metrics`, `/livez`)
- **База данных:** `orbit_users` в PostgreSQL (роль `orbit_user`, создаётся скриптом `postgres-init`)
- **Healthcheck:** стандартный gRPC Health Check для сервиса `orbit.users.v1.UserService` (`/bin/grpc_health_probe -addr=127.0.0.1:50052 -service=orbit.users.v1.UserService`)

---

## 4. Переменные окружения

В Docker Compose (`infra/compose.yaml`) сервис получает единый `POSTGRES_DSN`, порт gRPC и адрес метрик:

```ini
USERS_GRPC_PORT=50052
METRICS_ADDR=0.0.0.0:9096
POSTGRES_DSN=postgres://orbit_user:change_this_app_password@postgres:5432/orbit_users?sslmode=disable
```

При локальном запуске процесса на хосте без `POSTGRES_DSN` (или `DATABASE_URL`) сервис также поддерживает раздельные переменные `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE` (и резервное имя `GRPC_PORT`):

```ini
USERS_GRPC_PORT=50052
METRICS_ADDR=0.0.0.0:9096

DB_HOST=localhost
DB_PORT=5434
DB_USER=orbit_user
DB_PASSWORD=change_this_app_password
DB_NAME=orbit_users
DB_SSLMODE=disable
```

Не коммитьте пароли. О роли `orbit_user` и инициализации базы `orbit_users` см. [инфраструктуру Compose](../../contracts-infra/infra.md).

См. также: [история задачи разработки users-service](../../project/tasks/users-service.md), [OpenAPI-сводка](../api/openapi.md).
