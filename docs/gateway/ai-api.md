# AI API: потоковый ответ, диалоги и оценки

Все маршруты AI требуют **Bearer access JWT** (через Email OTP или Google OAuth). Полные пути относительно Gateway (`http://localhost:8080`):

| Метод | Путь | Ответ | Назначение |
|---|---|---|---|
| `POST` | `/api/v1/ai/chat/stream` | `text/event-stream` (SSE) | Потоковый ответ ассистента |
| `POST` | `/api/v1/ai/chat/cancel` | JSON `{"cancelled": true/false}` | Отмена активной генерации |
| `GET` | `/api/v1/ai/conversations` | JSON `GetConversationsResponse` | Список диалогов пользователя (`limit`, `offset`) |
| `GET` | `/api/v1/ai/conversations/{id}` | JSON `GetConversationHistoryResponse` | Сообщения выбранного диалога (`limit`) |
| `DELETE` | `/api/v1/ai/conversations/{id}` | JSON `{"success": true/false}` | Удаление диалога |
| `PUT` | `/api/v1/ai/conversations/{id}/messages/{message_id}/feedback` | JSON `SubmitFeedbackResponse` | Лайк/дизлайк ответа и причина |

Gateway проверяет JWT, валидирует JSON (до 64 KiB) и передаёт `user_id` в `ai-service` через metadata `x-user-id`. Напрямую `user_id` из тела запроса сервер не принимает.

---

## 1. Потоковая генерация (`POST /api/v1/ai/chat/stream`)

```json
{
  "message_id": "a38c291f-0e42-4b2a-8cf1-9b16999a4c8a",
  "query": "Какого числа выплачивается академическая стипендия?"
}
```

- `message_id` — обязательный UUIDv4 новой попытки (генерируется на клиенте).
- `query` — непустой текст, максимум 4000 символов после обрезки пробелов.
- `conversation_id` — необязательный UUIDv4. Если не передавать, `ai-service` заведёт новый диалог и вернёт его ID в событии `start`.
- `model` — необязательная строка (`AI_ALLOWED_MODELS`). Если не передавать (или передать `auto`), сработает роутинг модели по роли запроса. Переданная модель входит в ключ идемпотентности — для другой модели нужен новый `message_id`.

Пример запуска в Bash:

```bash
curl -N --fail-with-body http://localhost:8080/api/v1/ai/chat/stream \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  --data '{"message_id":"a38c291f-0e42-4b2a-8cf1-9b16999a4c8a","query":"Где находится корпус на Автозаводской?"}'
```

В PowerShell вместо `curl` вызывайте `curl.exe`.

### События SSE

Каждое событие состоит из `event: <stage>` и `data: <JSON>`:

| Стадия (`stage`) | Что делает клиент |
|---|---|
| `start` | Сохраняет `conversation_id` и `message_id` |
| `retrieval` | Показывает индикатор поиска по базе знаний вуза (приходит, когда включился RAG) |
| `delta` | Дописывает `delta_text` к текущему ответу |
| `citations` | Показывает список источников (`document_title`, `section`, `url`, `relevance_score`, `excerpt`) |
| `done` | Финализирует ответ; может содержать `duration_ms` (длительность генерации в мс) |
| `cancelled` | Генерация была прервана |
| `error` | Показывает `error_message` и учитывает `error_code` (`1001` upstream unavailable, `1002` timeout, `1003` rate limit, `1004` context too large, `1005` content filtered, `1099` internal) |

`HTTP 200` на старте SSE не гарантирует успешный финал: ошибка может прийти уже внутри потока через `event: error`. Если поток оборвался без `done` или `error`, ответ нельзя считать завершённым.

### Ошибки и повторные запросы (replay)

До первого чанка Gateway возвращает обычные HTTP-ошибки: `400`, `401`, `403`, `409`, `413`, `503`, `504`.

| Ситуация с тем же `message_id` | Поведение |
|---|---|
| Попытка уже завершилась успехом, тот же пользователь и payload | Отдаётся сохранённый ответ/цитаты без нового вызова LLM |
| Другой `query` / `conversation_id` / `model` | Конфликт, `400 INVALID_ARGUMENT` |
| Предыдущая попытка ещё выполняется | `409 ALREADY_EXISTS` |
| Предыдущая попытка завершилась ошибкой или отменой | `400 FAILED_PRECONDITION` — нужен новый `message_id` |

---

## 2. Отмена (`POST /api/v1/ai/chat/cancel`)

```bash
curl --fail-with-body http://localhost:8080/api/v1/ai/chat/cancel \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  --data '{"conversation_id":"d290f1ee-6c54-4b01-90e6-d701748f0851","message_id":"a38c291f-0e42-4b2a-8cf1-9b16999a4c8a"}'
```

`message_id` обязателен, `conversation_id` — опционален. `cancelled: false` может означать, что генерация уже успела закончиться. Закрытие самого HTTP-соединения стрима также отменяет RPC.

---

## 3. История и удаление диалогов

- **`GET /api/v1/ai/conversations?limit=20&offset=0`**
  Возвращает `conversations` (`id`, `title`, `created_at`, `updated_at`, `messages_count`, `last_message_snippet`) и `total_count`. По умолчанию `limit=20`, `offset=0`.
- **`GET /api/v1/ai/conversations/{id}?limit=50`**
  `id` — UUIDv4 диалога. Возвращает `messages`: `id`, `role`, `content`, `citations`, `created_at`, `duration_ms`, `feedback_rating` (`up` / `down`), `feedback_reason`. Если диалог чужой или не найден, вернутся `403` или `404`.
- **`DELETE /api/v1/ai/conversations/{id}`**
  Удаляет диалог пользователя (с проверкой владельца) и возвращает `{"success": true}`.

---

## 4. Оценка ответов (`PUT .../feedback`)

Позволяет поставить, обновить или снять оценку для конкретного сообщения ассистента:

```json
{
  "rating": "down",
  "reason": "inaccurate",
  "comment": "Неверно указан корпус"
}
```

- `rating`: `"up"`, `"down"` или `"none"` (пустая строка тоже снимает оценку).
- `reason`: `""`, `"inaccurate"`, `"incomplete"`, `"wrong_source"` или `"other"`.
- `comment`: до `500` символов после `TrimSpace`.

Ответ `200 OK`: `{"saved": true, "rating": "down", "reason": "inaccurate"}`. Негативные оценки (`down`) подхватываются `ops-bot` и попадают в топик `#feedback` для разбора.

---

## 5. Рекомендации для клиентов (Web и Flutter)

Браузерный `EventSource` не умеет отправлять `POST` и кастомные заголовки, поэтому на вебе используйте `fetch` с `ReadableStream`. Во Flutter (как в `mobile/lib/features/assistant/`) читайте байтовый поток из Dio и буферизуйте незавершённые UTF-8 строки и `event:`/`data:` блоки между сетевыми чанками.

**Исходники:** [handler.go](https://github.com/orbit-polytech/gateway-service/blob/main/internal/transport/http/v1/ai/handler.go), [conversations.go](https://github.com/orbit-polytech/gateway-service/blob/main/internal/transport/http/v1/ai/conversations.go), [feedback.go](https://github.com/orbit-polytech/gateway-service/blob/main/internal/transport/http/v1/ai/feedback.go), [dto.go](https://github.com/orbit-polytech/gateway-service/blob/main/internal/transport/http/v1/ai/dto.go).
