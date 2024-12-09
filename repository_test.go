package task_test

import (
	"context"
	"encoding/json"
	"github.com/nanostack-dev/task"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGivenValidTask_WhenCreatedAndRetrieved_ThenDataMatches(t *testing.T) {
	ctx := context.Background()
	repo := &task.Repository{DB: TestDb}

	// Given
	payload := map[string]interface{}{
		"title":       "Test Task",
		"description": "This is a test task",
	}
	payloadJSON, _ := json.Marshal(payload)
	taskData := &task.Task{
		Payload:  payloadJSON,
		Status:   "pending",
		Priority: 1,
	}

	// When
	createdTask, err := repo.CreateTask(ctx, taskData)

	// Then
	assert.NoError(t, err)
	assert.NotNil(t, createdTask)
	assert.NotZero(t, createdTask.ID)

	retrievedTask, err := repo.GetTask(ctx, createdTask.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedTask)

	var retrievedPayload map[string]interface{}
	json.Unmarshal(retrievedTask.Payload, &retrievedPayload)

	assert.Equal(t, payload["title"], retrievedPayload["title"])
	assert.Equal(t, payload["description"], retrievedPayload["description"])
	assert.Equal(t, taskData.Status, retrievedTask.Status)
	assert.Equal(t, taskData.Priority, retrievedTask.Priority)
}

func TestGivenExistingTask_WhenUpdated_ThenChangesAreReflected(t *testing.T) {
	ctx := context.Background()
	repo := &task.Repository{DB: TestDb}

	// Given
	payload := map[string]interface{}{
		"title":       "Initial Task",
		"description": "Initial description",
	}
	payloadJSON, _ := json.Marshal(payload)
	taskData := &task.Task{
		Payload:  payloadJSON,
		Status:   "pending",
		Priority: 2,
	}
	createdTask, _ := repo.CreateTask(ctx, taskData)

	// When
	createdTask.Status = "completed"
	createdTask.Priority = 3
	err := repo.UpdateTask(
		ctx, createdTask.ID, task.TaskUpdates{
			Status:   &createdTask.Status,
			Priority: &createdTask.Priority,
		},
	)

	// Then
	assert.NoError(t, err)

	updatedTask, err := repo.GetTask(ctx, createdTask.ID)
	assert.NoError(t, err)
	assert.NotNil(t, updatedTask)

	var updatedPayload map[string]interface{}
	json.Unmarshal(updatedTask.Payload, &updatedPayload)
	assert.Equal(t, createdTask.Status, updatedTask.Status)
	assert.Equal(t, createdTask.Priority, updatedTask.Priority)
}

func TestGivenExistingTask_WhenDeleted_ThenCannotBeRetrieved(t *testing.T) {
	ctx := context.Background()
	repo := &task.Repository{DB: TestDb}

	// Given
	payload := map[string]interface{}{
		"title":       "Task to Delete",
		"description": "This task will be deleted",
	}
	payloadJSON, _ := json.Marshal(payload)
	taskData := &task.Task{
		Payload:  payloadJSON,
		Status:   "pending",
		Priority: 1,
	}
	createdTask, _ := repo.CreateTask(ctx, taskData)

	// When
	err := repo.DeleteTask(ctx, createdTask.ID)

	// Then
	assert.NoError(t, err)

	deletedTask, err := repo.GetTask(ctx, createdTask.ID)
	assert.NoError(t, err)
	assert.Nil(t, deletedTask)
}

func TestGivenMultipleTasks_WhenFilteredByStatus_ThenOnlyMatchingTasksAreReturned(t *testing.T) {
	//TODO: Implement this test
}
