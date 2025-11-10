package neo4go

import (
	"context"
	"fmt"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type neo4jStorage struct {
	driver   neo4j.DriverWithContext
	database string
	logger   Logger
}

func newNeo4jStorage(driver neo4j.DriverWithContext, database string, logger Logger) *neo4jStorage {
	return &neo4jStorage{
		driver:   driver,
		database: database,
		logger:   logger,
	}
}

func (s *neo4jStorage) Init(ctx context.Context) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeWrite,
		DatabaseName: s.database,
	})
	defer session.Close(ctx)

	query := `
		CREATE CONSTRAINT schema_migration_version IF NOT EXISTS
		FOR (m:SchemaMigration)
		REQUIRE m.version IS UNIQUE
	`

	_, err := session.Run(ctx, query, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDatabaseConnection, err)
	}

	s.logger.Info("initialized schema migration tracking")
	return nil
}

func (s *neo4jStorage) GetAppliedMigrations(ctx context.Context) ([]MigrationRecord, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeRead,
		DatabaseName: s.database,
	})
	defer session.Close(ctx)

	query := `
		MATCH (m:SchemaMigration)
		RETURN m.version AS version, m.name AS name, m.applied_at AS applied_at, m.checksum AS checksum
		ORDER BY m.version
	`

	result, err := session.Run(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseConnection, err)
	}

	var records []MigrationRecord
	for result.Next(ctx) {
		record := result.Record()

		version, _ := record.Get("version")
		name, _ := record.Get("name")
		appliedAt, _ := record.Get("applied_at")
		checksum, _ := record.Get("checksum")

		records = append(records, MigrationRecord{
			Version:   int(version.(int64)),
			Name:      name.(string),
			AppliedAt: appliedAt.(time.Time),
			Checksum:  checksum.(string),
		})
	}

	if err := result.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseConnection, err)
	}

	return records, nil
}

func (s *neo4jStorage) RecordMigration(ctx context.Context, migration Migration) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeWrite,
		DatabaseName: s.database,
	})
	defer session.Close(ctx)

	query := `
		CREATE (m:SchemaMigration {
			version: $version,
			name: $name,
			applied_at: datetime(),
			checksum: $checksum
		})
	`

	params := map[string]any{
		"version":  migration.Version,
		"name":     migration.Name,
		"checksum": migration.Checksum,
	}

	_, err := session.Run(ctx, query, params)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDatabaseConnection, err)
	}

	s.logger.Info("recorded migration", "version", migration.Version, "name", migration.Name)
	return nil
}

func (s *neo4jStorage) RemoveMigration(ctx context.Context, version int) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeWrite,
		DatabaseName: s.database,
	})
	defer session.Close(ctx)

	query := `
		MATCH (m:SchemaMigration {version: $version})
		DELETE m
	`

	params := map[string]any{
		"version": version,
	}

	_, err := session.Run(ctx, query, params)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDatabaseConnection, err)
	}

	s.logger.Info("removed migration record", "version", version)
	return nil
}

func (s *neo4jStorage) GetCurrentVersion(ctx context.Context) (int, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeRead,
		DatabaseName: s.database,
	})
	defer session.Close(ctx)

	query := `
		MATCH (m:SchemaMigration)
		RETURN m.version AS version
		ORDER BY m.version DESC
		LIMIT 1
	`

	result, err := session.Run(ctx, query, nil)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrDatabaseConnection, err)
	}

	if result.Next(ctx) {
		record := result.Record()
		version, _ := record.Get("version")
		return int(version.(int64)), nil
	}

	return 0, nil
}

func (s *neo4jStorage) Close() error {
	return s.driver.Close(context.Background())
}
