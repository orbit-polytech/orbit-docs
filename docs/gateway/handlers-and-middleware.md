# Обработчики и Middleware в Gateway

В этом разделе рассматривается устройство роутера, middleware и типовой код обработчиков в `gateway-service`.

---

## 1. Текущая маршрутизация и middleware

В `main` используется **`net/http.ServeMux`** ([`internal/transport/http/router.go`](https://github.com/orbit-polytech/gateway-service/blob/main/internal/transport/http/router.go)). Все бизнес-маршруты начинаются с префикса `/api/v1`:

- поиск групп и расписание регистрируются без JWT middleware;
- публичные точки Auth (`send-code`, `verify-code`, `refresh`, `logout`, `oauth/*/login`, `oauth/*/callback`, `oauth/exchange`, `identities/link/finish`) работают без Access JWT, проверяя свои токены/state/cookies внутри обработчиков;
- защищённые маршруты Auth (`/auth/me`, `PATCH /auth/preferences`, `POST /auth/email/change/begin`, `POST /auth/email/change/confirm`, `/auth/identities/link/begin`, `GET /identities`, `DELETE /identities/{provider}`), маршруты Users (`GET/PUT /api/v1/users/profile`) и все 6 маршрутов AI (`/api/v1/ai/...`) явно оборачиваются в `middleware.Auth(validator)`;
- `POST /api/v1/ai/chat/stream` дополнительно обёрнут в `middleware.AIRateLimit` (Lua-скрипт скользящего окна + cooldown в Redis под префиксом `orbit:ai:rl`, по умолчанию `15` запросов в `1h` и `5s` cooldown; при недоступности Redis — fallback на in-memory);
- `POST /api/v1/support/issues` использует `middleware.OptionalAuth(validator)` и `middleware.SupportIssueRateLimit` (префикс `orbit:support:rl`, `5` запросов за `15m` и `30s` cooldown по `user:<id>` или `ip:<ip>`).

Цепочка общих middleware на основном роутере: `RequestID` → `Logger` (JSON `log/slog` с `service="gateway-service"`) → `Metrics` (`orbit_gateway_http_requests_total`, `orbit_gateway_http_request_duration_seconds`, `orbit_gateway_http_in_flight_requests` по `r.Pattern`) → `Recovery` → `CORS`.

Для Schedule handler читает `group`, необязательные `date_from`/`date_to`, создаёт контекст с `SCHEDULE_RPC_TIMEOUT`, вызывает gRPC и преобразует DTO. Выбор недели остаётся в usecase Schedule. См. [полный API](schedule-api.md).

## 2. Учебный вариант маршрутизации на `go-chi/chi`

!!! info "Учебный пример, не текущий роутер"
    Пример ниже сохранён для обучения. `go-chi/chi`, `Tasks` и Telegram-эндпоинт `/auth/telegram` не подключены в текущем Gateway. При этом Redis rate limiting для AI (`AIRateLimit`) и поддержки (`SupportIssueRateLimit`) уже работает в реальном `net/http.ServeMux`. Копировать код ниже как список текущих маршрутов нельзя.

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

## 3. Безопасный приём JSON (`httputil.DecodeJSONBody`)

HTTP-обработчик обязан защищать сервис от атак типа Slowloris и переполнения памяти слишком большим телом запроса. В реальном `gateway-service/internal/transport/http/httputil/json.go` используются строгие лимиты:

- **`16 KiB` (`16 * 1024`)** — лимит по умолчанию (`DecodeJSONBody`) с обязательным `Content-Type: application/json` (иначе `415 UNSUPPORTED_MEDIA_TYPE`), `DisallowUnknownFields()`, проверкой на ровно один JSON-объект и статусом `413 PAYLOAD_TOO_LARGE` при превышении размера;
- **`64 KiB` (`64 * 1024`)** — увеличенный лимит для AI-запросов (`internal/transport/http/v1/ai/handler.go`, через `DecodeJSONBodyWithOptions`).

Ниже показан упрощённый учебный шаблон для понимания механики (`http.MaxBytesReader` + `DisallowUnknownFields`):

```go
func readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
    // В реальном коде Gateway: 16 KiB по умолчанию (64 KiB для AI)
    r.Body = http.MaxBytesReader(w, r.Body, 16*1024)

    dec := json.NewDecoder(r.Body)
    // Запрещаем неизвестные поля: защита от опечаток и скрытых полей
    dec.DisallowUnknownFields()

    return dec.Decode(dst)
}
```

---

## 4. Типовой учебный обработчик эндпоинта

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


## Метаданные текущего API расписания

Gateway использует `contracts@v0.5.12` (Schedule API совместим с `v0.5.0`) и явно маппит `ScheduleStatus` в HTTP DTO: `READY`, `PENDING_SYNC` или `UNSPECIFIED`. Передаются `published_from` и `published_to`, если они известны. Ожидание фоновой синхронизации — это `200`, а не gRPC-ошибка. См. [публичное API расписания](schedule-api.md).

## AI, Users и Support handlers

- **AI** (`internal/transport/http/v1/ai`): `stream.go` (валидация, SSE-поток), `cancel.go`, `conversations.go` (список, история, удаление) и `feedback.go` (лайки/дизлайки). Подробности — в [AI API](ai-api.md).
- **Users** (`internal/transport/http/v1/users`): `GetProfile` и `UpdateProfile` для `/api/v1/users/profile`.
- **Support** (`internal/transport/http/v1/support`): `CreateIssue` для `POST /api/v1/support/issues` (валидация категории/длины и отправка в `ops-bot`).
