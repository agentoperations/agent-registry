package cli

import (
	"encoding/json"
	"fmt"

	"github.com/agentregistry/agent-registry/internal/model"
	"github.com/spf13/cobra"
)

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <kind> <name> [version]",
		Short: "Get an artifact by name and optional version",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, ok := model.ParseKind(args[0])
			if !ok {
				return fmt.Errorf("invalid kind: %s", args[0])
			}
			name := args[1]
			version := "latest"
			if len(args) == 3 {
				version = args[2]
			}

			artifact, err := apiClient.GetArtifact(string(kind.Plural()), name, version)
			if err != nil {
				return err
			}

			out, _ := json.MarshalIndent(artifact, "", "  ")
			fmt.Println(string(out))
			return nil
		},
	}
}
