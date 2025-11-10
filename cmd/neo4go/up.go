package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.kirha.ai/neo4go"
)

func newUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Run all pending migrations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := getConfigFromEnv()
			if err != nil {
				return err
			}

			migrator, err := neo4go.New(cfg)
			if err != nil {
				return fmt.Errorf("failed to create migrator: %w", err)
			}
			defer func() {
				_ = migrator.Close()
			}()

			if err := migrator.Up(cmd.Context()); err != nil {
				return fmt.Errorf("failed to run migrations: %w", err)
			}

			fmt.Println("All migrations applied successfully")
			return nil
		},
	}
}
