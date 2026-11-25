# Мониторинг, логи и алерты (Observability)

В стеке `infra` настроен локальный контур наблюдаемости: метрики сервисов и баз данных в **Prometheus**, централизованный сбор логов через **Grafana Alloy** в **Loki**, визуализация в **Grafana** и маршрутизация алертов через **Alertmanager** в [`ops-bot`](../reference/services/ops-bot.md).

## Как устроен контур

```text
auth-service (:9091/metrics)      ──┐
schedule-service (:9092/metrics)  ──┤
ai-service (:9093/metrics)        ──┼──▶ prometheus (:9090) ──▶ alertmanager (:9093) ──▶ ops-bot (:8081/alerts/webhook)
gateway-service (:9094/metrics)   ──┤         │
users-service (:9096/metrics)     ──┤         ▼
postgres-exporter (:9187)         ──┤      grafana (:3000) ◀── loki (:3100) ◀── alloy (/var/run/docker.sock)
redis-exporter (:9121)            ──┘
```

Все сервисы при запуске в Compose пишут логи в `stdout` в формате JSON с полем `service` (у Go-сервисов через стандартный `log/slog`, у `ai-service` через JSON-форматтер в production-режиме). Alloy автоматически забирает логи всех контейнеров проекта `orbit-infra`, извлекает `level`, `request_id` и `trace_id` (конфигурация — `monitoring/alloy/config.alloy`) и складывает их в Loki.

## Компоненты и порты

| Сервис | Роль | Внутри сети | На хосте (по умолчанию) |
|---|---|---|---|
| `prometheus` | Сбор метрик каждые `15s` (retention `15d`) и оценка alert rules | `prometheus:9090` | `127.0.0.1:9090` |
| `alertmanager` | Группировка алертов и отправка вебхука в `ops-bot` | `alertmanager:9093` | `127.0.0.1:9093` |
| `loki` | Хранилище структурированных логов | `loki:3100` | Не публикуется |
| `alloy` | Сборщик Docker-логов (`loki.source.docker`) | `alloy:12345` | Не публикуется |
| `grafana` | Веб-интерфейс с дашбордами и источниками Prometheus/Loki | `grafana:3000` | `127.0.0.1:3000` |
| `postgres-exporter` | Метрики PostgreSQL (роль `orbit_metrics` с правом `pg_monitor`) | `postgres-exporter:9187` | Не публикуется |
| `redis-exporter` | Метрики Redis (`redis_exporter`) | `redis-exporter:9121` | Не публикуется |

### Изоляция `/metrics` у микросервисов

Порты метрик слушают только внутри Docker-сети `infra` и не пробрасываются на наружный HTTP-периметр:

| Сервис | Внутренний адрес | Что отдаёт |
|---|---|---|
| `auth-service` | `auth-service:9091` | `/metrics` (`grpcprom`), `/livez` |
| `schedule-service` | `schedule-service:9092` | `/metrics` (`grpcprom` и метрики синка `orbit_schedule_sync_*`), `/livez` |
| `ai-service` | `ai-service:9093` | `/metrics` (токены, квоты, длительность и TTFT LLM, RAG-исходы, gRPC) |
| `gateway-service` | `gateway-service:9094` | `/metrics` (HTTP RED-метрики `orbit_gateway_http_*`), `/livez` |
| `users-service` | `users-service:9096` | `/metrics` (`grpcprom`), `/livez` |

Наружный порт `gateway-service:8080`, клиентский прокси `web:8088` и туннель Ngrok намеренно не открывают `/metrics` — запрос `GET /metrics` на `:8080` возвращает `404` (а на `:8088` отдаётся SPA fallback).

## Дашборды Grafana

Datasources (`Prometheus` и `Loki`) и дашборды из `monitoring/grafana/` провиженятся автоматически при старте контейнера Grafana. Три готовых дашборда лежат в папке **Orbit Polytech**:

- **Orbit — System Overview** (`orbit-overview.json`): состояние таргетов `up`, RPS и доля 5xx на Gateway, latency по сервисам, коннекты PostgreSQL и память Redis.
- **Orbit — AI & LLM Tokens / Quotas** (`orbit-llm-tokens.json`): расход токенов по провайдерам, моделям и ролям (`router`, `casual`, `knowledge_rag`, `campus_navigation`, `general_study`), остатки лимитов Groq и OpenRouter, ошибки `rate_limit`.
- **Orbit — Centralized Logs (Loki)** (`orbit-logs.json`): живой поток логов с фильтрацией по `service`, `level` и `request_id`.

Логин и пароль администратора Grafana задаются в `infra/.env` переменными `GRAFANA_ADMIN_USER` и `GRAFANA_ADMIN_PASSWORD`.

## Правила алертов

Правила лежат в `monitoring/prometheus/alerts/orbit-alerts.yml`, маршрутизация — в `monitoring/alertmanager/alertmanager.yml`:

| Алерт | Условие | Severity | Топик в `ops-bot` |
|---|---|---|---|
| `ServiceDown` | `up == 0` дольше 2 минут | `critical` | `#alerts` |
| `GatewayHigh5xxRate` | Доля `5xx` на Gateway > 5% за 5 минут | `critical` | `#alerts` |
| `AiServiceHighLatency` | p95 генерации LLM > 15s за 3 минуты | `warning` | `#alerts` |
| `LlmQuotaLowWarning` | Остаток квоты токенов < 15% за 1 минуту | `warning` | `#llm-quotas` |
| `LlmQuotaLowCritical` | Остаток квоты токенов < 5% за 30 секунд | `critical` | `#llm-quotas` |
| `LlmRateLimitHitSpike` | Более 3 ответов `rate_limit` за 5 минут | `warning` | `#llm-quotas` |
| `PostgresHighConnections` | Использовано > 80% `max_connections` за 3 минуты | `warning` | `#alerts` |
| `RedisMemoryHigh` | Использовано > 85% `maxmemory` за 3 минуты | `warning` | `#alerts` |

Для пары `warning`/`critical` по одному `alertname` и `service` настроено `inhibit_rules`: критический алерт глушит предупреждение. Alertmanager авторизуется в `ops-bot` Bearer-токеном из файла `/tmp/webhook_token`, который создаётся при старте контейнера из `OPS_BOT_WEBHOOK_TOKEN`.

## Запуск и быстрая проверка

Поднять только мониторинг (поверх уже работающего стенда):

```bash
docker compose --profile monitoring up -d --wait
```

Или запустить весь стек сразу — приложения, AI, мониторинг, туннель и `ops-bot`:

```bash
docker compose --profile all up -d --wait --wait-timeout 180
```

Что проверить после запуска:

1. В `http://127.0.0.1:9090/targets` все **8** таргетов (`prometheus`, `gateway-service`, `auth-service`, `users-service`, `schedule-service`, `ai-service`, `postgres`, `redis`) в состоянии `UP`.
2. `curl -i http://127.0.0.1:8080/metrics` возвращает `404 Not Found` (метрики не торчат наружу).
3. В Grafana (`http://127.0.0.1:3000`) открываются дашборды в папке **Orbit Polytech**, а в `Orbit — Centralized Logs (Loki)` идут свежие логи сервисов.

Дальше: [инфраструктура Compose](infra.md), [паспорт ops-bot](../reference/services/ops-bot.md), [порты и переменные](../reference/ports-and-env.md).
