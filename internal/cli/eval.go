package cli

import (
	"fmt"
	"os"

	"github.com/agentoperations/agent-registry/internal/model"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func newEvalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "eval",
		Short: "Manage evaluation records",
	}
	cmd.AddCommand(newEvalAttachCmd(), newEvalListCmd())
	return cmd
}

func newEvalAttachCmd() *cobra.Command {
	var category, provider, benchmark, evaluator, method string
	var score float64

	cmd := &cobra.Command{
		Use:   "attach <kind> <name> <version>",
		Short: "Attach an evaluation record",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, ok := model.ParseKind(args[0])
			if !ok {
				return fmt.Errorf("invalid kind: %s", args[0])
			}

			eval := &model.EvalRecord{
				Category: model.EvalCategory(category),
				Benchmark: model.Benchmark{
					Name:    benchmark,
					Version: "1.0.0",
				},
				Evaluator: model.Evaluator{
					Name: evaluator,
					Type: method,
				},
				Results: model.EvalResults{
					OverallScore: score,
				},
			}
			if provider != "" {
				eval.Provider = &model.EvalProvider{Name: provider}
			}

			result, err := apiClient.SubmitEval(string(kind.Plural()), args[1], args[2], eval)
			if err != nil {
				return err
			}

			fmt.Printf("Eval record attached: %s (category: %s, benchmark: %s, score: %.2f)\n",
				result.ID, result.Category, result.Benchmark.Name, result.Results.OverallScore)
			return nil
		},
	}

	cmd.Flags().StringVar(&category, "category", "functional", "Eval category (functional, safety, red-team, performance, custom)")
	cmd.Flags().StringVar(&provider, "provider", "", "Provider name (e.g., garak, eval-hub)")
	cmd.Flags().StringVar(&benchmark, "benchmark", "", "Benchmark name")
	cmd.Flags().Float64Var(&score, "score", 0, "Overall score (0.0-1.0)")
	cmd.Flags().StringVar(&evaluator, "evaluator", "cli", "Evaluator identity")
	cmd.Flags().StringVar(&method, "method", "automated", "Evaluation method (automated, human, hybrid)")
	return cmd
}

func newEvalListCmd() *cobra.Command {
	var category string

	cmd := &cobra.Command{
		Use:   "list <kind> <name> <version>",
		Short: "List evaluation records",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, ok := model.ParseKind(args[0])
			if !ok {
				return fmt.Errorf("invalid kind: %s", args[0])
			}

			evals, err := apiClient.ListEvals(string(kind.Plural()), args[1], args[2], category)
			if err != nil {
				return err
			}

			if len(evals) == 0 {
				fmt.Println("No evaluation records found.")
				return nil
			}

			header := []string{"ID", "Category", "Provider", "Benchmark", "Score", "Evaluator"}
			var rows [][]string
			for _, e := range evals {
				provName := ""
				if e.Provider != nil {
					provName = e.Provider.Name
				}
				rows = append(rows, []string{
					truncate(e.ID, 12),
					string(e.Category),
					provName,
					e.Benchmark.Name,
					fmt.Sprintf("%.2f", e.Results.OverallScore),
					e.Evaluator.Name,
				})
			}
			table := tablewriter.NewTable(os.Stdout, tablewriter.WithHeader(header))
			table.Bulk(rows)
			table.Render()
			return nil
		},
	}

	cmd.Flags().StringVar(&category, "category", "", "Filter by category")
	return cmd
}

