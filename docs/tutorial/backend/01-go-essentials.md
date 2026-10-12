# 01. Go в объёме проекта

Язык **Go** выбран основным языком для бэкенд-сервисов Orbit AI благодаря строгой типизации, высокой производительности, минимальному потреблению памяти и первоклассной поддержке конкурентности.

В этом руководстве разобраны ключевые концепции языка, необходимые для разработки микросервисов в проекте.

---

## 1. Структуры и методы (Value vs Pointer Receivers)

В Go нет классов и наследования. Данные описываются структурами (`struct`), а поведение — методами, привязанными к этим структурам через получатель (receiver).

### Правило выбора получателя:
1. **Value Receiver `(t Task)`**: используется для неизменяемых объектов (Value Objects) и методов чтения, где гарантируется отсутствие мутации данных. Метод оперирует независимой копией.
2. **Pointer Receiver `(t *Task)`**: используется для доменных сущностей (Entities), когда метод изменяет внутреннее состояние агрегата или структура содержит тяжелые поля.

```go
// Value Object: неизменяемый, получатель по значению
type Title struct {
    value string
}

func (t Title) String() string {
    return t.value
}

// Доменная сущность: состояние мутирует, получатель по указателю
type Task struct {
    title  Title
    status TaskStatus
}

func (t *Task) Complete() error {
    if t.status == TaskStatusCompleted {
        return ErrTaskAlreadyCompleted
    }
    t.status = TaskStatusCompleted
    return nil
}
```

---

## 2. Интерфейсы и принцип инверсии зависимостей

Интерфейсы в Go реализуются **неявно (implicit interfaces)**: структуре не нужно объявлять `implements MyInterface`. Если сигнатуры методов совпадают, тип автоматически удовлетворяет интерфейсу.

### Принципы проектирования интерфейсов в Orbit AI:
1. **Интерфейсы объявляются потребителем (Consumer-side interfaces)**:
   Интерфейс репозитория `TaskRepository` объявляется в пакете `usecase`, которому он нужен, а не в пакете `infrastructure/postgres`, который его реализует.
2. **Принцип разделения интерфейсов (Interface Segregation Principle, ISP)**:
   Вместо «толстых» монолитных интерфейсов создаются узкие контракты под конкретные сценарии:

```go
// Узкие специализированные интерфейсы:
type TaskSaver interface {
    Save(ctx context.Context, task *domain.Task) error
}

type TaskFinder interface {
    FindByID(ctx context.Context, id domain.TaskID) (*domain.Task, error)
}
```

> [!TIP] Золотое правило Go
> **«Принимай интерфейс, возвращай структуру»** (Accept interfaces, return structs). Функции-конструкторы `NewTaskUseCase(...)` должны возвращать конкретный указатель на структуру, а на вход принимать интерфейсы зависимостей.

---

## 3. Конкурентность, горутины и мьютексы

Go предоставляет легковесные потоки исполнения — **горутины** (`go doWork()`). В отличие от системных потоков ОС (весящих мегабайты), горутина стартует со стеком в 2–4 КБ.

Однако параллельный доступ к разделяемой памяти без синхронизации приводит к состоянию гонки (Data Race) и падению процесса.

### Защита состояния через `sync.RWMutex`:
В in-memory репозиториях и кэшах доступ к мапам защищается мьютексом:

```go
type InMemoryRepository struct {
    mu    sync.RWMutex
    tasks map[string]*domain.Task
}

// Чтение: блокирует запись, разрешает параллельное чтение другими горутинами
func (r *InMemoryRepository) FindByID(id string) (*domain.Task, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    task, exists := r.tasks[id]
    if !exists {
        return nil, ErrNotFound
    }
    return task.Clone(), nil
}

// Запись: монопольная блокировка
func (r *InMemoryRepository) Save(task *domain.Task) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.tasks[task.ID()] = task.Clone()
    return nil
}
```

> [!WARNING] Обязательный флаг `-race`
> Любой запуск тестов локально и в CI обязан содержать флаг `go test -race ./...`. Детектор гонок выявляет конфликты одновременного чтения и записи.

---

## 4. Контекст (`context.Context`)

Объект `context.Context` передаётся первым аргументом во все асинхронные операции, сетевые вызовы и методы репозиториев: `ctx context.Context`.

### Задачи контекста:
1. **Распространение отмены (Cancellation)**:
   Если клиент оборвал HTTP/gRPC соединение, контекст отменяется. Сервис не должен продолжать выполнять тяжелый SQL-запрос.
2. **Дедлайны и таймауты (Timeouts)**:
   Ограничивают максимальное время ожидания ответа от базы данных или внешнего сервиса.
3. **Передача метаданных запроса (Value)**:
   Передача идентификатора авторизованного пользователя (`x-user-id`) или сквозного ID трассировки (`trace-id`).

```go
func (uc *CreateTaskUseCase) Execute(ctx context.Context, input CreateTaskInput) (*CreateTaskOutput, error) {
    // Проверка, не отменён ли запрос до начала тяжелой работы
    if err := ctx.Err(); err != nil {
        return nil, err
    }

    // Передача контекста в порт сохранения
    if err := uc.repo.Save(ctx, task); err != nil {
        return nil, fmt.Errorf("сохранение задачи: %w", err)
    }

    return &CreateTaskOutput{Task: task}, nil
}
```

---

## 5. Идиоматичная обработка ошибок

В Go ошибки — это обычные значения, удовлетворяющие стандартному интерфейсу:
```go
type error interface {
    Error() string
}
```

В проекте запрещено игнорировать ошибки (`_ = doSomething()`) или выбрасывать `panic` в бизнес-логике.

```go
// Оборачивание контекстом через %w для сохранения цепочки ошибок
if err := task.Validate(); err != nil {
    return fmt.Errorf("валидация задачи: %w", err)
}

// Проверка типа ошибки через errors.Is:
if errors.Is(err, domain.ErrTaskNotFound) {
    // обработка отсутствия сущности
}
```

---

## 6. Примеры: как нельзя и как правильно

### ❌ Как нельзя:
```go
// ОШИБКА 1: Фоновая горутина без контекста и контроля
func SaveAsync(t Task) {
    go func() {
        db.Save(t) // Утечка горутины, нет обработки ошибок, нет отмены при shutdown
    }()
}

// ОШИБКА 2: Возврат интерфейса вместо структуры из конструктора
func NewTaskUseCase(r Repo) TaskUseCaseInterface { 
    return &TaskUseCase{r: r}
}

// ОШИБКА 3: Паника при ошибке конфигурации или валидации
func ParsePort(raw string) int {
    p, err := strconv.Atoi(raw)
    if err != nil {
        panic(err) // Аварийное падение вместо возврата error
    }
    return p
}
```

### ✅ Как правильно:
```go
// Конструктор возвращает конкретный указатель на структуру
func NewTaskUseCase(saver TaskSaver) *TaskUseCase {
    return &TaskUseCase{saver: saver}
}

// Метод принимает context и возвращает понятную ошибку
func (uc *TaskUseCase) Execute(ctx context.Context, input Input) error {
    if err := uc.saver.Save(ctx, input.Task); err != nil {
        return fmt.Errorf("сохранение задачи: %w", err)
    }
    return nil
}
```

---

## Резюме

1. Разделяйте Value Objects (иммутабельные структуры) и Entities (агрегаты с состоянием).
2. Объявляйте интерфейсы на стороне потребителя и держите их узкими (ISP).
3. Всегда защищайте разделяемую память мьютексами и проверяйте код через `go test -race`.
4. Пробрасывайте `ctx context.Context` первым аргументом во все методы слоёв Use Case, Infrastructure и Transport.
