package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// --8<-- [start:domain_errors]
var (
	ErrEmptyTaskID          = errors.New("идентификатор задачи не может быть пустым")
	ErrEmptyOwnerID         = errors.New("идентификатор владельца (студента) не может быть пустым")
	ErrEmptyTitle           = errors.New("название задачи не может быть пустым")
	ErrTitleTooShort        = errors.New("название задачи слишком короткое (минимум 3 символа)")
	ErrTitleTooLong         = errors.New("название задачи не может превышать 200 символов")
	ErrDescriptionTooLong   = errors.New("описание задачи не может превышать 2000 символов")
	ErrInvalidDeadline      = errors.New("дедлайн задачи должен быть строго в будущем")
	ErrInvalidPriority      = errors.New("указан некорректный или неопределённый приоритет задачи")
	ErrTaskNotFound         = errors.New("учебная задача не найдена")
	ErrTaskAlreadyCompleted = errors.New("нельзя изменить уже завершённую задачу")
	ErrVersionConflict      = errors.New("конфликт параллельного изменения задачи (оптимистичная блокировка)")
)

// --8<-- [end:domain_errors]

// --8<-- [start:value_objects]
// TaskID — Value Object уникального идентификатора задачи.
type TaskID struct {
	value string
}

func NewTaskID(raw string) (TaskID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return TaskID{}, ErrEmptyTaskID
	}
	return TaskID{value: trimmed}, nil
}

func (id TaskID) String() string { return id.value }
func (id TaskID) IsZero() bool   { return id.value == "" }

// OwnerID — Value Object идентификатора владельца задачи.
type OwnerID struct {
	value string
}

func NewOwnerID(raw string) (OwnerID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return OwnerID{}, ErrEmptyOwnerID
	}
	return OwnerID{value: trimmed}, nil
}

func (id OwnerID) String() string { return id.value }
func (id OwnerID) IsZero() bool   { return id.value == "" }

// Title — Value Object названия задачи с контролем бизнес-инвариантов.
type Title struct {
	value string
}

func NewTitle(raw string) (Title, error) {
	trimmed := strings.TrimSpace(raw)
	count := utf8.RuneCountInString(trimmed)
	if count == 0 {
		return Title{}, ErrEmptyTitle
	}
	if count < 3 {
		return Title{}, ErrTitleTooShort
	}
	if count > 200 {
		return Title{}, ErrTitleTooLong
	}
	return Title{value: trimmed}, nil
}

func (t Title) String() string { return t.value }
func (t Title) IsZero() bool   { return t.value == "" }

// Description — Value Object описания задачи с контролем бизнес-инвариантов.
type Description struct {
	value string
}

func NewDescription(raw string) (Description, error) {
	trimmed := strings.TrimSpace(raw)
	if utf8.RuneCountInString(trimmed) > 2000 {
		return Description{}, ErrDescriptionTooLong
	}
	return Description{value: trimmed}, nil
}

func (d Description) String() string { return d.value }
func (d Description) IsZero() bool   { return d.value == "" }

// Deadline — Value Object дедлайна, нормализованный в UTC.
type Deadline struct {
	value time.Time
}

func NewDeadline(t time.Time, now time.Time) (Deadline, error) {
	if t.IsZero() {
		return Deadline{}, ErrInvalidDeadline
	}
	utcTime := t.UTC()
	utcNow := now.UTC()
	if !utcTime.After(utcNow) {
		return Deadline{}, ErrInvalidDeadline
	}
	return Deadline{value: utcTime}, nil
}

func ReconstituteDeadline(t time.Time) Deadline {
	return Deadline{value: t.UTC()}
}

func (d Deadline) Time() time.Time { return d.value }
func (d Deadline) IsZero() bool    { return d.value.IsZero() }

// TaskPriority — строгое перечисление приоритетов задачи.
type TaskPriority int

const (
	TaskPriorityUnspecified TaskPriority = 0
	TaskPriorityLow         TaskPriority = 1
	TaskPriorityMedium      TaskPriority = 2
	TaskPriorityHigh        TaskPriority = 3
)

func (p TaskPriority) IsValid() bool {
	return p == TaskPriorityLow || p == TaskPriorityMedium || p == TaskPriorityHigh
}

func ParseTaskPriority(val int) (TaskPriority, error) {
	p := TaskPriority(val)
	if !p.IsValid() {
		return TaskPriorityUnspecified, fmt.Errorf("%w: недопустимое значение %d", ErrInvalidPriority, val)
	}
	return p, nil
}

func (p TaskPriority) String() string {
	switch p {
	case TaskPriorityLow:
		return "LOW"
	case TaskPriorityMedium:
		return "MEDIUM"
	case TaskPriorityHigh:
		return "HIGH"
	default:
		return "UNSPECIFIED"
	}
}

// TaskStatus — статус выполнения задачи.
type TaskStatus int

const (
	TaskStatusUnspecified TaskStatus = 0
	TaskStatusPending     TaskStatus = 1
	TaskStatusCompleted   TaskStatus = 2
)

func (s TaskStatus) IsValid() bool {
	return s == TaskStatusPending || s == TaskStatusCompleted
}

func (s TaskStatus) String() string {
	switch s {
	case TaskStatusPending:
		return "PENDING"
	case TaskStatusCompleted:
		return "COMPLETED"
	default:
		return "UNSPECIFIED"
	}
}

// --8<-- [end:value_objects]

// --8<-- [start:domain_entity]
// Task — доменная сущность с контролем владения и оптимистичной блокировкой (version).
type Task struct {
	id          TaskID
	ownerID     OwnerID
	title       Title
	description Description
	deadline    Deadline
	priority    TaskPriority
	status      TaskStatus
	version     int64
	createdAt   time.Time
	events      []TaskCreated
}

// NewTask создаёт новую задачу. Проверяет обязательные Value Objects на IsZero.
func NewTask(
	id TaskID,
	ownerID OwnerID,
	title Title,
	description Description,
	deadline Deadline,
	priority TaskPriority,
	now time.Time,
) (*Task, error) {
	if id.IsZero() {
		return nil, ErrEmptyTaskID
	}
	if ownerID.IsZero() {
		return nil, ErrEmptyOwnerID
	}
	if title.IsZero() {
		return nil, ErrEmptyTitle
	}
	if deadline.IsZero() {
		return nil, ErrInvalidDeadline
	}
	if !priority.IsValid() {
		return nil, ErrInvalidPriority
	}

	utcNow := now.UTC()
	t := &Task{
		id:          id,
		ownerID:     ownerID,
		title:       title,
		description: description,
		deadline:    deadline,
		priority:    priority,
		status:      TaskStatusPending,
		version:     1,
		createdAt:   utcNow,
	}

	t.events = append(t.events, TaskCreated{
		TaskID:     id,
		OwnerID:    ownerID,
		Title:      title,
		Deadline:   deadline,
		Priority:   priority,
		OccurredAt: utcNow,
	})

	return t, nil
}

// ReconstituteTask вызывается исключительно адаптерами инфраструктуры для восстановления из БД.
func ReconstituteTask(
	id TaskID,
	ownerID OwnerID,
	title Title,
	description Description,
	deadline Deadline,
	priority TaskPriority,
	status TaskStatus,
	version int64,
	createdAt time.Time,
) *Task {
	return &Task{
		id:          id,
		ownerID:     ownerID,
		title:       title,
		description: description,
		deadline:    deadline,
		priority:    priority,
		status:      status,
		version:     version,
		createdAt:   createdAt.UTC(),
	}
}

func (t *Task) ID() TaskID               { return t.id }
func (t *Task) OwnerID() OwnerID         { return t.ownerID }
func (t *Task) Title() Title             { return t.title }
func (t *Task) Description() Description { return t.description }
func (t *Task) Deadline() Deadline       { return t.deadline }
func (t *Task) Priority() TaskPriority   { return t.priority }
func (t *Task) Status() TaskStatus       { return t.status }
func (t *Task) Version() int64           { return t.version }
func (t *Task) CreatedAt() time.Time     { return t.createdAt }

// IncrementVersion увеличивает номер версии сущности при успешном сохранении в репозитории.
func (t *Task) IncrementVersion() {
	t.version++
}

// PullEvents возвращает накопленные доменные события и очищает внутренний буфер.
func (t *Task) PullEvents() []TaskCreated {
	events := t.events
	t.events = nil
	return events
}

// IsOwnedBy проверяет принадлежность задачи указанному пользователю.
func (t *Task) IsOwnedBy(actor OwnerID) bool {
	return t.ownerID == actor
}

// Complete переводит задачу в статус завершённой (версия инкрементируется репозиторием при сохранении).
func (t *Task) Complete() error {
	if t.status == TaskStatusCompleted {
		return ErrTaskAlreadyCompleted
	}
	t.status = TaskStatusCompleted
	return nil
}

// Reschedule переносит дедлайн задачи (версия инкрементируется репозиторием при сохранении).
func (t *Task) Reschedule(newDeadline Deadline, now time.Time) error {
	if t.status == TaskStatusCompleted {
		return ErrTaskAlreadyCompleted
	}
	if newDeadline.IsZero() || !newDeadline.Time().After(now.UTC()) {
		return ErrInvalidDeadline
	}
	t.deadline = newDeadline
	return nil
}

// --8<-- [end:domain_entity]

// --8<-- [start:domain_events]
// TaskCreated — неизменяемое доменное событие.
type TaskCreated struct {
	EventID    string
	TaskID     TaskID
	OwnerID    OwnerID
	Title      Title
	Deadline   Deadline
	Priority   TaskPriority
	OccurredAt time.Time
}

// --8<-- [end:domain_events]
