package usecase

import (
	"context"
	"fmt"

	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/domain"
)

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
		return nil, 0, fmt.Errorf("%w: лимит %d должен быть в диапазоне [1, 100]", ErrInvalidPagination, limit)
	}
	if offset < 0 {
		return nil, 0, fmt.Errorf("%w: смещение %d не может быть отрицательным", ErrInvalidPagination, offset)
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
