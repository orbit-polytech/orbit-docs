# Обработчики и Middleware в Gateway

В этом разделе рассматривается устройство роутера, middleware и типовой код обработчиков в `gateway-service`.

---

## 1. Маршрутизация на `go-chi/chi`

Все эндпоинты регистрируются в едином файле `internal/httpapi/router.go`. Маршруты разделены на публичные (без авторизации, но со строгим ограничением частоты по IP) и защищённые (требующие валидного JWT):

```go
func NewRouter(deps Dependencies) http.Handler {
    r := chi.NewRouter()

    // Глобальные middleware для каждого запроса
    r.Use(middleware.RequestID)
    r.Use(middleware.Recover(deps.Logger))
    r.Use(middleware.Logger(deps.Logger))
    r.Use(middleware.CORS(deps.Config.AllowedOrigins))

    // Проверка здоровья для Docker healthcheck
    r.Get("/healthz", healthHandler)

    r.Route("/v1", func(r chi.Router) {
        // Публичная группа: строгий лимит 10 запросов в минуту на IP
        r.Group(func(r chi.Router) {
            r.Use(deps.RateLimiter.PerIP(10, time.Minute))
            r.Post("/auth/telegram", deps.AuthHandler.LoginTelegram)
            r.Post("/auth/refresh", deps.AuthHandler.RefreshToken)
        })

        // Защищённая группа: требует валидного JWT и разрешает 120 запросов в минуту
        r.Group(func(r chi.Router) {
            r.Use(middleware.Authenticate(deps.JWTVerifier))
            r.Use(deps.RateLimiter.PerUser(120, time.Minute))

            r.Post("/auth/logout", deps.AuthHandler.Logout)

            // Задачи студента
            r.Get("/tasks", deps.TasksHandler.ListTasks)
            r.Post("/tasks", deps.TasksHandler.CreateTask)
            r.Patch("/tasks/{id}", deps.TasksHandler.UpdateTask)
        })
    })

    return r
}
```

---

## 2. Безопасный приём JSON (`readJSON`)

HTTP-обработчик обязан защищать сервис от атак типа Slowloris и переполнения памяти слишком большим телом запроса:

```go
func readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
    // Ограничиваем максимальный размер тела 1 МБ
    r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

    dec := json.NewDecoder(r.Body)
    // Запрещаем неизвестные поля: защита от опечаток и скрытых полей
    dec.DisallowUnknownFields()

    return dec.Decode(dst)
}
```

---

## 3. Типовой обработчик эндпоинта

Обработчик в Gateway делает ровно четыре действия:
1. Читает и декодирует JSON.
2. Проверяет базовую форму DTO (`Validate()`).
3. Вызывает gRPC-метод внутреннего сервиса.
4. Отправляет клиенту JSON-ответ.

```go
func (h *TasksHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
    var req dto.CreateTaskRequest
    if err := readJSON(w, r, &req); err != nil {
        writeError(w, r, errBadRequest(err))
        return
    }

    if err := req.Validate(); err != nil {
        writeError(w, r, errBadRequest(err))
        return
    }

    // Извлекаем x-user-id, подготовленный middleware авторизации
    userID := r.Context().Value(contextUserID).(string)

    // Пробрасываем контекст и вызываем gRPC-клиент tasks-service
    ctx := metadata.AppendToOutgoingContext(r.Context(), "x-user-id", userID)
    resp, err := h.tasksClient.CreateTask(ctx, &tasksv1.CreateTaskRequest{
        Title:          req.Title,
        DueAt:          timestamppb.New(req.DueAt),
        IdempotencyKey: r.Header.Get("Idempotency-Key"),
    })
    if err != nil {
        // Трансляция gRPC-статуса в HTTP-код
        writeError(w, r, err)
        return
    }

    writeJSON(w, http.StatusCreated, dto.TaskFromProto(resp.GetTask()))
}
```
