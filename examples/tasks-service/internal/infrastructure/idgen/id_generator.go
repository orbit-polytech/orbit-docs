package idgen

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/domain"
)

// --8<-- [start:id_generator]
// UUIDGenerator генерирует уникальные идентификаторы без паники.
type UUIDGenerator struct{}

func NewUUIDGenerator() *UUIDGenerator {
	return &UUIDGenerator{}
}

// NewTaskID генерирует новый доменный TaskID. Возвращает явную ошибку вместо panic.
func (g *UUIDGenerator) NewTaskID() (domain.TaskID, error) {
	newUUID, err := uuid.NewRandom()
	if err != nil {
		return domain.TaskID{}, fmt.Errorf("ошибка генератора случайных чисел при создании UUID: %w", err)
	}

	taskID, err := domain.NewTaskID(newUUID.String())
	if err != nil {
		return domain.TaskID{}, fmt.Errorf("ошибка создания TaskID из UUID: %w", err)
	}

	return taskID, nil
}

// NewEventID генерирует уникальный идентификатор события без паники.
func (g *UUIDGenerator) NewEventID() (string, error) {
	newUUID, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("ошибка генератора случайных чисел при создании UUID события: %w", err)
	}
	return newUUID.String(), nil
}

// --8<-- [end:id_generator]
