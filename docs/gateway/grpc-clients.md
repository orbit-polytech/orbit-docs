# Маршрутизация и клиенты внутренних сервисов в Gateway

В текущем `main` (`internal/app/wiring.go`) подключены четыре долгоживущих gRPC-клиента и один HTTP-клиент:
- `internal/infrastructure/authclient` (`AUTH_GRPC_ADDR`, в Compose `auth-service:50051`, таймауты `AUTH_RPC_TIMEOUT=5s`, `AUTH_SEND_CODE_TIMEOUT=12s`);
- `internal/infrastructure/usersclient` (`USERS_GRPC_ADDR`, в Compose `users-service:50052`, таймаут `USERS_RPC_TIMEOUT=5s`);
- `internal/infrastructure/scheduleclient` (`SCHEDULE_GRPC_ADDR`, в Compose `schedule-service:50054`, таймаут `SCHEDULE_RPC_TIMEOUT=5s`);
- `internal/infrastructure/aiclient` (`AI_GRPC_ADDR`, в Compose `ai-service:50057`, таймаут `AI_RPC_TIMEOUT=120s`, опциональный `INTERNAL_SECRET`);
- `internal/infrastructure/opsbotclient` — HTTP-клиент для отправки обращений из приложения (`POST /issues/webhook` в `OPS_BOT_WEBHOOK_URL`, в Compose `http://ops-bot:8081`, с Bearer `OPS_BOT_WEBHOOK_TOKEN` и таймаутом `5s`).

Пустые `date_from`/`date_to` передаются в Schedule без вычисления дат в Gateway; фактические границы ответа возвращаются клиенту. Общий маппер gRPC-ошибок находится в `internal/transport/http/httputil/grpc_error.go`.

!!! info "Ниже — упрощённая учебная схема расширения"
    Фабрика `internal/clients`, Tasks и код `writeError` ниже показывают возможную организацию будущих интеграций, а не реальное дерево репозитория. В текущем `gateway-service` клиенты разнесены по пакетам `internal/infrastructure/*client` и собираются в `internal/app/wiring.go` (`dialClients`), а ошибки маппятся функцией `httputil.WriteGRPCError`. См. [обзор Gateway](overview.md).

---

## 1. Фабрика gRPC-клиентов (`internal/clients`)

gRPC-соединения в Go используют HTTP/2 мультиплексирование, поэтому для каждого сервиса создаётся **одно долгоживущее соединение (`grpc.ClientConnInterface`)** при старте шлюза, которое разделяется между всеми входящими HTTP-запросами:

```go
package clients

import (
	"fmt"

	authv1 "github.com/orbit-polytech/contracts/gen/go/orbit/auth/v1"
	tasksv1 "github.com/orbit-polytech/contracts/gen/go/orbit/tasks/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Clients struct {
	Auth  authv1.AuthServiceClient
	Tasks tasksv1.TasksServiceClient
}

func DialAll(cfg Config) (*Clients, error) {
	// Подключение к auth-service
	authConn, err := grpc.NewClient(cfg.AuthAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("подключение к auth-service (%s): %w", cfg.AuthAddr, err)
	}

	// Подключение к tasks-service
	tasksConn, err := grpc.NewClient(cfg.TasksAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("подключение к tasks-service (%s): %w", cfg.TasksAddr, err)
	}

	return &Clients{
		Auth:  authv1.NewAuthServiceClient(authConn),
		Tasks: tasksv1.NewTasksServiceClient(tasksConn),
	}, nil
}
```

---

## 2. Управление таймаутами и контекстом

Каждый вызов микросервиса из HTTP-обработчика обязан иметь явно заданный таймаут (обычно 2–5 секунд). Без таймаута зависание одного внутреннего сервиса быстро исчерпает пул горутин шлюза и положит всю систему:

```go
ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
defer cancel()

resp, err := h.tasksClient.GetTask(ctx, &tasksv1.GetTaskRequest{Id: taskID})
```

Если время вышло, gRPC вернет код `codes.DeadlineExceeded`, который транслируется клиенту в HTTP-код `504 Gateway Timeout`.

---

## 3. Таблица маппинга ошибок (gRPC $\rightarrow$ HTTP)

Текущий `httputil.WriteGRPCError` преобразует статусы следующим образом. Полная таблица и JSON — в [справочнике ошибок](../reference/error-codes.md):

| Код статуса gRPC | HTTP статус | Описание для клиента |
|---|---|---|
| `codes.InvalidArgument` | **400 Bad Request** | Ошибка валидации параметров запроса |
| `codes.Unauthenticated` | **401 Unauthorized** | Отсутствует или недействителен токен |
| `codes.PermissionDenied` | **403 Forbidden** | Недостаточно прав роли пользователя |
| `codes.NotFound` | **404 Not Found** | Запрашиваемый ресурс не существует |
| `codes.AlreadyExists` | **409 Conflict** | Ресурс с таким ключом уже создан |
| `codes.Aborted` | **409 Conflict** | Конфликт параллельного изменения (оптимистичная блокировка) |
| `codes.FailedPrecondition` | **400 Bad Request** | Действие недопустимо в текущем состоянии |
| `codes.ResourceExhausted` | **429 Too Many Requests** | Превышен лимит запросов или квота (при `RetryInfo` добавляются `Retry-After` и `retry_after_sec`) |
| `codes.Unavailable` | **503 Service Unavailable** | Целевой микросервис временно недоступен |
| `codes.DeadlineExceeded` | **504 Gateway Timeout** | Время ожидания ответа сервиса истекло |
| `codes.Internal` | **500 Internal Error** | Непредвиденная внутренняя ошибка |


## Метаданные текущего API расписания

Gateway использует `contracts@v0.5.12` (Schedule API совместим с `v0.5.0`) и явно преобразует `ScheduleStatus` в HTTP DTO: `READY`, `PENDING_SYNC` или `UNSPECIFIED`. Передаются `published_from` / `published_to`, если известны. См. [публичное API расписания](schedule-api.md).

## Потоковый и unary AI RPC, Users и Ops Bot

- **AI (`aiclient`)**: долгоживущее соединение, `AI_GRPC_ADDR` и `AI_RPC_TIMEOUT` (`120s` в Compose). Для потоковой генерации контекст связан с HTTP-запросом и отменяется при разрыве соединения; для коротких unary-вызовов (`Cancel`, `GetConversations`, `GetConversationHistory`, `DeleteConversation`, `SubmitFeedback`) используются отдельные ограниченные таймауты (`5–10s`). Подробнее: [AI API](ai-api.md).
- **Users (`usersclient`)**: долгоживущее соединение к `users-service:50052` (`USERS_GRPC_ADDR`, `USERS_RPC_TIMEOUT=5s`) с health-проверкой сервиса `orbit.users.v1.UserService`.
- **Ops Bot (`opsbotclient`)**: HTTP-клиент (`http.Client` с таймаутом `5s`), вызывающий `POST /issues/webhook` с `Authorization: Bearer <OPS_BOT_WEBHOOK_TOKEN>`.
