package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// AgentctlConfig is the persistent configuration for the agentctl CLI.
// Stored at ~/.config/agentctl/config.yaml
type AgentctlConfig struct {
	Server string     `yaml:"server,omitempty"` // Registry server URL
	Init   InitConfig `yaml:"init,omitempty"`   // Defaults for agentctl init
}

type InitConfig struct {
	Provider string `yaml:"provider,omitempty"` // anthropic, openai
	API      string `yaml:"api,omitempty"`      // messages, chat-completions, responses
	Model    string `yaml:"model,omitempty"`    // Model name
	BaseURL  string `yaml:"baseUrl,omitempty"`  // Base URL for OpenAI-compatible providers
}

func configPath() string {
	if p := os.Getenv("AGENTCTL_CONFIG"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "agentctl", "config.yaml")
}

func loadConfig() *AgentctlConfig {
	cfg := &AgentctlConfig{}
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	yaml.Unmarshal(data, cfg)
	return cfg
}

func saveConfig(cfg *AgentctlConfig) error {
	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage agentctl configuration",
	}
	cmd.AddCommand(newConfigSetCmd(), newConfigShowCmd(), newConfigInitCmd())
	return cmd
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()
			data, _ := yaml.Marshal(cfg)
			fmt.Printf("Config file: %s\n\n", configPath())
			fmt.Println(string(data))
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config value (e.g., init.provider, init.model, init.baseUrl, init.api, server)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()
			key, value := args[0], args[1]

			switch key {
			case "server":
				cfg.Server = value
			case "init.provider":
				cfg.Init.Provider = value
			case "init.model":
				cfg.Init.Model = value
			case "init.baseUrl", "init.base-url":
				cfg.Init.BaseURL = value
			case "init.api":
				if value != "messages" && value != "chat-completions" && value != "responses" {
					return fmt.Errorf("invalid api: %s (use: messages, chat-completions, responses)", value)
				}
				cfg.Init.API = value
			default:
				return fmt.Errorf("unknown config key: %s", key)
			}

			if err := saveConfig(cfg); err != nil {
				return err
			}
			fmt.Printf("Set %s = %s\n", key, value)
			return nil
		},
	}
}

func newConfigInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Interactive config setup",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()

			fmt.Println("agentctl configuration setup")
			fmt.Println("Press Enter to keep current value.")
		fmt.Println()

			cfg.Server = promptWithDefault("Registry server URL", cfg.Server, "http://localhost:8080")
			cfg.Init.Provider = promptWithDefault("LLM provider (anthropic, openai)", cfg.Init.Provider, "anthropic")

			switch cfg.Init.Provider {
			case "anthropic":
				cfg.Init.API = promptWithDefault("API format (messages)", cfg.Init.API, "messages")
				cfg.Init.Model = promptWithDefault("Model", cfg.Init.Model, "claude-haiku-4-20250514")
			case "openai":
				cfg.Init.API = promptWithDefault("API format (chat-completions, responses)", cfg.Init.API, "chat-completions")
				cfg.Init.Model = promptWithDefault("Model", cfg.Init.Model, "gpt-4o-mini")
				cfg.Init.BaseURL = promptWithDefault("Base URL (blank for api.openai.com)", cfg.Init.BaseURL, "")
			}

			if err := saveConfig(cfg); err != nil {
				return err
			}
			fmt.Printf("\nConfig saved to %s\n", configPath())
			return nil
		},
	}
}

func promptWithDefault(label, current, fallback string) string {
	display := current
	if display == "" {
		display = fallback
	}
	if display != "" {
		fmt.Printf("  %s [%s]: ", label, display)
	} else {
		fmt.Printf("  %s: ", label)
	}

	var input string
	fmt.Scanln(&input)
	input = stringTrim(input)
	if input != "" {
		return input
	}
	if current != "" {
		return current
	}
	return fallback
}

func stringTrim(s string) string {
	return fmt.Sprintf("%s", fmt.Sprintf("%s", s))
}
