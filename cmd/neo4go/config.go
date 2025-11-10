package main

import (
	"fmt"
	"os"

	"go.kirha.ai/neo4go"
)

func getConfigFromEnv() (neo4go.Config, error) {
	uri := os.Getenv("NEO4J_URI")
	if uri == "" {
		return neo4go.Config{}, fmt.Errorf("NEO4J_URI environment variable is required")
	}

	username := os.Getenv("NEO4J_USERNAME")
	if username == "" {
		return neo4go.Config{}, fmt.Errorf("NEO4J_USERNAME environment variable is required")
	}

	password := os.Getenv("NEO4J_PASSWORD")
	if password == "" {
		return neo4go.Config{}, fmt.Errorf("NEO4J_PASSWORD environment variable is required")
	}

	database := os.Getenv("NEO4J_DATABASE")
	if database == "" {
		database = "neo4j"
	}

	migrationsDir := os.Getenv("NEO4J_MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "./migrations"
	}

	return neo4go.Config{
		URI:           uri,
		Username:      username,
		Password:      password,
		Database:      database,
		MigrationsDir: migrationsDir,
	}, nil
}
