package task

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/google/uuid"
	"sync"
	"time"
)

// Options for configuring task execution.
type TaskOptions struct {
	Priority    *int       // Optional priority of the task.
	ScheduledAt *time.Time // Optional time to schedule the task.
	Tx          *sql.Tx    // Optional transaction to use for task processing.
}

// Type of backoff strategy for retries.
type RetryBackoffStrategy string

const (
	// Fixed retry interval: wait_interval = base
	BackoffFixed RetryBackoffStrategy = "fixed"
	// Exponential retry interval: wait_interval = base * 2^n
	BackoffExponential RetryBackoffStrategy = "exponential"
)

// Initialization options for the task framework.
type TaskFrameworkConfig struct {
	DatabaseDSN  string           // Data source name for the task database.
	WorkerCount  int              // Number of workers to process tasks.
	PollInterval time.Duration    // Interval for polling the task queue.
	Logger       Logger           // Logger instance for logging activities.
	RetryConfig  *TaskRetryConfig // Default retry configuration for tasks.
}

// Retry configuration for tasks.
type TaskRetryConfig struct {
	MaxAttempts     *int                  // Maximum number of retry attempts.
	RetryInterval   *time.Duration        // Base interval between retries.
	BackoffStrategy *RetryBackoffStrategy // Strategy for calculating retry intervals.
	TimeLimit       *time.Duration        // Maximum time limit for retrying a task.
}

// Configuration for task subscriptions.
type TaskSubscription struct {
	Name        string                              // Name of the subscription.
	Handler     func(payload any, task *Task) error // Handler function for processing the task.
	RetryConfig *TaskRetryConfig                    // Retry configuration specific to the subscription.
}

var (
	defaultTaskSettings = &TaskFrameworkConfig{
		RetryConfig: &TaskRetryConfig{
			MaxAttempts:     ToPointer(3),
			RetryInterval:   ToPointer(5 * time.Second),
			BackoffStrategy: ToPointer(BackoffFixed),
			TimeLimit:       ToPointer(5 * time.Hour),
		},
	}
	defaultTaskOptions = TaskOptions{
		Priority: ToPointer(1),
	}
	repository    *Repository
	subscriptions map[string][]TaskSubscription = make(map[string][]TaskSubscription)
	taskQueue     chan *Task
	quit          chan struct{}
	wg            sync.WaitGroup
)

func computeBackoff(retryOptions TaskRetryConfig, retryCount int) time.Duration {
	var duration time.Duration
	switch *retryOptions.BackoffStrategy {
	case BackoffFixed:
		duration = *retryOptions.RetryInterval
	case BackoffExponential:
		duration = *retryOptions.RetryInterval * time.Duration(2<<retryCount)
	default:
		duration = *retryOptions.RetryInterval
	}
	if duration > *retryOptions.TimeLimit {
		return *retryOptions.TimeLimit
	}
	return duration
}

// InitNanostackTask InitNanostack initializes the task processing system.
func InitNanostackTask(options TaskFrameworkConfig) error {
	if options.Logger != nil {
		SetLogger(options.Logger)
	}
	if options.RetryConfig != nil {
		logger.Infof("Using custom retry configuration %+v", options.RetryConfig)
	}
	logger.Infof("Connecting to database")
	db, err := sql.Open("postgres", options.DatabaseDSN)

	if err != nil {
		return err
	}
	err = InitDB(db)
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

func StopNanostackTask() {
	close(quit)
	wg.Wait()
	close(taskQueue)
	subscriptions = make(map[string][]TaskSubscription)
	logger.Infof("Stopped task processing")
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
	return SubscribeWithOpts(name, handler, defaultTaskSettings.RetryConfig)
}

func SubscribeWithOpts[T any](
	name string, handler func(payload T, task *Task) error, opts *TaskRetryConfig,
) error {
	if opts == nil {
		opts = defaultTaskSettings.RetryConfig
	}
	var subscriptionsFn []TaskSubscription
	if subsFnExisting, exists := subscriptions[name]; exists {
		logger.Debugf(
			"Subscription for task %s already have len %d function and adding more", name,
			len(subsFnExisting),
		)
		subscriptionsFn = subsFnExisting
	}
	logger.Infof("Subscribing to task %s with retry config %+v", name, opts)
	subscriptionsFn = append(
		subscriptionsFn, TaskSubscription{
			Name: name,
			Handler: func(payload any, task *Task) error {
				var genericPayload T
				if err := json.Unmarshal(payload.(json.RawMessage), &genericPayload); err != nil {
					return err
				}
				return handler(genericPayload, task)
			},
			RetryConfig: opts,
		},
	)
	subscriptions[name] = subscriptionsFn
	return nil
}

// SendTaskTx sends a task to the queue using a transaction.
func SendTaskTx(ctx context.Context, tx *sql.Tx, name string, payload any) (string, error) {
	return SendTaskWithOpts(ctx, name, payload, TaskOptions{defaultTaskOptions.Priority, nil, tx})
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
	if opts.Tx != nil {
		logger.Debugf("Using transaction for task %s with payload %v", name, marshal)
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
	}
	createdTask, err := repository.CreateTask(SetTransaction(opts.Tx, ctx), task)
	if err != nil {
		return "", err
	}
	if task.ScheduledAt != nil {
		logger.Infof("Scheduled task %s at %s", createdTask.ID, *task.ScheduledAt)
	} else {
		logger.Infof("Created task %s", createdTask.ID)
	}
	return createdTask.ID, nil
}

// worker processes tasks from the queue.
func worker(workerNumber int) {
	defer wg.Done()
	for task := range taskQueue {
		logger.Infof("Worker %d processing task %s", workerNumber, task.ID)
		if subscriptionFns, exists := subscriptions[task.Name]; exists {
			logger.Infof("Starting task %s with payload %s", task.ID, task.Payload)
			for _, subscription := range subscriptionFns {

				if err := subscription.Handler(task.Payload, task); err != nil {
					logger.Errorf("Error processing task %s: %v", task.Name, err)
					task.IncrementRetryCount()
					status := Pending
					if !task.IsRetryable(*subscription.RetryConfig.MaxAttempts) {
						status = Failed
						logger.Errorf("Task %s has reached max retries", task.ID)
					}
					backOffComputed := computeBackoff(*subscription.RetryConfig, task.RetryCount)
					scheduledAt := time.Now().Add(backOffComputed)
					if err != nil {
						logger.Errorf("Failed to compute backoff: %v", err)
					}
					logger.Infof(
						"Retrying task %s in %s at %s", task.ID, backOffComputed, scheduledAt,
					)
					if err := repository.UpdateTask(
						context.Background(), task.ID,
						TaskUpdates{
							Status:      ToPointer(status),
							Priority:    ToPointer(task.Priority),
							ScheduledAt: ToPointer(scheduledAt),
							RetryCount:  ToPointer(task.RetryCount),
						},
					); err != nil {
						logger.Errorf("Failed to update task %s: %v", task.ID, err)
					}
				} else {
					logger.Infof("Task %s processed successfully", task.ID)
					if err := repository.UpdateTask(
						context.Background(), task.ID,
						TaskUpdates{
							Status:     ToPointer(Completed),
							RetryCount: ToPointer(task.RetryCount),
						},
					); err != nil {
						logger.Errorf("Failed to update task %s: %v", task.ID, err)
					}
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

func ReprocessTask(ctx context.Context, taskID string) error {
	return repository.UpdateTask(
		ctx, taskID, TaskUpdates{
			Status:     ToPointer(Pending),
			RetryCount: ToPointer(0),
		},
	)
}
