# AI API: потоковый ответ и отмена

Оба маршрута требуют **Bearer access JWT**, полученный через текущий Email OTP / Google OAuth. Профиль Users для входа не требуется. Полные пути:

| Метод | Путь | Ответ |
|---|---|---|
| POST | `/api/v1/ai/chat/stream` | SSE `text/event-stream` |
| POST | `/api/v1/ai/chat/cancel` | JSON `{"cancelled": true/false}` |

## Запрос генерации

```json
{
  "message_id": "a38c291f-0e42-4b2a-8cf1-9b16999a4c8a",
  "query": "Какие правила подтверждены загруженными документами?"
}
```

- `message_id` — обязательный UUIDv4 новой попытки; создаётся клиентом.
- `query` — непустой текст, максимум 4000 символов после удаления пробелов по краям.
- `conversation_id` — необязательный UUIDv4. Для нового диалога опустите поле и сохраните идентификатор из `start`.
- Максимум JSON тела — 64 KiB. Идентификатор пользователя не передаётся в теле: Gateway берёт его из JWT.

Пример Bash; `$ACCESS_TOKEN` задайте локально, не публикуйте команду с реальным токеном. Создавайте новый UUID для новой попытки:

```bash
curl -N --fail-with-body http://localhost:8080/api/v1/ai/chat/stream \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  --data '{"message_id":"a38c291f-0e42-4b2a-8cf1-9b16999a4c8a","query":"Какие правила подтверждены загруженными документами?"}'
```

В PowerShell используйте `curl.exe` и синтаксис переменных/кавычек PowerShell, а не Bash-переносы `\`.

## События SSE

Каждое событие содержит `event: <имя>` и `data: <JSON>`, заканчивается пустой строкой. Схематический фрагмент:

```text
event: start
data: {"stage":"start","conversation_id":"d290f1ee-6c54-4b01-90e6-d701748f0851","message_id":"a38c291f-0e42-4b2a-8cf1-9b16999a4c8a"}

event: delta
data: {"stage":"delta","delta_text":"Фрагмент ответа","conversation_id":"d290f1ee-6c54-4b01-90e6-d701748f0851","message_id":"a38c291f-0e42-4b2a-8cf1-9b16999a4c8a"}

event: done
data: {"stage":"done","conversation_id":"d290f1ee-6c54-4b01-90e6-d701748f0851","message_id":"a38c291f-0e42-4b2a-8cf1-9b16999a4c8a"}

```

| Событие | Обработка клиентом |
|---|---|
| `start` | Сохранить идентификаторы |
| `delta` | Добавить `delta_text` к текущему ответу |
| `citations` | Показать найденные источники, если они есть |
| `done` | Успешное завершение; результат сохранён |
| `error` | Завершение с ошибкой; показать `error_message`, учесть `error_code` |

Цитата содержит `document_title`, `section`, `url`, `relevance_score`, `excerpt`. Внутренний `source_id` во внешний контракт не входит. Цитаты передаются отдельным событием, а не в каждом `delta`. Их отсутствие возможно и не означает сбой SSE; см. [ограничения RAG](../python-ai/current-limitations.md).

**HTTP 200 ещё не означает успех генерации:** после открытия потока возможен `event: error`. Обрыв без `done`/`error` — незавершённый ответ, не успешный результат.

## Ошибки и повторные запросы

До первого полученного чанка Gateway может вернуть обычную HTTP/JSON ошибку: `400` валидация, `401` JWT, `403` владение, `409` активная попытка, `413` размер тела, `503` недоступность, `504` deadline. После начала SSE статус HTTP уже не меняется, ошибка передаётся событием.

| Повтор | Правило |
|---|---|
| Успешная попытка, тот же пользователь и payload | Сохранённый текст/цитаты, без нового вызова LLM |
| Тот же `message_id`, другой payload | Конфликт, `400 INVALID_ARGUMENT` до потока |
| Попытка ещё выполняется | `409 ALREADY_EXISTS` до потока |
| Попытка завершилась ошибкой или отменой | Новый `message_id`; старый даёт `FAILED_PRECONDITION` (`400`) |

Replay гарантирует содержимое, а не исходное разбиение текста по `delta`: сохранённый текст может прийти одним фрагментом. Повтор не обходит проверку владельца.

Коды внутри SSE: `1001` upstream unavailable, `1002` timeout, `1003` rate limit, `1004` context too large, `1005` content filtered, `1099` internal. Наличие кода в enum не означает, что каждая причина уже определяется каждым адаптером. Не смешивайте эти числа с HTTP status и строковым `error.code` JSON-ошибки.

## Отмена

Отправьте из второй вкладки/соединения с тем же JWT:

```bash
curl --fail-with-body http://localhost:8080/api/v1/ai/chat/cancel \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  --data '{"conversation_id":"d290f1ee-6c54-4b01-90e6-d701748f0851","message_id":"a38c291f-0e42-4b2a-8cf1-9b16999a4c8a"}'
```

`conversation_id` необязателен, `message_id` обязателен. `cancelled=false` не доказывает ошибку: активная генерация могла уже завершиться. Закрытие HTTP-соединения также отменяет RPC; клиент не должен ждать обязательного финального события после собственного отключения. Реестр отмены сейчас in-memory: надёжная адресная отмена между репликами ещё не реализована.

## Web и Flutter

В браузере используйте `fetch` с POST, Bearer header и чтением `ReadableStream`, а не нативный `EventSource`, рассчитанный на GET. Парсер должен сохранять незавершённые строки/UTF-8 между сетевыми чанками: сетевой чанк не равен SSE-событию. Отключите буферизацию на внешнем reverse proxy и согласуйте таймауты. В Flutter нужен HTTP-клиент с потоковым чтением и аналогичным SSE-парсером.

**Исходники:** [handler](https://github.com/orbit-polytech/gateway-service/blob/main/internal/transport/http/v1/ai/handler.go), [DTO](https://github.com/orbit-polytech/gateway-service/blob/main/internal/transport/http/v1/ai/dto.go), [usecase](https://github.com/orbit-polytech/ai-service/blob/main/src/orbit_ai/application/chat_stream_use_case.py).
