package inmemory

import (
	"context"
	"sort"
	"sync"

	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/domain"
	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/usecase"
)

// --8<-- [start:inmemory_repo]
// TaskRepository — потокобезопасная in-memory реализация хранилища с оптимистичной блокировкой (version).
type TaskRepository struct {
	mu    sync.RWMutex
	tasks map[string]*domain.Task
}

func NewTaskRepository() *TaskRepository {
	return &TaskRepository{
		tasks: make(map[string]*domain.Task),
	}
}

// Save сохраняет сущность с контролем версии (защита от race conditions и параллельных изменений).
func (r *TaskRepository) Save(ctx context.Context, task *domain.Task) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	idStr := task.ID().String()
	if existing, exists := r.tasks[idStr]; exists {
		// Оптимистичная блокировка (WHERE id=$1 AND version=$2):
		// Версия в БД обязана совпадать с загруженной версией сущности
		if existing.Version() != task.Version() {
			return domain.ErrVersionConflict
		}
		task.IncrementVersion()
	}

	r.tasks[idStr] = cloneTask(task)
	return nil
}

// FindByID возвращает глубокую копию задачи по её ID.
func (r *TaskRepository) FindByID(ctx context.Context, id domain.TaskID) (*domain.Task, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	task, exists := r.tasks[id.String()]
	if !exists {
		return nil, domain.ErrTaskNotFound
	}

	return cloneTask(task), nil
}

// List возвращает отсортированный срез задач владельца с пагинацией.
func (r *TaskRepository) List(ctx context.Context, filter usecase.ListFilter) ([]*domain.Task, int, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var matched []*domain.Task
	for _, t := range r.tasks {
		if t.OwnerID().String() == filter.OwnerID.String() {
			matched = append(matched, cloneTask(t))
		}
	}

	// Детерминированная сортировка по дате создания со вторичным ключом TaskID
	sort.Slice(matched, func(i, j int) bool {
		if matched[i].CreatedAt().Equal(matched[j].CreatedAt()) {
			return matched[i].ID().String() < matched[j].ID().String()
		}
		return matched[i].CreatedAt().Before(matched[j].CreatedAt())
	})

	total := len(matched)
	if filter.Offset >= total {
		return []*domain.Task{}, total, nil
	}

	end := filter.Offset + filter.Limit
	if end > total {
		end = total
	}

	return matched[filter.Offset:end], total, nil
}

// cloneTask — приватная функция репозитория для глубокого копирования сущности (изоляция указателей).
func cloneTask(t *domain.Task) *domain.Task {
	return domain.ReconstituteTask(
		t.ID(),
		t.OwnerID(),
		t.Title(),
		t.Description(),
		t.Deadline(),
		t.Priority(),
		t.Status(),
		t.Version(),
		t.CreatedAt(),
	)
}

// --8<-- [end:inmemory_repo]
