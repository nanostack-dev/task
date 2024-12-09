package task_test

import (
	"context"
	"database/sql"
	_ "github.com/lib/pq"
	"github.com/nanostack-dev/task"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"os"
	"testing"
	"time"
)

var TestDb *sql.DB
var dsn string

func TestMain(m *testing.M) {
	TestDb = SetupTestDB(m)
	code := m.Run()
	TestDb.Close()
	os.Exit(code)
}

// SetupTestDB sets up the test database using testcontainers.
func SetupTestDB(t *testing.M) *sql.DB {
	ctx := context.Background()

	// Create PostgreSQL container

	dbName := "task"
	dbUser := "postgres"
	dbPassword := "password"

	container, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second),
		),
	)
	if err != nil {
		panic("container setup failed")
	}

	// Connect to the database
	dsn, err = container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic("failed to get connection string")
	}

	TestDb, err = sql.Open("postgres", dsn)
	if err != nil {
		panic("failed to open database")
	}

	// Verify the connection
	if err := TestDb.Ping(); err != nil {
		panic("failed to ping database")
	}

	if err := task.InitDB(TestDb); err != nil {
		panic("InitDB failed")
	}

	return TestDb
}
