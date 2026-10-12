package usecase

import (
	"context"
	"fmt"

	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/domain"
)

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
