package neo4go

import (
	"context"
	"fmt"
	"io/fs"
	"os"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Config struct {
	URI           string
	Username      string
	Password      string
	Database      string
	MigrationsDir string
	MigrationsFS  fs.FS
	Logger        Logger
}

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

	m, err := newMigrator(driver, storage, filesystem, migrationsDir, database, logger)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func validateConfig(cfg Config) error {
	if cfg.URI == "" {
		return fmt.Errorf("%w: URI is required", ErrInvalidConfig)
	}

	if cfg.Username == "" {
		return fmt.Errorf("%w: Username is required", ErrInvalidConfig)
	}

	if cfg.Password == "" {
		return fmt.Errorf("%w: Password is required", ErrInvalidConfig)
	}

	if cfg.MigrationsDir == "" && cfg.MigrationsFS == nil {
		return fmt.Errorf("%w: either MigrationsDir or MigrationsFS must be provided", ErrInvalidConfig)
	}

	return nil
}
