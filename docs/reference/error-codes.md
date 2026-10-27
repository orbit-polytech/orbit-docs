# Коды ошибок текущего Gateway

HTTP-шлюз преобразует gRPC status в HTTP-код и JSON. Актуальная реализация — `gateway-service/internal/transport/http/httputil/grpc_error.go`.

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

Не следует ожидать `412`, `422` или `499` от текущего общего маппера. Специализированные HTTP-обработчики могут формировать собственные ошибки; ориентируйтесь на Swagger и код соответствующего endpoint.

## Формат ответа

```json
{
  "error": {
    "code": "UNAVAILABLE",
    "message": "сервис временно недоступен"
  }
}
```

`details` и `request_id` в текущий JSON DTO не входят. Идентификатор запроса передаётся в заголовке **`X-Request-ID`** и используется при поиске логов. Для `Internal`, `Unknown`, `Unavailable` и `DeadlineExceeded` Gateway скрывает внутренние сообщения upstream.

При `401` общий gRPC-маппер выставляет `WWW-Authenticate: Bearer`. При `ResourceExhausted` с gRPC `RetryInfo` возможен `Retry-After` с округлением задержки вверх до целой секунды.

## Ошибки расписания

| Доменная причина | gRPC | HTTP |
|---|---|---|
| `ErrInvalidInput`, `ErrInvalidDateRange`, `ErrDateRangeTooLong` | `InvalidArgument` | `400` |
| `ErrGroupNotFound` | `NotFound` | `404` |
| Нет расписания и невозможно подтвердить постановку обновления (`ErrScheduleNotReady` в ошибочном пути) | `Unavailable` | `503` |
| Истёкший deadline | `DeadlineExceeded` | `504` |

Отсутствие загруженного расписания само по себе не ошибка: при подтверждённом новом или существующем задании ответ — `200`, `metadata.status=PENDING_SYNC`. Если поставить задание невозможно и сохранённых данных нет — `503`. Недоступность gRPC upstream также даёт `503`.

Сохранённое расписание возвращается с `200 READY`, в том числе с `metadata.is_stale=true`; ошибка фонового обновления не должна скрывать доступные данные. `UNSPECIFIED` означает неизвестное состояние, не готовность и не ожидание. См. [API](../gateway/schedule-api.md) и [диагностику](../how-to/schedule-sync-diagnostics.md).

## Учебные и будущие сервисы

Доменные ошибки Tasks/Users и правила сокрытия чужого ресурса обсуждаются в учебных разделах и задачах. Они не доказывают наличие соответствующих маршрутов в текущем Gateway. При подключении сервиса его доменные ошибки сопоставляются с общим маппером явно; нельзя автоматически переносить из учебного примера коды вроде `TASK_NOT_FOUND` в опубликованный REST-контракт.
