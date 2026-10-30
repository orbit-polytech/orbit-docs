# Репозиторий contracts: Protobuf и Buf CLI

Согласованные gRPC-контракты экосистемы Orbit AI живут в едином репозитории [**`contracts`**](https://github.com/orbit-polytech/contracts). Сервисы не хранят копии `.proto` файлов у себя, а используют сгенерированный код.

---

## 1. Дерево каталогов `contracts`

```text
contracts/
├── proto/orbit/
│   ├── auth/v1/
│   │   ├── auth.proto
│   │   ├── oauth.proto
│   │   └── identity.proto
│   ├── ai/v1/
│   │   └── ai.proto
│   └── schedule/v1/
│       └── schedule.proto
├── gen/go/                 # Go
├── gen/python/orbit/       # Python, .pyi и py.typed
├── pyproject.toml          # устанавливаемый orbit-contracts
├── buf.yaml
├── buf.gen.yaml
├── Taskfile.yml
├── go.mod
└── README.md
```

Срез `v0.5.0`: пакет `orbit.schedule.v1` содержит `SearchGroups` и `GetGroupSchedule`. Ответ чтения содержит `date_from = 6` и `date_to = 7`; номера старых полей не меняются. В `ScheduleMetadata` добавлены `status = 4` (`UNSPECIFIED`, `READY`, `PENDING_SYNC`), `published_from = 5` и `published_to = 6`. Старые номера полей сохранены. Schedule сохраняет эту версию модуля; Gateway с AI использует `v0.5.2`. Users, Tasks, общие pagination-схемы и события добавляются после согласования; учебные proto Tasks находятся отдельно в `orbit-docs/examples/tasks-service`.

Подключение Go-модуля из корня репозитория сервиса:

```bash
go get github.com/orbit-polytech/contracts@v0.5.2
go mod tidy
```

Путь импорта пакета: `github.com/orbit-polytech/contracts/gen/go/orbit/schedule/v1`. Версия `v0.5.0` — версия Go-модуля, `orbit.schedule.v1` — версия protobuf-пакета; это разные уровни версионирования.


> [!IMPORTANT]
> Путь к `.proto` файлу строго соответствует пакету Protobuf: файл с декларацией `package orbit.tasks.v1;` обязан лежать в каталоге `proto/orbit/tasks/v1/`. Это требование Buf CLI и стандарт экосистемы Google.

---

## 2. Официальный Protobuf Style Guide

Мы следуем [Google Protobuf Style Guide](https://protobuf.dev/programming-guides/style/):

1. **Имена файлов**: `lower_snake_case.proto`.
2. **Имя пакета**: `orbit.<сервис>.v1`. Указание версии (`v1`, `v2`) обязательно.
3. **Имена сущностей**: `PascalCase` для сообщений, перечислений (enum) и сервисов.
4. **Имена полей**: `snake_case` в нижнем регистре: `due_at`, `user_id`, `idempotency_key`.
5. **Массивы**: `repeated` во множественном числе: `repeated Task tasks = 1;`.
6. **Перечисления (enum)**: имена значений в `UPPER_SNAKE_CASE` с обязательным префиксом названия самого enum. Нулевое значение строго `_UNSPECIFIED = 0`:
   ```proto
   enum TaskStatus {
     TASK_STATUS_UNSPECIFIED = 0;
     TASK_STATUS_OPEN = 1;
     TASK_STATUS_DONE = 2;
   }
   ```
7. **Изолированные Request и Response**: каждый RPC-метод обязан иметь свои собственные сообщения `XxxRequest` и `XxxResponse`, даже если на текущем этапе они не содержат полей. Это гарантирует возможность расширения API в будущем без поломки обратной совместимости.
8. **Временные типы**: для момента времени используйте `google.protobuf.Timestamp`, для длительности — `google.protobuf.Duration`. Календарные даты Schedule передаются строками `YYYY-MM-DD`, время пары — `HH:mm` вместе с `time_zone`. Это не момент UTC; нельзя заменять календарную дату timestamp без изменения семантики.
9. **Удаление полей**: номера полей Protobuf никогда не переиспользуются. При удалении поля его номер и имя объявляются зарезервированными: `reserved 4; reserved "old_field";`.

---

## 3. Конфигурация Buf v2

Файл `buf.yaml`:

```yaml
version: v2
modules:
  - path: proto
lint:
  use:
    - STANDARD
breaking:
  use:
    - FILE
```

Файл генерации кода `buf.gen.yaml`:

```yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go:v1.36.5
    out: gen/go
    opt:
      - paths=source_relative
  - remote: buf.build/grpc/go:v1.5.1
    out: gen/go
    opt:
      - paths=source_relative
```

Текущий `buf.gen.yaml` генерирует Go и Python, включая Python `.pyi`. Фрагмент выше показывает только Go-плагины; полный источник — репозиторий contracts. Dart-генерация пока не настроена.


Команды проверки и кодогенерации:

```bash
buf lint                 # Проверка стиля и оформления proto-схем
buf breaking --against 'https://github.com/orbit-polytech/contracts.git#tag=v0.4.0' # согласованная база сравнения
buf generate             # Кодогенерация в gen/go
```


## AI и Python-пакет

`orbit.ai.v1.AIService` содержит `ChatStream` (server streaming) и `CancelGeneration`. Стадии: START, DELTA, CITATIONS, DONE, ERROR. Контракт цитат содержит название, раздел, URL, score и excerpt; внутренний `source_id` не передаётся клиенту.

В `ai-service/pyproject.toml` зависимость `orbit-contracts` закреплена Git-тегом:

```toml
[tool.uv.sources]
orbit-contracts = { git = "https://github.com/orbit-polytech/contracts.git", tag = "v0.5.2" }
```

Для существующего checkout AI используйте `uv sync --locked`. Для изолированной проверки установки контракта: `python -m pip install "orbit-contracts @ git+https://github.com/orbit-polytech/contracts.git@v0.5.2"`. Editable-соседство `../contracts` не является требованием Docker-сборки. Версия Python distribution в `pyproject.toml` и Git release-tag — разные значения; нельзя автоматически считать `version=0.1.0` тегом контракта.

Пакет включает `gen/python/orbit`, `.pyi` и `py.typed`. Генерируемые файлы вручную не исправляются. После изменения proto запускаются `task generate` и `task check`, проверяется diff; совместимость Go и Python проверяется потребителями. Для breaking-check выбирайте согласованную базовую версию, не копируйте устаревший тег из примера вслепую.

Источники: [proto](https://github.com/orbit-polytech/contracts/blob/main/proto/orbit/ai/v1/ai.proto), [Buf](https://github.com/orbit-polytech/contracts/blob/main/buf.gen.yaml), [Python package](https://github.com/orbit-polytech/contracts/blob/main/pyproject.toml).
