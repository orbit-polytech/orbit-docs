# notification-service (Сервис уведомлений)

**Репозиторий:** [`orbit-polytech/notification-service`](https://github.com/orbit-polytech/notification-service)

---

## 1. Назначение
Асинхронный потребитель событий распределенной системы Orbit AI. Подписывается на очереди RabbitMQ, отслеживает наступление дедлайнов задач, переносы пар, оповещения кураторов и отправляет персонализированные Push-уведомления (Firebase Cloud Messaging / APNs) и Telegram-сообщения.

---

## 2. Технический стек
- **Язык:** Go 1.24+ (Clean Architecture)
- **Брокер сообщений:** RabbitMQ (AMQP 0-9-1 консьюмер с ручным Ack/Nack и DLQ)
- **База данных:** PostgreSQL 16 (`notification_db`) — история уведомлений и предпочтения пользователей
- **Интеграции:** FCM SDK (Google Firebase), Telegram Bot API

---

## 3. Выделенные порты и очереди
- **gRPC порт:** `50056`
- **База данных:** `notification_db` в PostgreSQL
- **Очереди RabbitMQ:**
  * `notification.task_events` (биндинг к exchange `tasks.events`, routing key: `task.*`)
  * `notification.schedule_events` (биндинг к exchange `schedule.events`)

---

## 4. Обязательные переменные окружения

```ini
APP_ENV=local
LOG_LEVEL=debug
GRPC_PORT=50056

DB_HOST=localhost
DB_PORT=5432
DB_USER=orbit
DB_PASSWORD=orbit_dev_secret
DB_NAME=notification_db
DB_SSLMODE=disable

RABBITMQ_URL=amqp://guest:guest@localhost:5672/
FCM_CREDENTIALS_JSON="{...}"
TELEGRAM_BOT_TOKEN="123456:ABC-DEF..."
```
