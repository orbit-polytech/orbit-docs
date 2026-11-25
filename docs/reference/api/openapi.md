# Внешнее HTTP API и Swagger

- **Базовый URL:** `http://localhost:8080/api/v1` (или через прокси `http://localhost:8088/api/v1` / HTTPS-домен туннеля Ngrok).
- **Swagger UI:** `http://localhost:8080/swagger/index.html` (или `http://localhost:8088/swagger/index.html`).
- **Схема JSON:** `http://localhost:8080/swagger/doc.json`.

Схема генерируется пакетом Swag из аннотаций в `gateway-service` (`api/docs`, формат Swagger 2.0). Ниже разделены реально работающие маршруты и проектные примеры на будущее.

!!! note "Синхронизация `swagger.yaml`"
    Сгенерированный файл `gateway-service/api/docs/swagger.yaml` сейчас отстаёт от роутера: в нём ещё не перегенерированы новые эндпоинты (`/users/profile`, `/auth/email/change/*`, `/support/issues`, `/ai/conversations` и `retry_after_sec` в `ErrorDetail`). Актуальный источник маршрутов — `gateway-service/internal/transport/http/router.go`.

## Текущее API Auth

| Метод | Путь | Назначение |
|---|---|---|
| `POST` | `/auth/send-code` | Запрос 6-значного Email OTP |
| `POST` | `/auth/verify-code` | Подтверждение OTP и вход/регистрация |
| `POST` | `/auth/refresh` | Ротация refresh-токена |
| `POST` | `/auth/logout` | Завершение сессии |
| `GET` | `/auth/oauth/{provider}/login` | Начало OAuth (поддерживается `google`, web и mobile PKCE) |
| `GET` / `POST` | `/auth/oauth/{provider}/callback` | OAuth callback; для mobile редиректит на `ru.mospolytech.orbit://auth/callback` |
| `POST` | `/auth/oauth/exchange` | Обмен одноразового кода мобильного OAuth + `code_verifier` на пару токенов |
| `GET` | `/auth/me` | Данные текущего аккаунта (`user_id`, `email`, `primary_group`, `favorite_groups`); требуется JWT |
| `PATCH` | `/auth/preferences` | Обновление `primary_group` и `favorite_groups`; требуется JWT |
| `POST` | `/auth/email/change/begin` | Запрос OTP на новый `new_email` (требует JWT и `reauth_proof` / свежий Bearer); возвращает `retry_after_seconds`, `expires_in_seconds` |
| `POST` | `/auth/email/change/confirm` | Подтверждение смены email по 6-значному `code` (требует JWT и `reauth_proof`); возвращает обновлённый `MeResponse` |
| `POST` | `/auth/identities/link/begin` | Начало привязки внешнего провайдера (web и mobile); требуется JWT |
| `POST` | `/auth/identities/link/finish` | Завершение привязки |
| `GET` | `/auth/identities` | Список привязанных провайдеров; требуется JWT |
| `DELETE` | `/auth/identities/{provider}` | Отвязка внешнего провайдера; требуется JWT |

Все пути указаны относительно `/api/v1`. Парольного `POST /api/v1/auth/login` нет — вход идёт через Email OTP или Google OAuth. Выбранные учебные группы аккаунта по-прежнему синхронизируются через предпочтения Auth (`primary_group`, `favorite_groups`), а пользовательский профиль (`display_name`, `username`, `academic_group`, `course`, `avatar_url`, `role`) уже вынесен в `users-service`.

## Текущее API Users (требует JWT)

| Метод | Путь | Назначение |
|---|---|---|
| `GET` | `/users/profile` | Получить профиль текущего пользователя из `users-service` по `account_id` из JWT (`404`, если профиль ещё не создан) |
| `PUT` | `/users/profile` | Создать или полностью обновить профиль (`display_name`, `username`, `academic_group`, `course`, `avatar_url`; `409`, если `username` уже занят) |

## Текущее API расписания (без JWT)

| Метод | Путь | Назначение |
|---|---|---|
| `GET` | `/schedule/groups?query=261&limit=20` | Поиск учебных групп |
| `GET` | `/schedule/groups/261-334` | Расписание на текущую неделю по Москве |
| `GET` | `/schedule/groups/261-334?date_from=2026-10-24` | Один день |
| `GET` | `/schedule/groups/261-334?date_from=2026-10-19&date_to=2026-10-25` | Произвольный период (до 93 дней включительно) |

Обе даты необязательны, передавать только `date_to` нельзя. Подробнее со схемами и примерами: [публичное API расписания](../../gateway/schedule-api.md). Общий формат ошибок — в [справочнике кодов ошибок](../error-codes.md).

## Текущее AI API (требует JWT)

| Метод | Путь | Назначение |
|---|---|---|
| `POST` | `/ai/chat/stream` | Потоковый ответ SSE (`start`, `retrieval`, `delta`, `citations`, `done` / `error`); защищён Redis rate limiter (`429` + `retry_after_sec`) |
| `POST` | `/ai/chat/cancel` | Отмена активной генерации |
| `GET` | `/ai/conversations` | Список диалогов пользователя (`limit`, `offset`) |
| `GET` | `/ai/conversations/{id}` | История сообщений диалога (с цитатами, длительностью и оценкой) |
| `DELETE` | `/ai/conversations/{id}` | Удаление диалога |
| `PUT` | `/ai/conversations/{id}/messages/{message_id}/feedback` | Оценка ответа (`up`, `down`, `none`, `reason`, `comment`) |

Подробное описание DTO, стадий потока, rate limiting (`429`, `retry_after_sec`) и правил идемпотентности — в [AI API](../../gateway/ai-api.md).

## Текущее API поддержки (`Support`, опциональный JWT)

| Метод | Путь | Назначение |
|---|---|---|
| `POST` | `/support/issues` | Отправка обращения или баг-репорта в `#feedback` через `ops-bot` (`category`: `schedule`/`assistant`/`profile`/`ui_bug`/`other`, `description`, `contact`, `app_version`, `platform`). Работает с JWT и без него (`OptionalAuth`), ограничен rate limiter (`429` + `retry_after_sec`) |

В ответах `429 Too Many Requests` (rate limiter или `ResourceExhausted` с `RetryInfo`) `ErrorDetail` включает необязательное поле `retry_after_sec`, а в заголовках возвращается `Retry-After` (и `X-RateLimit-*` для rate limiter).

Gateway использует `contracts@v0.5.12` (Schedule совместим с `v0.5.0`) и явно маппит `ScheduleStatus` в `READY`, `PENDING_SYNC` или `UNSPECIFIED`, передавая `published_from` и `published_to`.

---

## Будущие API (проектные примеры)

!!! warning "Не реализовано в текущем Gateway"
    Ниже оставлены черновые идеи API для будущего `tasks-service`. Эти маршруты не работают в `gateway-service` — сначала контракты будут согласованы в `contracts`.

### Управление задачами (`/api/v1/tasks`)

#### `GET /api/v1/tasks`
Список задач пользователя с пагинацией и фильтрацией (`limit`, `offset`, `status`).

```json
{
  "tasks": [
    {
      "id": "018f2345-6789-7abc-def0-123456789abc",
      "title": "Сдать отчет по лабораторной работе №3",
      "description": "Тема: Инверсия зависимостей в Clean Architecture",
      "status": "TASK_STATUS_OPEN",
      "priority": "TASK_PRIORITY_HIGH",
      "deadline": "2026-10-15T18:00:00Z",
      "version": 1,
      "created_at": "2026-10-12T10:00:00Z"
    }
  ],
  "total_count": 1
}
```

#### `POST /api/v1/tasks`
Создание новой задачи.

```json
{
  "title": "Подготовить презентацию для предзащиты",
  "description": "Слайды 1-10",
  "priority": "TASK_PRIORITY_URGENT",
  "deadline": "2026-10-20T12:00:00Z",
  "idempotency_key": "task-uuid-or-client-hash"
}
```
