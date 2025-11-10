package neo4go

import (
	"context"
	"fmt"
	"io/fs"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type migrator struct {
	driver     neo4j.DriverWithContext
	storage    Storage
	parser     *parser
	migrations []Migration
	database   string
	logger     Logger
}

func newMigrator(driver neo4j.DriverWithContext, storage Storage, filesystem fs.FS, migrationsDir string, database string, logger Logger) (*migrator, error) {
	p := newParser(filesystem)
	migrations, err := p.parseMigrations(migrationsDir)
	if err != nil {
		return nil, err
	}

	return &migrator{
		driver:     driver,
		storage:    storage,
		parser:     p,
		migrations: migrations,
		database:   database,
		logger:     logger,
	}, nil
}

func (m *migrator) Up(ctx context.Context) error {
	if err := m.storage.Init(ctx); err != nil {
		return err
	}

	applied, err := m.storage.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	appliedVersions := make(map[int]bool)
	for _, record := range applied {
		appliedVersions[record.Version] = true
	}

	for _, migration := range m.migrations {
		if appliedVersions[migration.Version] {
			m.logger.Debug("skipping already applied migration", "version", migration.Version, "name", migration.Name)
			continue
		}

		m.logger.Info("applying migration", "version", migration.Version, "name", migration.Name)

		if err := m.executeMigration(ctx, migration.UpSQL); err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
		}

		if err := m.storage.RecordMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		m.logger.Info("successfully applied migration", "version", migration.Version, "name", migration.Name)
	}

	return nil
}

func (m *migrator) Down(ctx context.Context) error {
	if err := m.storage.Init(ctx); err != nil {
		return err
	}

	currentVersion, err := m.storage.GetCurrentVersion(ctx)
	if err != nil {
		return err
	}

	if currentVersion == 0 {
		m.logger.Info("no migrations to rollback")
		return nil
	}

	var targetMigration *Migration
	for _, migration := range m.migrations {
		if migration.Version == currentVersion {
			targetMigration = &migration
			break
		}
	}

	if targetMigration == nil {
		return fmt.Errorf("%w: version %d", ErrMigrationNotFound, currentVersion)
	}

	m.logger.Info("rolling back migration", "version", targetMigration.Version, "name", targetMigration.Name)

	if err := m.executeMigration(ctx, targetMigration.DownSQL); err != nil {
		return fmt.Errorf("failed to rollback migration %d: %w", targetMigration.Version, err)
	}

	if err := m.storage.RemoveMigration(ctx, targetMigration.Version); err != nil {
		return fmt.Errorf("failed to remove migration record %d: %w", targetMigration.Version, err)
	}

	m.logger.Info("successfully rolled back migration", "version", targetMigration.Version, "name", targetMigration.Name)
	return nil
}

func (m *migrator) UpTo(ctx context.Context, targetVersion int) error {
	if err := m.storage.Init(ctx); err != nil {
		return err
	}

	if targetVersion < 0 {
		return ErrInvalidVersion
	}

	applied, err := m.storage.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	appliedVersions := make(map[int]bool)
	for _, record := range applied {
		appliedVersions[record.Version] = true
	}

	for _, migration := range m.migrations {
		if migration.Version > targetVersion {
			break
		}

		if appliedVersions[migration.Version] {
			m.logger.Debug("skipping already applied migration", "version", migration.Version, "name", migration.Name)
			continue
		}

		m.logger.Info("applying migration", "version", migration.Version, "name", migration.Name)

		if err := m.executeMigration(ctx, migration.UpSQL); err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
		}

		if err := m.storage.RecordMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		m.logger.Info("successfully applied migration", "version", migration.Version, "name", migration.Name)
	}

	return nil
}

func (m *migrator) DownTo(ctx context.Context, targetVersion int) error {
	if err := m.storage.Init(ctx); err != nil {
		return err
	}

	if targetVersion < 0 {
		return ErrInvalidVersion
	}

	applied, err := m.storage.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	for i := len(applied) - 1; i >= 0; i-- {
		record := applied[i]

		if record.Version <= targetVersion {
			break
		}

		var targetMigration *Migration
		for _, migration := range m.migrations {
			if migration.Version == record.Version {
				targetMigration = &migration
				break
			}
		}

		if targetMigration == nil {
			return fmt.Errorf("%w: version %d", ErrMigrationNotFound, record.Version)
		}

		m.logger.Info("rolling back migration", "version", targetMigration.Version, "name", targetMigration.Name)

		if err := m.executeMigration(ctx, targetMigration.DownSQL); err != nil {
			return fmt.Errorf("failed to rollback migration %d: %w", targetMigration.Version, err)
		}

		if err := m.storage.RemoveMigration(ctx, targetMigration.Version); err != nil {
			return fmt.Errorf("failed to remove migration record %d: %w", targetMigration.Version, err)
		}

		m.logger.Info("successfully rolled back migration", "version", targetMigration.Version, "name", targetMigration.Name)
	}

	return nil
}

func (m *migrator) Status(ctx context.Context) ([]MigrationStatus, error) {
	if err := m.storage.Init(ctx); err != nil {
		return nil, err
	}

	applied, err := m.storage.GetAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	appliedMap := make(map[int]MigrationRecord)
	for _, record := range applied {
		appliedMap[record.Version] = record
	}

	var statuses []MigrationStatus
	for _, migration := range m.migrations {
		status := MigrationStatus{
			Version:  migration.Version,
			Name:     migration.Name,
			Applied:  false,
			Checksum: migration.Checksum,
		}

		if record, exists := appliedMap[migration.Version]; exists {
			status.Applied = true
			appliedAt := record.AppliedAt
			status.AppliedAt = &appliedAt

			if record.Checksum != migration.Checksum {
				m.logger.Warn("checksum mismatch", "version", migration.Version, "name", migration.Name)
			}
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}

func (m *migrator) Version(ctx context.Context) (int, error) {
	if err := m.storage.Init(ctx); err != nil {
		return 0, err
	}

	return m.storage.GetCurrentVersion(ctx)
}

func (m *migrator) Close() error {
	return m.storage.Close()
}

func (m *migrator) executeMigration(ctx context.Context, sql string) error {
	if m.driver == nil {
		return nil
	}

	session := m.driver.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeWrite,
		DatabaseName: m.database,
	})
	defer session.Close(ctx)

	statements := m.splitStatements(sql)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}

			m.logger.Debug("executing statement", "statement", stmt)

			_, err := tx.Run(ctx, stmt, nil)
			if err != nil {
				return nil, fmt.Errorf("%w: %v", ErrTransactionFailed, err)
			}
		}
		return nil, nil
	})

	return err
}

func (m *migrator) splitStatements(sql string) []string {
	return strings.Split(sql, ";")
}
