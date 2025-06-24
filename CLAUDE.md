# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

VoidexForge is a Nakama-based game backend server that provides comprehensive game services through a plugin architecture. The project is primarily written in Go with TypeScript extensions for Nakama runtime.

## Common Development Commands

### Building the Project

```bash
# Build and run with Docker (production config)
docker compose up --build nakama

# Build and run with Docker (local development)
docker compose -f docker-compose-local.yml up --build nakama

# Build Go plugin directly
go build --trimpath --mod=vendor --buildmode=plugin -o ./backend.so

# Build TypeScript
npx tsc
```

### Running Tests

```bash
# Run all tests with Docker
docker compose -f docker-compose-tests.yml up test

# Run Go tests directly
go test -v -race ./...

# Run specific package tests
go test -v -race ./pamlogix/...

# Run a single test
go test -v -race ./pamlogix -run TestSpecificFunction
```

### Linting

```bash
# Run golangci-lint
golangci-lint run
```

### Managing Dependencies

```bash
# Update Go dependencies
env GO111MODULE=on GOPRIVATE="github.com" go mod vendor

# Install TypeScript dependencies
npm install
```

## Architecture Overview

### Core Structure

The project follows a modular plugin architecture where the main game logic resides in the `pamlogix/` directory. Each game feature is implemented as a separate module with its own initialization, RPC handlers, and data structures.

### Key Directories

- **pamlogix/**: Core game logic implementation
  - Each subdirectory represents a game system (achievements, economy, inventory, etc.)
  - Each system has init functions registered in `pamlogix.go`
  - RPC handlers follow naming convention: `Rpc<SystemName><Action>`

- **configs/**: JSON configuration files for each game system
  - Loaded at server startup
  - Defines game parameters, rewards, requirements, etc.

- **api/**: Bruno API collection for testing all endpoints
  - Organized by feature area
  - Contains example requests for all RPC endpoints

### Database Schema

The project uses CockroachDB (PostgreSQL-compatible) with custom tables for each game system. Tables are created during module initialization using SQL migrations in each module's init function.

### RPC Pattern

All game features expose RPC endpoints following this pattern:
```go
func Rpc<SystemName><Action>(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error)
```

### Configuration Loading

Configurations are loaded from JSON files in the `configs/` directory:
```go
configData := pambase.LoadConfig[ConfigType]("configs/filename.json")
```

### Testing Pattern

Tests use a mock Nakama module and test database:
```go
db := pamtest.NewDB(t)
defer db.Close()

nk := &test_utils.MockNakamaModule{}
// Configure mock expectations

// Run test
```

## Important Patterns

### Error Handling
- Always return proper error messages with context
- Use Nakama's error codes (e.g., `runtime.NewError("message", INVALID_ARGUMENT)`)

### Database Transactions
- Use transactions for operations that modify multiple tables
- Always handle rollback on error

### JSON Marshaling
- Request/response payloads are JSON strings
- Use proper struct tags for JSON marshaling

### User Authentication
- User ID is extracted from context: `ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)`
- Always validate user permissions before operations

## Development Workflow

1. **Adding a New Feature**:
   - Create new directory under `pamlogix/`
   - Implement init function and register in `pamlogix.go`
   - Add RPC handlers following naming convention
   - Create configuration JSON in `configs/`
   - Add tests in same directory
   - Update API documentation in `api/`

2. **Modifying Existing Features**:
   - Check existing tests before making changes
   - Update configuration files if needed
   - Ensure backward compatibility
   - Update API documentation

3. **Database Changes**:
   - Add migrations in module's init function
   - Use `IF NOT EXISTS` for table creation
   - Consider data migration for existing deployments

## Port Configuration

- **7349**: gRPC API port
- **7350**: HTTP API port (REST gateway)
- **7351**: Console port (admin interface)

## Environment Variables

Key environment variables used:
- `DATABASE_URL`: CockroachDB connection string
- `PAMLOGIX_CONFIGS_PATH`: Path to configuration files (default: `/nakama/configs`)

## Deployment

The project uses multi-stage Docker builds and is deployed to AWS ECR. GitHub Actions handle CI/CD with manual version tagging.

Production deployment connects to a remote CockroachDB cluster, while local development uses containerized CockroachDB.