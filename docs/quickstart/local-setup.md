# Запуск актуального стенда

Цель руководства — поднять текущий backend-стенд Orbit AI и проверить Gateway. Для обычного запуска не нужны Go, Python, Flutter, Buf или Goose: сервисы работают в Docker-контейнерах.

!!! info "Статус"
    Образы Orbit опубликованы в GHCR. Для приватных пакетов нужны персональная авторизация Docker и Read-доступ ко всем шести пакетам. Публикация образов не заменяет проверку запуска из чистого клона. Сводка компонентов — на странице [«Текущее состояние»](../project/current-state.md).

## 1. Требования

| Инструмент | Требование |
|---|---|
| Git | 2.40+ |
| Docker | Docker Engine/Desktop с Compose v2 |
| Task | Taskfile 3.x |

Рекомендуется не менее 4 ГБ свободной RAM и 5 ГБ свободного места для образов и volumes.

## 2. Клонирование и конфигурация

```bash
git clone https://github.com/orbit-polytech/infra.git
cd infra
cp .env.example .env
```

На Windows PowerShell:

```powershell
git clone https://github.com/orbit-polytech/infra.git
Set-Location infra
Copy-Item .env.example .env
```

Откройте `.env` и замените значения `change_this_*` и `your_secret_*` случайными локальными значениями.

Для запуска и healthcheck достаточно непустых значений Resend. Для **реальной отправки OTP** нужны действующий `RESEND_API_KEY` и разрешённый `RESEND_FROM_EMAIL`. Для Google OAuth дополнительно заполните Google credentials и установите `GOOGLE_OAUTH_ENABLED=true`.

Никогда не добавляйте локальный `.env` в Git.

## Доступ к приватным образам GHCR

Перед первым `task up` выполните [инструкцию авторизации в GHCR](ghcr-access.md): получите Read-доступ, создайте PAT classic с `read:packages`, войдите через `docker login ghcr.io -u YOUR_GITHUB_USERNAME` и проверьте скачивание всех шести образов. Для публичных пакетов этот шаг не нужен.

**PAT не записывается в `.env` и не передаётся контейнерам.** Авторизация в реестре не является авторизацией в API Orbit.

## 3. Запуск

```bash
task up
task status
```

Ожидаемый результат:

| Контейнер | Ожидаемый статус |
|---|---|
| `postgres` | `Up (healthy)` |
| `redis` | `Up (healthy)` |
| `jwt-keygen` | `Exited (0)` |
| `postgres-init` | `Exited (0)` |
| `auth-migrate` | `Exited (0)` |
| `schedule-migrate` | `Exited (0)` |
| `auth-service` | `Up (healthy)` |
| `schedule-service` | `Up (healthy)` |
| `gateway-service` | `Up (healthy)` |

`jwt-keygen`, `postgres-init`, `auth-migrate` и `schedule-migrate` — one-shot контейнеры. Их состояние `Exited (0)` означает успешное завершение, а не ошибку.

## 4. Проверка

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

Откройте Swagger UI:

- [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)

Доступные сейчас маршруты Auth имеют префикс `/api/v1/auth`:

- `POST /api/v1/auth/send-code`
- `POST /api/v1/auth/verify-code`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout`
- `GET /api/v1/auth/me`
- OAuth и identities endpoints

В текущем API нет `/v1/auth/dev-login` и `/v1/tasks`: эти сценарии относятся к целевой архитектуре и учебному примеру.

### Проверка публичного расписания

После первого запуска worker загружает группы в фоне. `healthy` не означает, что расписание каждой группы уже загружено. Без JWT и без профиля выполните:

```bash
curl "http://localhost:8080/api/v1/schedule/groups?query=261&limit=20"
curl "http://localhost:8080/api/v1/schedule/groups/261-334"
```

Второй запрос выбирает текущую неделю по Москве. В ответе проверяйте `date_from`, `date_to`, `time_zone`, `metadata` и `days`. Если расписание ещё не загружено, подтверждённое задание даёт `200 PENDING_SYNC`; после загрузки — `200 READY`. Без данных и доступной очереди возможен `503`. Пустой `READY` и `404` неизвестной группы — отдельные состояния. Проверяйте также `metadata.status`, `published_from` и `published_to`. Полные правила и диагностика — в [API расписания](../gateway/schedule-api.md).

## 5. Команды управления

| Команда | Назначение |
|---|---|
| `task up` | Запустить полный стенд |
| `task up:deps` | Запустить только PostgreSQL и Redis |
| `task db:init` | Повторно подготовить роль, базу Schedule и права, не удаляя volume |
| `task status` | Показать все контейнеры, включая завершившиеся one-shot |
| `task logs` | Следить за логами |
| `task down` | Остановить стенд, сохранив данные и JWT-ключи |
| `task clean` | Удалить контейнеры, данные и JWT-ключи |

!!! danger "Полный сброс"
    `task clean` удаляет PostgreSQL, Redis и JWT-ключи. После генерации новой пары ранее выданные JWT перестанут проходить проверку.

## 6. Локальная разработка сервисов

Backend-разработчик может собрать изменяемый сервис из соседнего репозитория:

```text
orbit/
├── infra/
├── auth-service/
├── gateway-service/
└── schedule-service/   # нужен только при его локальной разработке
```

В `infra`:

```bash
cp compose.override.yaml.example compose.override.yaml
```

Раскомментируйте `build` только у разрабатываемого сервиса, затем выполните:

```bash
task build
docker compose --profile apps up -d --pull never --wait --wait-timeout 120
```

Файл `compose.override.yaml` локальный и не должен попадать в Git. Если меняете миграции Schedule, подключите сборку `schedule-migrate` из того же исходного кода.

## 7. Обновление уже запущенного стенда

После публикации новых образов и обновления `infra` скачайте их явно: одного `task up` недостаточно, чтобы гарантированно обновить локальный `latest`.

```bash
git pull --ff-only
docker compose --profile apps pull
task up
task status
```

База Schedule на старом PostgreSQL volume подготавливается one-shot сервисом `postgres-init` до миграций. Не выполняйте `task clean` или `down -v` ради добавления Schedule. При ручной диагностике: `task up:deps`, затем `task db:init`, затем `task up`. Подробности о паролях и существующих базах — в [infra](../contracts-infra/infra.md).

## 8. Диагностика

```bash
task status
docker compose --profile apps logs auth-service
docker compose --profile apps logs gateway-service
docker compose --profile apps logs auth-migrate
docker compose --profile apps logs postgres-init schedule-migrate schedule-service
docker compose --profile apps logs jwt-keygen
```

| Симптом | Что проверить |
|---|---|
| Ошибка скачивания `ghcr.io/orbit-polytech/*` | Проверьте имя/тег и [авторизацию и Read-доступ к пакетам](ghcr-access.md) |
| Auth не стартует | `RESEND_API_KEY`, `RESEND_FROM_EMAIL`, `OTP_HMAC_SECRET`, DSN и Redis password |
| `auth-migrate` завершился не с нулём | Логи мигратора и доступность PostgreSQL |
| Gateway возвращает `503` на Auth | Health Auth и `AUTH_GRPC_ADDR` |
| `503` на расписании | Health Schedule, `SCHEDULE_GRPC_ADDR`, доступность Redis/очереди, логи синхронизации и `SCHEDULE_SYNC_ENABLED` |
| `postgres-init` / `schedule-migrate` завершился с ошибкой | Логи, существующая база Auth, admin credentials и права приложения; не удалять volume вслепую |
| После `task clean` токены не принимаются | Это ожидаемо: была создана новая пара JWT-ключей |
