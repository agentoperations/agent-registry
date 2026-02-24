package cli

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/agentoperations/agent-registry/pkg/client"
	"github.com/spf13/cobra"
)

var (
	serverURL string
	apiClient *client.Client
	uiFS      fs.FS
)

// SetUIFS sets the embedded UI filesystem for the server start command.
func SetUIFS(f fs.FS) {
	uiFS = f
}

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "agentctl",
		Short: "Agent Registry CLI",
		Long:  "agentctl is the command-line tool for the Agent Registry. Push, discover, and manage AI agents, skills, and MCP servers.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Skip client setup for server command
			if cmd.Name() == "start" || cmd.Name() == "version" {
				return nil
			}
			apiClient = client.New(serverURL)
			return nil
		},
		SilenceUsage: true,
	}

	cfgServerURL := loadConfig().Server
	defaultServer := coalesce(os.Getenv("AGENTCTL_SERVER"), cfgServerURL, "http://localhost:8080")
	root.PersistentFlags().StringVar(&serverURL, "server", defaultServer, "Registry server URL")

	root.AddCommand(
		newConfigCmd(),
		newInitCmd(),
		newPushCmd(),
		newImportCmd(),
		newGetCmd(),
		newListCmd(),
		newDeleteCmd(),
		newPromoteCmd(),
		newEvalCmd(),
		newInspectCmd(),
		newDepsCmd(),
		newSearchCmd(),
		newServerCmd(),
		newVersionCmd(),
	)

	return root
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("agentctl v0.1.0")
		},
	}
}
