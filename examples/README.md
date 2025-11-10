# Neo4go Examples

This directory contains examples demonstrating different ways to use neo4go.

## Embedded Example

The `embedded` directory shows how to use embedded migrations with `embed.FS`.

### Running the embedded example:

```bash
cd examples/embedded
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_USERNAME="neo4j"
export NEO4J_PASSWORD="password"
go run main.go
```

## CLI Example

The `cli` directory contains example migration files that can be used with the neo4go CLI.

### Running migrations with the CLI:

```bash
# Set environment variables
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_USERNAME="neo4j"
export NEO4J_PASSWORD="password"
export NEO4J_MIGRATIONS_DIR="./examples/cli/migrations"

# Run migrations
neo4go up

# Check status
neo4go status

# Get current version
neo4go version

# Rollback last migration
neo4go down
```

## Migration File Format

Migration files follow this format:

```cypher
-- +neo4go Up
CREATE CONSTRAINT user_id_unique IF NOT EXISTS
FOR (u:User) REQUIRE u.id IS UNIQUE;

-- +neo4go Down
DROP CONSTRAINT user_id_unique IF EXISTS;
```

### Naming Convention

Migration files must follow the pattern: `{version}_{name}.cypher`

Examples:
- `001_initial.cypher`
- `002_add_indexes.cypher`
- `003_add_relationships.cypher`

The version number is used for ordering and must be unique.
