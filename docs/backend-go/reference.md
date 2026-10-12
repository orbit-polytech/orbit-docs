# Справочник слоёв и правил статического анализа

В этом справочнике собраны строгие правила ограничений импортов, конфигурация линтера `depguard` и правила написания SQL-миграций.

---

## 1. Матрица разрешённых импортов

| Слой | Разрешённые импорты | Категорически запрещено импортировать |
|---|---|---|
| **`internal/domain`** | Только стандартная библиотека Go (`$gostd`): `time`, `errors`, `strings`, `unicode/utf8` | Любые внешние пакеты (`pgx`, `grpc`), другие слои сервиса (`app`, `adapters`, `transport`) |
| **`internal/app`** | `$gostd`, `internal/domain` | Любые адаптеры (`adapters/*`), транспорт (`transport/*`), библиотеки драйверов БД |
| **`internal/adapters/*`**| `$gostd`, `internal/domain`, `internal/app` (для интерфейсов портов), внешние библиотеки конкретной технологии (`pgx`, `go-redis`, `amqp`) | Другие адаптеры (например, `adapters/postgres` не может импортировать `adapters/redis`), транспорт `transport/*` |
| **`internal/transport/grpc`**| `$gostd`, `internal/domain`, `internal/app`, protobuf структуры (`pkg/api/...`), `google.golang.org/grpc` | Адаптеры инфраструктуры (`adapters/*`), прямое чтение базы данных |
| **`cmd/<service>`** | Имеет доступ ко всем слоям для связывания Composition Root | Бизнес-логика, прямые SQL-запросы |

---

## 2. Конфигурация depguard (`.golangci.yml`)

Для автоматического контроля слоев на уровне компилятора в GitHub Actions настроен линтер `depguard`:

```yaml
linters-settings:
  depguard:
    rules:
      domain_isolation:
        list-mode: strict
        files:
          - "!**/*_test.go"
          - "**/internal/domain/**"
        allow:
          - "$gostd"
      usecase_isolation:
        list-mode: strict
        files:
          - "!**/*_test.go"
          - "**/internal/app/**"
          - "**/internal/usecase/**"
        allow:
          - "$gostd"
          - "github.com/orbit-polytech/tasks-service/internal/domain"
```

Если новичок попытается импортировать базу данных в домен, линтер моментально заблокирует сборку PR в CI:
`import of "github.com/jackc/pgx/v5" is not allowed in domain layer according to rule 'domain_isolation'`.

---

## 3. Стандарты написания SQL-миграций

Миграции пишутся на чистом SQL для инструмента `goose`:

1. **Файлы создаются парами**: `NNNNNN_<name>.up.sql` и `NNNNNN_<name>.down.sql`.
2. **Имена таблиц и колонок**: пишутся строго в `snake_case` в нижнем регистре во множественном числе для таблиц (`tasks`, `students`, `course_events`).
3. **Первичные ключи**: строковые UUIDv4 / UUIDv7 (`VARCHAR(64)` или `UUID`) для распределённых систем.
4. **Временные метки**: строго `TIMESTAMPTZ` (с сохранением временной зоны), по умолчанию `NOW()`.
5. **Оптимистичная блокировка**: обязательное поле `version BIGINT NOT NULL DEFAULT 1`.
6. **Мягкое удаление (Soft Delete)**: поле `deleted_at TIMESTAMPTZ NULL DEFAULT NULL`.
