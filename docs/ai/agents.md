# Руководство для ИИ-агентов (AGENTS.md)

Этот документ предназначен для языковых моделей, автономных ИИ-агентов (Copilot, Claude Code, Cursor, Antigravity) и разработчиков, использующих ИИ-ассистентов для генерации кода в проекте Orbit AI.

---

## 1. Архитектурные правила (MUST / MUST NOT)

### Домен и архитектура (Go):
* **MUST**: Слой `internal/domain` зависит **исключительно от стандартной библиотеки Go (`$gostd`)**. Никаких исключений.
* **MUST NOT**: Никогда не импортируйте библиотеки баз данных (`pgx`, `database/sql`), gRPC (`google.golang.org/grpc`), HTTP или фреймворки внутрь `internal/domain`.
* **MUST NOT**: Никогда не импортируйте адаптеры (`internal/adapters/*`) внутрь слоя сценариев `internal/app` (или `internal/usecase`).
* **MUST**: Интерфейсы портов объявляются на стороне потребителя: порты инфраструктуры — в `internal/app/ports.go`, интерфейсы вызова usecase — внутри `internal/transport/grpc/handler.go`.
* **MUST**: Любой метод изменения агрегата (`task.Complete()`) инкапсулирует бизнес-инварианты. Инкремент версии агрегата выполняется **репозиторием** при сохранении, а не доменным методом.
* **MUST NOT**: Никогда не используйте `panic()` в продовом коде. Все ошибки возвращаются явно через сигнатуру `error`.

### Текущая реализация Schedule:
* **MUST**: Сохранять публичные SearchGroups/GetGroupSchedule без зависимости от профиля Users и JWT.
* **MUST**: Вычислять период по умолчанию в `internal/usecase`, не в Gateway: текущая неделя Пн–Вс в `Europe/Moscow`; только date_from — один день; только date_to — ошибка; максимум 93 дня включительно.
* **MUST**: Чтение использует общий Redis-кэш расписания с fallback в PostgreSQL; календарная нарезка выполняется в usecase. Запрос может поставить фоновое задание в Redis Streams, но не должен обращаться к источнику синхронно. Все исходящие запросы используют общий распределённый rate limiter.
* **MUST NOT**: Считать сессию, API-фильтр подгрупп, универсальную чётность недель, presets или grid реализованными. Redis-кэш и фоновое обновление по запросу уже есть; `PENDING_SYNC` допустим только при подтверждённом задании. Не выдавать пустой `READY` за отсутствие группы.

### Безопасность и авторизация:
* **MUST NOT**: Никогда не принимайте `user_id` или роль из входящего JSON/Protobuf DTO от клиента. Они берутся **строго из криптографически проверенного контекста Gateway** (`x-user-id`).
* **MUST**: Любой сценарий доступа к персональному ресурсу проверяет принадлежность (`task.IsOwnedBy(actor)`). При несовпадении возвращается `ErrTaskNotFound` (404), а не 403.
* **MUST**: Все SQL-запросы обязаны быть параметризованными (`$1`, `$2`). Конкатенация строк в SQL запрещена.
* **MUST**: Для подписи JWT используется алгоритм **Ed25519 (EdDSA)**. Симметричный HS256 запрещен.

### Python и AI:
* **MUST**: Все зависимости Python управляются через `uv` в `pyproject.toml`.
* **MUST**: Размерность эмбеддингов в PostgreSQL `pgvector` строго фиксирована: **`vector(768)`**.
* **MUST NOT**: Имена моделей нейросетей никогда не хардкодятся в коде — они читаются из переменных окружения `AI_CHAT_MODEL` и `AI_EMBEDDING_MODEL`.
* **MUST**: Сессия `httpx.AsyncClient` в `bot-service` создается один раз при старте приложения, а не на каждое сообщение.

### Flutter и мобилка:
* **MUST**: Использовать архитектуру MVVM + Command + Result. Логика живет во ViewModel, виджет только отображает состояние.
* **MUST**: Токены хранятся строго через `flutter_secure_storage`.

---

## 2. Пути к ключевым файлам

| Компонент / Слой | Канонический путь |
|---|---|
| **Proto-контракты** | `contracts/proto/orbit/<сервис>/v1/<сервис>.proto` |
| **Сгенерированный Go-код** | `contracts/gen/go/orbit/<сервис>/v1/` |
| **Точка сборки Go-сервиса** | `<сервис>/cmd/<сервис>/main.go` |
| **Доменные сущности Go** | `<сервис>/internal/domain/` |
| **Сценарии Use Case Go** | `<сервис>/internal/app/` (или `internal/usecase/`) |
| **Интерфейсы портов** | `<сервис>/internal/app/ports.go` |
| **Адаптер PostgreSQL** | `<сервис>/internal/adapters/postgres/` |
| **gRPC Обработчик** | `<сервис>/internal/transport/grpc/handler.go` |
| **Миграции PostgreSQL** | `<сервис>/migrations/` (Goose); provisioner: `infra/scripts/initdb/01-init-user.sh` |
| **Docker Compose** | `infra/compose.yaml` |

---

## 3. Команды валидации перед сдачей задачи

Перед тем как сообщить об успешном выполнении задачи, ИИ-агент обязан выполнить команды верификации:

```bash
# 1. Проверка Go-сервиса (тесты и линтер)
cd <service>
go test -count=1 ./...
golangci-lint run ./...

# 2. Проверка Protobuf-схем
cd contracts
buf lint
buf breaking --against 'https://github.com/orbit-polytech/contracts.git#tag=v0.4.0' # историческая база совместимости; не версия зависимости сервисов

# 3. Проверка Python-сервиса
cd <python-service>
uv run pytest
uv run ruff check .
uv run mypy src

# 4. Проверка Flutter-приложения
cd mobile
flutter test
flutter analyze

# 5. Проверка документации MkDocs
cd orbit-docs
python scripts/check-doc-links.py
mkdocs build --strict
```
