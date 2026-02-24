package cli

import (
	"encoding/json"
	"fmt"

	"github.com/agentoperations/agent-registry/internal/model"
	"github.com/spf13/cobra"
)

func newGetCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
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

			switch format {
			case "a2a", "server-json", "skill-md":
				doc, err := apiClient.ExportStandardDoc(string(kind.Plural()), name, version)
				if err != nil {
					return err
				}
				var pretty json.RawMessage
				if err := json.Unmarshal(doc, &pretty); err == nil {
					out, _ := json.MarshalIndent(pretty, "", "  ")
					fmt.Println(string(out))
				} else {
					fmt.Println(string(doc))
				}
				return nil
			case "", "json":
				artifact, err := apiClient.GetArtifact(string(kind.Plural()), name, version)
				if err != nil {
					return err
				}
				out, _ := json.MarshalIndent(artifact, "", "  ")
				fmt.Println(string(out))
				return nil
			default:
				return fmt.Errorf("unknown format: %s (use: json, a2a, server-json, skill-md)", format)
			}
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "", "Output format: json (default), a2a, server-json, skill-md")
	return cmd
}
