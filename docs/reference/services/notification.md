# notification-service (Сервис уведомлений)

**Репозиторий:** `orbit-polytech/notification-service` *(репозиторий на GitHub ещё не создан)*

!!! info "Запланировано"
    Сервис уведомлений и брокер RabbitMQ пока не реализованы и не входят в текущий стенд. Ниже приведён проектный паспорт на будущее.

---

## 1. Назначение
Асинхронный потребитель событий распределенной системы Orbit AI. Подписывается на очереди RabbitMQ, отслеживает наступление дедлайнов задач, переносы пар, оповещения кураторов и отправляет персонализированные Push-уведомления (Firebase Cloud Messaging / APNs) и Telegram-сообщения.

---

## 2. Технический стек
- **Язык:** Go 1.26+ (Clean Architecture)
- **Брокер сообщений:** RabbitMQ (AMQP 0-9-1 консьюмер с ручным Ack/Nack и DLQ)
- **База данных:** PostgreSQL 16 (`notification_db`) — история уведомлений и предпочтения пользователей
- **Интеграции:** FCM SDK (Google Firebase), Telegram Bot API

---

## 3. Выделенные порты и очереди
- **gRPC порт:** `50056`
- **База данных:** `notification_db` в PostgreSQL (уточняется при добавлении в `infra`)
- **Очереди RabbitMQ *(план)*:**
  * `notification.task_events` (биндинг к exchange `tasks.events`, routing key: `task.*`)
  * `notification.schedule_events` (биндинг к exchange `schedule.events`)

---

## 4. Обязательные переменные окружения

```ini
APP_ENV=local
LOG_LEVEL=debug
GRPC_PORT=50056

DB_HOST=localhost
DB_PORT=5434
DB_USER=orbit_user
DB_PASSWORD=change_this_app_password
DB_NAME=notification_db
DB_SSLMODE=disable

RABBITMQ_URL=amqp://guest:guest@localhost:5672/
FCM_CREDENTIALS_JSON="{...}"
TELEGRAM_BOT_TOKEN="123456:ABC-DEF..."
```
