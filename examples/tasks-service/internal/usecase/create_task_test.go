package usecase_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/domain"
	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func computeTestHash(input usecase.CreateTaskInput) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s|%s|%s|%d", input.Title, input.Description, input.Deadline.UTC().Format(time.RFC3339Nano), input.Priority)
	return hex.EncodeToString(h.Sum(nil))
}

func mustOwnerID(t *testing.T, raw string) domain.OwnerID {
	t.Helper()
	id, err := domain.NewOwnerID(raw)
	require.NoError(t, err)
	return id
}

func mustTaskID(t *testing.T, raw string) domain.TaskID {
	t.Helper()
	id, err := domain.NewTaskID(raw)
	require.NoError(t, err)
	return id
}

func mustTitle(t *testing.T, raw string) domain.Title {
	t.Helper()
	title, err := domain.NewTitle(raw)
	require.NoError(t, err)
	return title
}

func mustDescription(t *testing.T, raw string) domain.Description {
	t.Helper()
	desc, err := domain.NewDescription(raw)
	require.NoError(t, err)
	return desc
}

func mustDeadline(t *testing.T, target, now time.Time) domain.Deadline {
	t.Helper()
	dl, err := domain.NewDeadline(target, now)
	require.NoError(t, err)
	return dl
}

// --8<-- [start:usecase_mocks]
type mockClock struct {
	now time.Time
}

func (m *mockClock) Now() time.Time { return m.now }

type mockTaskIDGen struct {
	nextID string
	err    error
}

func (m *mockTaskIDGen) NewTaskID() (domain.TaskID, error) {
	if m.err != nil {
		return domain.TaskID{}, m.err
	}
	return domain.NewTaskID(m.nextID)
}

type mockEventIDGen struct {
	nextEvent string
	err       error
}

func (m *mockEventIDGen) NewEventID() (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.nextEvent, nil
}

type mockIdempStore struct {
	mu        sync.Mutex
	hashes    map[string]string
	completed map[string]domain.TaskID
	inFlight  map[string]bool
	err       error
}

func newMockIdempStore() *mockIdempStore {
	return &mockIdempStore{
		hashes:    make(map[string]string),
		completed: make(map[string]domain.TaskID),
		inFlight:  make(map[string]bool),
	}
}

func (m *mockIdempStore) Reserve(ctx context.Context, key string, requestHash string) (domain.TaskID, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return domain.TaskID{}, false, m.err
	}
	if prevHash, exists := m.hashes[key]; exists {
		if prevHash != requestHash {
			return domain.TaskID{}, false, usecase.ErrIdempotencyConflict
		}
		if m.inFlight[key] {
			return domain.TaskID{}, false, usecase.ErrIdempotencyInFlight
		}
		return m.completed[key], true, nil
	}
	m.hashes[key] = requestHash
	m.inFlight[key] = true
	return domain.TaskID{}, false, nil
}

func (m *mockIdempStore) Complete(ctx context.Context, key string, taskID domain.TaskID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.inFlight, key)
	m.completed[key] = taskID
	return nil
}

func (m *mockIdempStore) Release(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.inFlight, key)
	delete(m.hashes, key)
	return nil
}

type mockTaskRepo struct {
	tasks   map[string]*domain.Task
	saveErr error
	findErr error
	listErr error
}

func newMockTaskRepo() *mockTaskRepo {
	return &mockTaskRepo{tasks: make(map[string]*domain.Task)}
}

func (m *mockTaskRepo) Save(ctx context.Context, task *domain.Task) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.tasks[task.ID().String()] = task
	return nil
}

func (m *mockTaskRepo) FindByID(ctx context.Context, id domain.TaskID) (*domain.Task, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	task, exists := m.tasks[id.String()]
	if !exists {
		return nil, domain.ErrTaskNotFound
	}
	return task, nil
}

func (m *mockTaskRepo) List(ctx context.Context, filter usecase.ListFilter) ([]*domain.Task, int, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	var res []*domain.Task
	for _, t := range m.tasks {
		if t.OwnerID().String() == filter.OwnerID.String() {
			res = append(res, t)
		}
	}
	return res, len(res), nil
}

type mockEventPublisher struct {
	published []domain.TaskCreated
	err       error
}

func (m *mockEventPublisher) PublishTaskCreated(ctx context.Context, event domain.TaskCreated) error {
	if m.err != nil {
		return m.err
	}
	m.published = append(m.published, event)
	return nil
}

// --8<-- [end:usecase_mocks]

// --8<-- [start:usecase_tests]
func TestCreateTaskUseCase(t *testing.T) {
	fixedTime := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	clock := &mockClock{now: fixedTime}
	idGen := &mockTaskIDGen{nextID: "task-001"}
	evtGen := &mockEventIDGen{nextEvent: "evt-001"}
	ownerID := mustOwnerID(t, "student-123")

	t.Run("успешное создание задачи с метаданными события", func(t *testing.T) {
		repo := newMockTaskRepo()
		idempStore := newMockIdempStore()
		events := &mockEventPublisher{}
		uc := usecase.NewCreateTaskUseCase(repo, repo, idempStore, events, idGen, evtGen, clock)

		input := usecase.CreateTaskInput{
			Title:          "Сдать лабораторную работу",
			Description:    "Текст описания",
			Deadline:       fixedTime.Add(24 * time.Hour),
			Priority:       domain.TaskPriorityHigh,
			IdempotencyKey: "key-123",
		}

		task, err := uc.Execute(context.Background(), ownerID, input)
		require.NoError(t, err)
		require.NotNil(t, task)
		assert.Equal(t, "task-001", task.ID().String())
		assert.Equal(t, "student-123", task.OwnerID().String())
		assert.Equal(t, "Текст описания", task.Description().String())

		// Проверка опубликованного события
		require.Len(t, events.published, 1)
		evt := events.published[0]
		assert.Equal(t, "evt-001", evt.EventID)
		assert.Equal(t, "task-001", evt.TaskID.String())
		assert.Equal(t, fixedTime, evt.OccurredAt)
	})

	t.Run("дедупликация по ключу идемпотентности", func(t *testing.T) {
		repo := newMockTaskRepo()
		idempStore := newMockIdempStore()
		events := &mockEventPublisher{}
		uc := usecase.NewCreateTaskUseCase(repo, repo, idempStore, events, idGen, evtGen, clock)

		input := usecase.CreateTaskInput{
			Title:          "Повторный запрос",
			Description:    "Описание",
			Deadline:       fixedTime.Add(24 * time.Hour),
			Priority:       domain.TaskPriorityLow,
			IdempotencyKey: "idemp-unique-1",
		}

		// Первый вызов — создание задачи
		task1, err := uc.Execute(context.Background(), ownerID, input)
		require.NoError(t, err)
		require.Len(t, events.published, 1)

		// Второй вызов с тем же ключом и телом — возвращается та же задача без повторной публикации события
		task2, err := uc.Execute(context.Background(), ownerID, input)
		require.NoError(t, err)
		assert.Equal(t, task1.ID().String(), task2.ID().String())
		assert.Len(t, events.published, 1) // Новое событие НЕ публиковалось
	})

	t.Run("конфликт хэша запроса при повторном ключе (AIP-155)", func(t *testing.T) {
		repo := newMockTaskRepo()
		idempStore := newMockIdempStore()
		events := &mockEventPublisher{}
		uc := usecase.NewCreateTaskUseCase(repo, repo, idempStore, events, idGen, evtGen, clock)

		input1 := usecase.CreateTaskInput{
			Title:          "Первоначальная задача",
			Deadline:       fixedTime.Add(24 * time.Hour),
			Priority:       domain.TaskPriorityLow,
			IdempotencyKey: "shared-key",
		}
		_, err := uc.Execute(context.Background(), ownerID, input1)
		require.NoError(t, err)

		input2 := usecase.CreateTaskInput{
			Title:          "Другое название с тем же ключом",
			Deadline:       fixedTime.Add(24 * time.Hour),
			Priority:       domain.TaskPriorityHigh,
			IdempotencyKey: "shared-key",
		}
		_, err = uc.Execute(context.Background(), ownerID, input2)
		require.Error(t, err)
		assert.ErrorIs(t, err, usecase.ErrIdempotencyConflict)
	})

	t.Run("параллельные запросы с одним ключом идемпотентности", func(t *testing.T) {
		repo := newMockTaskRepo()
		idempStore := newMockIdempStore()
		events := &mockEventPublisher{}
		uc := usecase.NewCreateTaskUseCase(repo, repo, idempStore, events, idGen, evtGen, clock)

		input := usecase.CreateTaskInput{
			Title:          "Конкурентное создание",
			Deadline:       fixedTime.Add(24 * time.Hour),
			Priority:       domain.TaskPriorityMedium,
			IdempotencyKey: "conc-key-1",
		}

		const parallelCount = 5
		var successCount int64
		var inFlightCount int64
		var wg sync.WaitGroup
		startSignal := make(chan struct{})

		for i := 0; i < parallelCount; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-startSignal

				_, err := uc.Execute(context.Background(), ownerID, input)
				if err == nil {
					atomic.AddInt64(&successCount, 1)
				} else if errors.Is(err, usecase.ErrIdempotencyInFlight) {
					atomic.AddInt64(&inFlightCount, 1)
				}
			}()
		}

		close(startSignal)
		wg.Wait()

		// Все параллельные запросы либо завершаются дедупликацией (success), либо ловят in-flight.
		// Главная гарантия идемпотентности: в репозитории создана строго одна задача и ровно одно событие!
		assert.Equal(t, int64(parallelCount), successCount+inFlightCount)
		assert.Equal(t, 1, len(repo.tasks), "в репозитории должна быть создана ровно одна задача")
		assert.Equal(t, 1, len(events.published), "должно быть опубликовано ровно одно событие")
	})

	t.Run("активный in-flight запрос возвращает ErrIdempotencyInFlight", func(t *testing.T) {
		repo := newMockTaskRepo()
		idempStore := newMockIdempStore()
		events := &mockEventPublisher{}
		uc := usecase.NewCreateTaskUseCase(repo, repo, idempStore, events, idGen, evtGen, clock)

		input := usecase.CreateTaskInput{
			Title:          "Задача в процессе",
			Description:    "Описание",
			Deadline:       fixedTime.Add(24 * time.Hour),
			Priority:       domain.TaskPriorityMedium,
			IdempotencyKey: "inflight-test-key",
		}

		// Резервируем ключ вручную с тем же хэшем
		scopedKey := ownerID.String() + ":" + input.IdempotencyKey
		hash := computeTestHash(input)
		_, _, err := idempStore.Reserve(context.Background(), scopedKey, hash)
		require.NoError(t, err)

		// Повторный вызов через usecase пока ключ in-flight возвращает ErrIdempotencyInFlight
		_, err = uc.Execute(context.Background(), ownerID, input)
		require.Error(t, err)
		assert.ErrorIs(t, err, usecase.ErrIdempotencyInFlight)
	})

	t.Run("ошибка при отказе генератора ID", func(t *testing.T) {
		repo := newMockTaskRepo()
		idempStore := newMockIdempStore()
		events := &mockEventPublisher{}
		errGen := &mockTaskIDGen{err: errors.New("entropy error")}
		uc := usecase.NewCreateTaskUseCase(repo, repo, idempStore, events, errGen, evtGen, clock)

		input := usecase.CreateTaskInput{
			Title:    "Задача",
			Deadline: fixedTime.Add(24 * time.Hour),
			Priority: domain.TaskPriorityMedium,
		}

		_, err := uc.Execute(context.Background(), ownerID, input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ошибка генерации идентификатора задачи")
	})

	t.Run("ошибка при отказе генератора EventID", func(t *testing.T) {
		repo := newMockTaskRepo()
		idempStore := newMockIdempStore()
		events := &mockEventPublisher{}
		errEvtGen := &mockEventIDGen{err: errors.New("event uuid error")}
		uc := usecase.NewCreateTaskUseCase(repo, repo, idempStore, events, idGen, errEvtGen, clock)

		input := usecase.CreateTaskInput{
			Title:    "Задача",
			Deadline: fixedTime.Add(24 * time.Hour),
			Priority: domain.TaskPriorityMedium,
		}

		_, err := uc.Execute(context.Background(), ownerID, input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "генерация идентификатора события")
	})

	t.Run("ошибка при отказе хранилища (Save error)", func(t *testing.T) {
		repo := newMockTaskRepo()
		repo.saveErr = errors.New("db error")
		idempStore := newMockIdempStore()
		events := &mockEventPublisher{}
		uc := usecase.NewCreateTaskUseCase(repo, repo, idempStore, events, idGen, evtGen, clock)

		input := usecase.CreateTaskInput{
			Title:          "Задача",
			Deadline:       fixedTime.Add(24 * time.Hour),
			Priority:       domain.TaskPriorityMedium,
			IdempotencyKey: "err-key",
		}

		_, err := uc.Execute(context.Background(), ownerID, input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "не удалось сохранить задачу в хранилище")
		assert.Empty(t, events.published)

		// Ключ был освобождён через Release, поэтому повторный вызов возможен
		assert.False(t, idempStore.inFlight["student-123:err-key"])
	})
}

func TestTaskSecurityAndOwnershipAccess(t *testing.T) {
	fixedTime := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	clock := &mockClock{now: fixedTime}
	repo := newMockTaskRepo()

	ownerID := mustOwnerID(t, "legit-student")
	hackerID := mustOwnerID(t, "hacker-student")
	tID := mustTaskID(t, "task-secure")
	ttl := mustTitle(t, "Приватная задача")
	desc := mustDescription(t, "")
	dl := mustDeadline(t, fixedTime.Add(10*time.Hour), fixedTime)

	initialTask, err := domain.NewTask(tID, ownerID, ttl, desc, dl, domain.TaskPriorityHigh, fixedTime)
	require.NoError(t, err)
	_ = repo.Save(context.Background(), initialTask)

	t.Run("GetTask: чужой студент получает ErrTaskNotFound", func(t *testing.T) {
		getUC := usecase.NewGetTaskUseCase(repo)
		task, err := getUC.Execute(context.Background(), hackerID, "task-secure")
		assert.Nil(t, task)
		assert.ErrorIs(t, err, domain.ErrTaskNotFound)

		// Легитимный владелец получает задачу
		task, err = getUC.Execute(context.Background(), ownerID, "task-secure")
		require.NoError(t, err)
		assert.Equal(t, "task-secure", task.ID().String())
	})

	t.Run("CompleteTask: чужой студент получает ErrTaskNotFound", func(t *testing.T) {
		completeUC := usecase.NewCompleteTaskUseCase(repo, repo)
		_, err := completeUC.Execute(context.Background(), hackerID, "task-secure")
		assert.ErrorIs(t, err, domain.ErrTaskNotFound)
	})

	t.Run("RescheduleTask: чужой студент получает ErrTaskNotFound", func(t *testing.T) {
		reschedUC := usecase.NewRescheduleTaskUseCase(repo, repo, clock)
		_, err := reschedUC.Execute(context.Background(), hackerID, "task-secure", fixedTime.Add(20*time.Hour))
		assert.ErrorIs(t, err, domain.ErrTaskNotFound)
	})

	t.Run("ListTasks: пагинация валидирует границы", func(t *testing.T) {
		listUC := usecase.NewListTasksUseCase(repo)
		_, _, err := listUC.Execute(context.Background(), ownerID, 0, 0)
		assert.ErrorIs(t, err, usecase.ErrInvalidPagination)

		_, _, err = listUC.Execute(context.Background(), ownerID, 101, 0)
		assert.ErrorIs(t, err, usecase.ErrInvalidPagination)

		_, _, err = listUC.Execute(context.Background(), ownerID, 10, -1)
		assert.ErrorIs(t, err, usecase.ErrInvalidPagination)

		items, total, err := listUC.Execute(context.Background(), ownerID, 10, 0)
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, items, 1)
	})
}

// --8<-- [end:usecase_tests]
