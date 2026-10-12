# gateway-service (API Gateway)

**Репозиторий:** [`orbit-polytech/gateway-service`](https://github.com/orbit-polytech/gateway-service)

---

## 1. Назначение
Входная точка и периметр безопасности распределенной системы Orbit AI. Принимает внешние HTTP/REST запросы от мобильного приложения на Flutter, веб-интерфейсов и Telegram-бота, валидирует JWT-токены Ed25519, защищает от атак (Rate Limiting, CORS) и транслирует вызовы во внутренний gRPC-контур.

---

## 2. Технический стек
- **Язык:** Go 1.24+
- **HTTP-роутер:** `go-chi/chi/v5`
- **Протоколы:** Вход — HTTP/1.1 & HTTP/2 (REST/JSON), Выход — gRPC по HTTP/2
- **Кэш и безопасность:** Redis 7 (проверка блэклиста отозванных JWT по `jti`, Rate Limiting)
- **Контракты:** Клиенты gRPC из `orbit-contracts/gen/go`

---

## 3. Сетевые порты
- **Внешний порт (Host/Docker):** `8080` (HTTP REST & Swagger UI)
- **Метрики Prometheus:** `http://localhost:8080/metrics`
- **Проверка здоровья (Healthcheck):** `http://localhost:8080/healthz`

---

## 4. Обязательные переменные окружения

```ini
APP_ENV=local
LOG_LEVEL=debug
HTTP_PORT=8080

# Redis для блэклиста токенов
REDIS_ADDR=localhost:6379

# Публичный ключ для проверки подписи JWT Ed25519
JWT_PUBLIC_KEY_PEM="-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----"

# Адреса внутренних gRPC-сервисов
AUTH_GRPC_ADDR=localhost:50051
USERS_GRPC_ADDR=localhost:50052
SCHEDULE_GRPC_ADDR=localhost:50053
NAVIGATION_GRPC_ADDR=localhost:50054
TASKS_GRPC_ADDR=localhost:50055
AI_GRPC_ADDR=localhost:50057
```
