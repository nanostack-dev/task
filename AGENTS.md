# Agent Guide: Task Framework

Lightweight persistent task queue for Go applications.

## Key Features
- Database-backed task processing.
- Priority and retry management.

## Tech Stack
- **Language**: Go
- **Database**: PostgreSQL

## Best Practices
- Ensure handlers are idempotent to support retries.
- Monitor task queue health and worker performance.
- Use transactions when creating tasks alongside business logic.
