package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.kirha.ai/neo4go"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show migration status",
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

			statuses, err := migrator.Status(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to get status: %w", err)
			}

			fmt.Println("Migration Status:")
			fmt.Println("Version | Name                  | Applied | Applied At")
			fmt.Println("--------|------------------------|---------|-------------------------")

			for _, status := range statuses {
				applied := "No"
				appliedAt := "-"

				if status.Applied {
					applied = "Yes"
					if status.AppliedAt != nil {
						appliedAt = status.AppliedAt.Format("2006-01-02 15:04:05")
					}
				}

				fmt.Printf("%-7d | %-22s | %-7s | %s\n",
					status.Version,
					status.Name,
					applied,
					appliedAt,
				)
			}

			return nil
		},
	}
}
