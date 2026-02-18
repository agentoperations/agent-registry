package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/agentoperations/agent-registry/internal/model"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newPushCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "push <kind> <file>",
		Short: "Push an artifact from a YAML/JSON file",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			kindStr := args[0]
			filePath := args[1]

			kind, ok := model.ParseKind(kindStr)
			if !ok {
				return fmt.Errorf("invalid kind: %s (use: agent, skill, mcp-server)", kindStr)
			}

			data, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}

			// Parse YAML into a generic map first, then re-serialize as JSON
			var raw map[string]interface{}
			if err := yaml.Unmarshal(data, &raw); err != nil {
				return fmt.Errorf("parse YAML: %w", err)
			}
			jsonData, err := json.Marshal(raw)
			if err != nil {
				return fmt.Errorf("convert to JSON: %w", err)
			}

			var artifact model.RegistryArtifact
			if err := json.Unmarshal(jsonData, &artifact); err != nil {
				return fmt.Errorf("parse artifact: %w", err)
			}

			result, err := apiClient.CreateArtifact(string(kind.Plural()), &artifact)
			if err != nil {
				return err
			}

			fmt.Printf("Pushed %s %s@%s (status: %s)\n",
				result.Kind, result.Identity.Name, result.Identity.Version, result.Status)
			return nil
		},
	}
}
