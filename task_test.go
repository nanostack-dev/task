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

// Payload validation helper function
func validatePayload(t *testing.T, expected, actual MyPayload) {
	// Compare Title and Description directly
	assert.EqualValues(t, expected.Title, actual.Title)
	assert.EqualValues(t, expected.Description, actual.Description)

	// Normalize and compare DatePtr
	require.NotNil(t, expected.DatePtr)
	require.NotNil(t, actual.DatePtr)
	assert.True(t, expected.DatePtr.Equal(*actual.DatePtr), "DatePtr should match")

	// Compare SubPayload
	if expected.SubPayload != nil && actual.SubPayload != nil {
		assert.EqualValues(t, expected.SubPayload.SubTitle, actual.SubPayload.SubTitle)
	} else {
		assert.Equal(t, expected.SubPayload, actual.SubPayload)
	}
}

func initializeTaskFramework(t *testing.T, dsn string) {
	err := task.InitNanostackTask(
		task.TaskFrameworkConfig{
			DatabaseDSN:  dsn,
			WorkerCount:  5,
			PollInterval: 500 * time.Millisecond,
			Logger:       nil,
			RetryConfig:  nil,
		},
	)
	require.NoError(t, err, "Failed to initialize NanostackTask framework")
}

func subscribeToTask(
	t *testing.T,
	taskName string,
	handler func(payload MyPayload, taskReceived *task.Task) error,
) {
	err := task.SubscribeWithTask(taskName, handler)
	require.NoError(t, err, "Failed to subscribe to task")
}

func sendTasks(
	ctx context.Context, t *testing.T, taskName string, payload MyPayload, count int,
) sync.Map {
	var ids sync.Map
	for i := 0; i < count; i++ {
		taskID, err := task.SendTask(ctx, taskName, payload)
		require.NoError(t, err, "Failed to send task")
		ids.Store(taskID, true)
	}
	return ids
}

func sendScheduledTask(
	ctx context.Context, t *testing.T, taskName string, payload MyPayload, scheduledAt time.Time,
) string {
	taskID, err := task.SendTaskWithOpts(
		ctx, taskName, payload, task.TaskOptions{
			ScheduledAt: &scheduledAt,
		},
	)
	require.NoError(t, err, "Failed to send scheduled task")
	return taskID
}

func waitForTasks(t *testing.T, wg *sync.WaitGroup, timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
		task.StopNanostackTask()
	}()

	select {
	case <-done:
		// Test completed
	case <-time.After(timeout):
		t.Fatalf("Test timed out after %v", timeout)
	}
}

func TestTask(t *testing.T) {
	ctx := context.Background()
	initializeTaskFramework(t, dsn)

	expectedPayload := GeneratePayload()
	wg := sync.WaitGroup{}
	ids := sendTasks(ctx, t, taskName, expectedPayload, 10)

	subscribeToTask(
		t, taskName, func(payload MyPayload, taskReceived *task.Task) error {
			validatePayload(t, expectedPayload, payload)
			_, ok := ids.Load(taskReceived.ID)
			require.True(t, ok, "Task ID not found in the map")
			ids.Delete(taskReceived.ID)
			wg.Done()
			return nil
		},
	)

	wg.Add(10)
	waitForTasks(t, &wg, 10*time.Second)
}

func TestScheduleTask(t *testing.T) {
	ctx := context.Background()
	initializeTaskFramework(t, dsn)

	expectedPayload := GeneratePayload()
	now := time.Now()
	future := now.Add(5 * time.Second)

	taskID := sendScheduledTask(ctx, t, taskName, expectedPayload, future)
	wg := sync.WaitGroup{}
	wg.Add(1)

	subscribeToTask(
		t, taskName, func(payload MyPayload, taskReceived *task.Task) error {
			validatePayload(t, expectedPayload, payload)
			nowReceive := time.Now()
			assert.Less(
				t, nowReceive.Sub(future), 1*time.Second,
				"Task was not received within expected time",
			)
			assert.Equal(t, taskID, taskReceived.ID, "Task ID mismatch")
			wg.Done()
			return nil
		},
	)

	waitForTasks(t, &wg, 10*time.Second)
}
