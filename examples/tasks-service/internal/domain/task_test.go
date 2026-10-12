package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/domain"
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

func TestValueObjects(t *testing.T) {
	now := time.Now().UTC()

	t.Run("Title validation", func(t *testing.T) {
		_, err := domain.NewTitle("ab")
		require.ErrorIs(t, err, domain.ErrEmptyTitle)

		_, err = domain.NewTitle("   ")
		require.ErrorIs(t, err, domain.ErrEmptyTitle)

		_, err = domain.NewTitle(strings.Repeat("a", 201))
		require.ErrorIs(t, err, domain.ErrTitleTooLong)

		title := mustTitle(t, "Корректный заголовок")
		assert.Equal(t, "Корректный заголовок", title.String())
		assert.False(t, title.IsZero())
	})

	t.Run("Deadline validation and IsZero", func(t *testing.T) {
		_, err := domain.NewDeadline(now.Add(-1*time.Hour), now)
		require.ErrorIs(t, err, domain.ErrInvalidDeadline)

		dl := mustDeadline(t, now.Add(2*time.Hour), now)
		assert.False(t, dl.IsZero())

		zeroDL := domain.Deadline{}
		assert.True(t, zeroDL.IsZero())
	})

	t.Run("TaskPriority validation", func(t *testing.T) {
		assert.False(t, domain.TaskPriorityUnspecified.IsValid())
		assert.True(t, domain.TaskPriorityLow.IsValid())
		assert.True(t, domain.TaskPriorityMedium.IsValid())
		assert.True(t, domain.TaskPriorityHigh.IsValid())

		_, err := domain.ParseTaskPriority(99)
		require.Error(t, err)
	})

	t.Run("TaskID and OwnerID IsZero", func(t *testing.T) {
		_, err := domain.NewTaskID("   ")
		require.ErrorIs(t, err, domain.ErrEmptyTaskID)

		_, err = domain.NewOwnerID("   ")
		require.ErrorIs(t, err, domain.ErrEmptyOwnerID)

		tid := mustTaskID(t, "task-1")
		assert.False(t, tid.IsZero())

		oid := mustOwnerID(t, "student-1")
		assert.False(t, oid.IsZero())
	})
}

// --8<-- [start:domain_tests]
func TestTask_OwnershipAndLifecycle(t *testing.T) {
	now := time.Now().UTC()
	taskID := mustTaskID(t, "task-uuid-1")
	ownerID := mustOwnerID(t, "student-42")
	foreignOwner := mustOwnerID(t, "student-99")
	title := mustTitle(t, "Лабораторная работа 1")
	deadline := mustDeadline(t, now.Add(48*time.Hour), now)

	t.Run("успешное создание с начальной версией 1", func(t *testing.T) {
		task, err := domain.NewTask(taskID, ownerID, title, "Описание", deadline, domain.TaskPriorityHigh, now)
		require.NoError(t, err)
		require.NotNil(t, task)

		assert.Equal(t, int64(1), task.Version())
		assert.Equal(t, domain.TaskStatusPending, task.Status())
		assert.True(t, task.IsOwnedBy(ownerID))
		assert.False(t, task.IsOwnedBy(foreignOwner))
	})

	t.Run("ошибка при слишком длинном описании (>2000 символов)", func(t *testing.T) {
		tooLongDesc := strings.Repeat("x", 2001)
		_, err := domain.NewTask(taskID, ownerID, title, tooLongDesc, deadline, domain.TaskPriorityMedium, now)
		require.ErrorIs(t, err, domain.ErrDescriptionTooLong)
	})

	t.Run("завершение задачи и инкремент версии", func(t *testing.T) {
		task, err := domain.NewTask(taskID, ownerID, title, "Описание", deadline, domain.TaskPriorityMedium, now)
		require.NoError(t, err)

		err = task.Complete()
		require.NoError(t, err)
		assert.Equal(t, domain.TaskStatusCompleted, task.Status())
		assert.Equal(t, int64(2), task.Version())

		// Повторное завершение возвращает ErrTaskAlreadyCompleted
		err = task.Complete()
		require.ErrorIs(t, err, domain.ErrTaskAlreadyCompleted)
	})

	t.Run("перенос дедлайна и инкремент версии", func(t *testing.T) {
		task, err := domain.NewTask(taskID, ownerID, title, "Описание", deadline, domain.TaskPriorityMedium, now)
		require.NoError(t, err)

		newDeadline := mustDeadline(t, now.Add(72*time.Hour), now)
		err = task.Reschedule(newDeadline, now)
		require.NoError(t, err)
		assert.Equal(t, newDeadline.Time(), task.Deadline().Time())
		assert.Equal(t, int64(2), task.Version())

		// Перенос дедлайна в прошлое возвращает ошибку
		pastDeadline := domain.ReconstituteDeadline(now.Add(-10 * time.Hour))
		err = task.Reschedule(pastDeadline, now)
		require.ErrorIs(t, err, domain.ErrInvalidDeadline)
	})

	t.Run("нельзя перенести дедлайн завершённой задачи", func(t *testing.T) {
		task, err := domain.NewTask(taskID, ownerID, title, "Описание", deadline, domain.TaskPriorityMedium, now)
		require.NoError(t, err)

		require.NoError(t, task.Complete())

		newDeadline := mustDeadline(t, now.Add(72*time.Hour), now)
		err = task.Reschedule(newDeadline, now)
		require.ErrorIs(t, err, domain.ErrTaskAlreadyCompleted)
	})
}

// --8<-- [end:domain_tests]
