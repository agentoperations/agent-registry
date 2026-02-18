package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const manifestSystemPrompt = `You are an expert at analyzing software projects and generating Agent Registry manifests.

Given a project's source code, dependencies, Dockerfile, and documentation, generate a COMPLETE registry manifest in YAML format.

You must determine:
1. **kind**: Is this an "agent" (autonomous AI system with runtime), "skill" (prompt-based capability, no runtime), or "mcp-server" (MCP protocol server with tools)?
2. **identity**: Extract name (namespace/name format), version, title, and description from the project
3. **artifacts**: OCI image reference from Dockerfile or provided image ref
4. **metadata**: tags, category, license, authors, repository URL
5. **capabilities** (agents): protocols, input/output modalities, streaming, multi-turn, context window
6. **runtime** (agents): environment variables, ports, health checks, resource requirements
7. **content** (skills): entrypoint, trigger phrases
8. **transport** (mcp-servers): stdio/http/sse, command, tools list
9. **bom**: ALL dependencies — models (with provider, role), tools (MCP servers used), skills referenced, orchestration framework, memory/session config

Rules:
- Output ONLY the YAML manifest, no explanations before or after
- Use real values extracted from the code, don't make up fake data
- For the OCI reference: if an --image flag was provided use that, otherwise construct from Dockerfile
- Version: extract from the project (package.json version, pyproject.toml, git tags, Dockerfile tags). Default to "0.1.0" if unclear
- Namespace: use the GitHub org/user if visible in the repo URL, otherwise use "default"
- Be thorough with the BOM — scan imports to find ALL model providers, ALL MCP connections, ALL framework dependencies
- Include environment variables from Dockerfile ENV, .env.example, or code os.getenv/os.environ calls
- Detect the orchestration framework: langchain, crewai, adk, autogen, llama-stack, or custom`

func newInitCmd() *cobra.Command {
	var (
		projectPath string
		modelName   string
		provider    string
		apiFormat   string
		baseURL     string
		outputFile  string
		imageRef    string
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Auto-generate a registry manifest by analyzing your project with an LLM",
		Long: `Scans your project directory (source code, Dockerfile, dependencies, README)
and uses an LLM to generate a complete agent/skill/mcp-server registry manifest.

No YAML writing required — the LLM analyzes your code and generates everything.

Supports three API formats:
  - Anthropic Messages API (--api messages)
  - OpenAI Chat Completions API (--api chat-completions)
  - OpenAI Responses API (--api responses)

Configure defaults once with:  agentctl config set init.provider openai
                                agentctl config set init.model gpt-4o-mini
                                agentctl config set init.api responses

Then just run:  agentctl init --path ./my-agent -o manifest.yaml

Examples:
  # Auto-detect from env vars + config file
  agentctl init --path ./my-agent

  # Anthropic Messages API
  agentctl init --path ./my-agent --provider anthropic --model claude-haiku-4-20250514

  # OpenAI Chat Completions
  agentctl init --path ./my-agent --provider openai --api chat-completions --model gpt-4o-mini

  # OpenAI Responses API
  agentctl init --path ./my-agent --provider openai --api responses --model gpt-4o-mini

  # Ollama (OpenAI-compatible, no API key)
  agentctl init --path ./my-agent --provider openai --base-url http://localhost:11434/v1 --model llama3

  # vLLM
  agentctl init --path ./my-agent --provider openai --base-url http://my-vllm:8000/v1 --model my-model`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectPath == "" {
				projectPath = "."
			}

			llm, err := resolveLLMConfig(provider, apiFormat, baseURL, modelName)
			if err != nil {
				return err
			}

			fmt.Printf("Scanning project at %s...\n", projectPath)
			projectContext, err := scanProject(projectPath)
			if err != nil {
				return fmt.Errorf("scan project: %w", err)
			}

			if imageRef != "" {
				projectContext += fmt.Sprintf("\n\n## OCI Image Reference\n%s\n", imageRef)
			}

			fmt.Printf("Collected %d bytes of project context\n", len(projectContext))
			fmt.Printf("Using %s/%s via %s API\n", llm.Provider, llm.Model, llm.API)

			manifest, err := llm.Generate(context.Background(), projectContext)
			if err != nil {
				return fmt.Errorf("generate manifest: %w", err)
			}

			if outputFile != "" {
				if err := os.WriteFile(outputFile, []byte(manifest), 0644); err != nil {
					return fmt.Errorf("write output: %w", err)
				}
				fmt.Printf("\nManifest written to %s\n", outputFile)
				fmt.Printf("Review it, then: agentctl push <kind> %s\n", outputFile)
			} else {
				fmt.Println("\n--- Generated Manifest ---")
				fmt.Println(manifest)
				fmt.Println("--- End ---")
				fmt.Println("\nSave to a file and push: agentctl push <kind> manifest.yaml")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&projectPath, "path", "p", ".", "Path to the project directory")
	cmd.Flags().StringVar(&modelName, "model", "", "Model name (overrides config)")
	cmd.Flags().StringVar(&provider, "provider", "", "LLM provider: anthropic, openai (overrides config)")
	cmd.Flags().StringVar(&apiFormat, "api", "", "API format: messages, chat-completions, responses (overrides config)")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "Base URL for OpenAI-compatible APIs (overrides config)")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (default: stdout)")
	cmd.Flags().StringVar(&imageRef, "image", "", "OCI image reference (e.g., ghcr.io/acme/my-agent:1.0.0)")
	return cmd
}

// --- LLM provider config ---

type llmConfig struct {
	Provider string // anthropic, openai
	API      string // messages, chat-completions, responses
	Model    string
	BaseURL  string
	APIKey   string
}

// resolveLLMConfig merges: flags > config file > env vars > defaults
func resolveLLMConfig(flagProvider, flagAPI, flagBaseURL, flagModel string) (*llmConfig, error) {
	fileCfg := loadConfig()

	cfg := &llmConfig{
		Provider: coalesce(flagProvider, fileCfg.Init.Provider),
		API:      coalesce(flagAPI, fileCfg.Init.API),
		Model:    coalesce(flagModel, fileCfg.Init.Model),
		BaseURL:  coalesce(flagBaseURL, fileCfg.Init.BaseURL),
	}

	// Auto-detect provider from env if still empty
	if cfg.Provider == "" {
		switch {
		case os.Getenv("ANTHROPIC_API_KEY") != "":
			cfg.Provider = "anthropic"
		case os.Getenv("OPENAI_API_KEY") != "":
			cfg.Provider = "openai"
		case cfg.BaseURL != "":
			cfg.Provider = "openai"
		default:
			return nil, fmt.Errorf("no LLM provider configured.\n\nSet up with:  agentctl config init\nOr set:       ANTHROPIC_API_KEY or OPENAI_API_KEY\nOr use:       --provider and --base-url flags")
		}
	}

	// Apply provider-specific defaults
	switch cfg.Provider {
	case "anthropic":
		cfg.APIKey = os.Getenv("ANTHROPIC_API_KEY")
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY is required for provider 'anthropic'")
		}
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://api.anthropic.com"
		}
		if cfg.Model == "" {
			cfg.Model = "claude-haiku-4-20250514"
		}
		if cfg.API == "" {
			cfg.API = "messages"
		}

	case "openai":
		cfg.APIKey = os.Getenv("OPENAI_API_KEY")
		if cfg.BaseURL == "" {
			if cfg.APIKey == "" {
				return nil, fmt.Errorf("OPENAI_API_KEY is required when using OpenAI without --base-url")
			}
			cfg.BaseURL = "https://api.openai.com"
		}
		if cfg.Model == "" {
			cfg.Model = "gpt-4o-mini"
		}
		if cfg.API == "" {
			cfg.API = "chat-completions"
		}

	default:
		return nil, fmt.Errorf("unknown provider: %s (use 'anthropic' or 'openai')", cfg.Provider)
	}

	// Validate API format
	validAPIs := map[string]bool{"messages": true, "chat-completions": true, "responses": true}
	if !validAPIs[cfg.API] {
		return nil, fmt.Errorf("unknown api format: %s (use: messages, chat-completions, responses)", cfg.API)
	}

	return cfg, nil
}

func (c *llmConfig) Generate(ctx context.Context, projectContext string) (string, error) {
	switch c.API {
	case "messages":
		return c.callMessages(ctx, projectContext)
	case "chat-completions":
		return c.callChatCompletions(ctx, projectContext)
	case "responses":
		return c.callResponses(ctx, projectContext)
	default:
		return "", fmt.Errorf("unknown api: %s", c.API)
	}
}

// --- Anthropic Messages API: POST /v1/messages ---

func (c *llmConfig) callMessages(ctx context.Context, projectContext string) (string, error) {
	reqBody := map[string]interface{}{
		"model":      c.Model,
		"max_tokens": 4096,
		"system":     manifestSystemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": "Analyze this project and generate a complete registry manifest:\n\n" + projectContext},
		},
	}

	body, _ := json.Marshal(reqBody)
	url := strings.TrimSuffix(c.BaseURL, "/") + "/v1/messages"

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	respBody, err := doHTTP(req)
	if err != nil {
		return "", err
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}
	if len(result.Content) == 0 {
		return "", fmt.Errorf("empty response")
	}
	return stripCodeFences(result.Content[0].Text), nil
}

// --- OpenAI Chat Completions API: POST /v1/chat/completions ---

func (c *llmConfig) callChatCompletions(ctx context.Context, projectContext string) (string, error) {
	reqBody := map[string]interface{}{
		"model":      c.Model,
		"max_tokens": 4096,
		"messages": []map[string]string{
			{"role": "system", "content": manifestSystemPrompt},
			{"role": "user", "content": "Analyze this project and generate a complete registry manifest:\n\n" + projectContext},
		},
	}

	body, _ := json.Marshal(reqBody)
	url := strings.TrimSuffix(c.BaseURL, "/") + "/v1/chat/completions"

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	respBody, err := doHTTP(req)
	if err != nil {
		return "", err
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("empty response")
	}
	return stripCodeFences(result.Choices[0].Message.Content), nil
}

// --- OpenAI Responses API: POST /v1/responses ---

func (c *llmConfig) callResponses(ctx context.Context, projectContext string) (string, error) {
	reqBody := map[string]interface{}{
		"model":        c.Model,
		"instructions": manifestSystemPrompt,
		"input":        "Analyze this project and generate a complete registry manifest:\n\n" + projectContext,
	}

	body, _ := json.Marshal(reqBody)
	url := strings.TrimSuffix(c.BaseURL, "/") + "/v1/responses"

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	respBody, err := doHTTP(req)
	if err != nil {
		return "", err
	}

	var result struct {
		Output []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}
	for _, out := range result.Output {
		if out.Type == "message" {
			for _, c := range out.Content {
				if c.Type == "output_text" && c.Text != "" {
					return stripCodeFences(c.Text), nil
				}
			}
		}
	}
	return "", fmt.Errorf("no text output in responses API result")
}

// --- Shared HTTP helper ---

func doHTTP(req *http.Request) ([]byte, error) {
	resp, err := (&http.Client{Timeout: 120 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, truncateErr(string(body), 300))
	}
	return body, nil
}

func truncateErr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// --- Project scanning ---

func scanProject(root string) (string, error) {
	var buf strings.Builder

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	buf.WriteString(fmt.Sprintf("# Project: %s\n\n", filepath.Base(absRoot)))

	priorityFiles := []string{
		"README.md", "README", "readme.md",
		"Dockerfile", "Containerfile",
		"docker-compose.yaml", "docker-compose.yml",
		"requirements.txt", "pyproject.toml", "setup.py", "setup.cfg",
		"go.mod", "go.sum",
		"package.json", "tsconfig.json",
		"Cargo.toml",
		".env.example", ".env.template", ".env.sample",
		"SKILL.md", "skill.md",
		"agent.json", "agent.yaml",
		"mcp.json",
	}

	for _, name := range priorityFiles {
		path := filepath.Join(root, name)
		content, err := readFileSafe(path, 8000)
		if err != nil {
			continue
		}
		buf.WriteString(fmt.Sprintf("## File: %s\n```\n%s\n```\n\n", name, content))
	}

	sourceBudget := 30000
	sourceCollected := 0
	sourceExts := map[string]bool{
		".py": true, ".go": true, ".ts": true, ".js": true, ".rs": true,
		".java": true, ".yaml": true, ".yml": true, ".toml": true,
	}
	skipDirs := map[string]bool{
		"node_modules": true, ".git": true, "__pycache__": true, ".venv": true,
		"venv": true, "vendor": true, ".tox": true, "dist": true, "build": true,
		".mypy_cache": true, ".pytest_cache": true, "target": true,
	}

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || sourceCollected >= sourceBudget {
			return filepath.SkipDir
		}
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(info.Name()))
		if !sourceExts[ext] || info.Size() > 20000 {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		for _, pf := range priorityFiles {
			if rel == pf {
				return nil
			}
		}
		content, err := readFileSafe(path, 5000)
		if err != nil {
			return nil
		}
		buf.WriteString(fmt.Sprintf("## File: %s\n```%s\n%s\n```\n\n", rel, ext[1:], content))
		sourceCollected += len(content)
		return nil
	})

	buf.WriteString("## Directory Structure\n```\n")
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && skipDirs[info.Name()] {
			return filepath.SkipDir
		}
		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			return nil
		}
		depth := strings.Count(rel, string(os.PathSeparator))
		if depth > 3 {
			return nil
		}
		indent := strings.Repeat("  ", depth)
		if info.IsDir() {
			buf.WriteString(fmt.Sprintf("%s%s/\n", indent, info.Name()))
		} else {
			buf.WriteString(fmt.Sprintf("%s%s\n", indent, info.Name()))
		}
		return nil
	})
	buf.WriteString("```\n")

	return buf.String(), nil
}

func readFileSafe(path string, maxBytes int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	data := make([]byte, maxBytes)
	n, err := f.Read(data)
	if err != nil && err != io.EOF {
		return "", err
	}
	content := string(data[:n])
	if n == maxBytes {
		content += "\n... (truncated)"
	}
	return content, nil
}

func stripCodeFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```yaml") {
		s = strings.TrimPrefix(s, "```yaml")
	} else if strings.HasPrefix(s, "```yml") {
		s = strings.TrimPrefix(s, "```yml")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
