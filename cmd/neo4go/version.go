package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.kirha.ai/neo4go"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show current migration version",
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

			version, err := migrator.Version(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to get version: %w", err)
			}

			if version == 0 {
				fmt.Println("No migrations applied yet")
			} else {
				fmt.Printf("Current version: %d\n", version)
			}

			return nil
		},
	}
}
