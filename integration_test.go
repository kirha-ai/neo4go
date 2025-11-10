//go:build integration
// +build integration

package neo4go

import (
	"context"
	"os"
	"testing"
	"testing/fstest"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func getTestConfig() Config {
	return Config{
		URI:           getEnv("NEO4J_URI", "bolt://localhost:7687"),
		Username:      getEnv("NEO4J_USERNAME", "neo4j"),
		Password:      getEnv("NEO4J_PASSWORD", "testpassword"),
		Database:      getEnv("NEO4J_DATABASE", "neo4j"),
		MigrationsDir: "./test_migrations",
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func cleanupDatabase(t *testing.T, cfg Config) {
	t.Helper()

	driver, err := neo4j.NewDriverWithContext(
		cfg.URI,
		neo4j.BasicAuth(cfg.Username, cfg.Password, ""),
	)
	if err != nil {
		t.Fatalf("failed to create driver for cleanup: %v", err)
	}
	defer driver.Close(context.Background())

	ctx := context.Background()
	session := driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeWrite,
		DatabaseName: cfg.Database,
	})
	defer session.Close(ctx)

	_, _ = session.Run(ctx, "MATCH (m:SchemaMigration) DELETE m", nil)
	_, _ = session.Run(ctx, "MATCH (n) DETACH DELETE n", nil)
}

func verifyVersion(t *testing.T, ctx context.Context, migrator Migrator, expected int) {
	t.Helper()
	version, err := migrator.Version(ctx)
	if err != nil {
		t.Fatalf("failed to get version: %v", err)
	}
	if version != expected {
		t.Errorf("expected version %d, got %d", expected, version)
	}
}

func verifyMigrationStatuses(t *testing.T, ctx context.Context, migrator Migrator, expectedCount int) {
	t.Helper()
	statuses, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}

	if len(statuses) != expectedCount {
		t.Fatalf("expected %d migrations, got %d", expectedCount, len(statuses))
	}

	for _, status := range statuses {
		if !status.Applied {
			t.Errorf("migration %d should be applied", status.Version)
		}
		if status.AppliedAt == nil {
			t.Errorf("migration %d should have applied_at timestamp", status.Version)
		}
	}
}

func verifyNeo4jConstraints(t *testing.T, ctx context.Context, cfg Config, minConstraints int) {
	t.Helper()
	driver, err := neo4j.NewDriverWithContext(
		cfg.URI,
		neo4j.BasicAuth(cfg.Username, cfg.Password, ""),
	)
	if err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}
	defer driver.Close(ctx)

	session := driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeRead,
		DatabaseName: cfg.Database,
	})
	defer session.Close(ctx)

	result, err := session.Run(ctx, "SHOW CONSTRAINTS", nil)
	if err != nil {
		t.Fatalf("failed to query constraints: %v", err)
	}

	constraintCount := 0
	for result.Next(ctx) {
		constraintCount++
	}

	if constraintCount < minConstraints {
		t.Errorf("expected at least %d constraints, got %d", minConstraints, constraintCount)
	}
}

func TestIntegrationFullMigrationCycle(t *testing.T) {
	cfg := getTestConfig()
	cfg.MigrationsFS = fstest.MapFS{
		"001_create_users.cypher": &fstest.MapFile{
			Data: []byte(`-- +neo4go Up
CREATE CONSTRAINT user_id_unique IF NOT EXISTS FOR (u:User) REQUIRE u.id IS UNIQUE;
CREATE INDEX user_email_idx IF NOT EXISTS FOR (u:User) ON (u.email);

-- +neo4go Down
DROP INDEX user_email_idx IF EXISTS;
DROP CONSTRAINT user_id_unique IF EXISTS;`),
		},
		"002_create_posts.cypher": &fstest.MapFile{
			Data: []byte(`-- +neo4go Up
CREATE CONSTRAINT post_id_unique IF NOT EXISTS FOR (p:Post) REQUIRE p.id IS UNIQUE;
CREATE INDEX post_created_at_idx IF NOT EXISTS FOR (p:Post) ON (p.created_at);

-- +neo4go Down
DROP INDEX post_created_at_idx IF EXISTS;
DROP CONSTRAINT post_id_unique IF EXISTS;`),
		},
	}
	cfg.MigrationsDir = ""

	cleanupDatabase(t, cfg)
	defer cleanupDatabase(t, cfg)

	migrator, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create migrator: %v", err)
	}
	defer migrator.Close()

	ctx := context.Background()

	t.Run("initial version should be 0", func(t *testing.T) {
		verifyVersion(t, ctx, migrator, 0)
	})

	t.Run("apply all migrations", func(t *testing.T) {
		if err := migrator.Up(ctx); err != nil {
			t.Fatalf("failed to run migrations: %v", err)
		}
		verifyVersion(t, ctx, migrator, 2)
	})

	t.Run("verify migrations are recorded", func(t *testing.T) {
		verifyMigrationStatuses(t, ctx, migrator, 2)
	})

	t.Run("verify constraints exist in Neo4j", func(t *testing.T) {
		verifyNeo4jConstraints(t, ctx, cfg, 2)
	})

	t.Run("rollback last migration", func(t *testing.T) {
		if err := migrator.Down(ctx); err != nil {
			t.Fatalf("failed to rollback: %v", err)
		}
		verifyVersion(t, ctx, migrator, 1)
	})

	t.Run("re-apply rolled back migration", func(t *testing.T) {
		if err := migrator.Up(ctx); err != nil {
			t.Fatalf("failed to re-apply migration: %v", err)
		}
		verifyVersion(t, ctx, migrator, 2)
	})
}

func TestIntegrationUpTo(t *testing.T) {
	cfg := getTestConfig()
	cfg.MigrationsFS = fstest.MapFS{
		"001_first.cypher": &fstest.MapFile{
			Data: []byte(`-- +neo4go Up
CREATE CONSTRAINT c1 IF NOT EXISTS FOR (n:TestNode1) REQUIRE n.id IS UNIQUE;

-- +neo4go Down
DROP CONSTRAINT c1 IF EXISTS;`),
		},
		"002_second.cypher": &fstest.MapFile{
			Data: []byte(`-- +neo4go Up
CREATE CONSTRAINT c2 IF NOT EXISTS FOR (n:TestNode2) REQUIRE n.id IS UNIQUE;

-- +neo4go Down
DROP CONSTRAINT c2 IF EXISTS;`),
		},
		"003_third.cypher": &fstest.MapFile{
			Data: []byte(`-- +neo4go Up
CREATE CONSTRAINT c3 IF NOT EXISTS FOR (n:TestNode3) REQUIRE n.id IS UNIQUE;

-- +neo4go Down
DROP CONSTRAINT c3 IF EXISTS;`),
		},
	}
	cfg.MigrationsDir = ""

	cleanupDatabase(t, cfg)
	defer cleanupDatabase(t, cfg)

	migrator, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create migrator: %v", err)
	}
	defer migrator.Close()

	ctx := context.Background()

	if err := migrator.UpTo(ctx, 2); err != nil {
		t.Fatalf("failed to migrate to version 2: %v", err)
	}

	version, err := migrator.Version(ctx)
	if err != nil {
		t.Fatalf("failed to get version: %v", err)
	}

	if version != 2 {
		t.Errorf("expected version 2, got %d", version)
	}

	statuses, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}

	if statuses[0].Applied && statuses[1].Applied && !statuses[2].Applied {
		return
	}
	t.Error("expected migrations 1 and 2 to be applied, but not 3")
}

func TestIntegrationDownTo(t *testing.T) {
	cfg := getTestConfig()
	cfg.MigrationsFS = fstest.MapFS{
		"001_first.cypher": &fstest.MapFile{
			Data: []byte(`-- +neo4go Up
CREATE CONSTRAINT c1 IF NOT EXISTS FOR (n:TestNode1) REQUIRE n.id IS UNIQUE;

-- +neo4go Down
DROP CONSTRAINT c1 IF EXISTS;`),
		},
		"002_second.cypher": &fstest.MapFile{
			Data: []byte(`-- +neo4go Up
CREATE CONSTRAINT c2 IF NOT EXISTS FOR (n:TestNode2) REQUIRE n.id IS UNIQUE;

-- +neo4go Down
DROP CONSTRAINT c2 IF EXISTS;`),
		},
		"003_third.cypher": &fstest.MapFile{
			Data: []byte(`-- +neo4go Up
CREATE CONSTRAINT c3 IF NOT EXISTS FOR (n:TestNode3) REQUIRE n.id IS UNIQUE;

-- +neo4go Down
DROP CONSTRAINT c3 IF EXISTS;`),
		},
	}
	cfg.MigrationsDir = ""

	cleanupDatabase(t, cfg)
	defer cleanupDatabase(t, cfg)

	migrator, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create migrator: %v", err)
	}
	defer migrator.Close()

	ctx := context.Background()

	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("failed to run all migrations: %v", err)
	}

	if err := migrator.DownTo(ctx, 1); err != nil {
		t.Fatalf("failed to rollback to version 1: %v", err)
	}

	version, err := migrator.Version(ctx)
	if err != nil {
		t.Fatalf("failed to get version: %v", err)
	}

	if version != 1 {
		t.Errorf("expected version 1, got %d", version)
	}

	statuses, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}

	if statuses[0].Applied && !statuses[1].Applied && !statuses[2].Applied {
		return
	}
	t.Error("expected only migration 1 to be applied")
}

func TestIntegrationChecksumVerification(t *testing.T) {
	cfg := getTestConfig()
	cfg.MigrationsFS = fstest.MapFS{
		"001_test.cypher": &fstest.MapFile{
			Data: []byte(`-- +neo4go Up
CREATE CONSTRAINT test_c IF NOT EXISTS FOR (n:Test) REQUIRE n.id IS UNIQUE;

-- +neo4go Down
DROP CONSTRAINT test_c IF EXISTS;`),
		},
	}
	cfg.MigrationsDir = ""

	cleanupDatabase(t, cfg)
	defer cleanupDatabase(t, cfg)

	migrator, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create migrator: %v", err)
	}

	ctx := context.Background()

	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("failed to run migration: %v", err)
	}

	statuses, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}

	originalChecksum := statuses[0].Checksum
	migrator.Close()

	cfg.MigrationsFS = fstest.MapFS{
		"001_test.cypher": &fstest.MapFile{
			Data: []byte(`-- +neo4go Up
CREATE CONSTRAINT test_c_modified IF NOT EXISTS FOR (n:Test) REQUIRE n.id IS UNIQUE;

-- +neo4go Down
DROP CONSTRAINT test_c_modified IF EXISTS;`),
		},
	}

	migrator2, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create second migrator: %v", err)
	}
	defer migrator2.Close()

	statuses2, err := migrator2.Status(ctx)
	if err != nil {
		t.Fatalf("failed to get status from second migrator: %v", err)
	}

	newChecksum := statuses2[0].Checksum

	if originalChecksum == newChecksum {
		t.Error("checksums should differ when migration content changes")
	}
}

func TestIntegrationIdempotency(t *testing.T) {
	cfg := getTestConfig()
	cfg.MigrationsFS = fstest.MapFS{
		"001_test.cypher": &fstest.MapFile{
			Data: []byte(`-- +neo4go Up
CREATE CONSTRAINT idem_c IF NOT EXISTS FOR (n:Idem) REQUIRE n.id IS UNIQUE;

-- +neo4go Down
DROP CONSTRAINT idem_c IF EXISTS;`),
		},
	}
	cfg.MigrationsDir = ""

	cleanupDatabase(t, cfg)
	defer cleanupDatabase(t, cfg)

	migrator, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create migrator: %v", err)
	}
	defer migrator.Close()

	ctx := context.Background()

	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("first Up() failed: %v", err)
	}

	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("second Up() failed (should be idempotent): %v", err)
	}

	version, err := migrator.Version(ctx)
	if err != nil {
		t.Fatalf("failed to get version: %v", err)
	}

	if version != 1 {
		t.Errorf("expected version 1, got %d", version)
	}
}

func TestIntegrationTransactionRollback(t *testing.T) {
	cfg := getTestConfig()
	cfg.MigrationsFS = fstest.MapFS{
		"001_valid.cypher": &fstest.MapFile{
			Data: []byte(`-- +neo4go Up
CREATE CONSTRAINT valid_c IF NOT EXISTS FOR (n:Valid) REQUIRE n.id IS UNIQUE;

-- +neo4go Down
DROP CONSTRAINT valid_c IF EXISTS;`),
		},
		"002_invalid.cypher": &fstest.MapFile{
			Data: []byte(`-- +neo4go Up
CREATE CONSTRAINT invalid_c IF NOT EXISTS FOR (n:Invalid) REQUIRE n.id IS UNIQUE;
THIS IS INVALID CYPHER THAT WILL FAIL;

-- +neo4go Down
DROP CONSTRAINT invalid_c IF EXISTS;`),
		},
	}
	cfg.MigrationsDir = ""

	cleanupDatabase(t, cfg)
	defer cleanupDatabase(t, cfg)

	migrator, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create migrator: %v", err)
	}
	defer migrator.Close()

	ctx := context.Background()

	err = migrator.Up(ctx)
	if err == nil {
		t.Fatal("expected error from invalid migration, got nil")
	}

	version, err := migrator.Version(ctx)
	if err != nil {
		t.Fatalf("failed to get version: %v", err)
	}

	if version != 1 {
		t.Errorf("expected version 1 (only valid migration applied), got %d", version)
	}
}

func TestIntegrationConcurrentMigrations(t *testing.T) {
	cfg := getTestConfig()
	cfg.MigrationsFS = fstest.MapFS{
		"001_test.cypher": &fstest.MapFile{
			Data: []byte(`-- +neo4go Up
CREATE CONSTRAINT conc_c IF NOT EXISTS FOR (n:Conc) REQUIRE n.id IS UNIQUE;

-- +neo4go Down
DROP CONSTRAINT conc_c IF EXISTS;`),
		},
	}
	cfg.MigrationsDir = ""

	cleanupDatabase(t, cfg)
	defer cleanupDatabase(t, cfg)

	ctx := context.Background()

	results := make(chan error, 3)

	for i := 0; i < 3; i++ {
		go func() {
			migrator, err := New(cfg)
			if err != nil {
				results <- err
				return
			}
			defer migrator.Close()

			time.Sleep(time.Millisecond * 50)
			results <- migrator.Up(ctx)
		}()
	}

	successCount := 0
	for i := 0; i < 3; i++ {
		err := <-results
		if err == nil {
			successCount++
		}
	}

	if successCount == 0 {
		t.Fatal("at least one migration should succeed")
	}

	migrator, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create verification migrator: %v", err)
	}
	defer migrator.Close()

	version, err := migrator.Version(ctx)
	if err != nil {
		t.Fatalf("failed to get final version: %v", err)
	}

	if version != 1 {
		t.Errorf("expected final version 1, got %d", version)
	}
}
