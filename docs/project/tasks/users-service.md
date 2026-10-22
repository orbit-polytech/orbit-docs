# Техническое задание: Разработка сервиса профилей (users-service) и gRPC-контрактов

- **Исполнитель**: Илья Мозжухин (Backend / Database Engineer)
- **Tech Lead & Software Architect:** Данил Мусатов
- **Проект**: Orbit AI
- **Стек**: Go 1.26, gRPC / Protobuf, PostgreSQL 16 (драйвер `pgx/v5`), Goose v3.24.1, Docker, GitHub Actions

---

## Как работать с этим заданием

Илья, предлагаю тебе взять `users-service`. Ниже общий план, чтобы было понятно, к какому результату идём. Сначала подними стенд и посмотри модель и контракт. Затем напиши вопросы и предложения — до согласования не нужно пытаться реализовать все этапы сразу.

План сохраняет подробные примеры, но они не заменяют обсуждение продуктовых решений. Названия и категории определяем под наши потребности, а не требования университета. Статус страницы — **задание в работе / проект решений**, не описание уже реализованного Users.

**Первый результат, который жду:** работающий локальный стенд, замечания к полям профиля и предложение по контракту. Дальше согласуем первый PR. Структуру проекта и архитектурные правила можно изучать в существующих учебных разделах `orbit-docs`.

## 1. Контекст и архитектурные границы

Микросервис `users-service` отвечает за хранение и управление профилями пользователей Orbit AI. ФИО, группа, курс и категории ниже — предлагаемые поля продукта; их обязательность и смысл согласуем до выпуска контракта. Отдельное хранилище произвольных предпочтений пока не входит в реализацию.

### Границы ответственности между Auth и Users
1. `auth-service` управляет только идентификацией и безопасностью: учетные записи, OTP-коды, привязки Google OAuth, сессии и генерация Ed25519 JWT-токенов.
2. `users-service` управляет данными человека: ФИО, аватар, академическая группа, курс, роль и предпочтения.
3. Базы данных сервисов изолированы. В схеме `users-service` нет внешних ключей (Foreign Keys) на базу `auth-service`. Связь между системами осуществляется логически через `account_id` (UUID), который API Gateway извлекает из проверенного JWT-токена и передает в gRPC-запросе.

### Гостевой режим

- Просмотр открытых данных, например публичного расписания группы, не требует аккаунта, JWT или записи в Users.
- Гость может выбрать группу и сохранить её локально на устройстве. Синхронизация выбора с профилем после входа — отдельный сценарий, не часть этого backend-задания.
- Users обслуживает профили пользователей с аккаунтом и не является обязательной зависимостью публичных маршрутов.
- Gateway защищает личные маршруты JWT middleware, а открытые регистрирует отдельно. Ограничение частоты, проверка ввода и фильтрация публичных данных нужны и для гостей.
- Маршруты расписания и сами публичные данные не реализуются в рамках Users.

### Жизненный цикл профиля — предлагаемый вариант

До слияния контракта согласуем следующие правила:

1. Регистрация в Auth автоматически не создаёт профиль.
2. `GetProfile` отсутствующего профиля возвращает `NotFound`; GET ничего не создаёт.
3. `UpdateProfile` в этом задании означает создание или **полную замену редактируемых полей**. Ему соответствует HTTP PUT, а не PATCH.
4. При первом корректном сохранении создаётся профиль; при последующих сохраняются `id`, `account_id` и `created_at`.
5. Пустая строка очищает необязательное текстовое поле. Отсутствие `course` очищает курс. Обязательные поля должны передаваться при каждом PUT.
6. `updated_at` выставляется сервером при успешной записи. Специальную семантику no-op timestamp для первой версии не вводим.
7. Для одного `account_id` существует не более одного профиля. Сохранение атомарно; конкурентные записи не создают дубли. Первая версия — last-write-wins, без контроля версии профиля.

Название `UpdateProfile` оставляем для преемственности исходного плана; поведение create-or-replace обязательно описываем в protobuf-комментариях и Swagger. При желании переименуем метод до первого релиза, но не меняем смысл без согласования.

### Роли и границы доверия

- Предлагаемый `UserRole` сохраняем в черновике, но это **не разрешение назначать себе права**. Категория профиля и административные полномочия — разные понятия.
- `role` отсутствует в запросе изменения. Начальное значение — `UNSPECIFIED`; механизм назначения категорий не входит в этот этап.
- Наличие значения `ADMIN` в профильном enum само по себе не предоставляет доступ к административным операциям. До использования роли для авторизации нужна отдельно согласованная модель доступа.
- Если категории сейчас не нужны, исключим их до первого выпуска контракта. Не добавляем роли только ради полноты примера.
- HTTP-клиент не задаёт `account_id`: Gateway получает его из проверенного JWT-контекста. Серверные поля и `role` не принимаются в HTTP DTO профиля.
- Users доступен доверенным внутренним клиентам. Порт gRPC не публикуется на хост по умолчанию; проверка UUID не заменяет проверку прав.
- Для production за пределами доверенной локальной сети нужно отдельно определить аутентификацию внутренних клиентов и защиту транспорта.

### Распределение работы

| Зона | Ответственный |
|---|---|
| Проект contracts, реализация Users, миграции, тесты, Docker и CI | Илья |
| Согласование и выпуск contracts, ревью | Данил |
| Общая БД/роли в infra, Compose и интеграция Gateway | Данил, с участием Ильи в проверках |

Интеграционные шаги ниже — общая работа, а не требование Илье самостоятельно менять все репозитории.


---

## 2. Требования к Git Workflow и коммитам

В проекте действует строгое правило атомарности изменений:

1. **Запрещены монолитные коммиты** вида «реализовал весь сервис» или «добавил код».
2. Коммиты делаем логическими и удобными для ревью. Примеры разделения (не требование коммита на каждый файл):
   - Создана схема БД и миграция — отдельный коммит.
   - Описаны доменные модели и интерфейс репозитория — отдельный коммит.
   - Реализован репозиторий на pgx — отдельный коммит.
   - Добавлены тесты репозитория — отдельный коммит.
   - Написан usecase — отдельный коммит.
   - Реализован gRPC-хэндлер — отдельный коммит.
3. Формат сообщений — Conventional Commits:
   - `feat(proto): define users v1 messages and service`
   - `feat(db): add profiles migration`
   - `feat(domain): define user profile entity and errors`
   - `feat(postgres): implement profile repository`
   - `test(postgres): add repository integration tests`
   - `feat(usecase): add get and update profile usecases`
   - `feat(grpc): implement users service gRPC server`
4. Вся разработка ведется в ветке `feature/init-users-service` (в репозитории `contracts` — в ветке `feature/users-v1`).
5. Draft PR открываем рано. Контракт, рабочий срез сервиса и поставку проверяем поэтапно, не ждём конца всей работы. После ревью и успешных проверок — **Squash and merge** по принятому workflow. Не требуется держать всю разработку одним огромным PR.

---

## 3. Стадии и пошаговый план реализации

### Стадия 0. Подготовка окружения и проверка инфра-стенда
**Цель**: Убедиться, что базовый стек Orbit AI запускается на локальной машине разработчика.

- **Шаг 0.1**. Принять приглашение в GitHub-организацию `orbit-polytech`.
- **Шаг 0.2**. Получить Read-доступ ко всем четырём private-пакетам и создать собственный PAT classic с `read:packages`. Токен не сохранять в `.env`, Git или переписке; при необходимости авторизовать его для SSO организации. [Инструкция GHCR](https://orbit-polytech.github.io/orbit-docs/quickstart/ghcr-access/).
- **Шаг 0.3**. Авторизовать локальный Docker в реестре GitHub:
  ```bash
  docker login ghcr.io -u <github_username>
  ```
- **Шаг 0.4**. Склонировать репозиторий `infra`, скопировать `.env.example` в `.env` и поднять стенд:
  ```bash
  git clone https://github.com/orbit-polytech/infra.git
  cd infra
  cp .env.example .env
  # Заполнить локальные параметры в .env; секреты получить отдельно.
  task up
  task status
  ```
  *Критерий завершения*: сервисы `postgres`, `redis`, `auth-service`, `gateway-service` имеют статус `Up (healthy)`, мигратор и генератор ключей завершились с кодом `0`. Swagger UI отвечает по адресу `http://localhost:8080/swagger/index.html`.

---

### Стадия 1. Определение контрактов в репозитории `contracts`
**Цель**: Спроектировать gRPC API до написания серверного кода.

- **Шаг 1.1**. В склонированном репозитории `contracts` создать ветку:
  ```bash
  git checkout -b feature/users-v1
  ```
- **Шаг 1.2**. Создать спецификацию `proto/orbit/users/v1/users.proto`:
  ```protobuf
  syntax = "proto3";

  package orbit.users.v1;

  option go_package = "github.com/orbit-polytech/contracts/gen/go/orbit/users/v1;usersv1";

  import "google/protobuf/timestamp.proto";

  enum UserRole {
    USER_ROLE_UNSPECIFIED = 0;
    USER_ROLE_STUDENT = 1;
    USER_ROLE_TEACHER = 2;
    USER_ROLE_ADMIN = 3;
  }

  message UserProfile {
    string id = 1;
    string account_id = 2;
    string first_name = 3;
    string last_name = 4;
    string middle_name = 5;
    string avatar_url = 6;
    string academic_group = 7;
    optional int32 course = 8;
    UserRole role = 9;
    google.protobuf.Timestamp created_at = 10;
    google.protobuf.Timestamp updated_at = 11;
  }

  message GetProfileRequest {
    string account_id = 1;
  }

  message GetProfileResponse {
    UserProfile profile = 1;
  }

  // Создаёт профиль либо полностью заменяет его редактируемые поля.
  // Пустые необязательные строки и отсутствие course очищают значения.
  message UpdateProfileRequest {
    string account_id = 1;
    string first_name = 2;
    string last_name = 3;
    string middle_name = 4;
    string avatar_url = 5;
    string academic_group = 6;
    optional int32 course = 7;
  }

  message UpdateProfileResponse {
    UserProfile profile = 1;
  }

  message GetBatchProfilesRequest {
    repeated string account_ids = 1;
  }

  message GetBatchProfilesResponse {
    repeated UserProfile profiles = 1;
  }

  service UserService {
    rpc GetProfile(GetProfileRequest) returns (GetProfileResponse);
    rpc UpdateProfile(UpdateProfileRequest) returns (UpdateProfileResponse);
    // Внутренний метод; не публикуется как открытый HTTP endpoint.
    rpc GetBatchProfiles(GetBatchProfilesRequest) returns (GetBatchProfilesResponse);
  }
  ```
**Ограничения GetBatchProfiles:** внутренний метод только для согласованных доверенных потребителей; максимум 100 входных ID до удаления дублей; любой невалидный UUID или превышение лимита → `InvalidArgument`. Отсутствующие профили пропускаются, дубли удаляются, оставшиеся идут в порядке первого появления ID. Метод не получает публичного HTTP-маршрута; полные профили выдаются только при согласованной необходимости. Если первого потребителя нет, реализацию batch можно вынести в следующий PR после обсуждения, не выдавая незавершённый метод за готовый.

- **Шаг 1.3**. Сверить команды с актуальным Taskfile contracts и запустить форматирование, lint и предусмотренную проверку совместимости. Ожидаемые команды:
  ```bash
  task format
  task lint
  ```
- **Шаг 1.4**. Сгенерировать Go-код:
  ```bash
  task generate
  ```
  Убедиться, что в каталоге `gen/go/orbit/users/v1/` появились файлы `users.pb.go` и `users_grpc.pb.go`.
- **Шаг 1.5**. Закоммитить изменения: `feat(proto): define users v1 contracts and generate go code`. Запушить ветку и открыть PR. После мерджа я публикую согласованную версию contracts. Затем подключаем её точным тегом в Users.

---

### Стадия 2. Инициализация репозитория `users-service`
**Цель**: Развернуть структуру проекта в соответствии со стандартами Clean Architecture.

- **Шаг 2.1**. Инициализировать модуль:
  ```bash
  go mod init github.com/orbit-polytech/users-service
  ```
- **Шаг 2.2**. Подключить основные зависимости:
  ```bash
  # Подставить реальный согласованный тег, не запускать с буквальным placeholder.
  go get github.com/orbit-polytech/contracts@<published-version>
  # pgx/v5, grpc и uuid подключить версиями, согласованными с текущим
  # toolchain, генерируемым кодом и принятыми зависимостями проекта.
  ```
- **Шаг 2.3**. Сформировать структуру каталогов:
  ```text
  users-service/
  ├── cmd/
  │   └── users/
  │       └── main.go
  ├── internal/
  │   ├── config/
  │   │   ├── config.go
  │   │   └── config_test.go
  │   ├── domain/
  │   │   ├── profile.go
  │   │   └── errors.go
  │   ├── infrastructure/
  │   │   └── postgres/
  │   │       ├── pool.go
  │   │       ├── profile_repository.go
  │   │       └── profile_repository_test.go
  │   ├── usecase/
  │   │   ├── ports.go
  │   │   ├── get_profile.go
  │   │   ├── get_profile_test.go
  │   │   ├── update_profile.go
  │   │   └── update_profile_test.go
  │   └── transport/
  │       └── grpc/
  │           ├── server.go
  │           ├── handler.go
  │           ├── handler_test.go
  │           └── errors.go
  ├── migrations/
  │   └── 00001_create_profiles.sql
  ├── Taskfile.yml
  ├── .golangci.yml
  ├── Dockerfile
  ├── Dockerfile.migrate
  ├── .dockerignore
  ├── .env.example
  ├── .gitignore
  ├── README.md
  ├── go.mod
  └── go.sum
  ```
**Расположение кода:** domain не импортирует protobuf, gRPC и pgx; usecase содержит сценарии и нужные им порты; postgres реализует порт и SQL; transport преобразует DTO и ошибки; main только собирает зависимости. Интерфейс репозитория размещаем у потребителя в `internal/usecase/ports.go`. Примеры и объяснения слоёв остаются в учебной документации.

- **Шаг 2.4**. Закоммитить: `chore: initialize users-service project structure`.

---

### Стадия 3. Схема данных PostgreSQL и миграции Goose
**Цель**: Спроектировать реляционную модель, написать SQL-миграцию и проверить ее обратимость.

**Изоляция БД:** для Users используем отдельную базу (предлагается `orbit_users`) и отдельную историю Goose. Один локальный PostgreSQL-сервер может обслуживать обе базы, но DSN Auth не используется. Данил подготавливает базу/роли, включая существующий volume: initdb-скрипты автоматически повторно не запускаются. Миграционная роль имеет необходимые DDL-права, runtime-роль — только требуемые права чтения/записи. Пароли не коммитятся.

- **Шаг 3.1**. Создать миграцию `migrations/00001_create_profiles.sql`:
  ```sql
  -- +goose Up
  -- +goose StatementBegin
  CREATE TABLE profiles (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      account_id UUID NOT NULL UNIQUE,
      first_name VARCHAR(100) NOT NULL CHECK (char_length(btrim(first_name)) > 0),
      last_name VARCHAR(100) NOT NULL DEFAULT '',
      middle_name VARCHAR(100) NOT NULL DEFAULT '',
      avatar_url TEXT NOT NULL DEFAULT '' CHECK (octet_length(avatar_url) <= 2048),
      academic_group VARCHAR(50) NOT NULL DEFAULT '',
      course SMALLINT CHECK (course BETWEEN 1 AND 6),
      role VARCHAR(20) NOT NULL DEFAULT 'unspecified'
          CHECK (role IN ('unspecified', 'student', 'teacher', 'admin')),
      created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
      updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
  );

  -- UNIQUE(account_id) уже создаёт индекс.
  -- Индекс по academic_group добавим, когда появится соответствующий запрос.
  -- +goose StatementEnd

  -- +goose Down
  -- +goose StatementBegin
  DROP TABLE IF EXISTS profiles;
  -- +goose StatementEnd
  ```
Здесь имя обязательно, фамилия и остальные текстовые поля необязательны. Курс nullable, а диапазон 1–6 — предлагаемое продуктовое ограничение, которое согласуем до первого релиза. Пробелы нормализуются приложением; длины текстовых полей проверяются в Unicode code points. При изменении согласованных правил одновременно меняются контракт, domain, SQL и тесты.

`avatar_url` — пусто либо абсолютный HTTPS URL длиной не более 2048 байт. Users не скачивает URL и не проксирует ресурс. Роль не меняется обычным Upsert; при создании всегда `unspecified`.

- **Шаг 3.2**. Проверить миграцию только на отдельной одноразовой базе разработки/тестов. Down удаляет таблицу и данные — не запускать на общей БД. Пример для Bash (переменная должна быть экспортирована; `.env` не загружается сама):
  ```bash
  goose -dir migrations postgres "$USERS_MIGRATION_DSN" up
  goose -dir migrations postgres "$USERS_MIGRATION_DSN" status
  goose -dir migrations postgres "$USERS_MIGRATION_DSN" down
  goose -dir migrations postgres "$USERS_MIGRATION_DSN" up
  ```
- **Шаг 3.3**. Закоммитить: `feat(db): add profiles table goose migration`.

---

### Стадия 4. Доменный слой и репозиторий PostgreSQL
**Цель**: Реализовать интерфейс доступа к данным через пул `pgxpool.Pool`.

- **Шаг 4.1**. Создать `internal/domain/profile.go` и `errors.go`:
  - Структура `Profile` с проверкой бизнес-правил (непустое имя, валидный курс).
  - Ошибки `ErrProfileNotFound`, `ErrInvalidInput`.
- **Шаг 4.2**. Описать интерфейс репозитория в `internal/usecase/ports.go` (импортируя domain):
  ```go
  type ProfileRepository interface {
      GetByAccountID(ctx context.Context, accountID uuid.UUID) (*domain.Profile, error)
      GetBatchByAccountIDs(ctx context.Context, accountIDs []uuid.UUID) ([]*domain.Profile, error)
      Upsert(ctx context.Context, profile *domain.Profile) (*domain.Profile, error)
  }
  ```
- **Шаг 4.3**. Реализовать `internal/infrastructure/postgres/profile_repository.go` с безопасными параметризованными запросами (`$1, $2`).
- **Шаг 4.4**. Написать интеграционные тесты на отдельной реальной PostgreSQL-базе: создание/чтение, NotFound, замена и очистка полей, nullable course, уникальность account_id, конкурентные записи и batch по согласованным правилам. Проверить сохранение id/created_at и серверный updated_at. Upsert атомарный, не меняет role и возвращает сохранённую модель через RETURNING. Тесты не чистят чужие базы; в CI отсутствие тестового DSN должно завершать job ошибкой, а не молча пропускать проверки.
- **Шаг 4.5**. Закоммитить: `feat(postgres): implement profile repository with pgxpool`.

---

### Стадия 5. Use Cases (Бизнес-логика)
**Цель**: Реализовать прикладные сценарии независимо от транспортного протокола.

- **Шаг 5.1**. Написать `internal/usecase/get_profile.go` (валидация UUID, вызов репозитория).
- **Шаг 5.2**. Написать `internal/usecase/update_profile.go` (валидация входных полей, сохранение изменений).
- **Шаг 5.3**. Покрыть use cases unit-тестами с mock/fake-репозиторием: обязательное имя, границы длин, Unicode, UUID, nullable course, URL, ошибки repository и отмена context. Если batch остаётся в первом релизе, добавить отдельный use case и тесты; handler не обращается к repository напрямую.
- **Шаг 5.4**. Закоммитить: `feat(usecase): implement get and update profile usecases`.

---

### Стадия 6. Транспортный уровень gRPC
**Цель**: Предоставить gRPC API согласно сгенерированному интерфейсу.

- **Шаг 6.1**. Реализовать хэндлеры в `internal/transport/grpc/handler.go`:
  - `GetProfile`, `UpdateProfile`, `GetBatchProfiles`.
  - Преобразование доменных моделей в Protobuf DTO.
- **Шаг 6.2**. Настроить единый mapper: NotFound, InvalidArgument, Internal; отмена/дедлайн сохраняются как Canceled/DeadlineExceeded. Неожиданные ошибки не раскрывают SQL, DSN и внутренние сообщения.
- **Шаг 6.3**. Подключить `grpc.health.v1`: начальный SERVING только после готовности БД, NOT_SERVING при остановке. Не считать probe непрерывной проверкой БД, если динамическое обновление health не реализовано.
- **Шаг 6.4**. Написать тесты хэндлеров через in-memory транспорт `google.golang.org/grpc/test/bufconn`.
- **Шаг 6.5**. Закоммитить: `feat(grpc): implement gRPC user service handler and health check`.

---

### Стадия 7. Точка входа, Graceful Shutdown и Dockerfile
**Цель**: Собрать исполняемый бинарник и подготовить контейнеризацию.

- **Шаг 7.1**. В `cmd/users/main.go` настроить чтение переменных окружения, инициализацию пула БД, запуск gRPC сервера и корректную остановку по сигналам `SIGINT/SIGTERM` с ограничением времени GracefulStop и принудительной остановкой по таймауту. Не логировать секреты и полные профили.
- **Шаг 7.2**. Подготовить `Dockerfile` по эталону проекта:
  - Использовать `# syntax=docker/dockerfile:1`.
  - Multi-stage: загрузка `grpc_health_probe`, компиляция в `golang:1.26-alpine` с BuildKit cache mounts (`--mount=type=cache`), минимальный финальный образ на `alpine:3.21`.
  - Запуск под non-root пользователем `USER 10001:10001`, предлагаемый внутренний порт `50052` (перед выпуском сверить карту портов).
  - Probe закреплённой версии под целевую архитектуру с проверкой checksum; CA certificates в runtime.
  - `.dockerignore` исключает секреты, `.env`/`.env.*`, ключи и временные файлы; безопасный `.env.example` можно явно разрешить.
- **Шаг 7.3**. Подготовить `Dockerfile.migrate` (Goose + директория `migrations/`).
- **Шаг 7.4**. Настроить CI: PostgreSQL 16, миграции, реальные integration tests, `go test -race`, lint и сборка двух Docker-образов. На PR нет публикации и права packages:write. После успешных проверок main/релизов — публикация согласованных образов `users-service` и `users-migrate` в GHCR по принятому workflow. В README описать env, команды, создание профиля и ограничения. Runtime получает только свой DSN и необходимые настройки; JWT-ключи, Redis и Resend Users не нужны.
- **Шаг 7.5**. Закоммитить: `feat(docker): add multi-stage Dockerfile and CI workflow`.

---

### Стадия 8. Интеграция в infra и сдача работы
**Цель**: Вместе включить сервис в общий запуск и проверить через Gateway. Изменения infra/Gateway ведёт Данил, Илья предоставляет параметры и участвует в проверке.

- **Шаг 8.1**. Подготовить отдельную базу/роли Users и добавить в Compose сервис Users и его мигратор. Базовый Compose запускает опубликованные образы без соседних исходников; gRPC доступен во внутренней сети. Users ждёт успешных миграций и готовности БД.
- **Шаг 8.2**. В `gateway-service` настроить клиент к `users-service` и зарегистрировать маршруты:
  - `GET /api/v1/users/me`
  - `PUT /api/v1/users/me`
- **Шаг 8.3**. Выполнить сквозную проверку:
  1. Авторизоваться через Auth и получить JWT-токен.
  2. GET до заполнения профиля → 404, без создания записи.
  3. PUT корректного профиля → 200 с сохранённым профилем; затем GET → те же данные.
  4. Повторный PUT не создаёт дубликат и не меняет id/created_at. Пустые необязательные поля и отсутствие курса очищают их.
  5. Невалидные поля → 400 без частичного сохранения; запрос без/с невалидным JWT → 401.
  6. Второй аккаунт не может прочитать или изменить профиль первого; account_id из клиентского тела не определяет владельца.
  7. Публичные данные приложения доступны гостю без создания профиля (проверять на реально существующем публичном маршруте; будущие маршруты расписания не выдавать за реализованные).
  8. После перезапуска профиль сохраняется. Чистый запуск из одного infra скачивает Users и мигратор без override и локальных сборок.
- **Шаг 8.4**. Провести финальный прогон проверок:
  ```bash
  golangci-lint run ./...
  go test -v -race -cover ./...
  ```
- **Шаг 8.5**. Завершить ранее открытый PR в Users, приложить результаты проверок, известные ограничения и ссылки на интеграционные PR. Я провожу финальное ревью; после успешных проверок выполняем Squash-слияние.

---

## 4. Контрольные точки и приёмка

### Сначала — обсуждение

Не нужно за вечер реализовывать весь сервис. Жду от Ильи: результат запуска стенда, замечания к полям и ролям, предложение по жизненному циклу и черновик контракта. Открытые решения фиксируем до слияния proto.

### Users готов к приёмке

- [ ] Контракт согласован, опубликован и подключён точной версией.
- [ ] Get/Update реализованы с описанной create-or-replace семантикой.
- [ ] Профили не смешиваются между аккаунтами, поля и ошибки валидируются.
- [ ] SQL и protobuf не нарушают границы слоёв.
- [ ] Миграции и repository tests работают на отдельной Users-базе в CI.
- [ ] Unit/bufconn tests проходят; batch либо полностью реализован по требованиям, либо явно вынесен из первого релиза до публикации контракта.
- [ ] Образы сервиса и мигратора собираются, README и безопасный env-шаблон готовы.
- [ ] В PR указаны реально выполненные проверки, а не только обещания.

### Общая интеграция готова

- [ ] Данил подключил Users к Gateway и infra.
- [ ] Мигратор завершается с кодом 0, сервис healthy.
- [ ] Проверены два аккаунта и отрицательные сценарии доступа.
- [ ] Образы доступны команде, чистый запуск проверен.
- [ ] Swagger и текущее состояние документации обновлены.

## 5. Что остаётся за рамками

Загрузка аватаров, пользовательский поиск, выдача ролей/прав, RabbitMQ, автоматическая синхронизация регистрации/удаления аккаунтов и полноценное production service-to-service security требуют отдельных задач. До production нужно определить удаление профиля и политику персональных данных.

## 6. Где хранится задание

Рекомендуемый путь в `orbit-docs`: `docs/project/tasks/users-service.md`, название страницы — **«Разработка users-service»**. Навигация: **Проект → Задачи → Разработка users-service**.

Эта страница описывает конкретную работу и её этапы. Постоянный паспорт сервиса остаётся в `reference/services/`; после выполнения в него переносится описание фактического поведения. Текущие статусы реализации, PR и обсуждение ведём в GitHub Issues/Projects и даём на них ссылки здесь. Название страницы не привязываем навсегда к одному исполнителю.
