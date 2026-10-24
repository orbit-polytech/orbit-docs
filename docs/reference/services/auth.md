# auth-service

**Репозиторий:** [`orbit-polytech/auth-service`](https://github.com/orbit-polytech/auth-service)

**Статус:** работает сейчас; контейнеризация находится в `main` и используется полным Compose-стеком.

## Назначение

Auth отвечает за аккаунты и способы входа, а не за пользовательские профили.

Реализовано:

- Email OTP;
- JWT Ed25519;
- refresh-сессии и ротация;
- Google OAuth/OIDC;
- linking, listing и unlinking внешних identities;
- OAuth state и re-auth deadlines в Redis;
- gRPC health service.

## Не отвечает за

- имя, аватар и учебную группу пользователя;
- расписание, задачи и навигацию;
- бизнес-роли ресурсов других сервисов.

Эти данные принадлежат будущему `users-service` и соответствующим bounded contexts.

## Интерфейсы и зависимости

| Категория | Значение |
|---|---|
| Транспорт | gRPC `:50051` |
| Данные | PostgreSQL 16 |
| Временное состояние | Redis 7 |
| Почта | Resend |
| OAuth | Google |
| Подпись | Ed25519 private key |

## Основные env

```ini
AUTH_GRPC_HOST=0.0.0.0
AUTH_GRPC_PORT=50051
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

Реальные секреты не добавляются в Git.

## Следующие возможности

Telegram identity может быть добавлена в будущем как ещё один провайдер после отдельного security review. Она не является текущим способом входа.
