package cli

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"go.kirha.ai/neo4go"
)

func NewCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new migration file",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]

			cfg, err := neo4go.GetConfigFromEnv()
			if err != nil {
				return err
			}

			migrator, err := neo4go.New(cfg)
			if err != nil {
				return err
			}
			defer migrator.Close()

			ctx := context.Background()
			migration, err := migrator.Create(ctx, name)
			if err != nil {
				return err
			}

			filePath := filepath.Join(cfg.MigrationsDir, fmt.Sprintf("%d_%s.cypher", migration.Version, migration.Name))
			fmt.Printf("Created migration: %s\n", filePath)
			return nil
		},
	}
}
