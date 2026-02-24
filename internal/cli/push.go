package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agentoperations/agent-registry/internal/model"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newPushCmd() *cobra.Command {
	var namespace string
	var ociRef string

	cmd := &cobra.Command{
		Use:   "push <kind> <file>",
		Short: "Push an artifact from a YAML/JSON file or standard document",
		Long: `Push an artifact to the registry. Accepts:
  - A full registry manifest (YAML/JSON with identity + kind + artifacts)
  - An A2A AgentCard JSON file (for agents, with --namespace and --oci)
  - An MCP server.json file (for mcp-servers, with --namespace and --oci)
  - A SKILL.md file or directory (for skills, with --namespace and --oci)`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			kindStr := args[0]
			filePath := args[1]

			kind, ok := model.ParseKind(kindStr)
			if !ok {
				return fmt.Errorf("invalid kind: %s (use: agents, skills, mcp-servers)", kindStr)
			}

			data, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}

			var artifact model.RegistryArtifact

			if namespace != "" && ociRef != "" {
				// Standard document mode: wrap in envelope
				artifact.Kind = kind
				artifact.Identity.Name = namespace
				artifact.Artifacts = []model.OCIReference{{OCI: ociRef}}

				switch kind {
				case model.KindAgent:
					artifact.AgentCard = json.RawMessage(data)
				case model.KindMCPServer:
					artifact.ServerJson = json.RawMessage(data)
				case model.KindSkill:
					artifact.SkillMd = json.RawMessage(data)
				}
			} else {
				// Legacy mode: parse as full manifest
				ext := strings.ToLower(filepath.Ext(filePath))
				if ext == ".json" {
					if err := json.Unmarshal(data, &artifact); err != nil {
						return fmt.Errorf("parse JSON: %w", err)
					}
				} else {
					var raw map[string]interface{}
					if err := yaml.Unmarshal(data, &raw); err != nil {
						return fmt.Errorf("parse YAML: %w", err)
					}
					jsonData, err := json.Marshal(raw)
					if err != nil {
						return fmt.Errorf("convert to JSON: %w", err)
					}
					if err := json.Unmarshal(jsonData, &artifact); err != nil {
						return fmt.Errorf("parse artifact: %w", err)
					}
				}
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

	cmd.Flags().StringVar(&namespace, "namespace", "", "Registry namespace (required for standard doc push)")
	cmd.Flags().StringVar(&ociRef, "oci", "", "OCI reference (required for standard doc push)")
	return cmd
}
