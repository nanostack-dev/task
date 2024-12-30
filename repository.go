package task

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Repository struct {
	DB Querier
}

// CreateTask inserts a new task into the database.
func (r *Repository) CreateTask(ctx context.Context, task *Task) (*Task, error) {
	query := `
		INSERT INTO tasks (%s)
		VALUES (%s)
		RETURNING %s;
	`
	querier := GetQueryOrTransaction(ctx, r.DB)
	query = fmt.Sprintf(query, TaskAllColumns[0], TaskAllColumns[1], TaskAllColumns[0])
	logger.Debugf("Query: %s", query)
	row := querier.QueryRowContext(
		ctx, query,
		task.ID,
		task.Name,
		task.Payload,
		task.Status,
		task.Priority,
		task.CreatedAt,
		task.UpdatedAt,
		task.ScheduledAt,
		task.RetryCount,
	)

	createdTask := &Task{}
	err := row.Scan(
		&createdTask.ID,
		&createdTask.Name,
		&createdTask.Payload,
		&createdTask.Status,
		&createdTask.Priority,
		&createdTask.CreatedAt,
		&createdTask.UpdatedAt,
		&createdTask.ScheduledAt,
		&createdTask.RetryCount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	return createdTask, nil
}

type TaskUpdates struct {
	Status      *TaskStatus
	Priority    *int
	ScheduledAt *time.Time
	RetryCount  *int
}

// GetTask retrieves a task by its ID.
func (r *Repository) GetTask(ctx context.Context, id string) (*Task, error) {
	query := `
		SELECT %s
		FROM tasks WHERE id = $1;
	`
	query = fmt.Sprintf(query, TaskAllColumns)
	task := &Task{}
	err := r.DB.QueryRowContext(ctx, query, id).Scan(
		&task.ID,
		&task.Payload,
		&task.Status,
		&task.Priority,
		&task.CreatedAt,
		&task.UpdatedAt,
		&task.ScheduledAt,
		&task.RetryCount,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Task not found
		}
		return nil, fmt.Errorf("failed to retrieve task: %w", err)
	}
	return task, nil
}
func (r *Repository) UpdateTask(ctx context.Context, taskID string, updates TaskUpdates) error {
	query := "UPDATE tasks SET"
	args := []interface{}{}
	argIndex := 1
	now := time.Now().UTC() // Ensure UTC format for PostgreSQL

	if updates.Status != nil {
		query += fmt.Sprintf(" status = $%d,", argIndex)
		args = append(args, *updates.Status)
		argIndex++
	}

	if updates.Priority != nil {
		query += fmt.Sprintf(" priority = $%d,", argIndex)
		args = append(args, *updates.Priority)
		argIndex++
	}

	if updates.ScheduledAt != nil {
		query += fmt.Sprintf(" scheduled_at = $%d,", argIndex)
		args = append(args, *updates.ScheduledAt)
		argIndex++
	}

	query += fmt.Sprintf(" updated_at = $%d,", argIndex)
	args = append(args, now)
	argIndex++

	// Remove trailing comma
	query = query[:len(query)-1]

	// Add WHERE clause
	query += fmt.Sprintf(" WHERE id = $%d", argIndex)
	args = append(args, taskID)

	// Execute update
	result, err := r.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Check rows affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no task found with id %s", taskID)
	}

	return nil
}

// DeleteTask deletes a task by its ID.
func (r *Repository) DeleteTask(ctx context.Context, id string) error {
	query := `DELETE FROM tasks WHERE id = $1;`
	result, err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		return fmt.Errorf("task not found")
	}
	return nil
}

type TaskPriorityFilter struct {
	Min *int
	Max *int
	Eq  *int
}

type TaskFilter struct {
	Status          *TaskStatus
	PriorityFilter  *TaskPriorityFilter
	Name            *string
	ScheduledBefore *time.Time
}

type QueryOptions struct {
	Sort   []SortOption
	Limit  int
	Locked bool
}

type SortOption struct {
	Field     string // Column name
	Direction string // ASC or DESC
}

func (r *Repository) SearchTasks(
	ctx context.Context, filter TaskFilter, opts QueryOptions,
) ([]*Task, error) {
	query := `SELECT ` + TaskAllColumns[0] + ` FROM tasks`
	args := []interface{}{}
	conditions := []string{}
	argIndex := 1

	// Filtering
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *filter.Status)
		argIndex++
	}

	if filter.PriorityFilter != nil {
		if filter.PriorityFilter.Min != nil {
			conditions = append(conditions, fmt.Sprintf("priority >= $%d", argIndex))
			args = append(args, *filter.PriorityFilter.Min)
			argIndex++
		}
		if filter.PriorityFilter.Max != nil {
			conditions = append(conditions, fmt.Sprintf("priority <= $%d", argIndex))
			args = append(args, *filter.PriorityFilter.Max)
			argIndex++
		}
		if filter.PriorityFilter.Eq != nil {
			conditions = append(conditions, fmt.Sprintf("priority = $%d", argIndex))
			args = append(args, *filter.PriorityFilter.Eq)
			argIndex++
		}
	}

	if filter.Name != nil {
		conditions = append(conditions, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *filter.Name)
		argIndex++
	}

	if filter.ScheduledBefore != nil {
		conditions = append(
			conditions,
			fmt.Sprintf("scheduled_at IS NULL OR scheduled_at <= $%d", argIndex),
		)
		args = append(args, *filter.ScheduledBefore)
		argIndex++
	}

	// Add conditions to query
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Sorting
	if len(opts.Sort) > 0 {
		query += " ORDER BY "
		sortClauses := []string{}
		for _, sort := range opts.Sort {
			sortClauses = append(sortClauses, fmt.Sprintf("%s %s", sort.Field, sort.Direction))
		}
		query += strings.Join(sortClauses, ", ")
	}

	// Limit
	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, opts.Limit)
		argIndex++
	}

	// Locking
	if opts.Locked {
		query += " FOR UPDATE SKIP LOCKED"
	}
	logger.Debugf("Query: %s", query)
	// Execute query
	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	// Parse results
	var tasks []*Task
	for rows.Next() {
		task := &Task{}
		if err := rows.Scan(task.GetAllTaskValues()...); err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

type LastTaskOptions struct {
	Limit int
}
