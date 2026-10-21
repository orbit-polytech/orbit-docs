# Запуск актуального стенда

Цель руководства — поднять текущий backend-стенд Orbit AI и проверить Gateway. Для обычного запуска не нужны Go, Python, Flutter, Buf или Goose: сервисы работают в Docker-контейнерах.

!!! info "Статус"
    Полный Compose-стек проверен в feature-ветках контейнеризации. Автономный запуск из чистого клона `infra` станет общедоступным после публикации образов в GHCR. Текущий статус указан на странице [«Текущее состояние»](../project/current-state.md).

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
| `auth-migrate` | `Exited (0)` |
| `auth-service` | `Up (healthy)` |
| `gateway-service` | `Up (healthy)` |

`jwt-keygen` и `auth-migrate` — one-shot контейнеры. Их состояние `Exited (0)` означает успешное завершение, а не ошибку.

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

## 5. Команды управления

| Команда | Назначение |
|---|---|
| `task up` | Запустить полный стенд |
| `task up:deps` | Запустить только PostgreSQL и Redis |
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
└── gateway-service/
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

Файл `compose.override.yaml` локальный и не должен попадать в Git.

## 7. Диагностика

```bash
task status
docker compose --profile apps logs auth-service
docker compose --profile apps logs gateway-service
docker compose --profile apps logs auth-migrate
docker compose --profile apps logs jwt-keygen
```

| Симптом | Что проверить |
|---|---|
| Ошибка скачивания `ghcr.io/orbit-polytech/*` | Образы ещё не опубликованы либо package требует авторизации; используйте локальный override |
| Auth не стартует | `RESEND_API_KEY`, `RESEND_FROM_EMAIL`, `OTP_HMAC_SECRET`, DSN и Redis password |
| `auth-migrate` завершился не с нулём | Логи мигратора и доступность PostgreSQL |
| Gateway возвращает `503` | Health Auth и значение `AUTH_GRPC_ADDR` внутри Compose |
| После `task clean` токены не принимаются | Это ожидаемо: была создана новая пара JWT-ключей |
