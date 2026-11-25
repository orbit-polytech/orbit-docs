# Руководство для ИИ-агентов (AGENTS.md)

Документ для языковых моделей, автономных агентов и разработчиков в экосистеме Orbit AI.

---

## 1. Архитектурные правила (MUST / MUST NOT)

### Домен и архитектура (Go):
* **MUST**: Слой `internal/domain` зависит **только от стандартной библиотеки Go (`$gostd`)**. Без исключений.
* **MUST NOT**: Не импортируйте драйверы БД (`pgx`, `database/sql`), gRPC (`google.golang.org/grpc`), HTTP или фреймворки внутрь `internal/domain`.
* **MUST NOT**: Не импортируйте адаптеры (`internal/adapters/*` или `internal/infrastructure/*`) внутрь слоя сценариев (`internal/app` / `internal/usecase`).
* **MUST**: Интерфейсы портов объявляются на стороне потребителя.
* **MUST**: Методы изменения агрегата инкапсулируют инварианты, а инкремент версии (оптимистичная блокировка) выполняется репозиторием при сохранении.
* **MUST NOT**: Не используйте `panic()` в рабочем коде (за исключением критических проверок конфигурации при сборке зависимостей на старте, где это явно принято в модуле). Ошибки возвращайте явно через `error`.

### Текущая реализация Schedule:
* **MUST**: Сохранять публичные `SearchGroups` и `GetGroupSchedule` без зависимости от профиля Users и JWT.
* **MUST**: Вычислять период по умолчанию в `internal/usecase` (текущая неделя Пн–Вс в `Europe/Moscow`; только `date_from` — один день; только `date_to` — ошибка; максимум 93 дня включительно).
* **MUST**: Чтение идёт через общий Redis-кэш с fallback в PostgreSQL, обновление — в фоне через Redis Streams и общий распределённый rate limiter.
* **MUST NOT**: Считать сессию, фильтр подгрупп, универсальную чётность, presets или `/grid` реализованными.

### Безопасность и авторизация:
* **MUST NOT**: Не доверяйте `user_id` или ролям из входящего JSON/query от клиента. Они берутся **строго из проверенного контекста Gateway** (`x-user-id`).
* **MUST**: Проверяйте владение ресурсом (`PermissionDenied` в AI, `NotFound` в учебном Tasks — соблюдайте контракт конкретного сервиса).
* **MUST**: Все SQL-запросы параметризованы (`$1`, `$2` или bind-параметры SQLAlchemy). Конкатенация строк в SQL запрещена.
* **MUST**: Подпись JWT — **Ed25519 (EdDSA)**. Симметричный HS256 запрещён.

### Наблюдаемость и изоляция портов:
* **MUST**: Порты `/metrics` и `/livez` (`auth :9091`, `schedule :9092`, `ai :9093`, `gateway :9094`, `users :9096`) слушают только во внутренней сети Docker. Не публикуйте `/metrics` на наружном HTTP-порту Gateway `:8080` и не открывайте через туннель Ngrok.
* **MUST**: Структурированные логи в Go (`log/slog` с JSONHandler) и Python содержат поле `service` и не логируют секреты, токены, DSN с паролями или личную переписку студентов.

### Python, AI и ETL:
* **MUST**: Зависимости в `ai-service`, `knowledge-pipeline` и `ops-bot` управляются через `uv` (`pyproject.toml` + `uv.lock`).
* **MUST**: Конфигурация провайдеров LLM (`AI_PROVIDER`, `GROQ_API_KEY`, `OPENROUTER_API_KEY`, `GEMINI_API_KEY`), маршрутизация ролей (`src/orbit_ai/config/model_routing.json`) и модель эмбеддингов (`EMBEDDING_MODEL_NAME`, 384 измерения для текущего FastEmbed) задаются через настройки, а не хардкодом в бизнес-логике. Смена модели эмбеддингов требует переиндексации.
* **MUST**: Основной корпус базы знаний готовится в `knowledge-pipeline` (`orbit-etl`) и загружается в `ai-service` через `ingest_documents --manifest ...`.
* **MUST**: При реализации будущего `bot-service` сессия `httpx.AsyncClient` создаётся один раз при старте.

### Flutter и мобилка:
* **MUST**: Clean Architecture + BLoC/Cubit и Feature-First (`lib/features/<feature>/{data,domain,presentation}` — включая `schedule`, `auth`, `assistant`, `profile`, `support`, `shell`). Виджеты не лезут в сеть или хранилище напрямую.
* **MUST**: Учитывайте обе целевые поставки `mobile`: Android APK и Web/PWA (`Dockerfile.web`, `web/nginx.conf`, `DesktopShellSidebar`, `IndexedStack`).
* **MUST**: Токены хранятся только через `flutter_secure_storage`. Адрес шлюза настраивается через `--dart-define=BASE_URL=...` (без суффикса `/api/v1`; в Web за Nginx-прокси `/api/` проксируется на `gateway-service`).
* **MUST**: При изменении кода перед открытием PR поднимайте и видимую версию, и build number в `mobile/pubspec.yaml` (например, `0.1.27-beta+28` → `0.1.28-beta+29`) и проверяйте скриптом `python tool/check_version.py`.
* **MUST**: В релизе не используйте обход `usesCleartextTraffic` или проброс секретов в клиент.

---

## 2. Ключевые пути в репозиториях

| Компонент / Слой | Канонический путь |
|---|---|
| **Proto-контракты** | `contracts/proto/orbit/<сервис>/v1/<сервис>.proto` |
| **Сгенерированный Go-код** | `contracts/gen/go/orbit/<сервис>/v1/` |
| **Сгенерированный Python-код** | `contracts/gen/python/orbit/` |
| **Точка сборки Go-сервиса** | `<сервис>/cmd/<сервис>/main.go` |
| **Доменные сущности Go** | `<сервис>/internal/domain/` |
| **Сценарии Use Case Go** | `<сервис>/internal/app/` или `internal/usecase/` |
| **Адаптер PostgreSQL** | `<сервис>/internal/adapters/postgres/` или `internal/infrastructure/postgres/` |
| **gRPC-обработчик** | `<сервис>/internal/transport/grpc/` |
| **Миграции PostgreSQL** | Go: `<сервис>/migrations/` (Goose); AI: `ai-service/migrations/` (Alembic); init-роли: `infra/scripts/initdb/01-init-user.sh` |
| **Docker Compose и мониторинг** | `infra/compose.yaml`, `infra/monitoring/` |
| **Мобильное приложение и Web/PWA** | `mobile/lib/{app,core,design_system,features}` (вкл. `profile`, `support`), `mobile/web/`, `mobile/Dockerfile.web` |

---

## 3. Команды проверки перед сдачей задачи

Перед отчётом о готовности задачи прогоните проверки полностью (не только часть тестов):

```bash
# 1. Go-сервис
cd <service>
go test -race -count=1 ./...
golangci-lint run ./...

# 2. Protobuf-контракты
cd contracts
buf lint
buf breaking --against 'https://github.com/orbit-polytech/contracts.git#tag=v0.5.12' # актуальный тег проверяйте по git tag

# 3. Python-сервис (ai-service / ops-bot / knowledge-pipeline)
cd <python-service>
uv run ruff check .
uv run ruff format --check .
uv run mypy src
uv run pytest

# 4. Мобильное приложение (на Windows перед запуском задайте TEMP/TMP не на C:, например F:\Temp)
cd mobile
flutter analyze --fatal-infos
flutter test
python tool/check_version.py

# 5. Документация MkDocs
cd orbit-docs
python scripts/check-doc-links.py
mkdocs build --strict
```

## Памятка по контрактам и стенду

- Актуальные теги `contracts` на сервисах: `gateway-service` и `users-service` используют **`v0.5.12`**, `auth-service` — `v0.5.11`, `ai-service` — `v0.5.10`, `schedule-service` — `v0.5.0`.
- В `infra` (`v0.6.0`) команда `task up` поднимает только профиль `apps` (PostgreSQL, Redis, Auth, `users-service`, Schedule, Gateway и `web` на `:8088`). Для совместного стенда с AI, мониторингом, `ops-bot` и Ngrok (`web:80`) используйте `--profile all`.
- Интеграционные тесты AI с БД запускайте только против `POSTGRES_TEST_DSN` (`orbit_ai_test`), не против рабочей `orbit_ai` (миграции Alembic на `orbit_ai_test` применяются отдельно, `ai-migrate` обновляет только `orbit_ai`).
- При изменении API или переменных окружения сразу обновляйте [AI API](../gateway/ai-api.md), [порты и env](../reference/ports-and-env.md), [ограничения AI](../python-ai/current-limitations.md) и `docs/llms.txt` (при изменении структуры страниц в `mkdocs.yml`).
