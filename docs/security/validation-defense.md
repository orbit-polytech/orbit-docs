# Валидация входных данных и защита от атак

В этом руководстве описываются стандарты защиты микросервисов Orbit AI от распространённых уязвимостей (OWASP Top 10).

---

## 1. Защита от прямого обращения к чужим объектам (IDOR)

**Insecure Direct Object Reference (IDOR)** — наиболее частая уязвимость в студенческих проектах. Она возникает, когда сервис принимает идентификатор ресурса (`task_id` или `grade_id`) и возвращает данные без проверки, принадлежит ли этот объект текущему пользователю.

### Неправильно:
```go
// УЯЗВИМО: любой студент может передать чужой ID и увидеть чужие задачи!
func (uc *GetTaskUseCase) Execute(ctx context.Context, id domain.TaskID) (*domain.Task, error) {
    return uc.finder.FindByID(ctx, id)
}
```

### Правильно:
```go
// БЕЗОПАСНО: сервис проверяет владение ресурсом
func (uc *GetTaskUseCase) Execute(ctx context.Context, actor domain.OwnerID, id domain.TaskID) (*domain.Task, error) {
    task, err := uc.finder.FindByID(ctx, id)
    if err != nil {
        return nil, err
    }

    // Если задача принадлежит другому студенту — возвращаем 404 Not Found!
    if !task.IsOwnedBy(actor) {
        return nil, domain.ErrTaskNotFound
    }

    return task, nil
}
```

> [!TIP]
> Для новых сервисов вроде `tasks-service` рекомендуется возвращать `ErrTaskNotFound` (`404`), а не `PermissionDenied` (`403`), чтобы не подтверждать злоумышленнику существование чужого ID. При этом учитывайте, что в текущем `ai-service` при обращении к чужому диалогу или сообщению возвращается `PermissionDeniedError` (HTTP `403 PERMISSION_DENIED`), а при отсутствии записи — `404 NOT_FOUND`.

---

## 2. Защита от SQL-инъекций

В Go и Python категорически запрещено склеивать SQL-запросы из строк пользователя через конкатенацию или форматирование (`fmt.Sprintf`, f-strings).

### Всегда используем плейсхолдеры СУБД:
```go
// Правильно (pgx параметризованный запрос):
row := pool.QueryRow(ctx, "SELECT id, title FROM tasks WHERE id = $1 AND owner_id = $2", taskID, ownerID)
```

В таком варианте драйвер СУБД передает параметры отдельно от текста запроса, и внедрить вредоносный SQL физически невозможно.

---

## 3. Защита от подмены привилегий (Mass Assignment) и ограничение тела

Системные поля (`role`, `account_id`, владелец диалога, статусы) **никогда не принимаются из входящего JSON DTO клиента**:
- Текущие Ed25519 JWT-токены **не несут claim `role`** (они содержат `sub`, `iss`, `aud`, `iat`, `exp`, `auth_time`). Системные роли (`unspecified`, `student`, `teacher`, `admin`) хранятся в `users-service` (`profiles.role`) и отсутствуют во входящем DTO `PUT /api/v1/users/profile`, поэтому клиент не может сам назначить себе роль через API профиля.
- Идентификатор пользователя `account_id` / `x-user-id` извлекается в Gateway исключительно из проверенного JWT, а не из тела запроса или клиентских заголовков.
- Входной JSON в Gateway декодируется через `httputil.DecodeJSONBody` с `http.MaxBytesReader` (`16 KiB` по умолчанию, `64 KiB` для AI), обязательной проверкой `Content-Type: application/json` (на обычных JSON-маршрутах) и `DisallowUnknownFields()`.
