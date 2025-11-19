package neo4go

import (
	"fmt"
	"io/fs"
	"os"
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

func GetConfigFromEnv() (Config, error) {
	uri := os.Getenv("NEO4J_URI")
	if uri == "" {
		return Config{}, fmt.Errorf("NEO4J_URI environment variable is required")
	}

	username := os.Getenv("NEO4J_USERNAME")
	if username == "" {
		return Config{}, fmt.Errorf("NEO4J_USERNAME environment variable is required")
	}

	password := os.Getenv("NEO4J_PASSWORD")
	if password == "" {
		return Config{}, fmt.Errorf("NEO4J_PASSWORD environment variable is required")
	}

	database := os.Getenv("NEO4J_DATABASE")
	if database == "" {
		database = "neo4j"
	}

	migrationsDir := os.Getenv("NEO4J_MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "./migrations"
	}

	return Config{
		URI:           uri,
		Username:      username,
		Password:      password,
		Database:      database,
		MigrationsDir: migrationsDir,
	}, nil
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
