# neo4go

[![CI](https://github.com/kirha-ai/neo4go/actions/workflows/ci.yml/badge.svg)](https://github.com/kirha-ai/neo4go/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/go.kirha.ai/neo4go)](https://goreportcard.com/report/go.kirha.ai/neo4go)
[![GoDoc](https://godoc.org/go.kirha.ai/neo4go?status.svg)](https://godoc.org/go.kirha.ai/neo4go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A Neo4j schema migration tool for Go, inspired by [pressly/goose](https://github.com/pressly/goose) 

## Features

- 🚀 Simple and intuitive API
- 📦 Embedded migrations support with `embed.FS`
- 🔄 Up/Down migrations with version tracking
- 🔒 Transaction safety for individual migrations
- ✅ Migration checksum verification
- 📝 Structured logging support
- 🛠️ CLI tool for running migrations
- 🎯 Designed specifically for Neo4j

## Installation

### As a Library

```bash
go get go.kirha.ai/neo4go
```

### CLI Tool

```bash
go install go.kirha.ai/neo4go/cmd/neo4go@latest
```

Or download pre-built binaries from the [releases page](https://github.com/kirha-ai/neo4go/releases).

## Quick Start

### 1. Create Migration Files

Create a migrations directory and add your first migration:

**`migrations/001_initial.cypher`**

```cypher
-- +neo4go Up
CREATE CONSTRAINT user_id_unique IF NOT EXISTS
FOR (u:User) REQUIRE u.id IS UNIQUE;

CREATE INDEX user_email_idx IF NOT EXISTS
FOR (u:User) ON (u.email);

-- +neo4go Down
DROP CONSTRAINT user_id_unique IF EXISTS;
DROP INDEX user_email_idx IF EXISTS;
```

### 2. Programmatic Usage

```go
package main

import (
    "context"
    "embed"
    "log"

    "go.kirha.ai/neo4go"
)

//go:embed migrations/*.cypher
var migrationsFS embed.FS

func main() {
    ctx := context.Background()

    migrator, err := neo4go.New(neo4go.Config{
        URI:          "bolt://localhost:7687",
        Username:     "neo4j",
        Password:     "password",
        Database:     "neo4j",
        MigrationsFS: migrationsFS,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer migrator.Close()

    if err := migrator.Up(ctx); err != nil {
        log.Fatal(err)
    }

    log.Println("Migrations applied successfully!")
}
```

### 3. CLI Usage

Set environment variables:

```bash
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_USERNAME="neo4j"
export NEO4J_PASSWORD="password"
export NEO4J_DATABASE="neo4j"
export NEO4J_MIGRATIONS_DIR="./migrations"
```

Run migrations:

```bash
# Apply all pending migrations
neo4go up

# Rollback the last migration
neo4go down

# Show migration status
neo4go status

# Show current version
neo4go version

# Migrate to a specific version
neo4go up-to 5

# Rollback to a specific version
neo4go down-to 3

# Create a new migration file
neo4go create add_user_indexes
```

## Migration File Format

Migration files must follow the naming convention: `{version}_{name}.cypher`

Each migration file contains two sections:

```cypher
-- +neo4go Up
-- Your "up" migration statements here
CREATE CONSTRAINT ...;
CREATE INDEX ...;

-- +neo4go Down
-- Your "down" migration statements here
DROP CONSTRAINT ...;
DROP INDEX ...;
```

### Best Practices

1. **Use IF EXISTS/IF NOT EXISTS**: Always use these clauses to make migrations idempotent
2. **One concern per migration**: Keep migrations focused on a single change
3. **Test rollbacks**: Always test that your down migrations work correctly
4. **Sequential versioning**: Use simple incrementing numbers (001, 002, 003) or timestamps
5. **Descriptive names**: Use clear, descriptive names for your migrations

## Configuration

### Config Struct

```go
type Config struct {
    URI           string    // Neo4j connection URI (required)
    Username      string    // Neo4j username (required)
    Password      string    // Neo4j password (required)
    Database      string    // Database name (default: "neo4j")
    MigrationsDir string    // Directory containing migrations (mutually exclusive with MigrationsFS)
    MigrationsFS  fs.FS     // Embedded filesystem (mutually exclusive with MigrationsDir)
    Logger        Logger    // Custom logger implementation (optional)
}
```

### Environment Variables (CLI)

- `NEO4J_URI` - Connection URI (required)
- `NEO4J_USERNAME` - Username (required)
- `NEO4J_PASSWORD` - Password (required)
- `NEO4J_DATABASE` - Database name (default: "neo4j")
- `NEO4J_MIGRATIONS_DIR` - Migrations directory (default: "./migrations")

## API Reference

### Migrator Interface

```go
type Migrator interface {
    Up(ctx context.Context) error
    Down(ctx context.Context) error
    UpTo(ctx context.Context, version int) error
    DownTo(ctx context.Context, version int) error
    Status(ctx context.Context) ([]MigrationStatus, error)
    Version(ctx context.Context) (int, error)
    Close() error
}
```

### Methods

#### Up

Applies all pending migrations in order.

```go
err := migrator.Up(ctx)
```

#### Down

Rolls back the most recent migration.

```go
err := migrator.Down(ctx)
```

#### UpTo

Migrates up to a specific version.

```go
err := migrator.UpTo(ctx, 5)
```

#### DownTo

Rolls back down to a specific version.

```go
err := migrator.DownTo(ctx, 3)
```

#### Status

Returns the status of all migrations.

```go
statuses, err := migrator.Status(ctx)
for _, status := range statuses {
    fmt.Printf("Version %d: Applied=%v\n", status.Version, status.Applied)
}
```

#### Version

Returns the current migration version.

```go
version, err := migrator.Version(ctx)
```

## Custom Logger

Implement the `Logger` interface to use your own logging solution:

```go
type Logger interface {
    Debug(msg string, args ...any)
    Info(msg string, args ...any)
    Warn(msg string, args ...any)
    Error(msg string, args ...any)
}
```

Example with [slog](https://pkg.go.dev/log/slog):

```go
type slogAdapter struct {
    logger *slog.Logger
}

func (s *slogAdapter) Debug(msg string, args ...any) {
    s.logger.Debug(msg, args...)
}

func (s *slogAdapter) Info(msg string, args ...any) {
    s.logger.Info(msg, args...)
}

func (s *slogAdapter) Warn(msg string, args ...any) {
    s.logger.Warn(msg, args...)
}

func (s *slogAdapter) Error(msg string, args ...any) {
    s.logger.Error(msg, args...)
}

// Usage
migrator, err := neo4go.New(neo4go.Config{
    // ... other config
    Logger: &slogAdapter{logger: slog.Default()},
})
```

## Version Tracking

neo4go tracks applied migrations in your Neo4j database using `:SchemaMigration` nodes:

```cypher
(:SchemaMigration {
    version: 1,
    name: "initial",
    applied_at: datetime(),
    checksum: "abc123..."
})
```

A unique constraint on `version` ensures no duplicate migrations are applied.

## Error Handling

neo4go provides descriptive error types:

- `ErrNoMigrations` - No migration files found
- `ErrInvalidVersion` - Invalid version number
- `ErrMigrationNotFound` - Migration file not found
- `ErrChecksumMismatch` - Migration file has been modified
- `ErrInvalidMigrationFile` - Invalid migration file format
- `ErrDatabaseConnection` - Database connection error
- `ErrTransactionFailed` - Migration transaction failed

Use `errors.Is()` to check for specific errors:

```go
if errors.Is(err, neo4go.ErrNoMigrations) {
    // Handle no migrations case
}
```

## Transaction Safety

Each migration runs within a Neo4j transaction. If any statement in a migration fails, the entire migration is rolled back, and the migration is not recorded as applied.

## Examples

See the [examples](./examples) directory for:

- Embedded migrations example
- CLI usage example
- Custom logger example

## Development

### Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose (for integration tests)

### Running Tests

```bash
# Unit tests only (fast, no dependencies)
make test

# Integration tests with Docker (starts Neo4j automatically)
make test-integration-local

# Integration tests (requires Neo4j already running)
make test-integration

See [TESTING.md](./TESTING.md) for detailed testing guide.

### Docker Management

```bash
# Start Neo4j in Docker
make docker-up

# Stop Neo4j
make docker-down
```

### Building

```bash
# Build CLI
make build

# Install CLI locally
make install
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by [pressly/goose](https://github.com/pressly/goose)
- Built with [neo4j-go-driver](https://github.com/neo4j/neo4j-go-driver)

## Support

- Documentation: [GoDoc](https://godoc.org/go.kirha.ai/neo4go)
- Issues: [GitHub Issues](https://github.com/kirha-ai/neo4go/issues)
