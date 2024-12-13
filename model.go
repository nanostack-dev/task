package task

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type TaskStatus string

const (
	Pending   TaskStatus = "pending"
	Running   TaskStatus = "running"
	Retrying  TaskStatus = "retrying"
	Completed TaskStatus = "completed"
	Failed    TaskStatus = "failed"
	Canceled  TaskStatus = "canceled"
)

var TaskAllColumns []string = getAllTaskColumns()

// Task represents a task entity.
type Task struct {
	ID          string          `json:"id" db:"id"`
	Name        string          `json:"name" db:"name"`
	Payload     json.RawMessage `json:"payload" db:"payload"`
	Status      TaskStatus      `json:"status" db:"status"`
	Priority    int             `json:"priority" db:"priority"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
	ScheduledAt *time.Time      `json:"scheduled_at" db:"scheduled_at"`
	RetryCount  int             `json:"retry_count" db:"retry_count"`
}

func getAllTaskColumns() []string {
	//Use reflection to get all the fields of the struct `db` tag
	var columns []string
	t := reflect.TypeOf(Task{})
	paramsQuery := ""
	for i := 0; i < t.NumField(); i++ {
		columns = append(columns, t.Field(i).Tag.Get("db"))
		paramsQuery += "$" + strconv.Itoa(i+1) + ", "
	}
	//join the columns with a comma
	return []string{
		strings.Join(columns, ", "),
		strings.TrimSuffix(paramsQuery, ", "),
	}
}

func (t *Task) IsRetryable(maxRetry int) bool {
	return t.RetryCount < maxRetry
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
	}
}
