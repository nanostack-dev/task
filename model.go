package task

import (
	"encoding/json"
	"time"
)

type TaskStatus string

const (
	Pending   TaskStatus = "pending"
	Running   TaskStatus = "running"
	Completed TaskStatus = "completed"
	Failed    TaskStatus = "failed"
	Canceled  TaskStatus = "canceled"
)

// Task represents a task entity.
type Task struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Payload     json.RawMessage `json:"payload"`
	Status      TaskStatus      `json:"status"`
	Priority    int             `json:"priority"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	ScheduledAt *time.Time      `json:"scheduled_at"`
	RetryCount  int             `json:"retry_count"`
	MaxRetry    int             `json:"max_retry"`
}

func (t *Task) IsRetryable() bool {
	return t.RetryCount < t.MaxRetry
}

func (t *Task) IncrementRetryCount() {
	t.RetryCount++
}

func (t *Task) GetAllTaskValues() []interface{} {
	return []interface{}{
		&t.ID,
		&t.Name,
		&t.Payload,
		&t.Status,
		&t.Priority,
		&t.CreatedAt,
		&t.UpdatedAt,
		&t.ScheduledAt,
		&t.RetryCount,
		&t.MaxRetry,
	}
}
func GetAllTaskColumns() string {
	return "id, name, payload, status, priority, created_at, updated_at, scheduled_at, retry_count, max_retry"
}
