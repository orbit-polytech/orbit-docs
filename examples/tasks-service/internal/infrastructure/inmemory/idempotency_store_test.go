package inmemory_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/infrastructure/inmemory"
	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdempotencyStore(t *testing.T) {
	ctx := context.Background()
	taskID := mustTaskID(t, "t-100")

	t.Run("Reserve and Complete flow", func(t *testing.T) {
		store := inmemory.NewIdempotencyStore()

		// 1. Первый запрос: резервация свободна
		existingID, completed, err := store.Reserve(ctx, "key-1", "hash-abc")
		require.NoError(t, err)
		assert.False(t, completed)
		assert.True(t, existingID.IsZero())

		// 2. Повторный вызов пока запрос in-flight -> ErrIdempotencyInFlight
		_, _, err = store.Reserve(ctx, "key-1", "hash-abc")
		assert.ErrorIs(t, err, usecase.ErrIdempotencyInFlight)

		// 3. Завершение запроса
		require.NoError(t, store.Complete(ctx, "key-1", taskID))

		// 4. Повторный вызов после Complete -> возвращает сохранённый taskID
		existingID, completed, err = store.Reserve(ctx, "key-1", "hash-abc")
		require.NoError(t, err)
		assert.True(t, completed)
		assert.Equal(t, taskID.String(), existingID.String())
	})

	t.Run("Payload conflict (AIP-155)", func(t *testing.T) {
		store := inmemory.NewIdempotencyStore()

		_, _, err := store.Reserve(ctx, "key-conflict", "hash-1")
		require.NoError(t, err)
		require.NoError(t, store.Complete(ctx, "key-conflict", taskID))

		// Вызов с тем же ключом, но другим хэшем тела -> ErrIdempotencyConflict
		_, _, err = store.Reserve(ctx, "key-conflict", "hash-2")
		assert.ErrorIs(t, err, usecase.ErrIdempotencyConflict)
	})

	t.Run("Release unlocks key", func(t *testing.T) {
		store := inmemory.NewIdempotencyStore()

		_, _, err := store.Reserve(ctx, "key-release", "hash-1")
		require.NoError(t, err)

		// Сброс резервации при ошибке
		require.NoError(t, store.Release(ctx, "key-release"))

		// Ключ снова доступен для выполнения
		_, completed, err := store.Reserve(ctx, "key-release", "hash-1")
		require.NoError(t, err)
		assert.False(t, completed)
	})

	t.Run("LRU eviction preserves recent keys", func(t *testing.T) {
		const maxKeys = 3
		store := inmemory.NewIdempotencyStoreWithConfig(maxKeys, time.Hour)

		// Добавляем 3 ключа
		for i := 1; i <= 3; i++ {
			k := fmt.Sprintf("k-%d", i)
			_, _, err := store.Reserve(ctx, k, "hash")
			require.NoError(t, err)
			require.NoError(t, store.Complete(ctx, k, mustTaskID(t, fmt.Sprintf("t-%d", i))))
		}

		// Обращаемся к k-1, делая его свежим (MRU)
		_, completed, err := store.Reserve(ctx, "k-1", "hash")
		require.NoError(t, err)
		assert.True(t, completed)

		// Добавляем 4-й ключ -> должен быть вытеснен наименее используемый k-2!
		_, _, err = store.Reserve(ctx, "k-4", "hash")
		require.NoError(t, err)
		require.NoError(t, store.Complete(ctx, "k-4", mustTaskID(t, "t-4")))

		// k-2 вытеснен (теперь он считается новым)
		_, completedK2, err := store.Reserve(ctx, "k-2", "hash")
		require.NoError(t, err)
		assert.False(t, completedK2, "k-2 должен был быть вытеснен по LRU")

		// k-1 остался на месте
		_, completedK1, err := store.Reserve(ctx, "k-1", "hash")
		require.NoError(t, err)
		assert.True(t, completedK1, "k-1 не должен был быть вытеснен")
	})

	t.Run("Concurrent atomic reservation: 5 workers", func(t *testing.T) {
		store := inmemory.NewIdempotencyStore()
		const workers = 5
		var reservedCount int64
		var inFlightCount int64
		var wg sync.WaitGroup

		startSignal := make(chan struct{})

		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-startSignal

				_, completed, err := store.Reserve(ctx, "conc-key", "conc-hash")
				if err == nil && !completed {
					atomic.AddInt64(&reservedCount, 1)
				} else if errors.Is(err, usecase.ErrIdempotencyInFlight) {
					atomic.AddInt64(&inFlightCount, 1)
				}
			}()
		}

		close(startSignal)
		wg.Wait()

		assert.Equal(t, int64(1), reservedCount, "ровно одна горутина должна успешно зарезервировать ключ")
		assert.Equal(t, int64(workers-1), inFlightCount, "остальные горутины должны получить ErrIdempotencyInFlight")
	})
}
