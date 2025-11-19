package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.kirha.ai/neo4go"
)

func NewDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Rollback the last migration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := neo4go.GetConfigFromEnv()
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

			if err := migrator.Down(cmd.Context()); err != nil {
				return fmt.Errorf("failed to rollback migration: %w", err)
			}

			fmt.Println("Migration rolled back successfully")
			return nil
		},
	}
}
