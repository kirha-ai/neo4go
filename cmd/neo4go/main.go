package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := newRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "neo4go",
		Short: "Neo4j schema migration tool",
		Long:  "neo4go is a schema migration tool for Neo4j databases",
	}

	cmd.AddCommand(newUpCmd())
	cmd.AddCommand(newDownCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newUpToCmd())
	cmd.AddCommand(newDownToCmd())

	return cmd
}
