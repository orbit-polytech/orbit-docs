package idgen_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/orbit-polytech/orbit-docs/examples/tasks-service/internal/infrastructure/idgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUUIDGenerator(t *testing.T) {
	gen := idgen.NewUUIDGenerator()

	t.Run("NewTaskID generates valid TaskID without panic", func(t *testing.T) {
		taskID, err := gen.NewTaskID()
		require.NoError(t, err)
		assert.False(t, taskID.IsZero())

		parsed, err := uuid.Parse(taskID.String())
		require.NoError(t, err)
		assert.Equal(t, uuid.Version(4), parsed.Version())
	})

	t.Run("NewEventID generates valid UUID without panic", func(t *testing.T) {
		evtID, err := gen.NewEventID()
		require.NoError(t, err)
		assert.NotEmpty(t, evtID)

		parsed, err := uuid.Parse(evtID)
		require.NoError(t, err)
		assert.Equal(t, uuid.Version(4), parsed.Version())
	})
}
