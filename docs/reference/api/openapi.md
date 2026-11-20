# Внешнее HTTP API и Swagger

- **Базовый URL:** `http://localhost:8080/api/v1` (или HTTPS-домен туннеля Ngrok).
- **Swagger UI:** `http://localhost:8080/swagger/index.html`.
- **Схема JSON:** `http://localhost:8080/swagger/doc.json`.

Схема генерируется пакетом Swag из аннотаций в `gateway-service` (`api/docs`, формат Swagger 2.0). Ниже разделены реально работающие маршруты и проектные примеры на будущее.

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
| `POST` | `/auth/identities/link/begin` | Начало привязки внешнего провайдера (web и mobile); требуется JWT |
| `POST` | `/auth/identities/link/finish` | Завершение привязки |
| `GET` | `/auth/identities` | Список привязанных провайдеров; требуется JWT |
| `DELETE` | `/auth/identities/{provider}` | Отвязка внешнего провайдера; требуется JWT |

Все пути указаны относительно `/api/v1`. Парольного `POST /api/v1/auth/login` нет — вход идёт через Email OTP или Google OAuth. Расширенные профили студентов и роли старост появятся позже в `users-service`, а пока выбранные группы временно хранятся в предпочтениях аккаунта Auth.

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
| `POST` | `/ai/chat/stream` | Потоковый ответ SSE (`start`, `retrieval`, `delta`, `citations`, `done` / `error`) |
| `POST` | `/ai/chat/cancel` | Отмена активной генерации |
| `GET` | `/ai/conversations` | Список диалогов пользователя (`limit`, `offset`) |
| `GET` | `/ai/conversations/{id}` | История сообщений диалога (с цитатами, длительностью и оценкой) |
| `DELETE` | `/ai/conversations/{id}` | Удаление диалога |
| `PUT` | `/ai/conversations/{id}/messages/{message_id}/feedback` | Оценка ответа (`up`, `down`, `none`, `reason`, `comment`) |

Подробное описание DTO, стадий потока и правил идемпотентности — в [AI API](../../gateway/ai-api.md).

Gateway использует `contracts@v0.5.9` (Schedule совместим с `v0.5.0`) и явно маппит `ScheduleStatus` в `READY`, `PENDING_SYNC` или `UNSPECIFIED`, передавая `published_from` и `published_to`.

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
