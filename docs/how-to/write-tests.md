# Как написать юнит- и интеграционный тест

В проекте Orbit AI высокое качество кода обеспечивается двухуровневой стратегией тестирования: быстрые изолированные **юнит-тесты** для бизнес-логики и сценариев, а также **интеграционные тесты** для проверки реального взаимодействия с PostgreSQL, Redis и RabbitMQ.

---

```mermaid
flowchart TD
    subgraph Pyramid ["Пирамида тестирования сервиса"]
        direction TB
        E2E["E2E тесты (Мало)<br/>Сквозные вызовы через Gateway и Docker"]
        Integration["Интеграционные тесты (Средне)<br/>Реальный PostgreSQL, pgvector, RabbitMQ в CI"]
        Unit["Юнит-тесты (Много)<br/>Бизнес-инварианты Domain и Usecase (< 1 сек, без внешних БД)"]
    end
    Unit --> Integration --> E2E
```

| Тип теста | Где находится | Зависимости | Скорость выполнения |
| :--- | :--- | :--- | :--- |
| **Юнит-тесты** | Рядом с файлом (`*_test.go`) | In-memory, моки портов | < 1 секунды |
| **Интеграционные** | `internal/infrastructure/postgres/...` | Реальный Docker Postgres | 5–15 секунд |

---

## 2. Написание юнит-теста для слоя Usecase

Юнит-тесты сценариев проверяют бизнес-правила и обработку ошибок портов. Мы используем легковесные in-memory заглушки интерфейсов (портов ISP).

### Пример теста `CreateTaskUseCase`

```go
package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/orbit-polytech/tasks-service/internal/domain"
	"github.com/orbit-polytech/tasks-service/internal/infrastructure/inmemory"
	"github.com/orbit-polytech/tasks-service/internal/usecase"
)

// Мок генератора идентификаторов для детерминированных тестов
type stubIDGen struct {
	nextID domain.TaskID
	err    error
}

func (s *stubIDGen) NewTaskID() (domain.TaskID, error) {
	return s.nextID, s.err
}

func TestCreateTaskUseCase_Success(t *testing.T) {
	ctx := context.Background()
	repo := inmemory.NewTaskRepository()
	publisher := inmemory.NewEventPublisher(10)
	idStore := inmemory.NewIdempotencyStore()
	
	validTaskID, _ := domain.NewTaskID("018f2345-6789-7abc-def0-123456789abc")
	idGen := &stubIDGen{nextID: validTaskID}

	uc := usecase.NewCreateTaskUseCase(repo, publisher, idGen, idStore)

	actor, _ := domain.NewOwnerID("user-123")
	futureDeadline := time.Now().UTC().Add(24 * time.Hour)

	input := usecase.CreateTaskInput{
		Actor:          actor,
		Title:          "Подготовить доклад по архитектуре",
		Description:    "Раздел Clean Architecture",
		Priority:       domain.PriorityHigh,
		Deadline:       &futureDeadline,
		IdempotencyKey: "unique-key-1",
	}

	// 1. Выполнение сценария
	task, err := uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("неожиданная ошибка создания задачи: %v", err)
	}

	// 2. Проверка инвариантов созданной задачи
	if task.ID() != validTaskID {
		t.Errorf("ожидался ID %s, получено %s", validTaskID, task.ID())
	}
	if task.Status() != domain.StatusOpen {
		t.Errorf("ожидался статус OPEN, получено %s", task.Status())
	}

	// 3. Проверка публикации доменного события
	events := publisher.Events()
	if len(events) != 1 {
		t.Fatalf("ожидалось 1 событие TaskCreated, получено %d", len(events))
	}
	if events[0].TaskID != validTaskID {
		t.Errorf("событие содержит неверный TaskID: %s", events[0].TaskID)
	}
}

func TestCreateTaskUseCase_Idempotency(t *testing.T) {
	ctx := context.Background()
	repo := inmemory.NewTaskRepository()
	publisher := inmemory.NewEventPublisher(10)
	idStore := inmemory.NewIdempotencyStore()
	validTaskID, _ := domain.NewTaskID("018f2345-6789-7abc-def0-123456789abc")
	idGen := &stubIDGen{nextID: validTaskID}

	uc := usecase.NewCreateTaskUseCase(repo, publisher, idGen, idStore)
	actor, _ := domain.NewOwnerID("user-123")

	input := usecase.CreateTaskInput{
		Actor:          actor,
		Title:          "Идемпотентная задача",
		Priority:       domain.PriorityLow,
		IdempotencyKey: "dup-key-999",
	}

	// Первый вызов — создание
	task1, err := uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("первый вызов завершился ошибкой: %v", err)
	}

	// Второй вызов с тем же ключом идемпотентности
	task2, err := uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("повторный вызов завершился ошибкой: %v", err)
	}

	// Должна вернуться та же самая задача без генерации второго события
	if task1.ID() != task2.ID() {
		t.Errorf("ID задач не совпадают при повторном ключе идемпотентности")
	}
	if len(publisher.Events()) != 1 {
		t.Errorf("при повторном ключе не должно публиковаться новое событие, получено %d", len(publisher.Events()))
	}
}
```

---

## 3. Запуск тестов с детектором состояний гонки (Race Detector)

Все юнит-тесты в Orbit AI **обязаны** проходить проверку детектором гонок Go:

```bash
go test -v -race -count=1 ./...
```

Флаг `-race` компилирует код со встроенным инструментом ThreadSanitizer и ловит одновременный доступ к разделяемой памяти без синхронизации. Флаг `-count=1` отключает кэширование результатов тестов.

---

## 4. Написание интеграционного теста базы данных

Интеграционные тесты работают с реальным PostgreSQL и изолируются директивой сборки `//go:build integration` в первой строке файла:

```go
//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/orbit-polytech/tasks-service/internal/domain"
	"github.com/orbit-polytech/tasks-service/internal/infrastructure/postgres"
)

func TestPostgresTaskRepository_SaveAndFind(t *testing.T) {
	ctx := context.Background()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL не задан, пропускаем интеграционный тест")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("ошибка подключения к тестовой БД: %v", err)
	}
	defer pool.Close()

	repo := postgres.NewTaskRepository(pool)

	taskID, _ := domain.NewTaskID("018f2345-6789-7abc-def0-123456789abc")
	ownerID, _ := domain.NewOwnerID("test-user-1")
	title, _ := domain.NewTitle("Интеграционный тест БД")
	now := time.Now().UTC()

	task, err := domain.NewTask(taskID, ownerID, title, "описание", domain.PriorityMedium, nil, now)
	if err != nil {
		t.Fatalf("ошибка создания доменной сущности: %v", err)
	}

	// 1. Сохранение в Postgres
	if err := repo.Save(ctx, task); err != nil {
		t.Fatalf("ошибка сохранения в реальную БД: %v", err)
	}

	// 2. Поиск по ID
	found, err := repo.FindByID(ctx, taskID)
	if err != nil {
		t.Fatalf("ошибка чтения из БД: %v", err)
	}

	if found.Title() != task.Title() {
		t.Errorf("ожидался заголовок %s, получено %s", task.Title(), found.Title())
	}
}
```

Запуск интеграционных тестов:

```bash
# Обычный запуск проигнорирует файлы с тегом integration
go test ./...

# Явный запуск интеграционного набора
TEST_DATABASE_URL="postgres://orbit:orbit_dev_secret@localhost:5432/tasks_test_db?sslmode=disable" \
go test -tags=integration -v ./...
```

---

## 5. Сравнение: Как нельзя и Как правильно

<div class="grid" markdown>

<div class="card" markdown>
### :x: Как нельзя

- **Использовать `time.Sleep(100 * time.Millisecond)` для ожидания горутины.** Это делает тесты медленными и нестабильными (Flaky Tests). Используйте `sync.WaitGroup`, каналы или таймауты контекста.
- **Запускать тесты без флага `-race`.** Скрытые состояния гонки проявятся только под нагрузкой в продакшене.
- **Подключаться к реальной базе данных в юнит-тестах слоя Usecase.** Юнит-тесты должны запускаться за миллисекунды без запущенного Docker.
- **Писать один гигантский тест на 300 строк.** Разбивайте проверки на изолированные сценарии с помощью `t.Run("название сценария", func(t *testing.T) { ... })`.
</div>

<div class="card" markdown>
### :white_check_mark: Как правильно

- **Проверять бизнес-поведение, а не внутренние приватные переменные.**
- **Использовать простые In-memory реализации интерфейсов (портов ISP).**
- **Всегда запускать `go test -race -count=1 ./...` перед открытием Pull Request.**
- **Изолировать тесты к внешним ресурсам директивой `//go:build integration`.**
</div>

</div>
