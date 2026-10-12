package inmemory

import (
	"container/list"
	"context"
	"sync"
	"time"

	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/domain"
	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/usecase"
)

// --8<-- [start:idempotency_store]
const (
	defaultMaxIdempotencyKeys = 10000
	defaultIdempotencyTTL     = 24 * time.Hour
)

type idempotencyEntry struct {
	key         string
	requestHash string
	taskID      domain.TaskID
	inFlight    bool
	createdAt   time.Time
	element     *list.Element
}

// IdempotencyStore реализует потокобезопасное in-memory хранилище ключей идемпотентности
// с поддержкой атомарной резервации (AIP-155), TTL и вытеснением по алгоритму LRU.
type IdempotencyStore struct {
	mu      sync.Mutex
	maxKeys int
	ttl     time.Duration
	entries map[string]*idempotencyEntry
	lru     *list.List
}

func NewIdempotencyStore() *IdempotencyStore {
	return NewIdempotencyStoreWithConfig(defaultMaxIdempotencyKeys, defaultIdempotencyTTL)
}

func NewIdempotencyStoreWithConfig(maxKeys int, ttl time.Duration) *IdempotencyStore {
	if maxKeys <= 0 {
		maxKeys = defaultMaxIdempotencyKeys
	}
	return &IdempotencyStore{
		maxKeys: maxKeys,
		ttl:     ttl,
		entries: make(map[string]*idempotencyEntry, maxKeys),
		lru:     list.New(),
	}
}

// Reserve атомарно резервирует ключ для выполнения или возвращает уже существующую задачу.
func (s *IdempotencyStore) Reserve(ctx context.Context, key string, requestHash string) (domain.TaskID, bool, error) {
	if err := ctx.Err(); err != nil {
		return domain.TaskID{}, false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	if entry, exists := s.entries[key]; exists {
		// Проверка устаревания по TTL
		if s.ttl > 0 && now.Sub(entry.createdAt) > s.ttl {
			s.removeEntry(entry)
		} else {
			s.lru.MoveToFront(entry.element)

			// Проверка неизменности тела запроса (AIP-155)
			if entry.requestHash != requestHash {
				return domain.TaskID{}, false, usecase.ErrIdempotencyConflict
			}

			// Если запрос ещё в обработке параллельной горутиной
			if entry.inFlight {
				return domain.TaskID{}, false, usecase.ErrIdempotencyInFlight
			}

			// Запрос уже успешно завершён ранее
			return entry.taskID, true, nil
		}
	}

	// Вытеснение по LRU при достижении предела ёмкости
	if len(s.entries) >= s.maxKeys {
		s.evictOldest()
	}

	elem := s.lru.PushFront(key)
	entry := &idempotencyEntry{
		key:         key,
		requestHash: requestHash,
		inFlight:    true,
		createdAt:   now,
		element:     elem,
	}
	s.entries[key] = entry

	return domain.TaskID{}, false, nil
}

// Complete фиксирует успешное выполнение операции и связывает ключ с созданной задачей.
func (s *IdempotencyStore) Complete(ctx context.Context, key string, taskID domain.TaskID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry, exists := s.entries[key]; exists {
		entry.taskID = taskID
		entry.inFlight = false
		s.lru.MoveToFront(entry.element)
	}

	return nil
}

// Release снимает резервацию незавершённого запроса в случае ошибки сценария.
func (s *IdempotencyStore) Release(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entry, exists := s.entries[key]; exists && entry.inFlight {
		s.removeEntry(entry)
	}

	return nil
}

func (s *IdempotencyStore) removeEntry(entry *idempotencyEntry) {
	s.lru.Remove(entry.element)
	delete(s.entries, entry.key)
}

func (s *IdempotencyStore) evictOldest() {
	// В первую очередь вытесняем старейший завершённый запрос с конца списка LRU
	for elem := s.lru.Back(); elem != nil; elem = elem.Prev() {
		key := elem.Value.(string)
		if entry, exists := s.entries[key]; exists && !entry.inFlight {
			s.removeEntry(entry)
			return
		}
	}

	// Если все элементы in-flight, вытесняем самый старый
	if oldest := s.lru.Back(); oldest != nil {
		key := oldest.Value.(string)
		if entry, exists := s.entries[key]; exists {
			s.removeEntry(entry)
		}
	}
}

// --8<-- [end:idempotency_store]
