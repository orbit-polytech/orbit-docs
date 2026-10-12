# Запуск проекта с нуля на своём компьютере

Цель этого руководства: за 20–30 минут получить полностью работающий микросервисный стек на своей машине и успешно отправить первый запрос через `gateway-service`. 

Никаких сложных облачных сервисов и Kubernetes для разработки не требуется: вся база данных, брокеры сообщений и сервисы поднимаются локально через Docker.

---

## 1. Системные требования и инструменты

Убедитесь, что на вашем компьютере установлены следующие версии инструментов:

| Инструмент | Минимальная версия | Проверка в терминале |
|---|---|---|
| **Git** | 2.40+ | `git --version` |
| **Docker Engine & Compose** | Compose 2.22+ (с поддержкой `watch`) | `docker compose version` |
| **Go** | 1.25+ | `go version` |
| **Python** | 3.12+ | `python3 --version` |
| **uv** (менеджер пакетов Python) | свежая | `uv --version` |
| **Flutter** | stable (3.24+) | `flutter doctor` |
| **Buf CLI** | 1.40+ (v2 config) | `buf --version` |
| **grpcurl** | свежая | `grpcurl --version` |
| **Make** | любая | `make --version` |

### Рекомендация для пользователей Windows
На Windows рекомендуется работать внутри **WSL 2 (Ubuntu)**. 
Репозитории клонируйте внутрь файловой системы WSL (например, в `~/orbit`), а не в смонтированный диск `/mnt/c/...`, иначе сборка Go и дисковый ввод-вывод Docker будут работать заметно медленнее.

Установка консольных Go-утилит одной командой:

```bash
go install github.com/bufbuild/buf/cmd/buf@latest
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

Убедитесь, что путь `$(go env GOPATH)/bin` добавлен в системную переменную `PATH`.

---

## 2. Шаг 1. Клонирование репозиториев

Репозитории клонируются в одну общую рабочую директорию рядом друг с другом. Docker Compose в репозитории `infra` находит исходный код сервисов по относительным путям `../<сервис>`:

```bash
mkdir -p ~/orbit && cd ~/orbit

# Инфраструктура, контракты и базовые сервисы
git clone git@github.com:orbit-polytech/infra.git
git clone git@github.com:orbit-polytech/contracts.git
git clone git@github.com:orbit-polytech/gateway-service.git
git clone git@github.com:orbit-polytech/auth-service.git
git clone git@github.com:orbit-polytech/tasks-service.git
# Остальные сервисы (schedule, navigation, ai, mobile) клонируются по мере работы
```

Итоговая структура папок на диске:

```text
~/orbit/
├── infra/
├── contracts/
├── gateway-service/
├── auth-service/
├── tasks-service/
└── mobile/
```

---

## 3. Шаг 2. Переменные окружения и ключи

Перейдите в папку инфраструктуры и скопируйте шаблон переменных окружения:

```bash
cd ~/orbit/infra
cp .env.example .env
```

Значения по умолчанию из `.env.example` уже оптимизированы для локальной разработки. Секретный ключ Google Gemini (`GEMINI_API_KEY`) и токен Telegram-бота (`TELEGRAM_BOT_TOKEN`) каждый разработчик генерирует для себя индивидуально и вносит только в свой локальный `.env`. Файл `.env` никогда не коммитится в Git.

Сгенерируйте пару асимметричных ключей **Ed25519** для подписи JWT:

```bash
make keys
# Создаст пару: secrets/jwt_ed25519 (приватный) и secrets/jwt_ed25519.pub (публичный)
```

Директория `secrets/` также занесена в `.gitignore`.

---

## 4. Шаг 3. Запуск инфраструктурных контейнеров

Запустите контейнеры баз данных и брокеров:

```bash
docker compose up -d postgres redis rabbitmq jaeger
docker compose ps
```

Дождитесь, пока контейнеры перейдут в статус `healthy`. Быстрая проверка доступности:

```bash
# Проверка PostgreSQL 16
docker compose exec postgres psql -U orbit -d orbit -c 'SELECT 1;'

# Проверка Redis 7
docker compose exec redis redis-cli ping
# Ответ: PONG

# Проверка веб-интерфейсов:
# RabbitMQ Management: http://localhost:15672 (логин/пароль: guest / guest)
# Jaeger UI трассировки: http://localhost:16686
```

---

## 5. Шаг 4. Накатывание миграций баз данных

Каждый сервис изолированно управляет своей схемой БД в общей инстанции PostgreSQL (`search_path=auth`, `search_path=tasks` и т.д.):

```bash
cd ~/orbit/auth-service && make migrate-up
cd ~/orbit/tasks-service && make migrate-up
```

---

## 6. Шаг 5. Запуск сервисов

Есть два основных сценария локальной разработки:

### Сценарий А: Запуск всех сервисов в Docker (удобно для фронтендеров и новичков)
```bash
cd ~/orbit/infra
docker compose --profile core up -d
# Поднимет auth-service, tasks-service и gateway-service
```

### Сценарий Б: Сервис разрабатывается на хосте, остальное в Docker (для бэкендеров)
Если вы прямо сейчас редактируете `tasks-service`, остановите его docker-контейнер и запустите код локально на хосте под отладчиком в VS Code:

```bash
cd ~/orbit/infra
docker compose --profile core up -d
docker compose stop tasks-service

# Запуск сервиса на хосте
cd ~/orbit/tasks-service
cp .env.example .env
make run
```

Шлюз `gateway-service` автоматически свяжется с сервисом на хост-машине через `host.docker.internal:50055`.

Для режима «меняю код на диске — контейнер сам пересобирается на лету» используйте встроенную функцию Docker:

```bash
docker compose --profile core watch
```

---

## 7. Шаг 6. Проверка работы и первый запрос

### Проверка gRPC напрямую через `grpcurl`:
*(gRPC reflection активирован только в режиме `APP_ENV=local`)*

```bash
grpcurl -plaintext localhost:50055 list
grpcurl -plaintext localhost:50055 grpc.health.v1.Health/Check
```

### Проверка через единый HTTP Gateway (как это делает приложение):

```bash
# 1. Получаем тестовый токен через dev-логин (доступен только при APP_ENV=local)
curl -s -X POST http://localhost:8080/v1/auth/dev-login \
  -H 'Content-Type: application/json' \
  -d '{"telegram_id": 1001}' | tee /tmp/login.json

TOKEN=$(jq -r .access_token /tmp/login.json)

# 2. Создаем задачу через HTTP API с ключом идемпотентности
curl -s -X POST http://localhost:8080/v1/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -H "Idempotency-Key: $(uuidgen)" \
  -d '{"title": "Сдать 1-ю лабораторную по базам данных", "due_at": "2026-11-01T18:00:00Z"}'

# 3. Запрашиваем список задач текущего студента
curl -s http://localhost:8080/v1/tasks -H "Authorization: Bearer $TOKEN" | jq .
```

Откройте панель **Jaeger** на [http://localhost:16686](http://localhost:16686), выберите сервис `gateway` и найдите созданный трейс. Вы увидите наглядную диаграмму того, как запрос прошел через шлюз, gRPC-интерцепторы и обратился к PostgreSQL.

---

## 8. Шаг 7. Запуск мобильного приложения на Flutter

```bash
cd ~/orbit/mobile
flutter pub get
flutter run -t lib/main_development.dart --dart-define=API_URL=http://localhost:8080
```

> [!NOTE]
> Для Android-эмулятора локальный хост доступен по специальному IP: `10.0.2.2`. 
> Запуск для Android: `--dart-define=API_URL=http://10.0.2.2:8080`.

---

## Решение типовых проблем

| Симптом | Причина | Что сделать |
|---|---|---|
| `bind: address already in use (5432)` | Порт занят системным Postgres | Остановите локальную службу: `sudo service postgresql stop` или проверьте `lsof -i :5432`. |
| Сервис падает с `connection refused` к Postgres | БД ещё не успела пройти инициализацию | Проверьте в `compose.yaml` условие `condition: service_healthy`. |
| `relation "tasks.tasks" does not exist` | Не применены миграции | Запустите шаг 4: `make migrate-up` в директории сервиса. |
| Gateway возвращает `401 Unauthorized` | Истёк 15-минутный access-токен или были пересозданы ключи в `secrets/` | Получите новый токен через `/v1/auth/dev-login`. |
| Gateway возвращает `503 Service Unavailable` | Целевой микросервис не запущен | Проверьте `docker compose ps` или запустите сервис на хосте. |

Полный сброс и очистка всех контейнеров и томов:

```bash
cd ~/orbit/infra
docker compose --profile "*" down -v
```
