package cli

import (
	"fmt"

	"github.com/agentregistry/agent-registry/internal/model"
	"github.com/spf13/cobra"
)

func newInspectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "inspect <kind> <name> <version>",
		Short: "Show artifact details, eval summary, and promotion history",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, ok := model.ParseKind(args[0])
			if !ok {
				return fmt.Errorf("invalid kind: %s", args[0])
			}

			result, err := apiClient.Inspect(string(kind.Plural()), args[1], args[2])
			if err != nil {
				return err
			}

			a := result.Artifact
			fmt.Printf("%s %s@%s\n", a.Kind, a.Identity.Name, a.Identity.Version)
			fmt.Printf("  Status:   %s\n", a.Status)
			fmt.Printf("  Title:    %s\n", a.Identity.Title)
			fmt.Printf("  Created:  %s\n", a.CreatedAt)
			if a.PublishedAt != "" {
				fmt.Printf("  Published: %s\n", a.PublishedAt)
			}

			// Eval summary
			fmt.Printf("\nEval Records: %d\n", result.EvalSummary.TotalRecords)
			if result.EvalSummary.TotalRecords > 0 {
				for cat, count := range result.EvalSummary.Categories {
					fmt.Printf("  %-14s %d record(s)\n", cat, count)
				}
				fmt.Printf("  Average score: %.2f\n", result.EvalSummary.AverageScore)
			}

			// Promotion history
			if len(result.Promotions) > 0 {
				fmt.Printf("\nPromotion History:\n")
				for _, p := range result.Promotions {
					comment := ""
					if p.Comment != "" {
						comment = fmt.Sprintf(" — %s", p.Comment)
					}
					fmt.Printf("  %s → %s  (%s)%s\n", p.From, p.To, p.Timestamp, comment)
				}
			}

			return nil
		},
	}
}
