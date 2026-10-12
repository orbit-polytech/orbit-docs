package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/domain"
)

// Ошибки слоя Use Case
var (
	ErrInvalidPagination   = errors.New("некорректные параметры пагинации (лимит от 1 до 100, смещение >= 0)")
	ErrIdempotencyConflict = errors.New("ключ идемпотентности уже использован с другими параметрами запроса")
	ErrIdempotencyInFlight = errors.New("параллельный запрос с данным ключом идемпотентности уже обрабатывается")
)

// --8<-- [start:usecase_ports]
// Interface Segregation Principle (ISP):
// Каждый сценарий зависит строго от тех интерфейсов, которые использует.

// TaskSaver — порт для сохранения или обновления задачи в хранилище.
type TaskSaver interface {
	Save(ctx context.Context, task *domain.Task) error
}

// TaskFinder — порт для поиска задачи по её идентификатору.
type TaskFinder interface {
	FindByID(ctx context.Context, id domain.TaskID) (*domain.Task, error)
}

// ListFilter — параметры пагинации для конкретного владельца задач.
type ListFilter struct {
	OwnerID domain.OwnerID
	Limit   int
	Offset  int
}

// TaskLister — порт для получения отсортированного списка задач с пагинацией.
type TaskLister interface {
	List(ctx context.Context, filter ListFilter) ([]*domain.Task, int, error)
}

// EventPublisher — порт публикации доменных событий во внешнюю шину (RabbitMQ).
type EventPublisher interface {
	PublishTaskCreated(ctx context.Context, event domain.TaskCreated) error
}

// TaskIDGenerator — порт генерации доменных идентификаторов задач.
type TaskIDGenerator interface {
	NewTaskID() (domain.TaskID, error)
}

// EventIDGenerator — порт генерации строковых идентификаторов событий без паник.
type EventIDGenerator interface {
	NewEventID() (string, error)
}

// IdempotencyStore — порт атомарной резервации и сохранения ключей идемпотентности (AIP-155).
type IdempotencyStore interface {
	Reserve(ctx context.Context, key string, requestHash string) (existingID domain.TaskID, completed bool, err error)
	Complete(ctx context.Context, key string, taskID domain.TaskID) error
	Release(ctx context.Context, key string) error
}

// Clock — абстракция системного времени для детерминированного тестирования.
type Clock interface {
	Now() time.Time
}

// NoopIdempotencyStore — реализация паттерна Null Object для безопасной работы без nil-проверок.
type NoopIdempotencyStore struct{}

func (NoopIdempotencyStore) Reserve(ctx context.Context, key, requestHash string) (domain.TaskID, bool, error) {
	return domain.TaskID{}, false, nil
}

func (NoopIdempotencyStore) Complete(ctx context.Context, key string, taskID domain.TaskID) error {
	return nil
}

func (NoopIdempotencyStore) Release(ctx context.Context, key string) error {
	return nil
}

// --8<-- [end:usecase_ports]
