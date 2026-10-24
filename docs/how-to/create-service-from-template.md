# Как создать новый микросервис из шаблона

Данное руководство описывает стандартизированный процесс создания нового Go-микросервиса в экосистеме Orbit AI на базе эталонного репозитория-шаблона `orbit-service-template`.


!!! info "Учебный сценарий"
    Существование и доступность `orbit-service-template` нужно проверить перед использованием. Дерево готового Tasks сохранено в `orbit-docs/examples/tasks-service`. Порт `50059` ниже — только пример для локального прототипа; согласуйте свободный порт до подключения. `50054` уже занят Schedule. Новому сервису не нужны Redis/RabbitMQ и actor-interceptor, если этого не требует его сценарий; публичное расписание — пример сервиса без actor.

---

## 1. Концепция и предварительные условия

Все бэкенд-сервисы Orbit AI (например, `schedule-service`, `navigation-service`, `notification-service`) строятся по единой структуре портов и адаптеров (Clean Architecture). Чтобы не писать заново бойлерплейт конфигурации, структурированного логирования (`slog`), gRPC-сервера и перехватчиков (аутентификация, recovery, логирование), используется репозиторий-шаблон:

- **Шаблон:** `https://github.com/orbit-polytech/orbit-service-template`
- **Целевая организация:** `https://github.com/orbit-polytech`
- **Соглашение по именованию:** `kebab-case` с суффиксом `-service` (например, `study-service`).

Перед началом убедитесь, что на рабочей машине установлены:

- Go 1.24+ (`go version`)
- Git 2.40+ (`git version`)
- Docker и Docker Compose v2 (`docker compose version`)

---

## 2. Пошаговый алгоритм

### Шаг 1. Создание репозитория из шаблона на GitHub

1. Перейдите на страницу шаблона `https://github.com/orbit-polytech/orbit-service-template`.
2. Нажмите зелёную кнопку **«Use this template»** -> **«Create a new repository»**.
3. В поле **Repository name** укажите имя вашего сервиса, например: `study-service`.
4. Выберите владельца: организация `orbit-polytech`.
5. Установите видимость: **Public**.
6. Нажмите **Create repository from template**.

---

### Шаг 2. Клонирование и переименование Go-модуля

Склонируйте созданный репозиторий на локальную машину:

```bash
git clone https://github.com/orbit-polytech/study-service.git
cd study-service
```

Замените базовый путь Go-модуля с шаблона на имя нового репозитория:

```bash
# Переименовываем имя модуля в go.mod
go mod edit -module github.com/orbit-polytech/study-service

# Обновляем пути импортов во всех внутренних файлах сервиса
# На Linux / macOS:
find . -type f -name "*.go" -not -path "./vendor/*" -exec sed -i 's|github.com/orbit-polytech/orbit-service-template|github.com/orbit-polytech/study-service|g' {} +

# На Windows (PowerShell):
Get-ChildItem -Filter *.go -Recurse | ForEach-Object {
    (Get-Content $_.FullName) -replace 'github.com/orbit-polytech/orbit-service-template', 'github.com/orbit-polytech/study-service' | Set-Content $_.FullName
}

# Актуализируем зависимости и хэши
go mod tidy
```

---

### Шаг 3. Подключение сгенерированных контрактов gRPC

Если ваш сервис реализует собственный gRPC API, сначала убедитесь, что контракт описан в `contracts` (см. рецепт [Как добавить новый gRPC-метод](add-grpc-method.md)).

Подключите сгенерированный Go-пакет контрактов:

```bash
go get github.com/orbit-polytech/contracts/gen/go/orbit/study/v1@main
go mod tidy
```

---

### Шаг 4. Настройка локальной конфигурации окружения

В корне нового сервиса скопируйте файл шаблона окружения:

```bash
cp .env.example .env
```

Отредактируйте `.env`, назначив сервису уникальные выделенные порты согласно сводной таблице [Порты и переменные окружения](../reference/ports-and-env.md):

```ini
APP_ENV=local
LOG_LEVEL=debug

# Уникальный gRPC-порт для study-service
GRPC_PORT=50059

# Подключение к базе данных PostgreSQL (выделенная БД сервиса)
DB_HOST=localhost
DB_PORT=5432
DB_USER=orbit
DB_PASSWORD=orbit_dev_secret
DB_NAME=study_db
DB_SSLMODE=disable

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
```

---

### Шаг 5. Регистрация базы данных и сервиса в `infra`

Базовый `infra/compose.yaml` — потребитель готовых образов, а не сборочный контекст соседних репозиториев.

1. Согласуйте контракт, отдельную базу, порт и необходимые зависимости. Миграции Goose храните в новом сервисе. Создание базы/роли добавьте в provisioner `infra/scripts/initdb/01-init-user.sh` отдельным проверенным изменением; `init-databases.sql` сейчас не используется. Проверьте повторный запуск на существующем volume.
2. Настройте Dockerfile и CI публикации образа сервиса и при необходимости мигратора в GHCR. Выдайте потребителям Read-доступ.
3. В базовый `compose.yaml` добавьте `image:`, явный набор env, healthcheck и зависимости. Не добавляйте `build: context: ../...` и не предполагайте наличие RabbitMQ/`orbit-net` в текущем стенде.
4. Сборку исходников подключайте **только в локальном `compose.override.yaml`**. Иллюстрация структуры override:

```yaml
services:
  study-service:
    build:
      context: ../study-service
      dockerfile: Dockerfile
```

Это дополнение к заранее описанному базовому сервису, не полный запускаемый Compose. Добавьте такой фрагмент в шаблон override, сам локальный файл не коммитьте.

5. Добавьте новый образ в список [доступа GHCR](../quickstart/ghcr-access.md), обновите [порты/env](../reference/ports-and-env.md), статус, Quickstart и проверьте запуск из одного клона `infra`.

---

### Шаг 6. Проверка сборки и тестов

Убедитесь, что сервис компилируется, проходит линтеры и тесты:

```bash
# 1. Форматирование кода
go fmt ./...

# 2. Проверка статическим анализатором
golangci-lint run ./...

# 3. Запуск юнит-тестов с детектором гонок
go test -v -race ./...

# 4. Пробный локальный запуск точки сборки
go run ./cmd/study-service/main.go
```

При успешном запуске в консоли появится структурированный лог:

```json
{"time":"2026-10-12T12:00:00Z","level":"INFO","msg":"gRPC-сервер запущен","port":50059,"env":"local"}
```

---

## 3. Сравнение: Как нельзя и Как правильно

<div class="grid" markdown>

<div class="card" markdown>
### :x: Как нельзя

- **Клонировать чужой рабочий сервис (например, tasks-service) с `git clone`, оставляя его историю коммитов.** В репозиторий попадут чужие миграции, бизнес-логика и старые теги.
- **Оставлять базовое имя модуля `orbit-service-template` в `go.mod`.** При установке пакетов через `go get` в других модулях возникнет коллизия импортов.
- **Использовать один порт `50051` для всех сервисов.** Локальный запуск нескольких сервисов в Docker упадет с ошибкой `bind: address already in use`.
- **Подключаться к чужой БД `tasks_db` вместо создания изолированной `study_db`.** Это грубое нарушение паттерна *Database-per-Service* (ADR-002).
</div>

<div class="card" markdown>
### :white_check_mark: Как правильно

- **Создавать репозиторий через GitHub кнопку «Use this template».** Получается чистая история коммитов с первого дня.
- **Выполнять `go mod edit -module github.com/orbit-polytech/<service-name>`.** Все импорты строго изолированы и однозначны.
- **Сверять номер порта с реестром `reference/ports-and-env.md`** и фиксировать его в конфигурации приложения и `compose.yaml`.
- **Создавать отдельную БД в PostgreSQL с отдельным владельцем**, чтобы сервисы не связывались на уровне внешних ключей (Foreign Keys).
</div>

</div>
