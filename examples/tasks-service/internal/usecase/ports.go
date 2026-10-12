package usecase

import (
	"context"
	"time"

	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/domain"
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

// EventIDGenerator — порт генерации строковых идентификаторов событий.
type EventIDGenerator interface {
	NewEventID() string
}

// IdempotencyStore — порт проверки и сохранения ключей идемпотентности.
type IdempotencyStore interface {
	Get(ctx context.Context, key string) (domain.TaskID, bool, error)
	Save(ctx context.Context, key string, taskID domain.TaskID) error
}

// Clock — абстракция системного времени для детерминированного тестирования.
type Clock interface {
	Now() time.Time
}

// --8<-- [end:usecase_ports]
