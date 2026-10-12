package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/domain"
)

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
