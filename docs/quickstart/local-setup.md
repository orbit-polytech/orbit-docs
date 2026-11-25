# Запуск актуального стенда

Эта инструкция поможет локально поднять рабочий backend-стенд Orbit и проверить Gateway. Для обычного запуска Go, Python, Flutter, Buf или Goose ставить не нужно — сервисы работают в Docker.

!!! info "Доступ к образам"
    Образы Orbit лежат в приватном GitHub Container Registry (`ghcr.io`). Перед первым запуском нужно настроить вход и получить право **Read** на нужные пакеты — см. [инструкцию GHCR](ghcr-access.md). Актуальный статус всех сервисов — на странице [«Текущее состояние»](../project/current-state.md).

## 1. Что понадобится

| Инструмент | Требование |
|---|---|
| Git | 2.40+ |
| Docker | Docker Engine / Desktop с Compose v2 |
| Task | Taskfile 3.x |

Для комфортной работы со всем стеком (особенно вместе с AI и мониторингом) желательно иметь от 4–6 ГБ свободной RAM и 5–8 ГБ на диске.

## 2. Клонирование и `.env`

```bash
git clone https://github.com/orbit-polytech/infra.git
cd infra
cp .env.example .env
```

В Windows PowerShell:

```powershell
git clone https://github.com/orbit-polytech/infra.git
Set-Location infra
Copy-Item .env.example .env
```

Откройте `.env` и замените плейсхолдеры (`change_this_*`, `your_secret_*`) своими локальными паролями и секретами:

- для healthcheck Auth достаточно непустых значений `RESEND_API_KEY` и `RESEND_FROM_EMAIL`, но для **реальной отправки писем** с OTP нужен живой ключ Resend и подтверждённый отправитель;
- для Google OAuth заполните `AUTH_GOOGLE_CLIENT_ID`, `AUTH_GOOGLE_CLIENT_SECRET` и включите `GOOGLE_OAUTH_ENABLED=true`;
- если планируете запускать `all` или `monitoring`/`ops`, заполните также `GRAFANA_ADMIN_PASSWORD`, `METRICS_DB_PASSWORD`, `OPS_BOT_DB_PASSWORD`, `OPS_BOT_WEBHOOK_TOKEN` и ключи выбранного LLM-провайдера.

Файл `.env` в Git добавлять нельзя.

## 3. Запуск базового стенда (`apps`)

```bash
task up
task status
```

Что должно получиться:

| Контейнер | Ожидаемый статус |
|---|---|
| `postgres` | `Up (healthy)` |
| `redis` | `Up (healthy)` |
| `jwt-keygen` | `Exited (0)` |
| `postgres-init` | `Exited (0)` |
| `auth-migrate` | `Exited (0)` |
| `users-migrate` | `Exited (0)` |
| `schedule-migrate` | `Exited (0)` |
| `auth-service` | `Up (healthy)` |
| `users-service` | `Up (healthy)` |
| `schedule-service` | `Up (healthy)` |
| `gateway-service` | `Up (healthy)` |
| `web` | `Up (healthy)` |

`Exited (0)` для one-shot контейнеров (`jwt-keygen`, `postgres-init`, миграторы) — это норма, значит инициализация прошла успешно. Web/PWA-клиент после старта доступен на [http://localhost:8088](http://localhost:8088).

## 4. Проверка Gateway, Auth, профиля, расписания и Web

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

- Swagger UI: [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)
- Web / PWA клиент: [http://localhost:8088](http://localhost:8088) (Nginx раздаёт статику Flutter Web и проксирует `/api/`, `/healthz` и `/swagger/` на `gateway-service:8080`)

Основные маршруты Auth (префикс `/api/v1/auth`):
- `POST /api/v1/auth/send-code`
- `POST /api/v1/auth/verify-code`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout`
- `GET /api/v1/auth/me`
- `PATCH /api/v1/auth/preferences`
- `POST /api/v1/auth/email/change/begin`, `POST /api/v1/auth/email/change/confirm`
- `GET /api/v1/auth/oauth/{provider}/login`, `callback`, `POST /api/v1/auth/oauth/exchange`
- `POST /api/v1/auth/identities/link/begin`, `finish`, `GET /identities`, `DELETE /identities/{provider}`

Профиль пользователя (`users-service`, требуют JWT) и поддержка (`ops-bot`):
- `GET /api/v1/users/profile`, `PUT /api/v1/users/profile`
- `POST /api/v1/support/issues` (опциональный JWT; отправляет обращение из клиента в `ops-bot`)

Публичное расписание (работает без JWT):

```bash
curl "http://localhost:8080/api/v1/schedule/groups?query=261&limit=20"
curl "http://localhost:8080/api/v1/schedule/groups/261-334"
```

В PowerShell вместо алиаса `curl` вызывайте `curl.exe`. Первый запуск может вернуть `PENDING_SYNC` — это значит, что задача на выгрузку группы уже ушла фоновым воркерам `schedule-service`. Подробнее: [API расписания](../gateway/schedule-api.md).

## 5. Основные команды `Taskfile`

| Команда | Назначение |
|---|---|
| `task up` | Базовый профиль `apps` (PostgreSQL, Redis, миграции, Auth, `users-service`, Schedule, Gateway и `web :8088`; без AI, мониторинга и `ops-bot`) |
| `task up:deps` | Только `postgres` и `redis` |
| `task db:init` | Перезапуск `postgres-init` без удаления данных |
| `task build` | Сборка локально переопределённых сервисов профиля `apps` из `compose.override.yaml` (для `ai-service` или `ops-bot` используйте `docker compose --profile all build <service>`) |
| `task status` | Список контейнеров профиля `apps` |
| `task logs` | Логи профиля `apps` |
| `task down` | Остановка с сохранением томов и ключей |
| `task clean` | Полный сброс контейнеров и томов |
| `task tunnel:up` / `task tunnel:down` | Внешний HTTPS-туннель Ngrok (проксирует `web:80`: Web/PWA и `/api/` в `gateway-service`) |

!!! danger "Осторожно с `task clean`"
    `task clean` удаляет базы PostgreSQL, Redis, оба JWT-ключа (старые токены перестанут проверяться) и тома мониторинга.

## 6. Локальная разработка своего сервиса

Если вы правите код одного из сервисов и хотите проверить его в Compose вместо готового образа из GHCR:

```bash
cd infra
cp compose.override.yaml.example compose.override.yaml
# Уберите или закомментируйте build-блоки сервисов, которые не меняете
task build
docker compose --profile apps up -d --pull never --wait --wait-timeout 120
```

Не коммитьте `compose.override.yaml`. Если меняли миграции, оставьте в override также соответствующий `*-migrate` сервис. Обратите внимание: `task build` запускает `docker compose --profile apps build`, поэтому для локальной сборки `ai-service` или `ops-bot` вызывайте `docker compose --profile all build <service>` и поднимайте их через `--profile all`.

## 7. Обновление уже запущенного стенда

Тег `latest` обновляется, поэтому после `git pull` нужно явно подтянуть свежие образы:

```bash
git pull --ff-only
docker compose --profile apps pull
task up
task status
```

Если в `infra` добавили новые базы или роли (например, для AI, мониторинга или `ops-bot`), перед запуском профилей просто прогоните `task up:deps` и `task db:init` — удалять volume не нужно. Подробности: [infra](../contracts-infra/infra.md).

## 8. Подключить AI, мониторинг, `ops-bot` и туннель

- **Стенд с AI (Groq / OpenRouter / Gemini + pgvector):** см. [запуск с AI](../how-to/ai-local-stack.md).
- **Мониторинг и логи (Prometheus, Alertmanager, Loki, Alloy, Grafana):** см. [Observability](../contracts-infra/observability.md).
- **Операционный Telegram-бот (`ops-bot`):** см. [паспорт ops-bot](../reference/services/ops-bot.md).
- **Всё вместе одной командой:**

```bash
docker compose --profile all pull
docker compose --profile all up -d --wait --wait-timeout 180
docker compose --profile all ps -a
```

- **Наружный туннель Ngrok для тестов Web/PWA и API с телефона:** заполните `NGROK_AUTHTOKEN` (и при наличии `NGROK_URL`) в `.env` и выполните `task tunnel:up` (туннель проксирует контейнер `web:80`).

## 9. Если что-то пошло не так

| Симптом | Что проверить |
|---|---|
| Ошибка скачивания `ghcr.io/orbit-polytech/*` | Логин в `ghcr.io` и права **Read** на нужный пакет — см. [инструкцию GHCR](ghcr-access.md) |
| `auth-service` не стартует | `RESEND_API_KEY`, `RESEND_FROM_EMAIL`, `OTP_HMAC_SECRET`, `REDIS_PASSWORD`, `POSTGRES_DSN` |
| Gateway отвечает `503` на Auth или Schedule | Логи целевого сервиса и его healthcheck (`docker compose logs`) |
| Мониторинг ругается на БД/пароли после обновления `infra` | Запустите `task db:init`, чтобы `postgres-init` досоздал роли `orbit_metrics` и `orbit_ops_bot` |
| После `task clean` старые JWT не принимаются | Нормально: при чистом старте `jwt-keygen` сгенерировал новую пару ключей |
