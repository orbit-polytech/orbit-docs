# Репозиторий contracts: Protobuf и Buf CLI

Согласованные gRPC-контракты экосистемы Orbit AI живут в едином репозитории [**`contracts`**](https://github.com/orbit-polytech/contracts). Сервисы не хранят копии `.proto` файлов у себя, а подключают сгенерированный код.

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
│   ├── schedule/v1/
│   │   └── schedule.proto
│   └── users/v1/
│       └── users.proto
├── gen/go/                 # сгенерированный Go-код
├── gen/python/orbit/       # Python gRPC, .pyi и py.typed
├── pyproject.toml          # устанавливаемый пакет orbit-contracts
├── buf.yaml
├── buf.gen.yaml
├── Taskfile.yml
├── go.mod
└── README.md
```

Актуальный тег — **`v0.5.12`**:
- `gateway-service` и `users-service` используют `v0.5.12`;
- `auth-service` использует `v0.5.11` (OTP, смена email, JWT, Google OAuth, identities, `GetAccount`, `UpdateAccountPreferences`);
- `ai-service` использует `v0.5.10` (`ChatStream` с `retry_after_sec`, отмена, диалоги, оценки);
- `schedule-service` использует `v0.5.0` (`SearchGroups`, `GetGroupSchedule`, `date_from`/`date_to`, `ScheduleStatus`: `UNSPECIFIED`, `READY`, `PENDING_SYNC`, `published_from`/`published_to`);
- `proto/orbit/users/v1/users.proto` описывает контракт `UserService` для работающего `users-service` (`GetProfile`, `UpdateProfile`, внутренний `GetBatchProfiles`, поля `display_name`, `username`, `academic_group`, `course`, `avatar_url`, `role`);
- учебные схемы `Tasks` лежат отдельно в `orbit-docs/examples/tasks-service` и будут перенесены в `contracts` при проектировании production-сервиса.

Подключение Go-модуля из корня репозитория сервиса:

```bash
go get github.com/orbit-polytech/contracts@v0.5.12
go mod tidy
```

Пример импорта: `github.com/orbit-polytech/contracts/gen/go/orbit/schedule/v1`. При этом `v0.5.12` — тег Go-модуля, а `orbit.schedule.v1` — версия protobuf-пакета.

> [!IMPORTANT]
> Путь к `.proto` файлу обязан совпадать с именем пакета: файл с `package orbit.tasks.v1;` должен лежать в `proto/orbit/tasks/v1/`.

---

## 2. Protobuf Style Guide

Мы придерживаемся [Google Protobuf Style Guide](https://protobuf.dev/programming-guides/style/):

1. **Имена файлов:** `lower_snake_case.proto`.
2. **Имя пакета:** `orbit.<сервис>.v1`. Указание мажорной версии обязательно.
3. **Имена сущностей:** `PascalCase` для message, enum и service.
4. **Имена полей:** `snake_case` в нижнем регистре: `due_at`, `user_id`, `idempotency_key`.
5. **Списки:** `repeated` во множественном числе: `repeated Task tasks = 1;`.
6. **Перечисления (`enum`):** значения в `UPPER_SNAKE_CASE` с префиксом имени перечисления, нулевое значение — строго `_UNSPECIFIED = 0`:
   ```proto
   enum TaskStatus {
     TASK_STATUS_UNSPECIFIED = 0;
     TASK_STATUS_OPEN = 1;
     TASK_STATUS_DONE = 2;
   }
   ```
7. **Отдельные Request и Response:** у каждого RPC свои сообщения `XxxRequest` и `XxxResponse`, даже если они пока пустые — чтобы потом не сломать совместимость.
8. **Даты и время:** для моментов времени используйте `google.protobuf.Timestamp`, для длительности — `google.protobuf.Duration`. Календарные даты расписания передаются строками `YYYY-MM-DD`, время пар — `HH:mm` вместе с `time_zone`.
9. **Удаление полей:** номера удалённых полей не переиспользуются, а помечаются через `reserved 4; reserved "old_field";`.

---

## 3. Конфигурация Buf v2

`buf.yaml`:

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

Фрагмент `buf.gen.yaml` (полная конфигурация в репозитории `contracts` включает плагины и для Go, и для Python `.pyi`):

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

Dart-кодогенерация по `.proto` не используется — мобильное приложение общается с Gateway по HTTP/REST/SSE.

Команды проверки и генерации:

```bash
buf lint
buf breaking --against 'https://github.com/orbit-polytech/contracts.git#tag=v0.5.11' # актуальный базовый тег перед изменением
buf generate
```

---

## 4. `orbit.auth.v1`, `orbit.users.v1` и `orbit.ai.v1` (актуальный срез)

- **`orbit.auth.v1.AuthService`**: помимо `SendCode`, `VerifyCode`, `RefreshToken`, `Logout`, OAuth и управления внешними identities, содержит `GetAccount`, `UpdateAccountPreferences` (`primary_group`, `favorite_groups`), а также `BeginEmailChange` и `ConfirmEmailChange` (с `reauth_proof`).
- **`orbit.users.v1.UserService`**: `GetProfile`, `UpdateProfile` (`display_name`, `username`, `academic_group`, `course`, `avatar_url`; устаревшие `first_name`/`last_name`/`middle_name` помечены `deprecated`) и внутренний `GetBatchProfiles`.
- **`orbit.ai.v1.AIService`**:
  - `ChatStream` (server streaming со стадиями `START`, `RETRIEVAL`, `DELTA`, `CITATIONS`, `DONE`, `ERROR`, `CANCELLED`, полями `duration_ms` и `retry_after_sec`);
  - `CancelGeneration`;
  - `GetConversations`, `GetConversationHistory`, `DeleteConversation`;
  - `SubmitFeedback` (`FeedbackRating`: `UP`, `DOWN`, `UNSPECIFIED`).

В `ai-service/pyproject.toml` зависимость `orbit-contracts` закреплена Git-тегом `v0.5.10`:

```toml
[tool.uv.sources]
orbit-contracts = { git = "https://github.com/orbit-polytech/contracts.git", tag = "v0.5.10" }
```

В `ai-service` используйте `uv sync --locked`. Пакет включает `gen/python/orbit`, `.pyi` и `py.typed`. Руками сгенерированные файлы править нельзя: любые изменения делаются в `.proto` и перегенерируются через `task generate` и `task check`.

Исходники: [ai.proto](https://github.com/orbit-polytech/contracts/blob/main/proto/orbit/ai/v1/ai.proto), [auth.proto](https://github.com/orbit-polytech/contracts/blob/main/proto/orbit/auth/v1/auth.proto), [users.proto](https://github.com/orbit-polytech/contracts/blob/main/proto/orbit/users/v1/users.proto), [buf.gen.yaml](https://github.com/orbit-polytech/contracts/blob/main/buf.gen.yaml), [pyproject.toml](https://github.com/orbit-polytech/contracts/blob/main/pyproject.toml).
