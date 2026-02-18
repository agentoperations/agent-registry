package cli

import (
	"fmt"

	"github.com/agentoperations/agent-registry/internal/model"
	"github.com/spf13/cobra"
)

func newPromoteCmd() *cobra.Command {
	var to, comment string

	cmd := &cobra.Command{
		Use:   "promote <kind> <name> <version>",
		Short: "Promote an artifact to a new lifecycle status",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, ok := model.ParseKind(args[0])
			if !ok {
				return fmt.Errorf("invalid kind: %s", args[0])
			}
			if to == "" {
				return fmt.Errorf("--to is required")
			}

			req := &model.PromotionRequest{
				TargetStatus: model.Status(to),
				Comment:      comment,
			}

			result, err := apiClient.Promote(string(kind.Plural()), args[1], args[2], req)
			if err != nil {
				return err
			}

			fmt.Printf("Promoted %s %s@%s -> %s\n",
				result.Kind, result.Identity.Name, result.Identity.Version, result.Status)
			return nil
		},
	}

	cmd.Flags().StringVar(&to, "to", "", "Target status (evaluated, approved, published, deprecated, archived)")
	cmd.Flags().StringVar(&comment, "comment", "", "Promotion comment")
	return cmd
}
