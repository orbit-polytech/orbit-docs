# Маршрутизация и gRPC-клиенты в Gateway

В текущем `main` подключены Auth и Schedule: `internal/infrastructure/authclient` и `internal/infrastructure/scheduleclient`. Соединения долгоживущие; таймаут каждого RPC задаётся от HTTP-контекста. Schedule получает `SCHEDULE_GRPC_ADDR` (в Compose `schedule-service:50054`) и `SCHEDULE_RPC_TIMEOUT` (`5s`).

Пустые `date_from`/`date_to` передаются в Schedule без вычисления дат в Gateway; фактические границы ответа возвращаются клиенту. Общий маппер находится в `internal/transport/http/httputil/grpc_error.go`.

!!! info "Ниже — учебная схема расширения"
    Фабрика `internal/clients`, Tasks и код `writeError` ниже показывают возможную организацию будущих интеграций. Это не дерево текущего репозитория. Реальные файлы перечислены в [обзоре Gateway](overview.md).

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
| `codes.ResourceExhausted` | **429 Too Many Requests** | Превышен лимит запросов или квота |
| `codes.Unavailable` | **503 Service Unavailable** | Целевой микросервис временно недоступен |
| `codes.DeadlineExceeded` | **504 Gateway Timeout** | Время ожидания ответа сервиса истекло |
| `codes.Internal` | **500 Internal Error** | Непредвиденная внутренняя ошибка |


## Метаданные текущего API расписания

Gateway использует `contracts v0.5.0` и явно преобразует `ScheduleStatus` в HTTP DTO: `READY`, `PENDING_SYNC` или `UNSPECIFIED` для неизвестного значения. Передаются `published_from` / `published_to`, если известны. Ожидание подтверждённого фонового задания — успешный `200`, не gRPC-ошибка. Правила периода, кэш и постановка задач остаются в Schedule.

См. [публичное API расписания](schedule-api.md).
