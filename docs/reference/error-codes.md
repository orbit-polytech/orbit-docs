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
| `ErrScheduleNotReady` | `Unavailable` | `503` |
| Истёкший deadline | `DeadlineExceeded` | `504` |

`503` может означать отсутствие первого снимка, даже если контейнер Schedule — `healthy`. Устаревший, но сохранённый снимок может возвращаться с `200` и `metadata.is_stale=true`. См. [диагностику расписания](../gateway/schedule-api.md).

## Учебные и будущие сервисы

Доменные ошибки Tasks/Users и правила сокрытия чужого ресурса обсуждаются в учебных разделах и задачах. Они не доказывают наличие соответствующих маршрутов в текущем Gateway. При подключении сервиса его доменные ошибки сопоставляются с общим маппером явно; нельзя автоматически переносить из учебного примера коды вроде `TASK_NOT_FOUND` в опубликованный REST-контракт.
