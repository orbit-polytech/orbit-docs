package inmemory

import (
	"context"
	"log/slog"
	"sync"

	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/domain"
)

// --8<-- [start:inmemory_events]
const defaultMaxEvents = 1000

// EventPublisher — потокобезопасная in-memory реализация шины событий с ограниченным буфером (защита от утечки памяти).
type EventPublisher struct {
	mu        sync.Mutex
	logger    *slog.Logger
	maxEvents int
	events    []domain.TaskCreated
}

func NewEventPublisher(logger *slog.Logger) *EventPublisher {
	return &EventPublisher{
		logger:    logger,
		maxEvents: defaultMaxEvents,
		events:    make([]domain.TaskCreated, 0, 100),
	}
}

// PublishTaskCreated сохраняет неизменяемое событие по значению.
func (p *EventPublisher) PublishTaskCreated(ctx context.Context, event domain.TaskCreated) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Ограничение размера очереди (кольцевой буфер): вытесняем старые события при переполнении
	if len(p.events) >= p.maxEvents {
		p.events = p.events[1:]
	}

	p.events = append(p.events, event)

	if p.logger != nil {
		p.logger.InfoContext(ctx, "событие TaskCreated опубликовано (in-memory)",
			slog.String("task_id", event.TaskID.String()),
			slog.String("owner_id", event.OwnerID.String()),
			slog.String("title", event.Title.String()),
			slog.String("deadline", event.Deadline.Time().String()),
		)
	}
	return nil
}

// PublishedEvents возвращает копию опубликованных событий (для тестов).
func (p *EventPublisher) PublishedEvents() []domain.TaskCreated {
	p.mu.Lock()
	defer p.mu.Unlock()

	res := make([]domain.TaskCreated, len(p.events))
	copy(res, p.events)
	return res
}

// --8<-- [end:inmemory_events]
