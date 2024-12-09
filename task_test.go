package task_test

import (
	"context"
	"github.com/nanostack-dev/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

type SubPayload struct {
	SubTitle string
}
type MyPayload struct {
	Title       string
	Description string
	DatePtr     *time.Time
	SubPayload  *SubPayload
}

func GeneratePayload() MyPayload {
	now := time.Now()
	return MyPayload{
		Title:       "Title",
		Description: "Description",
		DatePtr:     &now,
		SubPayload: &SubPayload{
			SubTitle: "SubTitle",
		},
	}
}

var taskName = "test_task"

func TestTask(t *testing.T) {
	ctx := context.Background()
	err := task.InitNanostackTask(
		task.TaskInitOptions{
			DSN:          dsn,
			WorkerCount:  5,
			PollInterval: time.Duration(1) * time.Second,
		},
	)
	ids := make(map[string]bool)
	expectedPayload := GeneratePayload()
	wg := sync.WaitGroup{}
	err = task.SubscribeWithTask(
		taskName, func(payload MyPayload, taskReceived *task.Task) error {
			// Compare Title and Description directly
			assert.EqualValues(t, expectedPayload.Title, payload.Title)
			assert.EqualValues(t, expectedPayload.Description, payload.Description)

			// Normalize and compare DatePtr
			require.NotNil(t, expectedPayload.DatePtr)
			require.NotNil(t, payload.DatePtr)
			assert.True(t, expectedPayload.DatePtr.Equal(*payload.DatePtr), "DatePtr should match")

			// Compare SubPayload
			if expectedPayload.SubPayload != nil && payload.SubPayload != nil {
				assert.EqualValues(
					t, expectedPayload.SubPayload.SubTitle, payload.SubPayload.SubTitle,
				)
			} else {
				assert.Equal(t, expectedPayload.SubPayload, payload.SubPayload)
			}

			// Ensure task ID is correct
			_, ok := ids[taskReceived.ID]
			require.True(t, ok)
			ids[taskReceived.ID] = false
			wg.Done()
			return nil
		},
	)
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		taskID, err := task.SendTask(ctx, taskName, expectedPayload)
		require.NoError(t, err)
		ids[taskID] = true
	}
	wg.Wait()

	require.NoError(t, err)
}
