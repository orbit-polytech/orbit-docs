# users-service (Профили пользователей)

**Репозиторий:** [`orbit-polytech/users-service`](https://github.com/orbit-polytech/users-service)

!!! info "Статус: целевая архитектура"
    Этот паспорт описывает проектируемый сервис, а не готовую реализацию. Текущее задание, предлагаемые решения и критерии приёмки — на странице [«Разработка users-service»](../../project/tasks/users-service.md). Параметры паспорта уточняются при согласовании контракта и интеграции.

---

## 1. Назначение
Хранит и управляет профилями студентов, старост и преподавателей Московского Политеха: ФИО, учебная группа (например, `261-334`), факультет/институт, аватар, контакты, системные роли (`student`, `group_leader`, `curator`, `admin`).

---

## 2. Технический стек
- **Язык:** Go 1.24+ (Clean Architecture)
- **База данных:** PostgreSQL 16 (`users_db`), пул `pgxpool`
- **Миграции:** Goose CLI (`migrations/`)
- **Протокол:** gRPC (порт `50052`)
- **Контракт:** `orbit.users.v1.UserService`

---

## 3. Выделенные порты и база данных
- **gRPC порт:** `50052`
- **База данных:** `users_db` в PostgreSQL
- **Healthcheck:** `/grpc.health.v1.Health/Check`

---

## 4. Обязательные переменные окружения

```ini
APP_ENV=local
LOG_LEVEL=debug
GRPC_PORT=50052

DB_HOST=localhost
DB_PORT=5432
DB_USER=orbit
DB_PASSWORD=orbit_dev_secret
DB_NAME=users_db
DB_SSLMODE=disable
```
