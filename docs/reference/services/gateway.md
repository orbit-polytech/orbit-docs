# gateway-service

**Репозиторий:** [`orbit-polytech/gateway-service`](https://github.com/orbit-polytech/gateway-service)

**Статус:** работает сейчас; контейнеризация находится в feature-ветке.

## Назначение

Gateway — единственная HTTP-граница текущего backend-стенда. Он принимает внешние запросы, проверяет access JWT и вызывает Auth по gRPC.

## Реализовано

- `/healthz` и `/readyz`;
- Swagger UI;
- Email OTP endpoints;
- refresh и logout;
- `/api/v1/auth/me`;
- Google OAuth login/callback;
- linking, listing и unlinking identities;
- JWT middleware;
- CORS, request ID, recovery и структурированное логирование;
- отображение gRPC-ошибок в HTTP.

## Не отвечает за

- выпуск токенов и хранение сессий;
- доступ к PostgreSQL Auth;
- бизнес-логику профилей, задач или расписания.

## Порты и зависимости

| Категория | Значение |
|---|---|
| HTTP внутри контейнера | `:8080` |
| Локальный адрес | `127.0.0.1:8080` |
| Auth | `auth-service:50051` |
| JWT | Только публичный Ed25519-ключ |

## Основные env

```ini
GATEWAY_HOST=0.0.0.0
GATEWAY_PORT=8080
AUTH_GRPC_ADDR=auth-service:50051
JWT_PUBLIC_KEY_PATH=/run/secrets/jwt_public/jwt_ed25519.pub
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000
SWAGGER_ENABLED=true
COOKIE_SECURE=false
```

`COOKIE_SECURE=false` допустим только для локального HTTP-стенда. В окружении с HTTPS значение должно быть `true`.
