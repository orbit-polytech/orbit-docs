# Спецификация внешнего REST API (OpenAPI)

Шлюз `gateway-service` предоставляет единый унифицированный REST/JSON API для мобильного приложения на Flutter, веб-интерфейсов и сторонних интеграций.

- **Базовый URL (локально):** `http://localhost:8080/api/v1`
- **Интерактивный Swagger UI:** `http://localhost:8080/swagger/index.html`
- **Спецификация OpenAPI v3 (JSON):** `http://localhost:8080/swagger/doc.json`

---

## 1. Аутентификация (`/api/v1/auth`)

Все защищенные эндпоинты требуют передачи токена доступа в HTTP-заголовке:
`Authorization: Bearer <access_jwt_token>`

### `POST /api/v1/auth/login`
Вход в систему по email (или логину) и паролю учетной записи.

- **Request Body:**
```json
{
  "email": "student@mail.ru",
  "password": "SecurePassword123"
}
```

- **Response 200 OK:**
```json
{
  "access_token": "eyJhbGciOiJFZERTQ...",
  "refresh_token": "def018f2...",
  "expires_in": 900,
  "token_type": "Bearer"
}
```

---

## 2. Управление задачами (`/api/v1/tasks`)

### `GET /api/v1/tasks`
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

### `POST /api/v1/tasks`
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

## 3. Расписание занятий (`/api/v1/schedule`)

### `GET /api/v1/schedule/groups/{group_id}`
Получение расписания учебной группы на текущую неделю с учетом числителя и знаменателя.

- **Параметры пути:**
  * `group_id`: номер группы (например, `261-334`).

- **Response 200 OK:**
```json
{
  "group_id": "261-334",
  "is_even_week": false,
  "days": [
    {
      "day_of_week": 1,
      "date": "2026-10-12",
      "lessons": [
        {
          "pair_number": 1,
          "time_start": "09:00",
          "time_end": "10:30",
          "discipline": "Введение в проектную деятельность",
          "teacher": "Иванов И. И.",
          "campus": "Большая Семёновская (БС)",
          "room": "А-401"
        }
      ]
    }
  ]
}
```

---

## 4. AI RAG-ассистент (`/api/v1/ai/query`)

### `POST /api/v1/ai/query`
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
