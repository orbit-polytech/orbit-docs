# Контракты и Инфраструктура: API-First подход

В основе микросервисной архитектуры Orbit AI лежат два правила:
1. **API-First (сначала контракты):** любое изменение во взаимодействии между сервисами начинается со схемы в репозитории [**`contracts`**](https://github.com/orbit-polytech/contracts). Пока схема не согласована и не прошла `buf lint` и `buf breaking`, код сервиса не меняется.
2. **Единое окружение разработки:** весь локальный стенд поднимается через репозиторий [**`infra`**](https://github.com/orbit-polytech/infra).

---

## 1. Репозиторий `contracts`

Все Protobuf v3 схемы хранятся в одном месте — в `contracts`. Сервисы подключают сгенерированный код через стандартные зависимости: Go-модуль `github.com/orbit-polytech/contracts@<tag>` (`v0.5.9` для Gateway, `v0.5.5` для Auth, `v0.5.0` для Schedule) и Python-пакет `orbit-contracts` (`.pyi` и `py.typed`, тег `v0.5.9` для `ai-service`). Подробнее — [справочник contracts](contracts.md).

---

## 2. Репозиторий `infra`

`infra` (`v0.4.0`) поднимает PostgreSQL 16 (`pgvector`), Redis 7, Auth, Schedule, Gateway, AI, внешний HTTPS-туннель Ngrok, стек мониторинга и `ops-bot`.

Профили Docker Compose:
- `task up` — базовый профиль `apps` (PostgreSQL, Redis, Auth, Schedule, Gateway и one-shot сервисы);
- `--profile ai` — только AI и его зависимости;
- `--profile tunnel` (или `task tunnel:up`) — внешний туннель Ngrok для Gateway;
- `--profile monitoring` — Prometheus, Alertmanager, Loki, Grafana Alloy, Grafana и экспортеры;
- `--profile ops` — операционный Telegram-бот `ops-bot`;
- `--profile all` — полный стенд со всеми компонентами сразу.

RabbitMQ и Jaeger остаются на следующие этапы.

Дальше: [infra (Docker Compose)](infra.md), [Мониторинг и логи](observability.md), [запуск с AI](../how-to/ai-local-stack.md), [порты и env](../reference/ports-and-env.md), [доступ к GHCR](../quickstart/ghcr-access.md).
