package neo4go

import (
	"context"
	"fmt"
	"os"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func New(cfg Config) (Migrator, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	driver, err := neo4j.NewDriverWithContext(
		cfg.URI,
		neo4j.BasicAuth(cfg.Username, cfg.Password, ""),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabaseConnection, err)
	}

	if err := driver.VerifyConnectivity(context.Background()); err != nil {
		_ = driver.Close(context.Background())
		return nil, fmt.Errorf("%w: %v", ErrDatabaseConnection, err)
	}

	return NewWithDriver(driver, cfg)
}

func NewWithDriver(driver neo4j.DriverWithContext, cfg Config) (Migrator, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	logger := cfg.Logger
	if logger == nil {
		logger = newDefaultLogger()
	}

	filesystem := cfg.MigrationsFS
	isEmbedded := cfg.MigrationsFS != nil
	actualMigrationsDir := cfg.MigrationsDir
	if filesystem == nil {
		filesystem = os.DirFS(cfg.MigrationsDir)
	}

	migrationsDir := "."
	if cfg.MigrationsFS == nil && cfg.MigrationsDir != "" {
		migrationsDir = "."
	}

	database := cfg.Database
	if database == "" {
		database = "neo4j"
	}

	storage := newNeo4jStorage(driver, database, logger)

	m, err := newMigrator(driver, storage, filesystem, migrationsDir, database, logger, actualMigrationsDir, isEmbedded)
	if err != nil {
		return nil, err
	}

	return m, nil
}
