package inmemory

import (
	"context"
	"sync"

	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/domain"
)

// --8<-- [start:idempotency_store]
// ВНИМАНИЕ: In-memory реализация является учебным упрощением.
// В production-среде ключи идемпотентности хранятся в распределённом кэше (Redis)
// с обязательным временем жизни (TTL 24–72 часа, команда SET key value NX EX 86400).
const defaultMaxIdempotencyKeys = 10000

// IdempotencyStore реализует потокобезопасное in-memory хранилище ключей идемпотентности с контролем ёмкости.
type IdempotencyStore struct {
	mu      sync.RWMutex
	maxKeys int
	keys    map[string]domain.TaskID
}

func NewIdempotencyStore() *IdempotencyStore {
	return &IdempotencyStore{
		maxKeys: defaultMaxIdempotencyKeys,
		keys:    make(map[string]domain.TaskID),
	}
}

func (s *IdempotencyStore) Get(ctx context.Context, key string) (domain.TaskID, bool, error) {
	if err := ctx.Err(); err != nil {
		return domain.TaskID{}, false, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	id, exists := s.keys[key]
	return id, exists, nil
}

func (s *IdempotencyStore) Save(ctx context.Context, key string, taskID domain.TaskID) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Защита от утечки памяти (Unbounded Memory Growth): очистка при переполнении буфера
	if len(s.keys) >= s.maxKeys {
		s.keys = make(map[string]domain.TaskID, s.maxKeys/2)
	}

	s.keys[key] = taskID
	return nil
}

// --8<-- [end:idempotency_store]
