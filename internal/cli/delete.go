package cli

import (
	"fmt"

	"github.com/agentoperations/agent-registry/internal/model"
	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <kind> <name> <version>",
		Short: "Delete a draft artifact",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, ok := model.ParseKind(args[0])
			if !ok {
				return fmt.Errorf("invalid kind: %s", args[0])
			}
			if err := apiClient.DeleteArtifact(string(kind.Plural()), args[1], args[2]); err != nil {
				return err
			}
			fmt.Printf("Deleted %s %s@%s\n", kind, args[1], args[2])
			return nil
		},
	}
}
