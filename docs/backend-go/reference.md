# Справочник слоёв и правил статического анализа

В этом справочнике зафиксированы границы слоёв Go-сервисов, актуальная конфигурация **`golangci-lint` v2** и стандарты SQL-миграций **Goose**.

---

## 1. Матрица разрешённых импортов

| Слой | Разрешённые импорты | Запрещено импортировать |
|---|---|---|
| **`internal/domain`** | Стандартная библиотека Go (`time`, `errors`, `strings`, `unicode/utf8`) и базовые типы без привязки к БД/сети | Пакеты БД и транспорта (`pgx`, `grpc`, `net/http`), слои `usecase`, `infrastructure`, `transport` |
| **`internal/usecase`** | Стандартная библиотека Go, `internal/domain` | Реализации адаптеров (`internal/infrastructure/*`), транспорт (`internal/transport/*`), драйверы БД |
| **`internal/infrastructure/*`** | Стандартная библиотека, `internal/domain`, `internal/usecase` (интерфейсы портов), драйверы конкретной технологии (`pgx/v5`, `go-redis`, `oauth2`) | Транспорт `internal/transport/*`, другие адаптеры напрямую без интерфейсов |
| **`internal/transport/grpc`** (или `http`) | Стандартная библиотека, `internal/domain`, `internal/usecase`, сгенерированные контракты (`github.com/orbit-polytech/contracts/...`), `google.golang.org/grpc` | Прямые SQL-запросы и прямая работа с БД/драйверами |
| **`internal/app`** (Composition Root) | Все слои сервиса для связывания зависимостей | Бизнес-логика и SQL-запросы |

---

## 2. Статический анализ: `golangci-lint` v2 (`.golangci.yml`)

Во всех Go-сервисах используется **`golangci-lint` версии 2** (`version: "2"`). Линтер запускается локально через `task lint` (или `golangci-lint run ./...`) и в GitHub Actions.

Типовая конфигурация `.golangci.yml` (как в `gateway-service` и `auth-service`):

```yaml
version: "2"

run:
  tests: true

linters:
  enable:
    - errcheck
    - govet
    - ineffassign
    - staticcheck
    - unused
  exclusions:
    generated: lax
    presets:
      - comments
      - common-false-positives
      - legacy
      - std-error-handling
    paths:
      - third_party$
      - builtin$
      - examples$

formatters:
  enable:
    - gofmt
  exclusions:
    generated: lax
    paths:
      - third_party$
      - builtin$
      - examples$
```

!!! note "Контроль границ слоёв"
    В текущей конфигурации CI не используется линтер `depguard` — соблюдение матрицы импортов (`domain` ← `usecase` ← `infrastructure`/`transport`) проверяется при проектировании и на Pull Request Review.

---

## 3. Стандарты написания SQL-миграций (Goose)

Миграции пишутся на чистом SQL для **`goose`** — каждая миграция хранится в **одном** `.sql`-файле внутри `migrations/`:

1. **Единый файл с секциями `-- +goose Up` и `-- +goose Down`**:
   `00001_<name>.sql`, `00002_<name>.sql` (создаётся командой `goose -dir ./migrations -s create <name> sql` или `task migrate-create NAME=<name>`).
2. **Имена таблиц и колонок**: пишутся в `snake_case` в нижнем регистре во множественном числе для таблиц (`accounts`, `profiles`, `group_schedules`).
3. **Первичные ключи**: `UUID` (`DEFAULT gen_random_uuid()`).
4. **Временные метки**: `TIMESTAMPTZ` (с часовым поясом), по умолчанию `now()`.
5. **Откат (`-- +goose Down`)**: каждая миграция обязана корректно откатывать свои изменения и проверяться командой `task migrate-down` (для проверки в локальном стенде).
