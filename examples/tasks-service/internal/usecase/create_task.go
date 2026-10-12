package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/domain"
)

// --8<-- [start:create_task_command]
// CreateTaskInput — входные данные команды создания задачи.
type CreateTaskInput struct {
	Title          string
	Description    string
	Deadline       time.Time
	Priority       domain.TaskPriority
	IdempotencyKey string
}

// --8<-- [end:create_task_command]

// --8<-- [start:create_task_usecase]
// CreateTaskUseCase координирует создание задачи и дедупликацию по ключу идемпотентности.
type CreateTaskUseCase struct {
	saver      TaskSaver
	finder     TaskFinder
	idempStore IdempotencyStore
	events     EventPublisher
	taskIDGen  TaskIDGenerator
	eventIDGen EventIDGenerator
	clock      Clock
}

func NewCreateTaskUseCase(
	saver TaskSaver,
	finder TaskFinder,
	idempStore IdempotencyStore,
	events EventPublisher,
	taskIDGen TaskIDGenerator,
	eventIDGen EventIDGenerator,
	clock Clock,
) *CreateTaskUseCase {
	return &CreateTaskUseCase{
		saver:      saver,
		finder:     finder,
		idempStore: idempStore,
		events:     events,
		taskIDGen:  taskIDGen,
		eventIDGen: eventIDGen,
		clock:      clock,
	}
}

// Execute создаёт задачу для авторизованного владельца (actor) с защитой от дубликатов.
func (uc *CreateTaskUseCase) Execute(ctx context.Context, actor domain.OwnerID, input CreateTaskInput) (*domain.Task, error) {
	if actor.IsZero() {
		return nil, domain.ErrEmptyOwnerID
	}

	// 1. Проверка идемпотентности с изоляцией по владельцу (Multi-Tenant Scoping по Google AIP-155)
	scopedKey := ""
	if input.IdempotencyKey != "" {
		scopedKey = actor.String() + ":" + input.IdempotencyKey
	}

	if scopedKey != "" && uc.idempStore != nil {
		existingID, found, err := uc.idempStore.Get(ctx, scopedKey)
		if err != nil {
			return nil, fmt.Errorf("ошибка проверки ключа идемпотентности: %w", err)
		}
		if found {
			existingTask, err := uc.finder.FindByID(ctx, existingID)
			if err != nil {
				return nil, fmt.Errorf("ошибка загрузки существующей задачи по ключу идемпотентности: %w", err)
			}
			return existingTask, nil
		}
	}

	now := uc.clock.Now()

	// 2. Создание Value Objects
	title, err := domain.NewTitle(input.Title)
	if err != nil {
		return nil, fmt.Errorf("ошибка названия задачи: %w", err)
	}

	deadline, err := domain.NewDeadline(input.Deadline, now)
	if err != nil {
		return nil, fmt.Errorf("ошибка дедлайна: %w", err)
	}

	taskID, err := uc.taskIDGen.NewTaskID()
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации идентификатора задачи: %w", err)
	}

	// 3. Создание доменной сущности с инвариантами
	task, err := domain.NewTask(taskID, actor, title, input.Description, deadline, input.Priority, now)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания сущности задачи: %w", err)
	}

	// 4. Сохранение в хранилище через порт TaskSaver
	if err := uc.saver.Save(ctx, task); err != nil {
		return nil, fmt.Errorf("не удалось сохранить задачу в хранилище: %w", err)
	}

	// 5. Фиксация ключа идемпотентности
	if scopedKey != "" && uc.idempStore != nil {
		if err := uc.idempStore.Save(ctx, scopedKey, task.ID()); err != nil {
			return nil, fmt.Errorf("не удалось зафиксировать ключ идемпотентности: %w", err)
		}
	}

	// ВНИМАНИЕ: КРИТИЧЕСКИЙ АРХИТЕКТУРНЫЙ РИСК — ДВОЙНАЯ ЗАПИСЬ (DUAL-WRITE)!
	// Вызов repo.Save() с последующим events.Publish() не атомарен.
	// Если после сохранения в БД упадёт сеть или брокер — событие потеряется, а запись останется.
	// В главе 8 мы устраняем этот риск через паттерн Transactional Outbox.
	event := domain.TaskCreated{
		EventID:    uc.eventIDGen.NewEventID(),
		TaskID:     task.ID(),
		OwnerID:    task.OwnerID(),
		Title:      task.Title(),
		Deadline:   task.Deadline(),
		Priority:   task.Priority(),
		OccurredAt: now,
	}

	if err := uc.events.PublishTaskCreated(ctx, event); err != nil {
		return nil, fmt.Errorf("не удалось опубликовать событие создания задачи: %w", err)
	}

	return task, nil
}

// --8<-- [end:create_task_usecase]
