# gateway-service

**Репозиторий:** [`orbit-polytech/gateway-service`](https://github.com/orbit-polytech/gateway-service)

**Статус:** работает сейчас; контейнеризация и интеграция Schedule находятся в `main`.

## Назначение

Gateway — единственная HTTP-граница текущего backend-стенда. Он принимает внешние запросы, проверяет access JWT на защищённых маршрутах и вызывает Auth или Schedule по gRPC. Публичное расписание не требует токена.

## Реализовано

- `/healthz` и `/readyz`;
- Swagger UI;
- публичный поиск групп и чтение расписания: `GET /api/v1/schedule/groups` и `GET /api/v1/schedule/groups/{group}`;
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
| Schedule | `schedule-service:50054` |
| JWT | Только публичный Ed25519-ключ |

## Основные env

```ini
GATEWAY_HOST=0.0.0.0
GATEWAY_PORT=8080
AUTH_GRPC_ADDR=auth-service:50051
SCHEDULE_GRPC_ADDR=schedule-service:50054
SCHEDULE_RPC_TIMEOUT=5s
JWT_PUBLIC_KEY_PATH=/run/secrets/jwt_public/jwt_ed25519.pub
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000
SWAGGER_ENABLED=true
COOKIE_SECURE=false
```

`COOKIE_SECURE=false` допустим только для локального HTTP-стенда. В окружении с HTTPS значение должно быть `true`.

Подробнее: [публичное API расписания](../../gateway/schedule-api.md). Gateway не вычисляет неделю по умолчанию — это правило usecase Schedule.


## Метаданные текущего API расписания

Gateway использует `contracts v0.5.0` и явно преобразует `ScheduleStatus` в HTTP DTO: `READY`, `PENDING_SYNC` или `UNSPECIFIED` для неизвестного значения. Передаются `published_from` / `published_to`, если известны. Ожидание подтверждённого фонового задания — успешный `200`, не gRPC-ошибка. Правила периода, кэш и постановка задач остаются в Schedule.

См. [публичное API расписания](../../gateway/schedule-api.md).
