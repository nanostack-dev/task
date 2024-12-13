package task

import (
	"database/sql"
	"fmt"
)

const createTasksTableSQL = `
CREATE TABLE IF NOT EXISTS tasks (
	id uuid PRIMARY KEY,
	payload JSONB NOT NULL, 
	name VARCHAR(255) NOT NULL,
	status VARCHAR(50) DEFAULT 'pending',
	priority INTEGER DEFAULT 0,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    scheduled_at TIMESTAMP,
    retry_count INTEGER DEFAULT 0
);
CREATE INDEX IF NOT EXISTS tasks_status_priority_scheduled_at_created_at_idx 
ON tasks(status, priority DESC, scheduled_at, created_at ASC);
`

// InitDB initializes the database and creates the tasks table.
func InitDB(conn *sql.DB) error {
	_, err := conn.Exec(createTasksTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create tasks table: %w", err)
	}
	return nil
}
