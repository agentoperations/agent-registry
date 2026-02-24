# Standards-Based Schema Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Restructure the agent registry so that agents use A2A AgentCard, MCP servers use MCP server.json, and skills use Agent Skills (SKILL.md) as their native identity format, with the registry providing a governance envelope around them.

**Architecture:** The `RegistryArtifact` model gains three new `json.RawMessage` fields (`AgentCard`, `ServerJson`, `SkillMd`) plus origin URL strings. The `Identity` struct becomes optional (legacy mode). Field extraction at push time populates the existing indexed columns (name, title, description) from the standard document. Two new API endpoints (`/export`, `/import`) enable round-tripping standard documents.

**Tech Stack:** Go 1.24, Chi router, SQLite (modernc.org/sqlite), Cobra CLI, no new dependencies

---

### Task 1: Add standard document fields to the model

**Files:**
- Modify: `internal/model/artifact.go:100-128`

**Step 1: Add the new fields to RegistryArtifact**

Add three standard document fields and three origin URL fields to the `RegistryArtifact` struct. Keep `Identity` for backward compatibility but make it optional in the new flow.

```go
// In RegistryArtifact struct, after the Identity field:

// Standard identity documents (envelope model — one per kind)
AgentCard       json.RawMessage `json:"agentCard,omitempty" yaml:"agentCard,omitempty"`
AgentCardOrigin string          `json:"agentCardOrigin,omitempty" yaml:"agentCardOrigin,omitempty"`
ServerJson      json.RawMessage `json:"serverJson,omitempty" yaml:"serverJson,omitempty"`
ServerJsonOrigin string         `json:"serverJsonOrigin,omitempty" yaml:"serverJsonOrigin,omitempty"`
SkillMd         json.RawMessage `json:"skillMd,omitempty" yaml:"skillMd,omitempty"`
SkillMdOrigin   string          `json:"skillMdOrigin,omitempty" yaml:"skillMdOrigin,omitempty"`
```

**Step 2: Add a helper to extract identity from standard docs**

```go
// ExtractIdentity populates Identity fields from the standard document.
// Returns an error if the standard doc is missing required fields.
func (a *RegistryArtifact) ExtractIdentity() error {
	switch a.Kind {
	case KindAgent:
		return a.extractFromAgentCard()
	case KindMCPServer:
		return a.extractFromServerJson()
	case KindSkill:
		return a.extractFromSkillMd()
	}
	return fmt.Errorf("unknown kind: %s", a.Kind)
}

func (a *RegistryArtifact) extractFromAgentCard() error {
	if len(a.AgentCard) == 0 {
		return nil // legacy mode, identity already set
	}
	var card struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(a.AgentCard, &card); err != nil {
		return fmt.Errorf("parse agentCard: %w", err)
	}
	if card.Name == "" || card.Version == "" {
		return fmt.Errorf("agentCard missing required fields: name, version")
	}
	// Build identity from card + namespace
	ns := a.Identity.Name // may already have namespace prefix
	if ns == "" {
		ns = "_default"
	}
	a.Identity.Title = card.Name
	a.Identity.Description = card.Description
	a.Identity.Version = card.Version
	if !strings.Contains(a.Identity.Name, "/") {
		a.Identity.Name = ns + "/" + strings.ToLower(strings.ReplaceAll(card.Name, " ", "-"))
	}
	return nil
}

func (a *RegistryArtifact) extractFromServerJson() error {
	if len(a.ServerJson) == 0 {
		return nil
	}
	var sj struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(a.ServerJson, &sj); err != nil {
		return fmt.Errorf("parse serverJson: %w", err)
	}
	if sj.Name == "" || sj.Version == "" {
		return fmt.Errorf("serverJson missing required fields: name, version")
	}
	a.Identity.Version = sj.Version
	a.Identity.Description = sj.Description
	if sj.Title != "" {
		a.Identity.Title = sj.Title
	} else {
		a.Identity.Title = sj.Name
	}
	if !strings.Contains(a.Identity.Name, "/") {
		// Map reverse-DNS to namespace/name: "io.github.acme/weather" -> keep as-is if has /
		a.Identity.Name = sj.Name
	}
	return nil
}

func (a *RegistryArtifact) extractFromSkillMd() error {
	if len(a.SkillMd) == 0 {
		return nil
	}
	var sm struct {
		Name        string            `json:"name"`
		Description string            `json:"description"`
		Metadata    map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(a.SkillMd, &sm); err != nil {
		return fmt.Errorf("parse skillMd: %w", err)
	}
	if sm.Name == "" {
		return fmt.Errorf("skillMd missing required field: name")
	}
	a.Identity.Title = sm.Name
	a.Identity.Description = sm.Description
	if v, ok := sm.Metadata["version"]; ok {
		a.Identity.Version = v
	}
	if !strings.Contains(a.Identity.Name, "/") {
		a.Identity.Name = a.Namespace() + "/" + sm.Name
	}
	return nil
}

// StandardDocument returns the standard doc for this artifact's kind.
func (a *RegistryArtifact) StandardDocument() json.RawMessage {
	switch a.Kind {
	case KindAgent:
		return a.AgentCard
	case KindMCPServer:
		return a.ServerJson
	case KindSkill:
		return a.SkillMd
	}
	return nil
}

// HasStandardDocument returns true if a standard doc is present.
func (a *RegistryArtifact) HasStandardDocument() bool {
	return len(a.StandardDocument()) > 0
}
```

**Step 3: Add `fmt` to the import list**

The file already imports `encoding/json`, `strings`, `time`. Add `"fmt"` to the import block.

**Step 4: Run build to verify compilation**

Run: `cd /Users/azaalouk/agent-registry && go build ./...`
Expected: Compiles cleanly (new fields are additive, no breaking changes)

**Step 5: Commit**

```bash
git add internal/model/artifact.go
git commit -m "feat(model): add standard document fields to RegistryArtifact

Add AgentCard, ServerJson, SkillMd fields with origin URLs.
Add ExtractIdentity() to populate Identity from standard docs.
Add StandardDocument() and HasStandardDocument() helpers.
Backward compatible: Identity struct still works for legacy pushes."
```

---

### Task 2: Update service layer to handle standard documents on create

**Files:**
- Modify: `internal/service/registry.go:21-37`

**Step 1: Update CreateArtifact to extract identity from standard docs**

In `registryService.CreateArtifact`, after setting `a.Kind`, call `ExtractIdentity()` if a standard document is present:

```go
func (s *registryService) CreateArtifact(ctx context.Context, kind model.Kind, a *model.RegistryArtifact) (*model.RegistryArtifact, error) {
	a.Kind = kind
	a.Status = model.StatusDraft

	// If a standard document is present, extract identity fields from it
	if a.HasStandardDocument() {
		if err := a.ExtractIdentity(); err != nil {
			return nil, fmt.Errorf("extract identity from standard document: %w", err)
		}
	}

	// Validate: must have name and version after extraction
	if a.Identity.Name == "" || a.Identity.Version == "" {
		return nil, fmt.Errorf("artifact must have name and version (via identity or standard document)")
	}

	if _, err := s.store.CreateArtifact(ctx, a); err != nil {
		return nil, err
	}
	if err := s.store.UpdateLatestFlag(ctx, kind, a.Identity.Name); err != nil {
		return nil, fmt.Errorf("update latest flag: %w", err)
	}

	artifact, _, err := s.store.GetArtifact(ctx, kind, a.Identity.Name, a.Identity.Version)
	if err != nil {
		return nil, err
	}
	return artifact, nil
}
```

**Step 2: Run build**

Run: `cd /Users/azaalouk/agent-registry && go build ./...`
Expected: Compiles cleanly

**Step 3: Commit**

```bash
git add internal/service/registry.go
git commit -m "feat(service): extract identity from standard documents on create"
```

---

### Task 3: Add export handler and route

**Files:**
- Modify: `internal/handler/handler.go` (add ExportStandardDoc method)
- Modify: `internal/server/server.go:30-38` (add /export route)
- Modify: `internal/service/service.go` (add ExportStandardDoc to interface)
- Modify: `internal/service/registry.go` (implement ExportStandardDoc)

**Step 1: Add ExportStandardDoc to the service interface**

In `internal/service/service.go`, add to the `RegistryService` interface:

```go
ExportStandardDoc(ctx context.Context, kind model.Kind, name, version string) (json.RawMessage, string, error)
```

Add `"encoding/json"` to the import block.

**Step 2: Implement ExportStandardDoc in the service**

In `internal/service/registry.go`, add:

```go
func (s *registryService) ExportStandardDoc(ctx context.Context, kind model.Kind, name, version string) (json.RawMessage, string, error) {
	a, err := s.GetArtifact(ctx, kind, name, version)
	if err != nil {
		return nil, "", err
	}
	doc := a.StandardDocument()
	if len(doc) == 0 {
		return nil, "", fmt.Errorf("artifact has no standard document")
	}
	// Determine content type
	contentType := "application/json"
	return doc, contentType, nil
}
```

**Step 3: Add ExportStandardDoc handler**

In `internal/handler/handler.go`, add:

```go
func (h *Handler) ExportStandardDoc(w http.ResponseWriter, r *http.Request) {
	kind, ok := parseKind(w, r)
	if !ok {
		return
	}
	name := extractName(r)
	version := chi.URLParam(r, "version")

	doc, contentType, err := h.svc.ExportStandardDoc(r.Context(), kind, name, version)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "artifact not found")
			return
		}
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	w.Write(doc)
}
```

**Step 4: Add the route**

In `internal/server/server.go`, inside the version-specific route block (line ~31), add:

```go
r.Get("/export", h.ExportStandardDoc)
```

**Step 5: Run build**

Run: `cd /Users/azaalouk/agent-registry && go build ./...`
Expected: Compiles cleanly

**Step 6: Commit**

```bash
git add internal/handler/handler.go internal/server/server.go internal/service/service.go internal/service/registry.go
git commit -m "feat(api): add /export endpoint for standard document extraction

GET /{kind}/{ns}/{name}/versions/{ver}/export returns the pure
standard document (AgentCard, server.json, or SKILL.md) with no
governance wrapping. Deployment systems call this to serve the
A2A card at /.well-known/agent-card.json."
```

---

### Task 4: Add --format flag to CLI get command

**Files:**
- Modify: `internal/cli/get.go`
- Modify: `pkg/client/client.go` (add ExportStandardDoc method)

**Step 1: Add ExportStandardDoc to the client**

In `pkg/client/client.go`, add:

```go
func (c *Client) ExportStandardDoc(kind, name, version string) (json.RawMessage, error) {
	parts := splitName(name)
	path := fmt.Sprintf("/api/v1/%s/%s/versions/%s/export", kind, parts, version)

	req, err := http.NewRequest("GET", c.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var problem model.ProblemDetail
		if err := json.Unmarshal(body, &problem); err == nil && problem.Detail != "" {
			return nil, fmt.Errorf("%s (HTTP %d)", problem.Detail, resp.StatusCode)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}
```

**Step 2: Update get command with --format flag**

Replace `internal/cli/get.go` entirely:

```go
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

			// If a standard format is requested, use the export endpoint
			switch format {
			case "a2a", "server-json", "skill-md":
				doc, err := apiClient.ExportStandardDoc(string(kind.Plural()), name, version)
				if err != nil {
					return err
				}
				// Pretty-print the JSON
				var pretty json.RawMessage
				if err := json.Unmarshal(doc, &pretty); err == nil {
					out, _ := json.MarshalIndent(pretty, "", "  ")
					fmt.Println(string(out))
				} else {
					fmt.Println(string(doc))
				}
				return nil
			case "", "json":
				// Default: full artifact with governance envelope
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
```

**Step 3: Run build**

Run: `cd /Users/azaalouk/agent-registry && go build ./...`
Expected: Compiles cleanly

**Step 4: Commit**

```bash
git add internal/cli/get.go pkg/client/client.go
git commit -m "feat(cli): add --format flag to get command for standard doc export

agentctl get agents acme/my-agent 1.0.0 --format a2a
exports a pure A2A AgentCard JSON. Also supports server-json and skill-md."
```

---

### Task 5: Update push command to accept standard documents

**Files:**
- Modify: `internal/cli/push.go`

**Step 1: Add --namespace and --oci flags, detect standard doc input**

Replace `internal/cli/push.go`:

```go
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

			// Detect if this is a standard document or a full manifest
			ext := strings.ToLower(filepath.Ext(filePath))
			isStandardDoc := false

			if namespace != "" && ociRef != "" {
				// User provided --namespace and --oci, treat as standard doc
				isStandardDoc = true
			}

			if isStandardDoc {
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
				// Legacy: parse as full manifest
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
```

**Step 2: Run build**

Run: `cd /Users/azaalouk/agent-registry && go build ./...`
Expected: Compiles cleanly

**Step 3: Commit**

```bash
git add internal/cli/push.go
git commit -m "feat(cli): push accepts standard documents with --namespace and --oci

agentctl push agents agent-card.json --namespace acme --oci ghcr.io/acme/my-agent:1.0.0
Legacy full-manifest push still works without flags."
```

---

### Task 6: Add import command

**Files:**
- Create: `internal/cli/import.go`
- Modify: `internal/cli/root.go` (register command)

**Step 1: Create the import command**

Create `internal/cli/import.go`:

```go
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
		Long: `Import an agent from an A2A AgentCard URL, an MCP server from the MCP Registry,
or a skill from a SKILL.md repository.`,
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

	// Validate it's valid JSON
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
```

**Step 2: Register the command in root.go**

Read `internal/cli/root.go` to find where commands are registered, then add `rootCmd.AddCommand(newImportCmd())`.

**Step 3: Run build**

Run: `cd /Users/azaalouk/agent-registry && go build ./...`
Expected: Compiles cleanly

**Step 4: Commit**

```bash
git add internal/cli/import.go internal/cli/root.go
git commit -m "feat(cli): add import command for A2A AgentCard URLs

agentctl import --from-a2a https://agent.example.com/.well-known/agent-card.json \
  --namespace acme --oci ghcr.io/acme/my-agent:1.0.0"
```

---

### Task 7: Update example YAML files

**Files:**
- Modify: `examples/agent-tier0-minimal.yaml`
- Create: `examples/agent-a2a-card.json`
- Create: `examples/mcp-server-json.json`

**Step 1: Create an A2A AgentCard example**

Create `examples/agent-a2a-card.json` with a valid A2A AgentCard that can be pushed with `--namespace` and `--oci`.

**Step 2: Create a server.json example**

Create `examples/mcp-server-json.json` with a valid MCP server.json.

**Step 3: Update the existing minimal agent example**

Update `examples/agent-tier0-minimal.yaml` to show the envelope model with an embedded `agentCard` field alongside the legacy `identity` field.

**Step 4: Commit**

```bash
git add examples/
git commit -m "docs(examples): add A2A AgentCard and MCP server.json examples"
```

---

### Task 8: Update JSON schemas

**Files:**
- Modify: `schemas/registry-artifact.schema.json`
- Modify: `schemas/agent.schema.json`
- Modify: `schemas/mcp-server.schema.json`
- Modify: `schemas/skill.schema.json`

**Step 1: Add standard document fields to registry-artifact schema**

Add `agentCard`, `agentCardOrigin`, `serverJson`, `serverJsonOrigin`, `skillMd`, `skillMdOrigin` to the schema. Make `identity` no longer required (it can be derived from the standard doc).

**Step 2: Run build to verify nothing broke**

Run: `cd /Users/azaalouk/agent-registry && go build ./...`
Expected: Compiles cleanly (schemas are not compiled, but verify the build didn't break)

**Step 3: Commit**

```bash
git add schemas/
git commit -m "feat(schemas): add standard document fields to JSON schemas

identity is now optional when a standard document is present.
agentCard, serverJson, skillMd fields added to registry-artifact schema."
```

---

### Task 9: Update the README

**Files:**
- Modify: `README.md`

**Step 1: Update the How It Works section**

Add a section showing the standard document workflow alongside the legacy workflow. Show the new `--format` flag and `import` command.

**Step 2: Update the CLI command table**

Add `import` command. Update `push` to mention `--namespace`/`--oci` flags. Update `get` to mention `--format` flag.

**Step 3: Add a Standards section**

Brief paragraph explaining that agents use A2A AgentCard, MCP servers use MCP server.json, and skills use SKILL.md as their native identity format, with the registry providing the governance envelope.

**Step 4: Commit**

```bash
git add README.md
git commit -m "docs: update README with standards-based schema workflow"
```

---

### Task 10: End-to-end manual test

**Step 1: Build**

Run: `cd /Users/azaalouk/agent-registry && go build -o agentctl ./cmd/agentctl`

**Step 2: Start server**

Run: `./agentctl server start --port 8585 &`

**Step 3: Push an A2A AgentCard**

```bash
./agentctl push agents examples/agent-a2a-card.json \
  --namespace acme \
  --oci ghcr.io/acme/recipe-agent:1.0.0
```

Expected: `Pushed agent acme/recipe-agent@1.0.0 (status: draft)`

**Step 4: Export the standard doc**

```bash
./agentctl get agents acme/recipe-agent 1.0.0 --format a2a
```

Expected: Pure A2A AgentCard JSON output (no governance fields)

**Step 5: Push a legacy manifest**

```bash
./agentctl push agents examples/agent-tier0-minimal.yaml
```

Expected: Still works, backward compatible

**Step 6: Kill server**

Run: `kill %1`

**Step 7: Commit any fixes**

If any issues found, fix and commit.
