package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	if idempStore == nil {
		idempStore = NoopIdempotencyStore{}
	}
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

	// 1. Атомарная резервация ключа идемпотентности (Multi-Tenant Scoping по Google AIP-155)
	scopedKey := ""
	requestHash := ""
	if input.IdempotencyKey != "" {
		scopedKey = actor.String() + ":" + input.IdempotencyKey
		requestHash = computeRequestHash(input)

		existingID, completed, err := uc.idempStore.Reserve(ctx, scopedKey, requestHash)
		if err != nil {
			return nil, fmt.Errorf("ошибка проверки ключа идемпотентности: %w", err)
		}
		if completed {
			existingTask, err := uc.finder.FindByID(ctx, existingID)
			if err != nil {
				return nil, fmt.Errorf("ошибка загрузки существующей задачи по ключу идемпотентности: %w", err)
			}
			return existingTask, nil
		}
	}

	releaseReservation := func() {
		if scopedKey != "" {
			_ = uc.idempStore.Release(context.Background(), scopedKey)
		}
	}

	now := uc.clock.Now()

	// 2. Создание Value Objects с инвариантами
	title, err := domain.NewTitle(input.Title)
	if err != nil {
		releaseReservation()
		return nil, fmt.Errorf("ошибка названия задачи: %w", err)
	}

	desc, err := domain.NewDescription(input.Description)
	if err != nil {
		releaseReservation()
		return nil, fmt.Errorf("ошибка описания задачи: %w", err)
	}

	deadline, err := domain.NewDeadline(input.Deadline, now)
	if err != nil {
		releaseReservation()
		return nil, fmt.Errorf("ошибка дедлайна: %w", err)
	}

	taskID, err := uc.taskIDGen.NewTaskID()
	if err != nil {
		releaseReservation()
		return nil, fmt.Errorf("ошибка генерации идентификатора задачи: %w", err)
	}

	// 3. Создание доменной сущности (агрегат накапливает доменные события)
	task, err := domain.NewTask(taskID, actor, title, desc, deadline, input.Priority, now)
	if err != nil {
		releaseReservation()
		return nil, fmt.Errorf("ошибка создания сущности задачи: %w", err)
	}

	// 4. Сохранение в хранилище через порт TaskSaver
	if err := uc.saver.Save(ctx, task); err != nil {
		releaseReservation()
		return nil, fmt.Errorf("не удалось сохранить задачу в хранилище: %w", err)
	}

	// 5. Фиксация успешного завершения по ключу идемпотентности
	if scopedKey != "" {
		if err := uc.idempStore.Complete(ctx, scopedKey, task.ID()); err != nil {
			return nil, fmt.Errorf("не удалось зафиксировать ключ идемпотентности: %w", err)
		}
	}

	// 6. Публикация событий агрегата (DDD)
	for _, event := range task.PullEvents() {
		eventID, err := uc.eventIDGen.NewEventID()
		if err != nil {
			return nil, fmt.Errorf("генерация идентификатора события: %w", err)
		}
		event.EventID = eventID

		if err := uc.events.PublishTaskCreated(ctx, event); err != nil {
			return nil, fmt.Errorf("не удалось опубликовать событие создания задачи: %w", err)
		}
	}

	return task, nil
}

func computeRequestHash(input CreateTaskInput) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s|%s|%s|%d", input.Title, input.Description, input.Deadline.UTC().Format(time.RFC3339Nano), input.Priority)
	return hex.EncodeToString(h.Sum(nil))
}

// --8<-- [end:create_task_usecase]
