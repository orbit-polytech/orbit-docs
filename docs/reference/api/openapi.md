# Внешнее HTTP API и Swagger

- **База URL:** `http://localhost:8080/api/v1`.
- **Swagger UI:** `http://localhost:8080/swagger/index.html`.
- **Схема JSON:** `http://localhost:8080/swagger/doc.json`.

Точная схема поставляется вместе с Gateway из генерируемого `api/docs`. Не следует называть её OpenAPI 3 без проверки поля версии в JSON: текущая генерация Swag использует формат Swagger 2.0. Ниже разделены работающие маршруты и проектные примеры.

## Текущее API Auth

| Метод | Путь | Назначение |
|---|---|---|
| POST | `/auth/send-code` | Запрос Email OTP |
| POST | `/auth/verify-code` | Подтверждение OTP и вход/регистрация |
| POST | `/auth/refresh` | Ротация сессии |
| POST | `/auth/logout` | Завершение сессии |
| GET | `/auth/me` | Проверенный JWT-контекст; требуется Bearer JWT |
| GET | `/auth/oauth/{provider}/login` | Начало OAuth |
| GET / POST | `/auth/oauth/{provider}/callback` | OAuth callback |
| POST | `/auth/identities/link/begin` | Начало привязки; требуется Bearer JWT |
| POST | `/auth/identities/link/finish` | Завершение протокола привязки |
| GET | `/auth/identities` | Список identities; требуется Bearer JWT |
| DELETE | `/auth/identities/{provider}` | Отвязка; требуется Bearer JWT |

Пути в таблице относительны к `/api/v1`. Поддерживаемый внешний OAuth provider сейчас — Google. Отсутствие JWT middleware не отменяет проверки OTP, OAuth state, binding cookie, refresh-токена и других требований конкретного endpoint; формат запроса смотрите в Swagger.

**Парольного `POST /api/v1/auth/login` сейчас нет.** Текущий вход — Email OTP или Google OAuth. `/auth/me` не заменяет будущий профиль Users.

## Текущее API расписания — без JWT

| Метод | Путь | Назначение |
|---|---|---|
| GET | `/schedule/groups?query=261&limit=20` | Поиск групп |
| GET | `/schedule/groups/261-334` | Текущая неделя по Москве |
| GET | `/schedule/groups/261-334?date_from=2026-10-24` | Один день |
| GET | `/schedule/groups/261-334?date_from=2026-10-19&date_to=2026-10-25` | Явный период |

Обе даты необязательны; только `date_to` недопустим. Максимум — 93 дня включительно. В JSON используются `group_title`, `slot_number`, `subject`, `start_time`, `end_time`, `metadata`, `date_from`, `date_to`, `time_zone` — не старые проектные поля `group_id`, `pair_number`, `discipline` и `is_even_week`.

Полная схема с примерами, пустыми ответами и ошибками: [публичное API расписания](../../gateway/schedule-api.md). Общий [формат ошибок](../error-codes.md).

## Будущие API: проектные примеры

!!! warning "Не реализовано в текущем Gateway"
    Ниже сохранены идеи Tasks и AI. Пути, enum, поля и ответы не являются опубликованными контрактами. Перед реализацией нужно согласовать `contracts` и отдельно обновить Swagger. AI-помощник планируется только для авторизованных пользователей; гостевой доступ к расписанию не означает гостевой доступ к AI.

### Управление задачами (`/api/v1/tasks`)

#### `GET /api/v1/tasks`
Получение списка задач аутентифицированного пользователя с пагинацией и фильтрацией.

- **Query-параметры:**
  * `limit` (int, default: 20, max: 100): количество элементов на страницу.
  * `offset` (int, default: 0): смещение.
  * `status` (string, optional): фильтр (`TASK_STATUS_OPEN`, `TASK_STATUS_COMPLETED`).

- **Response 200 OK:**
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

- **Request Body:**
```json
{
  "title": "Подготовить презентацию для предзащиты",
  "description": "Слайды 1-10",
  "priority": "TASK_PRIORITY_URGENT",
  "deadline": "2026-10-20T12:00:00Z",
  "idempotency_key": "task-uuid-or-client-hash"
}
```

---


### AI RAG-ассистент (`/api/v1/ai/query`)

#### `POST /api/v1/ai/query`
Интеллектуальный вопрос ассистенту по регламентам, стипендиям, БРС и навигации Московского Политеха.

- **Request Body:**
```json
{
  "prompt": "Как оформить справку об обучении для военкомата и где её забрать?"
}
```

- **Response 200 OK:**
```json
{
  "answer": "Справку об обучении можно заказать онлайн через Личный кабинет студента (e.mospolytech.ru) в разделе «Студенческий офис». Готовая справка с гербовой печатью выдается в корпусе на Большой Семёновской, ауд. В-107 при предъявлении студенческого билета.",
  "sources": [
    {
      "document_title": "Регламент работы Единого деканата 2026",
      "relevance_score": 0.92
    }
  ]
}
```


## Метаданные текущего API расписания

Gateway использует `contracts v0.5.0` и явно преобразует `ScheduleStatus` в HTTP DTO: `READY`, `PENDING_SYNC` или `UNSPECIFIED` для неизвестного значения. Передаются `published_from` / `published_to`, если известны. Ожидание подтверждённого фонового задания — успешный `200`, не gRPC-ошибка. Правила периода, кэш и постановка задач остаются в Schedule.

См. [публичное API расписания](../../gateway/schedule-api.md).
