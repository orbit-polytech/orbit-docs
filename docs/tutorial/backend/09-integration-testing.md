# 09. Интеграционное тестирование (Postgres + RabbitMQ)

В то время как модульные тесты (глава 6.8) проверяют логику слоёв в памяти с помощью моков, **интеграционные тесты** проверяют работоспособность кода в связке с реальными инфраструктурными компонентами: СУБД **PostgreSQL 16** и брокером **RabbitMQ 3.13**.

---

## 1. Разделение тестов: Unit vs Integration

Чтобы запуск повседневных тестов оставался мгновенным (доли секунды), интеграционные тесты отделяются специальным тегом сборки Go:

```go
//go:build integration
```

### Команды запуска:
```bash
# Быстрые юнит-тесты без внешних зависимостей (запуск по умолчанию):
go test -v -race -count=1 ./...

# Интеграционные тесты с реальными PostgreSQL и RabbitMQ:
go test -v -race -tags=integration -count=1 ./...
```

---

## 2. Настройка тестового окружения и очистка данных

Интеграционные тесты подключаются к контейнерам, поднятым через `infra`:

```go
// internal/infrastructure/postgres/task_repository_integration_test.go
//go:build integration

package postgres_test

import (
    "context"
    "os"
    "testing"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/domain"
    "github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/infrastructure/postgres"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
    t.Helper()

    dsn := os.Getenv("TEST_POSTGRES_DSN")
    if dsn == "" {
        dsn = "postgres://tasks_user:tasks_pass@localhost:5432/tasks_db?sslmode=disable"
    }

    ctx := context.Background()
    pool, err := postgres.NewPool(ctx, dsn)
    require.NoError(t, err, "не удалось подключиться к тестовой БД")

    // Очистка таблиц перед каждым тестом и после него (Cleanup)
    cleanDB := func() {
        _, _ = pool.Exec(ctx, "TRUNCATE TABLE tasks, outbox_events CASCADE")
    }
    cleanDB()
    t.Cleanup(cleanDB)

    return pool
}
```

---

## 3. Тестирование оптимистичной блокировки в реальной БД

Тест проверяет, что две параллельные транзакции в реальном PostgreSQL не приводят к состоянию гонки:

```go
func TestPostgresTaskRepository_OptimisticLocking_Concurrent(t *testing.T) {
    pool := setupTestDB(t)
    repo := postgres.NewTaskRepository(pool)
    ctx := context.Background()
    now := time.Now().UTC()

    // 1. Создаем начальную задачу в БД (версия 1)
    taskID, _ := domain.NewTaskID("task-race-1")
    ownerID, _ := domain.NewOwnerID("student-race-1")
    title, _ := domain.NewTitle("Лабораторная работа по базам данных")
    deadline := domain.ReconstituteDeadline(now.Add(24 * time.Hour))

    task, err := domain.NewTask(taskID, ownerID, title, "Описание", deadline, domain.TaskPriorityHigh, now)
    require.NoError(t, err)
    require.NoError(t, repo.Save(ctx, task))

    // 2. Две горутины пытаются одновременно завершить задачу
    errCh := make(chan error, 2)

    go func() {
        t1, err := repo.FindByID(ctx, taskID)
        if err != nil {
            errCh <- err
            return
        }
        _ = t1.Complete() // version 1 -> 2
        errCh <- repo.Save(ctx, t1)
    }()

    go func() {
        t2, err := repo.FindByID(ctx, taskID)
        if err != nil {
            errCh <- err
            return
        }
        _ = t2.Complete() // version 1 -> 2
        errCh <- repo.Save(ctx, t2)
    }()

    // 3. Анализ результатов: ровно одна горутина завершилась успехом, а вторая получила ErrVersionConflict
    err1 := <-errCh
    err2 := <-errCh

    hasSuccess := (err1 == nil) || (err2 == nil)
    hasConflict := assert.ErrorIs(t, err1, domain.ErrVersionConflict) || assert.ErrorIs(t, err2, domain.ErrVersionConflict)

    assert.True(t, hasSuccess, "одна из горутин должна успешно обновить задачу")
    assert.True(t, hasConflict, "вторая горутина должна получить ErrVersionConflict")
}
```

---

## 4. Сквозное тестирование Outbox и доставки в RabbitMQ

Тест проверяет всю цепочку: атомарная запись в PostgreSQL -> вычитка воркером -> получение сообщения из очереди RabbitMQ:

![Сквозное тестирование Outbox и доставки в RabbitMQ](image/outbox-test-flow.png)


```go
func TestOutbox_EndToEnd_Delivery(t *testing.T) {
    pool := setupTestDB(t)
    repo := postgres.NewTaskRepository(pool)
    amqpConn := setupTestRabbitMQ(t)
    defer amqpConn.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // 1. Создание задачи с Outbox-событием
    event := domain.TaskCreated{
        EventID:    "event-test-1",
        TaskID:     mustTaskID(t, "task-1"),
        OwnerID:    mustOwnerID(t, "student-1"),
        Title:      mustTitle(t, "Тестовая задача"),
        OccurredAt: time.Now().UTC(),
    }
    require.NoError(t, repo.CreateWithOutbox(ctx, task, event))

    // 2. Запуск Outbox воркера
    worker := postgres.NewOutboxWorker(pool, amqpConn)
    require.NoError(t, worker.ProcessBatch(ctx))

    // 3. Подписчик проверяет получение сообщения из очереди
    msg, err := readOneMessageFromQueue(ctx, amqpConn, "tasks.v1.task.created")
    require.NoError(t, err)
    assert.Equal(t, event.EventID, msg.MessageId)
}
```

---

## 5. Антипаттерны: как нельзя и как правильно

### ❌ Как нельзя:
```go
// ОШИБКА 1: Запуск тестов на общей рабочей базе
// Тест вызывает TRUNCATE и стирает данные коллег по команде!

// ОШИБКА 2: Ожидание сообщений через time.Sleep
time.Sleep(2 * time.Second) // Флакует в медленном CI, замедляет пайплайн

// ОШИБКА 3: Забытая очистка данных между тестами
// Второй тест падает из-за того, что ID уже занят первым тестом
```

### ✅ Как правильно:
```go
// Изоляция через t.Cleanup:
t.Cleanup(func() {
    pool.Exec(ctx, "TRUNCATE tasks, outbox_events CASCADE")
})

// Ожидание асинхронных сообщений через каналы с контекстным таймаутом:
select {
case msg := <-msgCh:
    // проверка сообщения
case <-ctx.Done():
    t.Fatal("сообщение не пришло в очередь за отведённый таймаут")
}
```

---

## Резюме

1. Интеграционные тесты отделены тегом `//go:build integration`.
2. Тестовая БД изолируется и очищается через `t.Cleanup()`.
3. Оптимистичная блокировка верифицируется в условиях реальной конкуренции горутин.
4. Доставка через Outbox в RabbitMQ проверяется сквозным тестом.
