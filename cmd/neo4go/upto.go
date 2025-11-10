package main

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"go.kirha.ai/neo4go"
)

func newUpToCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up-to <version>",
		Short: "Migrate up to a specific version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			version, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid version number: %w", err)
			}

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

			if err := migrator.UpTo(cmd.Context(), version); err != nil {
				return fmt.Errorf("failed to migrate to version %d: %w", version, err)
			}

			fmt.Printf("Migrated to version %d successfully\n", version)
			return nil
		},
	}
}
