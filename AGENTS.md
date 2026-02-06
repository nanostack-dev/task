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

## Tools & MCP
- When working with external libraries, **use the Context7 MCP** for accurate usage and API details.

## Git Conventions
- **Commit Messages**: Follow [Conventional Commits](https://www.conventionalcommits.org/):
  - `feat: add priority queue support`
  - `fix: resolve task retry race condition`
  - `perf: optimize worker polling strategy`
- **Branch Naming**: When working on tracked tasks, include ticket number:
  - Format: `<type>/<TICKET-ID>-<description>`
  - Examples: `feat/TASK-5-priority-queue`, `fix/TASK-10-retry-bug`
  - For untracked work: `<type>/<description>` (e.g., `docs/add-examples`)
