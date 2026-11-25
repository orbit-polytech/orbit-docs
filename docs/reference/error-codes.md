# Коды ошибок текущего Gateway

HTTP-шлюз преобразует gRPC status в HTTP-код и JSON. Актуальная реализация — `gateway-service/internal/transport/http/httputil/grpc_error.go` и `gateway-service/internal/transport/http/httputil/errors.go`.

## Маппинг gRPC → HTTP

| gRPC | HTTP | `error.code` |
|---|---|---|
| `InvalidArgument` | `400` | `INVALID_ARGUMENT` |
| `NotFound` | `404` | `NOT_FOUND` |
| `AlreadyExists` | `409` | `ALREADY_EXISTS` |
| `PermissionDenied` | `403` | `PERMISSION_DENIED` |
| `Unauthenticated` | `401` | `UNAUTHENTICATED` |
| `ResourceExhausted` | `429` | `RESOURCE_EXHAUSTED` |
| `FailedPrecondition` | `400` | `FAILED_PRECONDITION` |
| `Aborted` | `409` | `ABORTED` |
| `OutOfRange` | `400` | `OUT_OF_RANGE` |
| `Unimplemented` | `501` | `UNIMPLEMENTED` |
| `Internal` | `500` | `INTERNAL` |
| `Unavailable` | `503` | `UNAVAILABLE` |
| `DeadlineExceeded` | `504` | `DEADLINE_EXCEEDED` |
| Остальные, включая `Unknown` и `Canceled` | `500` | `INTERNAL` |

Не следует ожидать `412`, `422` или `499` от текущего общего маппера. Специализированные HTTP-обработчики и middleware могут формировать собственные ошибки (например, `RATE_LIMIT` при `429`, `PAYLOAD_TOO_LARGE` при `413`, `UNSUPPORTED_MEDIA_TYPE` при `415`); ориентируйтесь на Swagger и код соответствующего endpoint.

## Формат ответа

```json
{
  "error": {
    "code": "UNAVAILABLE",
    "message": "сервис временно недоступен"
  }
}
```

Для ответов `429 Too Many Requests` (`ResourceExhausted` с gRPC `RetryInfo` или срабатывание Redis rate limiter на Gateway с кодом `RATE_LIMIT`) в `error` добавляется необязательное поле `retry_after_sec` (целое число секунд) и выставляется заголовок `Retry-After`:

```json
{
  "error": {
    "code": "RATE_LIMIT",
    "message": "Слишком частые запросы. Подождите 5 сек.",
    "retry_after_sec": 5
  }
}
```

При срабатывании rate limiter (`POST /api/v1/ai/chat/stream`, `POST /api/v1/support/issues`) Gateway также выставляет заголовки `X-RateLimit-Limit`, `X-RateLimit-Remaining` и `X-RateLimit-Reset`.

`details` и `request_id` в текущий JSON DTO не входят. Идентификатор запроса передаётся в заголовке **`X-Request-ID`** и используется при поиске логов. Для `Internal`, `Unknown`, `Unavailable` и `DeadlineExceeded` Gateway скрывает внутренние сообщения upstream.

При `401` общий gRPC-маппер выставляет `WWW-Authenticate: Bearer`. При `ResourceExhausted` с gRPC `RetryInfo` выставляется `Retry-After` и `retry_after_sec` с округлением задержки вверх до целой секунды.

## Ошибки расписания

| Доменная причина | gRPC | HTTP |
|---|---|---|
| `ErrInvalidInput`, `ErrInvalidDateRange`, `ErrDateRangeTooLong` | `InvalidArgument` | `400` |
| `ErrGroupNotFound` | `NotFound` | `404` |
| Нет расписания и невозможно подтвердить постановку обновления (`ErrScheduleNotReady` в ошибочном пути) | `Unavailable` | `503` |
| Истёкший deadline | `DeadlineExceeded` | `504` |

Отсутствие загруженного расписания само по себе не ошибка: при подтверждённом новом или существующем задании ответ — `200`, `metadata.status=PENDING_SYNC`. Если поставить задание невозможно и сохранённых данных нет — `503`. Недоступность gRPC upstream также даёт `503`.

Сохранённое расписание возвращается с `200 READY`, в том числе с `metadata.is_stale=true`; ошибка фонового обновления не должна скрывать доступные данные. `UNSPECIFIED` означает неизвестное состояние, не готовность и не ожидание. См. [API](../gateway/schedule-api.md) и [диагностику](../how-to/schedule-sync-diagnostics.md).

## Действующие маршруты Users и учебные примеры

Маршруты профилей `GET /api/v1/users/profile` и `PUT /api/v1/users/profile` уже работают в Gateway поверх `users-service` и используют общий gRPC-маппер (`400 INVALID_ARGUMENT` при ошибке валидации полей или зарезервированном `username`, `401 UNAUTHENTICATED`, `404 NOT_FOUND`, если профиль ещё не создан, `409 ALREADY_EXISTS`, если `@username` уже занят).

Доменные ошибки `Tasks` (например, `TASK_NOT_FOUND`) остаются учебным примером: при подключении будущего сервиса его доменные ошибки сопоставляются с общим маппером явно.


## Ошибки AI и SSE

До начала потока применяются HTTP/JSON ошибки (включая `429 RATE_LIMIT` от Redis rate limiter с `retry_after_sec`); после открытия SSE — `event: error` с числовым `error_code`, `error_message` и опциональным `retry_after_sec` (при `1003` rate limit). Поэтому `200` нельзя считать доказательством успешной генерации: дождитесь `done`. Повтор активного сообщения — `409`, конфликт payload — `400`, повтор финализированной ошибки/отмены со старым `message_id` — `400 FAILED_PRECONDITION`. Новая генерация требует нового UUIDv4.

Коды генерации `1001..1005`, `1099` не равны HTTP-кодам. Полные правила и ограничения — [AI API](../gateway/ai-api.md).
