package main

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"go.kirha.ai/neo4go"
)

func newDownToCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down-to <version>",
		Short: "Rollback down to a specific version",
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

			if err := migrator.DownTo(cmd.Context(), version); err != nil {
				return fmt.Errorf("failed to rollback to version %d: %w", version, err)
			}

			fmt.Printf("Rolled back to version %d successfully\n", version)
			return nil
		},
	}
}
