# Спецификация кодов ошибок

В системе Orbit AI ошибки строго типизированы и однозначно транслируются между тремя уровнями представления:
1. **Доменные ошибки Go / Python:** типизированные `var Err... = errors.New(...)`.
2. **Статусные коды gRPC:** стандартные коды пакета `google.golang.org/grpc/codes`.
3. **HTTP-коды шлюза Gateway:** канонические статусы RFC 9110 с телом ошибки в формате JSON.

---

## 1. Сводная таблица маппинга ошибок

| Доменная ошибка | gRPC Status Code | HTTP Status | Описание для клиента |
| :--- | :--- | :-: | :--- |
| `ErrTaskNotFound`<br/>`ErrUserNotFound` | `codes.NotFound` | **404 Not Found** | Запрашиваемый ресурс не найден или у вас нет прав на его просмотр. |
| `ErrEmptyTitle`<br/>`ErrInvalidDeadline`<br/>`ErrInvalidPriority` | `codes.InvalidArgument` | **400 Bad Request** | Некорректные параметры запроса (ошибка валидации схемы/инварианта). |
| `ErrInvalidStatusTransition`<br/>`ErrCannotCompleteCanceled` | `codes.FailedPrecondition` | **412 Precondition Failed** | Недопустимый переход состояния сущности (например, попытка завершить отмененную задачу). |
| `ErrVersionConflict` | `codes.Aborted` | **409 Conflict** | Оптимистичная блокировка: запись была обновлена другим параллельным процессом. |
| `ErrEmptyOwnerID`<br/>`ErrInvalidToken` | `codes.Unauthenticated` | **401 Unauthorized** | Отсутствует или недействителен токен аутентификации. |
| `ErrAccessDenied`<br/>`ErrInsufficientRoles` | `codes.PermissionDenied` | **403 Forbidden** | Пользователь аутентифицирован, но не обладает достаточными ролями (например, не староста). |
| `ErrRateLimitExceeded` | `codes.ResourceExhausted` | **429 Too Many Requests** | Превышен лимит запросов в секунду / минуту. |
| `context.DeadlineExceeded` | `codes.DeadlineExceeded` | **504 Gateway Timeout** | Время ожидания ответа от нижележащего микросервиса истекло. |
| `context.Canceled` | `codes.Canceled` | **499 Client Closed Request** | Клиент разорвал сетевое соединение до получения ответа. |
| Внутренняя непредвиденная ошибка | `codes.Internal` | **500 Internal Server Error** | Сбой на стороне сервера. Внутренние детали ошибки логируются, но скрываются от клиента. |

---

## 2. Стандартный формат тела ошибки в REST API (OpenAPI)

При возникновении ошибки шлюз `gateway-service` возвращает ответ со следующим телом:

```json
{
  "error": {
    "code": "TASK_NOT_FOUND",
    "message": "задача не найдена",
    "details": [],
    "request_id": "018f2345-6789-7abc-def0-123456789abc"
  }
}
```

- **`code`:** машиночитаемый строковый код ошибки для использования в мобильном приложении (BLoC / UI стейты).
- **`message`:** локализованное понятное сообщение для пользователя.
- **`request_id`:** уникальный идентификатор запроса для поиска трейса в логах и Jaeger.
