package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/domain"
)

// --8<-- [start:load_owned_helper]
// loadOwned централизованно загружает задачу и валидирует права доступа.
// Если задача принадлежит другому пользователю, возвращается ErrTaskNotFound, предотвращая перебор ID (IDOR).
func loadOwned(ctx context.Context, f TaskFinder, actor domain.OwnerID, rawID string) (*domain.Task, error) {
	if actor.IsZero() {
		return nil, domain.ErrEmptyOwnerID
	}

	taskID, err := domain.NewTaskID(rawID)
	if err != nil {
		return nil, fmt.Errorf("некорректный ID задачи: %w", err)
	}

	task, err := f.FindByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if !task.IsOwnedBy(actor) {
		return nil, domain.ErrTaskNotFound
	}

	return task, nil
}

// --8<-- [end:load_owned_helper]

// --8<-- [start:get_task_usecase]
// GetTaskUseCase сценарий получения задачи по ID.
type GetTaskUseCase struct {
	finder TaskFinder
}

func NewGetTaskUseCase(finder TaskFinder) *GetTaskUseCase {
	return &GetTaskUseCase{finder: finder}
}

func (uc *GetTaskUseCase) Execute(ctx context.Context, actor domain.OwnerID, rawID string) (*domain.Task, error) {
	return loadOwned(ctx, uc.finder, actor, rawID)
}

// --8<-- [end:get_task_usecase]

// --8<-- [start:list_tasks_usecase]
// ListTasksUseCase сценарий получения списка задач владельца с явной валидацией пагинации.
type ListTasksUseCase struct {
	lister TaskLister
}

func NewListTasksUseCase(lister TaskLister) *ListTasksUseCase {
	return &ListTasksUseCase{lister: lister}
}

func (uc *ListTasksUseCase) Execute(ctx context.Context, actor domain.OwnerID, limit, offset int) ([]*domain.Task, int, error) {
	if actor.IsZero() {
		return nil, 0, domain.ErrEmptyOwnerID
	}

	// Никаких тихих умолчаний: невалидные значения лимита и смещения возвращают ошибку
	if limit <= 0 || limit > 100 {
		return nil, 0, fmt.Errorf("%w: лимит %d должен быть в диапазоне [1, 100]", domain.ErrInvalidPagination, limit)
	}
	if offset < 0 {
		return nil, 0, fmt.Errorf("%w: смещение %d не может быть отрицательным", domain.ErrInvalidPagination, offset)
	}

	filter := ListFilter{
		OwnerID: actor,
		Limit:   limit,
		Offset:  offset,
	}

	tasks, total, err := uc.lister.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("получение списка задач: %w", err)
	}

	return tasks, total, nil
}

// --8<-- [end:list_tasks_usecase]

// --8<-- [start:complete_task_usecase]
// CompleteTaskUseCase сценарий завершения задачи.
type CompleteTaskUseCase struct {
	finder TaskFinder
	saver  TaskSaver
}

func NewCompleteTaskUseCase(finder TaskFinder, saver TaskSaver) *CompleteTaskUseCase {
	return &CompleteTaskUseCase{
		finder: finder,
		saver:  saver,
	}
}

func (uc *CompleteTaskUseCase) Execute(ctx context.Context, actor domain.OwnerID, rawID string) (*domain.Task, error) {
	task, err := loadOwned(ctx, uc.finder, actor, rawID)
	if err != nil {
		return nil, err
	}

	if err := task.Complete(); err != nil {
		return nil, err
	}

	if err := uc.saver.Save(ctx, task); err != nil {
		return nil, fmt.Errorf("сохранение завершённой задачи: %w", err)
	}

	return task, nil
}

// --8<-- [end:complete_task_usecase]

// --8<-- [start:reschedule_task_usecase]
// RescheduleTaskUseCase сценарий переноса дедлайна задачи.
type RescheduleTaskUseCase struct {
	finder TaskFinder
	saver  TaskSaver
	clock  Clock
}

func NewRescheduleTaskUseCase(finder TaskFinder, saver TaskSaver, clock Clock) *RescheduleTaskUseCase {
	return &RescheduleTaskUseCase{
		finder: finder,
		saver:  saver,
		clock:  clock,
	}
}

func (uc *RescheduleTaskUseCase) Execute(ctx context.Context, actor domain.OwnerID, rawID string, newDeadlineTime time.Time) (*domain.Task, error) {
	task, err := loadOwned(ctx, uc.finder, actor, rawID)
	if err != nil {
		return nil, err
	}

	now := uc.clock.Now()
	newDeadline, err := domain.NewDeadline(newDeadlineTime, now)
	if err != nil {
		return nil, fmt.Errorf("некорректный новый дедлайн: %w", err)
	}

	if err := task.Reschedule(newDeadline, now); err != nil {
		return nil, err
	}

	if err := uc.saver.Save(ctx, task); err != nil {
		return nil, fmt.Errorf("сохранение обновлённой задачи: %w", err)
	}

	return task, nil
}

// --8<-- [end:reschedule_task_usecase]
