# Репозиторий contracts: Protobuf и Buf CLI

Все gRPC-контракты и схемы событий экосистемы Orbit AI живут в едином репозитории [**`contracts`**](https://github.com/orbit-polytech/contracts). Сервисы не хранят копии `.proto` файлов у себя, а используют сгенерированный код.

---

## 1. Дерево каталогов `contracts`

```text
contracts/
├── proto/
│   └── orbit/
│       ├── common/
│       │   └── v1/
│       │       └── pagination.proto      # Общие структуры пагинации
│       ├── auth/
│       │   └── v1/
│       │       └── auth_service.proto    # Контракт сервиса авторизации
│       ├── users/
│       │   └── v1/
│       │       └── users_service.proto   # Профили и группы студентов
│       ├── tasks/
│       │   └── v1/
│       │       ├── tasks_service.proto   # CRUD операций над задачами
│       │       └── task.proto            # Модель учебной задачи
│       └── events/
│           └── v1/
│               └── task_events.proto     # Protobuf-схемы для RabbitMQ
├── gen/
│   ├── go/                               # Сгенерированный Go-код (коммитится в Git)
│   └── python/                           # Сгенерированный Python-код
├── buf.yaml                              # Конфигурация Buf v2 (линтер и breaking changes)
├── buf.gen.yaml                          # Генерация плагинами buf.build
├── go.mod
├── Makefile
└── README.md
```

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
8. **Временные типы**: строго `google.protobuf.Timestamp`, а для промежутков — `google.protobuf.Duration`. Использование обычных строк или `int64` миллисекунд запрещено.
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
managed:
  enabled: true
  override:
    - file_option: go_package_prefix
      value: github.com/orbit-polytech/contracts/gen/go
plugins:
  - remote: buf.build/protocolbuffers/go
    out: gen/go
    opt:
      - paths=source_relative
  - remote: buf.build/grpc/go
    out: gen/go
    opt:
      - paths=source_relative
inputs:
  - directory: proto
```

Команды проверки и кодогенерации:

```bash
buf lint                 # Проверка стиля и оформления proto-схем
buf breaking --against . # Проверка на случайное нарушение обратной совместимости
buf generate             # Кодогенерация в gen/go и gen/python
```
