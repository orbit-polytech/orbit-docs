# schedule-service (Расписание занятий)

**Репозиторий:** [`orbit-polytech/schedule-service`](https://github.com/orbit-polytech/schedule-service)

---

## 1. Назначение
Предоставляет актуальное расписание занятий для групп и преподавателей Московского Политеха: учет четной/нечетной недели (числитель/знаменатель), переносы пар, праздничные дни, связь с аудиториями и корпусами. Высокая скорость отдачи благодаря двухуровневому кэшированию в Redis.

---

## 2. Технический стек
- **Язык:** Go 1.24+ (Clean Architecture)
- **База данных:** PostgreSQL 16 (`schedule_db`)
- **Кэш:** Redis 7 (паттерн Cache-Aside + `singleflight.Group` + TTL Jitter)
- **Брокер сообщений:** RabbitMQ (публикация события `schedule.updated` при правках диспетчера)
- **Протокол:** gRPC (порт `50053`)

---

## 3. Выделенные порты и база данных
- **gRPC порт:** `50053`
- **База данных:** `schedule_db` в PostgreSQL
- **Кэш-ключи:** `schedule:group:<group_id>`

---

## 4. Обязательные переменные окружения

```ini
APP_ENV=local
LOG_LEVEL=debug
GRPC_PORT=50053

DB_HOST=localhost
DB_PORT=5432
DB_USER=orbit
DB_PASSWORD=orbit_dev_secret
DB_NAME=schedule_db
DB_SSLMODE=disable

REDIS_ADDR=localhost:6379
REDIS_SCHEDULE_TTL=30m

RABBITMQ_URL=amqp://guest:guest@localhost:5672/
```
