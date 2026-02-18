package main

import (
	"embed"
	"io/fs"

	"github.com/agentoperations/agent-registry/internal/cli"
)

//go:embed ui
var uiEmbed embed.FS

func main() {
	uiContent, _ := fs.Sub(uiEmbed, "ui")
	cli.SetUIFS(uiContent)
	cli.Execute()
}
