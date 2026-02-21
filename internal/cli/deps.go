package cli

import (
	"fmt"

	"github.com/agentregistry/agent-registry/internal/model"
	"github.com/spf13/cobra"
)

func newDepsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "deps <kind> <name> <version>",
		Short: "Show dependency graph",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, ok := model.ParseKind(args[0])
			if !ok {
				return fmt.Errorf("invalid kind: %s", args[0])
			}

			graph, err := apiClient.GetDependencies(string(kind.Plural()), args[1], args[2])
			if err != nil {
				return err
			}

			fmt.Printf("%s %s@%s\n", graph.Artifact.Kind, graph.Artifact.Name, graph.Artifact.Version)
			for _, dep := range graph.Dependencies {
				printDep(dep, "  ")
			}
			if len(graph.Dependencies) == 0 {
				fmt.Println("  (no dependencies declared in BOM)")
			}
			return nil
		},
	}
}

func printDep(n model.DependencyNode, indent string) {
	resolved := "resolved"
	if !n.Resolved {
		resolved = "UNRESOLVED"
	}
	ver := n.Version
	if ver == "" {
		ver = n.VersionConstraint
	}
	fmt.Printf("%s|- %s %s@%s [%s]\n", indent, n.Kind, n.Name, ver, resolved)
	for _, child := range n.Dependencies {
		printDep(child, indent+"  ")
	}
}
