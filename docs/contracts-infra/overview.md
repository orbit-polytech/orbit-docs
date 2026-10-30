# Контракты и Инфраструктура: API-First подход

В основе надежности микросервисной архитектуры Orbit AI лежат два базовых принципа:
1. **API-First (Сначала контракты)**: любые изменения во взаимодействии микросервисов начинаются с описания схемы в репозитории [**`contracts`**](https://github.com/orbit-polytech/contracts). Ни один разработчик не пишет код сервиса, пока схема интерфейса не согласована и не проверена линтером `buf`.
2. **Единое окружение разработки**: текущий согласованный стек поднимается одной командой через репозиторий [**`infra`**](https://github.com/orbit-polytech/infra).

---

## 1. Репозиторий `contracts`

Вместо того чтобы копировать `.proto` файлы вручную между репозиториями сервисов, все схемы Protobuf v3 централизованно хранятся в `contracts`.

С помощью **Buf CLI** схемы проверяются и генерируются в Go и Python. Текущий AI подключает `contracts@v0.5.2`; Python-пакет содержит gRPC-код, `.pyi` и `py.typed`. Go импортирует пакеты из `github.com/orbit-polytech/contracts/gen/go/orbit/...`, но версионируется корневой модуль. См. [дерево и команды](contracts.md).

---

## 2. Репозиторий `infra`

Репозиторий `infra` поднимает PostgreSQL 16 с pgvector, Redis 7, Auth, Schedule, Gateway и AI с one-shot сервисами. `task up` включает `apps` без AI; `ai` запускает AI и его зависимости, `all` — совместный стенд. Без профиля запускаются базовые зависимости.

Базовый Compose использует готовые GHCR-образы, локальная сборка — override. Host gRPC порты ограничены loopback. RabbitMQ/Jaeger остаются будущими этапами.

Далее: [infra](infra.md), [запуск с AI](../how-to/ai-local-stack.md), [порты](../reference/ports-and-env.md), [GHCR](../quickstart/ghcr-access.md).
