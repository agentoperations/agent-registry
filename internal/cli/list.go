package cli

import (
	"fmt"
	"os"

	"github.com/agentoperations/agent-registry/internal/model"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var status, category string

	cmd := &cobra.Command{
		Use:   "list <kind>",
		Short: "List artifacts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, ok := model.ParseKind(args[0])
			if !ok {
				return fmt.Errorf("invalid kind: %s", args[0])
			}

			artifacts, err := apiClient.ListArtifacts(string(kind.Plural()), status, category)
			if err != nil {
				return err
			}

			if len(artifacts) == 0 {
				fmt.Printf("No %s found.\n", kind.Plural())
				return nil
			}

			header := []string{"Name", "Version", "Status", "Category", "Title"}
			var rows [][]string
			for _, a := range artifacts {
				cat := ""
				if a.Metadata != nil {
					cat = a.Metadata.Category
				}
				rows = append(rows, []string{
					a.Identity.Name,
					a.Identity.Version,
					string(a.Status),
					cat,
					truncate(a.Identity.Title, 40),
				})
			}
			table := tablewriter.NewTable(os.Stdout,
				tablewriter.WithHeader(header),
			)
			table.Bulk(rows)
			table.Render()
			return nil
		},
	}

	cmd.Flags().StringVar(&status, "status", "", "Filter by status")
	cmd.Flags().StringVar(&category, "category", "", "Filter by category")
	return cmd
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
