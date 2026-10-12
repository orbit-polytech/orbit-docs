# Справочник контейнеров и инфраструктуры

В этом справочнике сведены все параметры портов, дефолтных кредов для разработки и очередей RabbitMQ.

---

## 1. Порты инфраструктурных контейнеров

| Контейнер | Назначение | Порт хоста | Локальные креды (dev) |
|---|---|---|---|
| **`postgres`** | PostgreSQL 16 + pgvector | `127.0.0.1:5432` | user: `orbit`, password: `orbit_local_pass`, db: `orbit` |
| **`redis`** | In-memory кэш и rate limiting | `127.0.0.1:6379` | пароль отсутствует (только loopback) |
| **`rabbitmq`** | AMQP брокер очередей | `127.0.0.1:5672` | guest / guest |
| **`rabbitmq` (UI)** | Панель управления брокером | `127.0.0.1:15672`| guest / guest |
| **`jaeger` (UI)** | Веб-интерфейс трассировки | `127.0.0.1:16686`| авторизация не требуется |
| **`jaeger` (OTLP)**| OTLP gRPC приёмник трейсов | `127.0.0.1:4317` | — |

---

## 2. Именование очередей и Routing Keys в RabbitMQ

Очереди сообщений именуются по стандарту: `<сервис-источник>.<сущность>.<событие>`

| Топик / Exchange | Routing Key | Очередь потребителя | Описание события |
|---|---|---|---|
| `orbit.events` | `tasks.task.created` | `notification.tasks` | Создана новая задача, запланировать напоминание студенту |
| `orbit.events` | `tasks.task.completed` | `notification.tasks` | Задача выполнена, отменить уведомления |
| `orbit.events` | `schedule.lesson.changed`| `notification.schedule`| Пара перенесена в другую аудиторию или отменена |
| `orbit.events` | `auth.user.registered` | `users.sync` | Первичная регистрация студента через Telegram |

Exchange во всех сервисах объявляется как `topic` с флагом `durable: true`.
