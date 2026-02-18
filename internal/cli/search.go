package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	var kind string

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search artifacts across all kinds",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")

			artifacts, err := apiClient.Search(query, kind)
			if err != nil {
				return err
			}

			if len(artifacts) == 0 {
				fmt.Printf("No results for: %s\n", query)
				return nil
			}

			header := []string{"Kind", "Name", "Version", "Status", "Title"}
			var rows [][]string
			for _, a := range artifacts {
				rows = append(rows, []string{
					string(a.Kind),
					a.Identity.Name,
					a.Identity.Version,
					string(a.Status),
					truncate(a.Identity.Title, 40),
				})
			}
			table := tablewriter.NewTable(os.Stdout, tablewriter.WithHeader(header))
			table.Bulk(rows)
			table.Render()
			return nil
		},
	}

	cmd.Flags().StringVar(&kind, "kind", "", "Filter by kind (agent, skill, mcp-server)")
	return cmd
}
