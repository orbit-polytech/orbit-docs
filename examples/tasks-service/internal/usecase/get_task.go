package usecase

import (
	"context"
	"fmt"

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
