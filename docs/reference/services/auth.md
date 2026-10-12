# auth-service (Сервис аутентификации)

**Репозиторий:** [`orbit-polytech/auth-service`](https://github.com/orbit-polytech/auth-service)

---

## 1. Назначение
Отвечает за регистрацию, вход пользователей по логину/паролю, безопасное хэширование паролей (Argon2id), выпуск цифровых пар токенов (Access JWT на базе Ed25519 и Refresh-токенов в БД), ротацию сессий и отзыв токенов (Logout) с занесением `jti` в Redis блэклист.

---

## 2. Технический стек
- **Язык:** Go 1.24+ (Clean Architecture)
- **База данных:** PostgreSQL 16 (`auth_db`), пул `pgxpool`
- **Миграции:** Goose CLI (`migrations/`)
- **Криптография:** `golang.org/x/crypto/argon2`, `crypto/ed25519`
- **Протокол:** gRPC (порт `50051`)

---

## 3. Выделенные порты и база данных
- **gRPC порт:** `50051`
- **База данных:** `auth_db` в PostgreSQL
- **Healthcheck:** `/grpc.health.v1.Health/Check`

---

## 4. Обязательные переменные окружения

```ini
APP_ENV=local
LOG_LEVEL=debug
GRPC_PORT=50051

DB_HOST=localhost
DB_PORT=5432
DB_USER=orbit
DB_PASSWORD=orbit_dev_secret
DB_NAME=auth_db
DB_SSLMODE=disable

# Приватный ключ Ed25519 (секрет сервиса auth-service)
JWT_PRIVATE_KEY_PEM="-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----"
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=168h

REDIS_ADDR=localhost:6379
```
