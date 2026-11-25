# Контракты и Инфраструктура: API-First подход

В основе микросервисной архитектуры Orbit AI лежат два правила:
1. **API-First (сначала контракты):** любое изменение во взаимодействии между сервисами начинается со схемы в репозитории [**`contracts`**](https://github.com/orbit-polytech/contracts). Пока схема не согласована и не прошла `buf lint` и `buf breaking`, код сервиса не меняется.
2. **Единое окружение разработки:** весь локальный стенд поднимается через репозиторий [**`infra`**](https://github.com/orbit-polytech/infra).

---

## 1. Репозиторий `contracts`

Все Protobuf v3 схемы хранятся в одном месте — в `contracts` (актуальный тег **`v0.5.12`**). Сервисы подключают сгенерированный код через стандартные зависимости: Go-модуль `github.com/orbit-polytech/contracts@<tag>` (`v0.5.12` для Gateway и Users, `v0.5.11` для Auth, `v0.5.0` для Schedule) и Python-пакет `orbit-contracts` (`.pyi` и `py.typed`, тег `v0.5.10` для `ai-service`). Подробнее — [справочник contracts](contracts.md).

---

## 2. Репозиторий `infra`

`infra` (`v0.6.0`) поднимает PostgreSQL 16 (`pgvector`), Redis 7, Auth, Users, Schedule, Gateway, Web/PWA (`web` на `:8088`), AI, внешний HTTPS-туннель Ngrok (`web:80`), стек мониторинга и `ops-bot`.

Профили Docker Compose:
- `task up` — базовый профиль `apps` (PostgreSQL, Redis, Auth, Users, Schedule, Gateway, Web и one-shot миграции/keygen);
- `--profile ai` — только AI и его зависимости;
- `--profile tunnel` (или `task tunnel:up`) — внешний туннель Ngrok к `web:80`;
- `--profile monitoring` — Prometheus (8 таргетов), Alertmanager, Loki, Grafana Alloy, Grafana и экспортеры;
- `--profile ops` — операционный Telegram-бот `ops-bot`;
- `--profile all` — полный стенд со всеми компонентами сразу.

RabbitMQ и Jaeger остаются на следующие этапы.

Дальше: [infra (Docker Compose)](infra.md), [Мониторинг и логи](observability.md), [запуск с AI](../how-to/ai-local-stack.md), [порты и env](../reference/ports-and-env.md), [доступ к GHCR](../quickstart/ghcr-access.md).
