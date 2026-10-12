# Маршрутизация и gRPC-клиенты в Gateway

Gateway выступает в роли клиента ко всем внутренним микросервисам Orbit AI. В этом разделе описывается организация gRPC-соединений, пулов, таймаутов и обработка сбоев.

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

Вспомогательная функция `writeError` в `internal/httpapi/errors.go` транслирует статусы по стандарту Google AIP-193:

| Код статуса gRPC | HTTP статус | Описание для клиента |
|---|---|---|
| `codes.InvalidArgument` | **400 Bad Request** | Ошибка валидации параметров запроса |
| `codes.Unauthenticated` | **401 Unauthorized** | Отсутствует или недействителен токен |
| `codes.PermissionDenied` | **403 Forbidden** | Недостаточно прав роли пользователя |
| `codes.NotFound` | **404 Not Found** | Запрашиваемый ресурс не существует |
| `codes.AlreadyExists` | **409 Conflict** | Ресурс с таким ключом уже создан |
| `codes.Aborted` | **409 Conflict** | Конфликт параллельного изменения (оптимистичная блокировка) |
| `codes.FailedPrecondition` | **422 Unprocessable** | Действие недопустимо в текущем состоянии |
| `codes.ResourceExhausted` | **429 Too Many Requests** | Превышен лимит запросов или квота |
| `codes.Unavailable` | **503 Service Unavailable** | Целевой микросервис временно недоступен |
| `codes.DeadlineExceeded` | **504 Gateway Timeout** | Время ожидания ответа сервиса истекло |
| `codes.Internal` | **500 Internal Error** | Непредвиденная внутренняя ошибка |
