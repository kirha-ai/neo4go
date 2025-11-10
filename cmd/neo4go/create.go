package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new migration file",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]

			migrationsDir := os.Getenv("NEO4J_MIGRATIONS_DIR")
			if migrationsDir == "" {
				migrationsDir = "./migrations"
			}

			if err := os.MkdirAll(migrationsDir, 0750); err != nil {
				return fmt.Errorf("failed to create migrations directory: %w", err)
			}

			version := time.Now().Unix()
			filename := fmt.Sprintf("%d_%s.cypher", version, name)
			filePath := filepath.Join(migrationsDir, filename)

			content := `-- +neo4go Up
-- Add your up migration statements here


-- +neo4go Down
-- Add your down migration statements here

`

			if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
				return fmt.Errorf("failed to create migration file: %w", err)
			}

			fmt.Printf("Created migration: %s\n", filePath)
			return nil
		},
	}
}
