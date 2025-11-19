package cli

import "github.com/spf13/cobra"

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "neo4go",
		Short: "Neo4j schema migration tool",
		Long:  "neo4go is a schema migration tool for Neo4j databases",
	}

	cmd.AddCommand(NewUpCmd())
	cmd.AddCommand(NewDownCmd())
	cmd.AddCommand(NewStatusCmd())
	cmd.AddCommand(NewVersionCmd())
	cmd.AddCommand(NewCreateCmd())
	cmd.AddCommand(NewUpToCmd())
	cmd.AddCommand(NewDownToCmd())

	return cmd
}
