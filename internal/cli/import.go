package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/agentoperations/agent-registry/internal/model"
	"github.com/spf13/cobra"
)

func newImportCmd() *cobra.Command {
	var fromA2A string
	var namespace string
	var ociRef string

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import an artifact from a standard document URL",
		Long: `Import an agent from an A2A AgentCard URL.

Examples:
  agentctl import --from-a2a https://agent.example.com/.well-known/agent-card.json \
    --namespace acme --oci ghcr.io/acme/my-agent:1.0.0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if fromA2A != "" {
				return importFromA2A(fromA2A, namespace, ociRef)
			}
			return fmt.Errorf("specify a source: --from-a2a <url>")
		},
	}

	cmd.Flags().StringVar(&fromA2A, "from-a2a", "", "Import agent from A2A AgentCard URL")
	cmd.Flags().StringVar(&namespace, "namespace", "", "Registry namespace")
	cmd.Flags().StringVar(&ociRef, "oci", "", "OCI reference for the artifact")

	return cmd
}

func importFromA2A(cardURL, namespace, ociRef string) error {
	if namespace == "" {
		return fmt.Errorf("--namespace is required")
	}
	if ociRef == "" {
		return fmt.Errorf("--oci is required")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(cardURL)
	if err != nil {
		return fmt.Errorf("fetch agent card: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch agent card: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read agent card: %w", err)
	}

	var card json.RawMessage
	if err := json.Unmarshal(body, &card); err != nil {
		return fmt.Errorf("invalid agent card JSON: %w", err)
	}

	artifact := &model.RegistryArtifact{
		Kind:            model.KindAgent,
		Identity:        model.Identity{Name: namespace},
		AgentCard:       card,
		AgentCardOrigin: cardURL,
		Artifacts:       []model.OCIReference{{OCI: ociRef}},
	}

	result, err := apiClient.CreateArtifact("agents", artifact)
	if err != nil {
		return err
	}

	fmt.Printf("Imported %s %s@%s from %s (status: %s)\n",
		result.Kind, result.Identity.Name, result.Identity.Version, cardURL, result.Status)
	return nil
}
