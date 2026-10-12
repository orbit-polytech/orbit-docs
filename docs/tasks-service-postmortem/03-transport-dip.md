# Урок 3: Принцип инверсии зависимостей (DIP) в транспорте

## 1. Как было сделано изначально (Жёсткая привязка к структурам Use Case)

В исходном gRPC-обработчике структура `TaskHandler` объявлялась с прямыми ссылками на конкретные типы структур сценариев:

```go
// ПЛОХО: Зависимость от конкретных реализаций usecase
type TaskHandler struct {
    tasksv1.UnimplementedTaskServiceServer
    createUC     *usecase.CreateTaskUseCase
    getUC        *usecase.GetTaskUseCase
    listUC       *usecase.ListTasksUseCase
    completeUC   *usecase.CompleteTaskUseCase
    rescheduleUC *usecase.RescheduleTaskUseCase
}
```

---

## 2. В чём скрытый дефект

Такой подход прямо нарушает букву **D** в акрониме SOLID — **Dependency Inversion Principle (DIP)**:
> «Модули верхних уровней не должны зависеть от модулей нижних уровней. Оба должны зависеть от абстракций.»

**Последствия:**
1. Невозможно написать изолированный модульный тест на `TaskHandler`: чтобы проверить обработку заголовков или валидацию gRPC, приходилось создавать полные экземпляры `CreateTaskUseCase` со всеми их зависимостями (репозитории, кэши, брокеры).
2. Нарушается идиома Go: интерфейсы в Go должны принадлежать **потребителю**, а не поставщику.

---

## 3. Как правильно делать в реальном сервисе

Интерфейсы сценариев объявляются прямо внутри транспортного пакета `internal/transport/grpc`:

```go
// ХОРОШО: Consumer-driven interfaces
type taskCreator interface {
    Execute(ctx context.Context, actor domain.OwnerID, input usecase.CreateTaskInput) (*domain.Task, error)
}

type taskGetter interface {
    Execute(ctx context.Context, actor domain.OwnerID, rawID string) (*domain.Task, error)
}

type TaskHandler struct {
    tasksv1.UnimplementedTaskServiceServer
    createUC taskCreator
    getUC    taskGetter
}
```

**Преимущества:**
* Хендлер теперь требует минимально необходимый контракт.
* В тестах `handler_test.go` можно подставить моковый `itemCreator` из 5 строк и мгновенно протестировать любые сетевые коды ошибок gRPC без поднятия реальных юзкейсов.
