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

func mustDeadline(t *testing.T, target, now time.Time) domain.Deadline {
	t.Helper()
	dl, err := domain.NewDeadline(target, now)
	require.NoError(t, err)
	return dl
}

func TestInMemoryTaskRepository(t *testing.T) {
	ctx := context.Background()
	repo := inmemory.NewTaskRepository()
	now := time.Now().UTC()

	ownerID := mustOwnerID(t, "student-test")
	title := mustTitle(t, "Задание 1")
	dl := mustDeadline(t, now.Add(5*time.Hour), now)
	tID := mustTaskID(t, "t-1")

	task, err := domain.NewTask(tID, ownerID, title, "Описание", dl, domain.TaskPriorityMedium, now)
	require.NoError(t, err)

	t.Run("Save and FindByID", func(t *testing.T) {
		err := repo.Save(ctx, task)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, tID)
		require.NoError(t, err)
		assert.Equal(t, task.ID().String(), found.ID().String())
		assert.Equal(t, task.Title().String(), found.Title().String())
		assert.Equal(t, int64(1), found.Version())
	})

	t.Run("Optimistic locking version conflict", func(t *testing.T) {
		// Попытка сохранить ту же версию снова возвращает конфликт
		err := repo.Save(ctx, task)
		assert.ErrorIs(t, err, domain.ErrVersionConflict)
	})

	t.Run("Clone isolation", func(t *testing.T) {
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
		for i := 2; i <= 3; i++ {
			id := mustTaskID(t, fmt.Sprintf("t-%d", i))
			ttl := mustTitle(t, fmt.Sprintf("Задание %d", i))
			tNew, err := domain.NewTask(id, ownerID, ttl, "", dl, domain.TaskPriorityLow, now.Add(time.Duration(i)*time.Minute))
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
		raceTaskID := mustTaskID(t, "race-task-1")
		raceTask, err := domain.NewTask(raceTaskID, ownerID, title, "", dl, domain.TaskPriorityHigh, now)
		require.NoError(t, err)
		require.NoError(t, repo.Save(ctx, raceTask))

		var successCount int64
		var conflictCount int64
		var wg sync.WaitGroup

		startSignal := make(chan struct{})

		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-startSignal

				// Обе горутины берут текущую версию задачи из хранилища (версия 1)
				tCopy, err := repo.FindByID(ctx, raceTaskID)
				if err != nil {
					return
				}
				_ = tCopy.Complete() // Версия становится 2

				saveErr := repo.Save(ctx, tCopy)
				if saveErr == nil {
					atomic.AddInt64(&successCount, 1)
				} else if errors.Is(saveErr, domain.ErrVersionConflict) {
					atomic.AddInt64(&conflictCount, 1)
				}
			}()
		}

		close(startSignal)
		wg.Wait()

		// Ровно один вызов успешен, второй получает ErrVersionConflict
		assert.Equal(t, int64(1), successCount, "ровно одна горутина должна успешно завершить задачу")
		assert.Equal(t, int64(1), conflictCount, "вторая горутина должна получить ErrVersionConflict")
	})

	t.Run("Concurrent access safety with result verification", func(t *testing.T) {
		var wg sync.WaitGroup
		const concTasksCount = 50

		for i := 0; i < concTasksCount; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				id := mustTaskID(t, fmt.Sprintf("conc-%d", idx))
				ttl := mustTitle(t, fmt.Sprintf("Конкурентная задача %d", idx))
				tConcurrent, err := domain.NewTask(id, ownerID, ttl, "", dl, domain.TaskPriorityLow, now)
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
			id := mustTaskID(t, fmt.Sprintf("conc-%d", idx(i)))
			found, err := repo.FindByID(ctx, id)
			require.NoError(t, err)
			assert.Equal(t, id.String(), found.ID().String())
		}
	})
}

func idx(i int) int { return i }
