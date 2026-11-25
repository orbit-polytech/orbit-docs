# auth-service

**Репозиторий:** [`orbit-polytech/auth-service`](https://github.com/orbit-polytech/auth-service)

**Статус:** работает сейчас в `main` (`contracts@v0.5.11`), входит в профиль Compose `apps` / `all`.

## Назначение и границы

Отвечает за аккаунты, способы аутентификации, выпуск JWT и управление внешними привязками.

Что уже реализовано:
- Email OTP (отправка через Resend, HMAC-хэширование, cooldown и TTL в Redis);
- смена email с подтверждением личности (`BeginEmailChange`, `ConfirmEmailChange` по OTP на новый адрес и проверкой `reauth_proof`);
- выпуск access JWT с подписью **Ed25519**;
- refresh-сессии в PostgreSQL с ротацией и отзывом всего семейства при повторном использовании старого токена;
- Google OAuth/OIDC (`BeginOAuthLogin`, `FinishOAuthLogin` — web и mobile);
- привязка, список и отвязка внешних identities (`BeginIdentityLink`, `FinishIdentityLink`, `ListLinkedIdentities`, `UnlinkIdentity`) с проверкой intent, binding cookie и re-auth окна;
- получение и обновление предпочтений учебных групп аккаунта (`GetAccount`, `UpdateAccountPreferences`: `primary_group` и `favorite_groups`, миграция `00005_add_account_preferences.sql`);
- структурированное JSON-логирование `log/slog` (`service="auth-service"`) и внутренний HTTP-сервер метрик `:9091` (`/metrics` с `grpcprom`, `/livez`);
- gRPC Health Checking Protocol.

За что **не отвечает**:
- пользовательский профиль (`display_name`, `@username`, курс, аватар и роль — за это отвечает работающий [`users-service`](users.md); при этом `primary_group` и `favorite_groups` сохраняются в `auth-service` для синхронизации расписания);
- расписание, задачи, AI и навигация.

## Интерфейсы и зависимости

| Категория | Значение |
|---|---|
| Транспорт | gRPC `:50051` |
| Metrics HTTP (внутренний) | `:9091` (`/metrics`, `/livez`) |
| База данных | PostgreSQL 16 (`orbit_auth`, миграции Goose в `auth-migrate`) |
| Кэш / временное состояние | Redis 7 (`OTP`, `OAuth state`, `re-auth`, rate limits) |
| Почта | Resend |
| OAuth | Google |
| Подпись | Приватный Ed25519-ключ (`jwt_private`) |

## Основные env

```ini
AUTH_GRPC_HOST=0.0.0.0
AUTH_GRPC_PORT=50051
METRICS_ADDR=0.0.0.0:9091
LOG_LEVEL=info
POSTGRES_DSN=postgres://...
REDIS_ADDR=redis:6379
REDIS_PASSWORD=...
OTP_HMAC_SECRET=...
RESEND_API_KEY=...
RESEND_FROM_EMAIL=...
JWT_PRIVATE_KEY_PATH=/run/secrets/jwt_private/jwt_ed25519.key
GOOGLE_OAUTH_ENABLED=false
AUTH_GOOGLE_CLIENT_ID=
AUTH_GOOGLE_CLIENT_SECRET=
AUTH_GOOGLE_REDIRECT_URI=http://localhost:8080/api/v1/auth/oauth/google/callback
```

Секреты заполняются только в локальном `.env` и не коммитятся.

## Дальнейшие планы

Вход через Telegram (Login Widget / Mini App `initData`) по-прежнему остаётся будущей интеграцией и требует отдельной проработки модели безопасности до добавления в `contracts`.

См. также: [Модель безопасности](../../security/overview.md), [Observability](../../contracts-infra/observability.md).
