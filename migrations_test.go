package task_test

import (
	"database/sql"
	"github.com/nanostack-dev/task"
	"testing"
)

func TestInitDB(t *testing.T) {

	// Initialize the database
	if err := task.InitDB(TestDb); err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}

	// Verify the table exists
	var tableName string
	err := TestDb.QueryRow(
		"SELECT table_name FROM information_schema." +
			"tables WHERE table_name = 'tasks';",
	).Scan(&tableName)
	if err != nil {
		if err == sql.ErrNoRows {
			t.Fatalf("tasks table was not created")
		}
		t.Fatalf("failed to query tasks table: %v", err)
	}

	if tableName != "tasks" {
		t.Fatalf("expected table name 'tasks', got %v", tableName)
	}
}
