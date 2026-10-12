# Рецепты: «Хочу добавить X, куда писать?»

Когда перед вами стоит задача в Go-сервисе, не нужно гадать, где создавать код. Используйте эту практическую шпаргалку.

---

## 1. Навигационная матрица изменений

| Что я хочу сделать | В какие файлы и папки писать код |
|---|---|
| **Добавить новое поле к сущности** (например, `priority` у задачи) | 1. Поле и методы в `internal/domain/task.go`.<br>2. SQL-миграция `migrations/00000N_add_priority.up.sql`.<br>3. Маппинг в `internal/adapters/postgres/task_repo.go`.<br>4. Описание поля в proto в репозитории `contracts`.<br>5. Маппинг DTO в `internal/transport/grpc/tasks_handler.go`. |
| **Добавить новое бизнес-правило** (например, «нельзя завершить отменённую задачу») | 1. Проверка в методе сущности `internal/domain/task.go`.<br>2. Новая доменная ошибка в `internal/domain/errors.go`.<br>3. Модульный юнит-тест в `internal/domain/task_test.go`. |
| **Реализовать новый сценарий** (например, «перенести дедлайн лабораторной») | 1. Новый файл `internal/app/reschedule_task.go`.<br>2. Юнит-тест со моками в `internal/app/reschedule_task_test.go`.<br>3. Вызов сценария из `internal/transport/grpc/tasks_handler.go`. |
| **Добавить новый RPC-метод** | 1. Сначала объявить метод в `.proto` в репозитории `contracts`.<br>2. Запустить `buf generate`.<br>3. Реализовать метод в `internal/transport/grpc/tasks_handler.go`. |
| **Написать новый SQL-запрос к базе** | 1. Метод в `internal/adapters/postgres/task_repo.go` (или `queries.sql` при использовании sqlc).<br>2. Интеграционный тест с реальной БД или pgxmock. |
| **Опубликовать событие в шину RabbitMQ** | 1. Доменный интерфейс `EventPublisher` в `internal/app/ports.go`.<br>2. Реализация AMQP-паблишера в `internal/adapters/rabbitmq/publisher.go`. |
| **Добавить кэширование в Redis** | 1. Интерфейс кэша в `internal/app/ports.go`.<br>2. Реализация на `go-redis` в `internal/adapters/redis/cache.go`. |
| **Добавить новую переменную окружения** | 1. Новое поле в структуре `internal/config/config.go`.<br>2. Строка с примером значения в `.env.example`. |
| **Добавить интерцептор** (логирование, трейсинг, авторизация) | 1. Функция интерцептора в `internal/transport/grpc/interceptors.go`.<br>2. Подключение в цепочку `grpc.ChainUnaryInterceptor` в `cmd/<service>/main.go`. |
| **Добавить маппинг ошибки в gRPC-статус** | `internal/transport/grpc/errors.go` (добавить ветку в `switch`). |
| **Написать вспомогательную функцию «на будущее»** | **Никуда.** Не создавайте абстрактный код про запас. Напишите приватную функцию ровно в том файле, где она используется прямо сейчас. |

---

## 2. Типовые сценарии

### Как добавить SQL-миграцию
Используем утилиту `goose` или `golang-migrate`:

```bash
# Создание пары файлов up и down
goose create add_deadline_to_tasks sql
```

Файл `000002_add_deadline_to_tasks.sql`:
```sql
-- +goose Up
ALTER TABLE tasks ADD COLUMN due_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- +goose Down
ALTER TABLE tasks DROP COLUMN due_at;
```

> [!IMPORTANT]
> Миграция `down` обязана полностью и без ошибок откатывать всё, что создала миграция `up`. Проверяйте откат локально через `make migrate-down`.

---

### Как протестировать usecase без поднятия базы данных
Благодаря разделению портов по принципу ISP (Interface Segregation), в тестах `internal/app/` мы передаём легковесные mock-структуры в памяти:

```go
type mockTaskSaver struct {
    savedTask *domain.Task
}

func (m *mockTaskSaver) Save(ctx context.Context, t *domain.Task) error {
    m.savedTask = t
    return nil
}

func TestCreateTask(t *testing.T) {
    saver := &mockTaskSaver{}
    uc := app.NewCreateTaskUseCase(saver, mockClock{})

    res, err := uc.Execute(context.Background(), "t-1", "Тестовая задача")
    if err != nil {
        t.Fatalf("неожиданная ошибка: %v", err)
    }
    if saver.savedTask == nil {
        t.Fatal("задача не была передана в репозиторий")
    }
}
```
Тест выполняется за доли миллисекунды, не требует Docker и не зависит от сети.
