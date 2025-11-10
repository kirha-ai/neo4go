package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"

	"go.kirha.ai/neo4go"
)

//go:embed migrations/*.cypher
var migrationsFS embed.FS

func main() {
	ctx := context.Background()

	uri := getEnv("NEO4J_URI", "bolt://localhost:7687")
	username := getEnv("NEO4J_USERNAME", "neo4j")
	password := getEnv("NEO4J_PASSWORD", "password")
	database := getEnv("NEO4J_DATABASE", "neo4j")

	migrator, err := neo4go.New(neo4go.Config{
		URI:          uri,
		Username:     username,
		Password:     password,
		Database:     database,
		MigrationsFS: migrationsFS,
	})
	if err != nil {
		log.Fatalf("Failed to create migrator: %v", err)
	}
	defer migrator.Close()

	fmt.Println("Running migrations...")
	if err := migrator.Up(ctx); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	version, err := migrator.Version(ctx)
	if err != nil {
		log.Fatalf("Failed to get version: %v", err)
	}

	fmt.Printf("Current migration version: %d\n", version)

	statuses, err := migrator.Status(ctx)
	if err != nil {
		log.Fatalf("Failed to get status: %v", err)
	}

	fmt.Println("\nMigration Status:")
	for _, status := range statuses {
		applied := "No"
		if status.Applied {
			applied = "Yes"
		}
		fmt.Printf("  Version %d (%s): Applied=%s\n", status.Version, status.Name, applied)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
