package inmemory_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/domain"
	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/infrastructure/inmemory"
	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestInMemoryTaskRepository(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()

	ownerID := mustOwnerID(t, "student-test")
	title := mustTitle(t, "Задание 1")
	desc := mustDescription(t, "Описание")
	dl := mustDeadline(t, now.Add(5*time.Hour), now)
	tID := mustTaskID(t, "t-1")

	t.Run("Save and FindByID", func(t *testing.T) {
		repo := inmemory.NewTaskRepository()
		task, err := domain.NewTask(tID, ownerID, title, desc, dl, domain.TaskPriorityMedium, now)
		require.NoError(t, err)

		err = repo.Save(ctx, task)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, tID)
		require.NoError(t, err)
		assert.Equal(t, task.ID().String(), found.ID().String())
		assert.Equal(t, task.Title().String(), found.Title().String())
		assert.Equal(t, int64(1), found.Version())
	})

	t.Run("Optimistic locking version conflict with stale entity", func(t *testing.T) {
		repo := inmemory.NewTaskRepository()
		task, err := domain.NewTask(tID, ownerID, title, desc, dl, domain.TaskPriorityMedium, now)
		require.NoError(t, err)
		require.NoError(t, repo.Save(ctx, task))

		staleTask := domain.ReconstituteTask(
			task.ID(), task.OwnerID(), task.Title(), task.Description(),
			task.Deadline(), task.Priority(), task.Status(), 0, task.CreatedAt(),
		)
		err = repo.Save(ctx, staleTask)
		assert.ErrorIs(t, err, domain.ErrVersionConflict)
	})

	t.Run("Reschedule and Complete in one transaction succeeds", func(t *testing.T) {
		repo := inmemory.NewTaskRepository()
		id := mustTaskID(t, "t-multi")
		tMulti, err := domain.NewTask(id, ownerID, title, desc, dl, domain.TaskPriorityMedium, now)
		require.NoError(t, err)
		require.NoError(t, repo.Save(ctx, tMulti))

		fresh, err := repo.FindByID(ctx, id)
		require.NoError(t, err)

		newDL := mustDeadline(t, now.Add(10*time.Hour), now)
		require.NoError(t, fresh.Reschedule(newDL, now))
		require.NoError(t, fresh.Complete())

		require.NoError(t, repo.Save(ctx, fresh), "множественные мутации перед сохранением не должны вызывать конфликт версий")
		assert.Equal(t, int64(2), fresh.Version())
	})

	t.Run("Clone isolation", func(t *testing.T) {
		repo := inmemory.NewTaskRepository()
		task, err := domain.NewTask(tID, ownerID, title, desc, dl, domain.TaskPriorityMedium, now)
		require.NoError(t, err)
		require.NoError(t, repo.Save(ctx, task))

		found, err := repo.FindByID(ctx, tID)
		require.NoError(t, err)

		// Модифицируем возвращенную сущность
		_ = found.Complete()

		// В репозитории состояние не изменилось (изоляция указателя)
		fresh, err := repo.FindByID(ctx, tID)
		require.NoError(t, err)
		assert.Equal(t, domain.TaskStatusPending, fresh.Status())
		assert.Equal(t, int64(1), fresh.Version())
	})

	t.Run("List with pagination", func(t *testing.T) {
		repo := inmemory.NewTaskRepository()
		for i := 1; i <= 3; i++ {
			id := mustTaskID(t, fmt.Sprintf("t-%d", i))
			ttl := mustTitle(t, fmt.Sprintf("Задание %d", i))
			tNew, err := domain.NewTask(id, ownerID, ttl, mustDescription(t, ""), dl, domain.TaskPriorityLow, now.Add(time.Duration(i)*time.Minute))
			require.NoError(t, err)
			require.NoError(t, repo.Save(ctx, tNew))
		}

		filter := usecase.ListFilter{
			OwnerID: ownerID,
			Limit:   2,
			Offset:  0,
		}
		items, total, err := repo.List(ctx, filter)
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, items, 2)
		assert.Equal(t, "t-1", items[0].ID().String())
		assert.Equal(t, "t-2", items[1].ID().String())
	})

	t.Run("Race condition: две параллельные горутины завершают одну задачу", func(t *testing.T) {
		repo := inmemory.NewTaskRepository()
		raceTaskID := mustTaskID(t, "race-task-1")
		raceTask, err := domain.NewTask(raceTaskID, ownerID, title, desc, dl, domain.TaskPriorityHigh, now)
		require.NoError(t, err)
		require.NoError(t, repo.Save(ctx, raceTask))

		var successCount int64
		var conflictCount int64
		var wg sync.WaitGroup

		startSignal := make(chan struct{})

		// Обе горутины получают исходный снимок задачи одной и той же версии (версия 1)
		tCopy1, err := repo.FindByID(ctx, raceTaskID)
		require.NoError(t, err)
		tCopy2, err := repo.FindByID(ctx, raceTaskID)
		require.NoError(t, err)

		require.NoError(t, tCopy1.Complete())
		require.NoError(t, tCopy2.Complete())

		wg.Add(2)
		go func() {
			defer wg.Done()
			<-startSignal
			saveErr := repo.Save(ctx, tCopy1)
			if saveErr == nil {
				atomic.AddInt64(&successCount, 1)
			} else if errors.Is(saveErr, domain.ErrVersionConflict) {
				atomic.AddInt64(&conflictCount, 1)
			}
		}()
		go func() {
			defer wg.Done()
			<-startSignal
			saveErr := repo.Save(ctx, tCopy2)
			if saveErr == nil {
				atomic.AddInt64(&successCount, 1)
			} else if errors.Is(saveErr, domain.ErrVersionConflict) {
				atomic.AddInt64(&conflictCount, 1)
			}
		}()

		close(startSignal)
		wg.Wait()

		// Ровно один вызов успешен, второй получает ErrVersionConflict
		assert.Equal(t, int64(1), successCount, "ровно одна горутина должна успешно завершить задачу")
		assert.Equal(t, int64(1), conflictCount, "вторая горутина должна получить ErrVersionConflict")
	})

	t.Run("Concurrent access safety with result verification", func(t *testing.T) {
		repo := inmemory.NewTaskRepository()
		var wg sync.WaitGroup
		const concTasksCount = 50

		for i := 0; i < concTasksCount; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				id := mustTaskID(t, fmt.Sprintf("conc-%d", idx))
				ttl := mustTitle(t, fmt.Sprintf("Конкурентная задача %d", idx))
				tConcurrent, err := domain.NewTask(id, ownerID, ttl, mustDescription(t, ""), dl, domain.TaskPriorityLow, now)
				if err != nil {
					t.Errorf("ошибка создания задачи: %v", err)
					return
				}
				_ = repo.Save(ctx, tConcurrent)
			}(i)
		}
		wg.Wait()

		// Проверяем, что все 50 задач успешно сохранены и читаются
		for i := 0; i < concTasksCount; i++ {
			id := mustTaskID(t, fmt.Sprintf("conc-%d", i))
			found, err := repo.FindByID(ctx, id)
			require.NoError(t, err)
			assert.Equal(t, id.String(), found.ID().String())
		}
	})
}
