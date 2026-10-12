# Как написать и применить миграцию базы данных

Для управления схемой реляционной базы данных PostgreSQL в микросервисах Orbit AI используется инструмент **Goose** (`github.com/pressly/goose/v3`). Все миграции пишутся на чистом SQL, версионируются в репозитории сервиса и строго поддерживают как накат (`Up`), так и безопасный откат (`Down`).

---

## 1. Подготовка и переменные окружения

Goose подключается к PostgreSQL с использованием строки DSN. Для удобства локальной работы задайте параметры в терминале или убедитесь, что они подгружаются из `.env`:

```bash
export GOOSE_DRIVER=postgres
export GOOSE_DBSTRING="host=localhost port=5432 user=orbit password=orbit_dev_secret dbname=tasks_db sslmode=disable"
```

Для установки утилиты локально:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
goose --version
```

---

## 2. Пошаговый процесс создания миграции

### Шаг 1. Создание нового файла миграции

Выполните команду генерации шаблона в директории `migrations/`:

```bash
goose -dir migrations create create_tasks_table sql
```

Goose создаст файл с таймстемпом в имени, например:  
`migrations/20261012120000_create_tasks_table.sql`.

---

### Шаг 2. Написание SQL-инструкций (Up и Down)

Откройте созданный файл. Структура должна содержать две секции, разделенные комментариями аннотаций:

```sql
-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS tasks (
    id UUID PRIMARY KEY,
    owner_id VARCHAR(64) NOT NULL,
    title VARCHAR(120) NOT NULL,
    description VARCHAR(1000) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL,
    priority VARCHAR(32) NOT NULL,
    deadline TIMESTAMPTZ,
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_tasks_status CHECK (status IN ('TASK_STATUS_OPEN', 'TASK_STATUS_IN_PROGRESS', 'TASK_STATUS_COMPLETED', 'TASK_STATUS_CANCELED')),
    CONSTRAINT chk_tasks_priority CHECK (priority IN ('TASK_PRIORITY_LOW', 'TASK_PRIORITY_MEDIUM', 'TASK_PRIORITY_HIGH', 'TASK_PRIORITY_URGENT'))
);

-- Индекс для выборки списка задач конкретного пользователя с фильтрацией по статусу
CREATE INDEX IF NOT EXISTS idx_tasks_owner_status ON tasks(owner_id, status);

-- Индекс для сортировки по дедлайну
CREATE INDEX IF NOT EXISTS idx_tasks_deadline ON tasks(deadline ASC NULLS LAST);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tasks;
-- +goose StatementEnd
```

> [!NOTE] Аннотации `-- +goose StatementBegin` и `-- +goose StatementEnd`
> Эти комментарии обязательны при создании составных инструкций, PL/pgSQL функций, триггеров или нескольких DDL-запросов, выполняемых в рамках одной транзакции. Они указывают парсеру Goose не разбивать блок по символу точки с запятой `;`.

---

### Шаг 3. Проверка статуса миграций

Перед накатом проверьте текущее состояние базы данных:

```bash
goose -dir migrations status
```

Вывод покажет все обнаруженные файлы и отметку `Pending` напротив новых:

```text
    Applied At                  Migration
    =======================================
    Pending                  -- 20261012120000_create_tasks_table.sql
```

---

### Шаг 4. Применение миграции (Up)

Выполните накат всех ожидающих миграций:

```bash
goose -dir migrations up
```

При успешном применении в консоли появится подтверждение:

```text
2026/10/12 12:01:00 OK   20261012120000_create_tasks_table.sql (12.4ms)
goose: no migrations to run. current version: 20261012120000
```

Goose автоматически создаёт и обновляет системную таблицу `goose_db_version`, отслеживая применённые версии.

---

### Шаг 5. Обязательная проверка отката (Down)

Каждый разработчик обязан локально проверить, что написанная секция `Down` действительно корректно удаляет созданные объекты и не падает с ошибками зависимостей:

```bash
# 1. Откатываем последнюю миграцию
goose -dir migrations down

# 2. Проверяем статус (должен снова стать Pending)
goose -dir migrations status

# 3. Накатываем обратно
goose -dir migrations up
```

---

## 3. Выполнение миграций внутри Docker Compose

Если вы не хотите устанавливать Goose локально на хост-машину, миграции можно запускать через временный контейнер в рамках общей сети:

```bash
docker run --rm \
  --network orbit-net \
  -v $(pwd)/migrations:/migrations \
  -e GOOSE_DRIVER=postgres \
  -e GOOSE_DBSTRING="host=postgres port=5432 user=orbit password=orbit_dev_secret dbname=tasks_db sslmode=disable" \
  pressly/goose:latest -dir /migrations up
```

---

## 4. Сравнение: Как нельзя и Как правильно

<div class="grid" markdown>

<div class="card" markdown>
### :x: Как нельзя

- **Редактировать уже влитую в ветку main миграцию.** Если ваши коллеги или тестовый стенд уже накатили миграцию, изменение файла сломает хэш и последовательность версий. Любые правки оформляются **только новой миграцией**.
- **Оставлять секцию `-- +goose Down` пустой или писать туда `-- TODO`.** В случае сбоя релиза в продакшене автоматический откат (Rollback) будет невозможен.
- **Удалять колонку таблицы (`DROP COLUMN`) в одном релизе с кодом, который её использует.** Это вызывает ошибки `column does not exist` в момент раскатки новых подов. Делайте переход в два шага: сначала удаление использования в коде, затем отдельной миграцией удаление колонки.
- **Хранить миграции разных микросервисов в одной общей папке.** У каждого сервиса своя изолированная база данных и своя папка `migrations/`.
</div>

<div class="card" markdown>
### :white_check_mark: Как правильно

- **Создавать новую миграцию для каждого изменения схемы** с осмысленным описанием в названии (`add_index_to_tasks_deadline.sql`).
- **Всегда проверять команду `goose down` на локальной БД перед отправкой Pull Request.**
- **Использовать безопасные конструкции:** `CREATE TABLE IF NOT EXISTS`, `DROP TABLE IF EXISTS`, `ADD COLUMN ... DEFAULT ...`.
- **Добавлять индексы на поля внешних связей и фильтрации:** `owner_id`, `status`, `created_at`.
</div>

</div>
