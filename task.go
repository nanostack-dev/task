package task

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"sync"
	"time"
)

var (
	defaultTaskOptions = TaskOptions{
		Priority: ToPointer(1),
		MaxRetry: ToPointer(3),
	}
	repository    *Repository
	subscriptions map[string]func(payload any, task *Task) error = make(
		map[string]func(
			payload any, task *Task,
		) error,
	)
	taskQueue chan *Task
	quit      chan struct{}
	wg        sync.WaitGroup
)

// TaskOptions represents task options.
type TaskOptions struct {
	Priority    *int
	ScheduledAt *time.Time
	MaxRetry    *int
}

// TaskInitOptions represents initialization options.
type TaskInitOptions struct {
	DSN          string
	WorkerCount  int
	PollInterval time.Duration
}

// InitNanostack initializes the task processing system.
func InitNanostackTask(options TaskInitOptions) error {
	// Initialize the repository
	logger.Infof("Connecting to database")
	db, err := sql.Open("postgres", options.DSN)

	if err != nil {
		return err
	}
	err = db.Ping()
	if err != nil {
		return err
	}
	repository = &Repository{DB: db}

	// Initialize task queue and quit channel
	taskQueue = make(chan *Task, 100)
	quit = make(chan struct{})

	// Start workers
	for i := 0; i < options.WorkerCount; i++ {
		wg.Add(1)
		go worker(i)
	}
	logger.Infof(
		"Started %d workers with poll interval %s", options.WorkerCount, options.PollInterval,
	)
	// Start event loop
	go eventLoop(options.PollInterval)

	return nil
}

// StopNanostack stops the task processing system.
func StopNanostackTask() {
	close(quit)
	wg.Wait()
	close(taskQueue)
}

// Subscribe registers a handler for a task name.
func Subscribe[T any](name string, handler func(payload T) error) error {
	return SubscribeWithTask(
		name, func(payload T, _ *Task) error {
			return handler(payload)
		},
	)
}

func SubscribeWithTask[T any](name string, handler func(payload T, task *Task) error) error {
	if _, ok := subscriptions[name]; ok {
		return fmt.Errorf("task %s already subscribed", name)
	}
	subscriptions[name] = func(payload any, task *Task) error {
		var genericPayload T
		if err := json.Unmarshal(payload.(json.RawMessage), &genericPayload); err != nil {
			return err
		}
		return handler(genericPayload, task)
	}
	return nil
}

// SendTask sends a task to the queue.
func SendTask(ctx context.Context, name string, payload any) (string, error) {
	return SendTaskWithOpts(ctx, name, payload, defaultTaskOptions)
}

// SendTaskWithOpts sends a task with specific options.
func SendTaskWithOpts(ctx context.Context, name string, payload any, opts TaskOptions) (
	string, error,
) {
	marshal, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	now := time.Now()
	task := &Task{
		ID:          uuid.NewString(),
		Name:        name,
		Payload:     json.RawMessage(marshal),
		Status:      Pending,
		Priority:    PointerOrDefaultValue(opts.Priority, 1),
		CreatedAt:   now,
		UpdatedAt:   now,
		ScheduledAt: opts.ScheduledAt,
		RetryCount:  0,
		MaxRetry:    PointerOrDefaultValue(opts.MaxRetry, 3),
	}
	createdTask, err := repository.CreateTask(ctx, task)
	if err != nil {
		return "", err
	}
	logger.Infof("Created task %s", createdTask.ID)
	return createdTask.ID, nil
}

// worker processes tasks from the queue.
func worker(workerNumber int) {
	defer wg.Done()
	for task := range taskQueue {
		logger.Infof("Worker %d processing task %s", workerNumber, task.ID)
		if handler, exists := subscriptions[task.Name]; exists {
			if err := handler(task.Payload, task); err == nil {
				logger.Errorf("Error processing task %s: %v", task.Name, err)
				task.IncrementRetryCount()
				status := Pending
				if !task.IsRetryable() {
					status = Failed
					logger.Errorf("Task %s has reached max retries", task.ID)
				}
				if err := repository.UpdateTask(
					context.Background(), task.ID,
					TaskUpdates{
						Status:      ToPointer(status),
						Priority:    ToPointer(task.Priority),
						ScheduledAt: task.ScheduledAt,
						RetryCount:  ToPointer(task.RetryCount),
					},
				); err != nil {
					logger.Errorf("Failed to update task %s: %v", task.ID, err)
				}
			}
		} else {
			logger.Warnf("No handler found for task %s", task.Name)
		}
		logger.Infof("Worker %d finished processing task %s", workerNumber, task.ID)
	}
}

// eventLoop polls for pending tasks and pushes them into the queue.
func eventLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-quit:
			return
		case <-ticker.C:
			logger.Debugf("Polling for pending tasks")
			pendingTasks, err := GetLastTask(context.Background())
			logger.Debugf("Found %d pending tasks", len(pendingTasks))
			if err != nil {
				logger.Errorf("Failed to retrieve pending tasks: %v", err)
				continue
			}
			for _, task := range pendingTasks {
				logger.Debugf("Pushing task %s to internal queue", task.ID)
				taskQueue <- task
			}
		}
	}
}

func GetLastTask(ctx context.Context) ([]*Task, error) {
	return repository.SearchTasks(
		ctx, TaskFilter{
			Status:          ToPointer(Pending),
			ScheduledBefore: ToPointer(time.Now()),
		}, QueryOptions{
			Limit:  5,
			Locked: true,
			Sort: []SortOption{
				{
					Field:     "priority",
					Direction: "DESC",
				},
				{
					Field:     "created_at",
					Direction: "ASC",
				},
			},
		},
	)
}
