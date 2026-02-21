#!/usr/bin/env bash
# Setup script — runs BEFORE showtime recording starts.
# Creates artifacts, builds agentctl, starts server.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
export AGENTCTL_SERVER=http://localhost:8585

# Build
cd "$ROOT_DIR"
go build -o /tmp/agentctl ./cmd/agentctl 2>/dev/null

# Clean slate
rm -f /tmp/demo-showtime.db
pkill -f "agentctl server start --port 8585" 2>/dev/null || true
sleep 1

# Start server
/tmp/agentctl server start --port 8585 --db /tmp/demo-showtime.db &>/dev/null &
sleep 2

# Create demo YAML files
mkdir -p /tmp/demo-artifacts

cat > /tmp/demo-artifacts/agent.yaml << 'EOF'
kind: agent
identity:
  name: acme/cluster-doctor
  version: "1.0.0"
  title: Cluster Doctor
  description: AI-powered Kubernetes cluster diagnostics agent
artifacts:
  - oci: "ghcr.io/acme/cluster-doctor:1.0.0"
    digest: "sha256:a1b2c3d4e5f6"
metadata:
  tags: ["kubernetes", "diagnostics", "sre", "incident-response"]
  category: devops
  license: Apache-2.0
  repository:
    url: https://github.com/acme/cluster-doctor
    source: github
  authors:
    - name: Platform Team
      email: platform@acme.dev
capabilities:
  protocols: ["mcp", "a2a"]
  inputModalities: ["text"]
  outputModalities: ["text"]
  streaming: true
  multiTurn: true
bom:
  models:
    - name: claude-sonnet-4-20250514
      provider: anthropic
      role: primary
  tools:
    - name: acme/kubernetes-mcp
      version: ">=2.0.0"
    - name: acme/prometheus-mcp
      version: ">=1.0.0"
  skills:
    - name: acme/k8s-troubleshooting
      version: ">=1.0.0"
EOF

cat > /tmp/demo-artifacts/skill.yaml << 'EOF'
kind: skill
identity:
  name: acme/k8s-troubleshooting
  version: "1.0.0"
  title: Kubernetes Troubleshooting
  description: Structured skill for diagnosing common Kubernetes issues
artifacts:
  - oci: "ghcr.io/acme/k8s-troubleshooting:1.0.0"
metadata:
  tags: ["kubernetes", "troubleshooting", "sre"]
  category: devops
  license: Apache-2.0
EOF

cat > /tmp/demo-artifacts/mcp-server.yaml << 'EOF'
kind: mcp-server
identity:
  name: acme/kubernetes-mcp
  version: "2.0.0"
  title: Kubernetes MCP Server
  description: MCP server providing Kubernetes cluster management tools
artifacts:
  - oci: "ghcr.io/acme/kubernetes-mcp:2.0.0"
metadata:
  tags: ["kubernetes", "mcp", "infrastructure"]
  category: devops
  license: Apache-2.0
transport:
  type: stdio
  command: kubernetes-mcp
tools:
  - name: get_pods
    description: List pods in a namespace
  - name: describe_pod
    description: Get detailed pod information
  - name: get_events
    description: Get cluster events
EOF

echo "Setup complete. Server running on :8585"
