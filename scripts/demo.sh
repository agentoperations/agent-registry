#!/usr/bin/env bash
# ============================================================================
#  Agent Registry — Full Narrated Demo
# ============================================================================
#
#  Usage:  ./scripts/demo.sh [--auto]
#
#  Walks through every agentctl capability with narration:
#
#    ACT 1  Setup & Server
#    ACT 2  Push Artifacts (agent, skill, MCP server)
#    ACT 3  Discovery (list, search, get)
#    ACT 4  Evaluation Records (safety, red-team, functional)
#    ACT 5  Promotion Lifecycle (draft → evaluated → approved → published)
#    ACT 6  Inspect & Trust Signals
#    ACT 7  Dependency Graph (BOM resolution)
#    ACT 8  Immutability Guardrails
#    ACT 9  Second Version & Versioning
#    ACT 10 REST API & Web UI
#    ACT 11 Cleanup Commands
#
#  Flags:
#    --auto    Run without pauses (for CI / recording)
#
# ============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
SERVER_PORT=8080
DB_FILE="/tmp/registry-demo.db"
ARCTL=""
AUTO=false

[[ "${1:-}" == "--auto" ]] && AUTO=true

# ── Colors ──────────────────────────────────────────────────────────────────

GREEN='\033[0;32m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
MAGENTA='\033[0;35m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

# ── Helpers ─────────────────────────────────────────────────────────────────

act() {
    echo ""
    echo -e "${BLUE}${BOLD}┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓${NC}"
    echo -e "${BLUE}${BOLD}┃  $1${NC}"
    echo -e "${BLUE}${BOLD}┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛${NC}"
    echo ""
}

narrate() {
    echo -e "${DIM}${MAGENTA}  $1${NC}"
}

info() {
    echo -e "  ${GREEN}$1${NC}"
}

warn() {
    echo -e "  ${YELLOW}$1${NC}"
}

run() {
    echo -e "\n${CYAN}  \$ $*${NC}"
    "$@" 2>&1 | sed 's/^/  /'
    echo ""
}

fail_expected() {
    echo -e "\n${CYAN}  \$ $*${NC}"
    if "$@" 2>&1 | sed 's/^/  /'; then
        echo -e "  ${RED}(expected failure but command succeeded)${NC}"
    else
        echo -e "  ${YELLOW}^ Expected failure — guardrail working correctly${NC}"
    fi
    echo ""
}

pause() {
    if [[ "$AUTO" == "false" ]]; then
        echo -e "${DIM}  Press Enter to continue...${NC}"
        read -r
    fi
}

separator() {
    echo -e "${DIM}  ─────────────────────────────────────────────────────${NC}"
}

# ── Cleanup on exit ─────────────────────────────────────────────────────────

cleanup() {
    if [[ -n "${SERVER_PID:-}" ]]; then
        kill "$SERVER_PID" 2>/dev/null || true
        wait "$SERVER_PID" 2>/dev/null || true
    fi
    rm -f "$DB_FILE"
    rm -rf "${TMPDIR:-}"
}
trap cleanup EXIT

# ============================================================================
#  PROLOGUE
# ============================================================================

echo ""
echo -e "${BOLD}${BLUE}"
echo '    _                    _     ____            _     _              '
echo '   / \   __ _  ___ _ __ | |_  |  _ \ ___  __ _(_)___| |_ _ __ _   _'
echo '  / _ \ / _` |/ _ \ `_ \| __| | |_) / _ \/ _` | / __| __| `__| | | |'
echo ' / ___ \ (_| |  __/ | | | |_  |  _ <  __/ (_| | \__ \ |_| |  | |_| |'
echo '/_/   \_\__, |\___|_| |_|\__| |_| \_\___|\__, |_|___/\__|_|   \__, |'
echo '        |___/                             |___/                |___/ '
echo -e "${NC}"
echo -e "${DIM}  A vendor-neutral registry for AI agents, skills, and MCP servers.${NC}"
echo -e "${DIM}  https://github.com/agentoperations/agent-registry${NC}"
echo ""
separator

narrate "This demo walks through the full agentctl workflow."
narrate "We will register a Kubernetes diagnostics agent that composes"
narrate "an MCP server (for cluster access) and a skill (troubleshooting"
narrate "playbook), then evaluate it, promote it through the governance"
narrate "lifecycle, and inspect its trust signals and dependency graph."
echo ""

pause

# ============================================================================
#  ACT 1 — BUILD & START SERVER
# ============================================================================

act "ACT 1: Build & Start the Registry Server"

narrate "agentctl is both the CLI client and the server in one binary."
narrate "The server uses SQLite for storage and serves a web UI alongside"
narrate "the REST API.  Let's build it and start a fresh instance."

separator
echo ""

info "Building agentctl from source..."
cd "$ROOT_DIR"
go build -o /tmp/agentctl ./cmd/agentctl 2>&1 | sed 's/^/  /'
ARCTL="/tmp/agentctl"
info "Built: /tmp/agentctl"

echo ""
run $ARCTL version
separator

narrate "Starting the registry server on port $SERVER_PORT with a fresh database."

rm -f "$DB_FILE"
$ARCTL server start --port $SERVER_PORT --db "$DB_FILE" &
SERVER_PID=$!
sleep 2

info "Server running on http://localhost:$SERVER_PORT (PID: $SERVER_PID)"
info "API endpoint: http://localhost:$SERVER_PORT/api/v1"
info "Web UI:       http://localhost:$SERVER_PORT"

echo ""
pause

# ============================================================================
#  ACT 2 — PUSH ARTIFACTS
# ============================================================================

act "ACT 2: Push Artifacts — Agent, Skill, MCP Server"

narrate "The registry manages three first-class artifact kinds:"
narrate ""
narrate "  1. AGENTS     — Autonomous AI systems with container runtimes"
narrate "  2. SKILLS     — Reusable prompt/playbook bundles (no runtime)"
narrate "  3. MCP SERVERS — Model Context Protocol servers with tools"
narrate ""
narrate "Each artifact is described by a YAML manifest. The registry stores"
narrate "metadata — the actual containers live in OCI registries (ghcr.io, etc)."
narrate ""
narrate "Let's push one of each to build a realistic dependency chain:"
narrate "  cluster-doctor (agent) --> kubernetes-mcp (MCP server)"
narrate "                         --> prometheus-mcp (MCP server)"
narrate "                         --> k8s-troubleshooting (skill)"

separator

# ── Create manifests ──

TMPDIR=$(mktemp -d)

# --- Agent manifest ---
cat > "$TMPDIR/agent.yaml" << 'YAML'
kind: agent
identity:
  name: acme/cluster-doctor
  version: "1.0.0"
  title: Cluster Doctor
  description: >-
    AI-powered Kubernetes cluster diagnostics agent. Analyzes pod failures,
    resource contention, and alert correlations to provide actionable
    remediation steps.
artifacts:
  - oci: "ghcr.io/acme/cluster-doctor:1.0.0"
    digest: "sha256:b5bb9d8014a0f9b1d61e21e796d78dccdf1352f23cd32812f4850b878ae4944c"
metadata:
  labels:
    team: platform-engineering
    tier: production
  tags: ["kubernetes", "diagnostics", "sre", "incident-response"]
  category: infrastructure/diagnostics
  authors:
    - name: Platform Engineering Team
      email: platform@acme.dev
  license: Apache-2.0
  repository:
    url: https://github.com/acme/cluster-doctor
    source: github
capabilities:
  protocols: ["mcp", "a2a"]
  inputModalities: ["text"]
  outputModalities: ["text"]
  streaming: true
  multiTurn: true
bom:
  models:
    - name: granite-3.1-8b-instruct
      provider: ibm
      role: primary
    - name: llama-3.1-70b-instruct
      provider: meta
      role: fallback
  tools:
    - name: acme/kubernetes-mcp
      version: ">=2.0.0"
    - name: acme/prometheus-mcp
      version: ">=1.0.0"
  skills:
    - name: acme/k8s-troubleshooting
      version: ">=1.0.0"
YAML

# --- Skill manifest ---
cat > "$TMPDIR/skill.yaml" << 'YAML'
kind: skill
identity:
  name: acme/k8s-troubleshooting
  version: "1.0.0"
  title: Kubernetes Troubleshooting
  description: >-
    Structured troubleshooting playbook for diagnosing common Kubernetes
    issues: CrashLoopBackOff, OOMKilled, ImagePullBackOff, pending pods,
    and network policy misconfigurations.
artifacts:
  - oci: "ghcr.io/acme/k8s-troubleshooting:1.0.0"
metadata:
  tags: ["kubernetes", "troubleshooting", "sre", "runbooks"]
  category: operations/troubleshooting
  license: Apache-2.0
  authors:
    - name: SRE Team
      email: sre@acme.dev
  repository:
    url: https://github.com/acme/k8s-troubleshooting
    source: github
YAML

# --- MCP Server manifest ---
cat > "$TMPDIR/mcp-server-k8s.yaml" << 'YAML'
kind: mcp-server
identity:
  name: acme/kubernetes-mcp
  version: "2.0.0"
  title: Kubernetes MCP Server
  description: >-
    MCP server providing read-only Kubernetes cluster management tools.
    Supports pod listing, resource description, event retrieval, and
    container log streaming.
artifacts:
  - oci: "ghcr.io/acme/kubernetes-mcp:2.0.0"
metadata:
  tags: ["kubernetes", "mcp", "infrastructure", "cluster-management"]
  category: infrastructure/kubernetes
  license: Apache-2.0
  authors:
    - name: Platform Engineering Team
      email: platform@acme.dev
  repository:
    url: https://github.com/acme/kubernetes-mcp
    source: github
tools:
  - name: get_pods
    description: List pods in a namespace with status and restart counts
  - name: describe_resource
    description: Get detailed YAML description of any Kubernetes resource
  - name: get_events
    description: Get cluster events filtered by namespace or resource
  - name: get_logs
    description: Stream or tail container logs from a pod
  - name: get_nodes
    description: List cluster nodes with capacity and allocatable resources
transport:
  type: stdio
  command: kubernetes-mcp
YAML

# --- Prometheus MCP Server ---
cat > "$TMPDIR/mcp-server-prom.yaml" << 'YAML'
kind: mcp-server
identity:
  name: acme/prometheus-mcp
  version: "1.5.0"
  title: Prometheus MCP Server
  description: >-
    MCP server for querying Prometheus metrics. Supports instant queries,
    range queries, alert enumeration, and target health inspection.
artifacts:
  - oci: "ghcr.io/acme/prometheus-mcp:1.5.0"
metadata:
  tags: ["prometheus", "metrics", "monitoring", "observability"]
  category: observability/metrics
  license: Apache-2.0
  authors:
    - name: Observability Team
      email: observability@acme.dev
  repository:
    url: https://github.com/acme/prometheus-mcp
    source: github
tools:
  - name: query_instant
    description: Execute an instant PromQL query
  - name: query_range
    description: Execute a range PromQL query over a time window
  - name: get_alerts
    description: Return currently firing and pending alerts
  - name: get_targets
    description: Return scrape target status and health
transport:
  type: streamable-http
  command: prometheus-mcp
YAML

echo ""
narrate "Pushing the MCP servers first (they are dependencies)..."
echo ""

info "1/4  Pushing MCP server: acme/kubernetes-mcp@2.0.0"
run $ARCTL push mcp-servers "$TMPDIR/mcp-server-k8s.yaml"

info "2/4  Pushing MCP server: acme/prometheus-mcp@1.5.0"
run $ARCTL push mcp-servers "$TMPDIR/mcp-server-prom.yaml"

narrate "Now the skill..."
echo ""

info "3/4  Pushing skill: acme/k8s-troubleshooting@1.0.0"
run $ARCTL push skills "$TMPDIR/skill.yaml"

narrate "And finally the agent that composes them all..."
echo ""

info "4/4  Pushing agent: acme/cluster-doctor@1.0.0"
run $ARCTL push agents "$TMPDIR/agent.yaml"

separator
info "All 4 artifacts pushed. Each starts in 'draft' status."

echo ""
pause

# ============================================================================
#  ACT 3 — DISCOVERY: LIST, SEARCH, GET
# ============================================================================

act "ACT 3: Discovery — List, Search, Get Details"

narrate "The registry provides multiple discovery mechanisms:"
narrate "  - list    List artifacts by kind, with optional status/category filters"
narrate "  - search  Cross-kind full-text search across names, descriptions, tags"
narrate "  - get     Fetch the full JSON manifest for a specific artifact+version"

separator

narrate "List all agents:"
run $ARCTL list agents

narrate "List all skills:"
run $ARCTL list skills

narrate "List all MCP servers:"
run $ARCTL list mcp-servers

separator

narrate "Search across ALL kinds for 'kubernetes':"
narrate "(This finds agents, skills, and MCP servers matching the query)"
run $ARCTL search kubernetes

separator

narrate "Search for 'prometheus' — should find just the MCP server:"
run $ARCTL search prometheus

separator

narrate "Get full details for the cluster-doctor agent:"
run $ARCTL get agents acme/cluster-doctor 1.0.0

echo ""
pause

# ============================================================================
#  ACT 4 — EVALUATION RECORDS
# ============================================================================

act "ACT 4: Attach Evaluation Records"

narrate "Before an artifact can be promoted beyond 'draft', it needs"
narrate "evaluation records. The registry supports multiple eval categories:"
narrate ""
narrate "  - safety      Toxicity, bias, harmful content detection"
narrate "  - red-team    Prompt injection, jailbreak, adversarial robustness"
narrate "  - functional  Task-specific accuracy and capability benchmarks"
narrate "  - performance Latency, throughput, resource usage"
narrate ""
narrate "Each eval record includes: provider, benchmark name, score (0-1),"
narrate "evaluator identity, and evaluation method (automated/human/hybrid)."

separator

narrate "Attaching a SAFETY eval — Garak toxicity detection scan:"
run $ARCTL eval attach agents acme/cluster-doctor 1.0.0 \
    --category safety \
    --provider garak \
    --benchmark toxicity-detection \
    --score 0.96 \
    --evaluator ci-safety-pipeline \
    --method automated

narrate "Attaching a RED-TEAM eval — Garak prompt injection resistance:"
run $ARCTL eval attach agents acme/cluster-doctor 1.0.0 \
    --category red-team \
    --provider garak \
    --benchmark prompt-injection \
    --score 0.91 \
    --evaluator ci-security-pipeline \
    --method automated

narrate "Attaching a FUNCTIONAL eval — k8s diagnostics accuracy benchmark:"
run $ARCTL eval attach agents acme/cluster-doctor 1.0.0 \
    --category functional \
    --provider eval-hub \
    --benchmark k8s-diagnostics-accuracy \
    --score 0.88 \
    --evaluator integration-tests \
    --method automated

separator

narrate "All eval records for cluster-doctor@1.0.0:"
run $ARCTL eval list agents acme/cluster-doctor 1.0.0

separator

narrate "Three evals attached: safety (0.96), red-team (0.91), functional (0.88)."
narrate "The agent is now eligible for promotion to 'evaluated' status."

echo ""
pause

# ============================================================================
#  ACT 5 — PROMOTION LIFECYCLE
# ============================================================================

act "ACT 5: Promotion Lifecycle"

narrate "Artifacts progress through a 6-stage governance lifecycle:"
narrate ""
narrate "  draft --> evaluated --> approved --> published --> deprecated --> archived"
narrate ""
narrate "Each transition has gates:"
narrate "  - draft -> evaluated     Requires at least one eval record"
narrate "  - evaluated -> approved  Requires review sign-off"
narrate "  - approved -> published  Artifact becomes discoverable & immutable"
narrate ""
narrate "Once published, the artifact's content is permanently locked."
narrate "Let's promote cluster-doctor through the full lifecycle."

separator

narrate "Step 1: draft -> evaluated"
narrate "(Gate: eval records must exist — we attached 3 above)"
run $ARCTL promote agents acme/cluster-doctor 1.0.0 \
    --to evaluated \
    --comment "Safety, red-team, and functional evals pass — all scores > 0.85"

narrate "Step 2: evaluated -> approved"
narrate "(Gate: human review — comment serves as audit trail)"
run $ARCTL promote agents acme/cluster-doctor 1.0.0 \
    --to approved \
    --comment "Reviewed and approved by platform team lead (@jane)"

narrate "Step 3: approved -> published"
narrate "(Gate: final sign-off — artifact becomes immutable and discoverable)"
run $ARCTL promote agents acme/cluster-doctor 1.0.0 \
    --to published \
    --comment "Production-ready. Deployed to shared agent registry."

separator

info "cluster-doctor@1.0.0 is now PUBLISHED."
narrate "The promotion history is recorded as an immutable audit trail."

echo ""
pause

# ============================================================================
#  ACT 6 — INSPECT & TRUST SIGNALS
# ============================================================================

act "ACT 6: Inspect Artifact — Status, Evals, Promotion History"

narrate "The 'inspect' command gives a consolidated view of an artifact:"
narrate "  - Current status and timestamps"
narrate "  - Evaluation summary (count, categories, average score)"
narrate "  - Full promotion history with comments"

separator

run $ARCTL inspect agents acme/cluster-doctor 1.0.0

separator

narrate "The inspect output shows:"
narrate "  - Status is 'published' with publication timestamp"
narrate "  - 3 eval records across safety, red-team, and functional categories"
narrate "  - Complete promotion trail: draft -> evaluated -> approved -> published"
narrate "  - Each promotion step has a timestamp and audit comment"

echo ""
pause

# ============================================================================
#  ACT 7 — DEPENDENCY GRAPH
# ============================================================================

act "ACT 7: Dependency Graph (BOM Resolution)"

narrate "Every artifact has a Bill of Materials (BOM) declaring its dependencies."
narrate "The 'deps' command resolves the full transitive dependency tree:"
narrate ""
narrate "  cluster-doctor (agent)"
narrate "    |-- granite-3.1-8b-instruct (model, primary)"
narrate "    |-- llama-3.1-70b-instruct (model, fallback)"
narrate "    |-- acme/kubernetes-mcp@>=2.0.0 (tool)"
narrate "    |-- acme/prometheus-mcp@>=1.0.0 (tool)"
narrate "    |-- acme/k8s-troubleshooting@>=1.0.0 (skill)"
narrate ""
narrate "Dependencies registered in the registry are marked 'resolved'."
narrate "Missing dependencies are marked 'UNRESOLVED'."

separator

run $ARCTL deps agents acme/cluster-doctor 1.0.0

separator

narrate "The dependency graph is extracted from the BOM section of the manifest."
narrate "This enables supply chain analysis: you can audit every model, tool,"
narrate "and skill that an agent depends on before deploying it."

echo ""
pause

# ============================================================================
#  ACT 8 — IMMUTABILITY GUARDRAILS
# ============================================================================

act "ACT 8: Immutability Guardrails"

narrate "Published artifacts are IMMUTABLE. The registry enforces this"
narrate "by rejecting delete and update operations on non-draft artifacts."
narrate ""
narrate "Let's verify this by trying to delete the published agent:"

separator

fail_expected $ARCTL delete agents acme/cluster-doctor 1.0.0

separator

narrate "The delete was rejected because the artifact is published."
narrate "This is by design — published artifacts must remain stable"
narrate "for all downstream consumers. To make changes, publish a new version."

echo ""
pause

# ============================================================================
#  ACT 9 — SECOND VERSION & VERSIONING
# ============================================================================

act "ACT 9: Versioning — Publishing a New Version"

narrate "The registry supports multiple versions of the same artifact."
narrate "Each version has its own lifecycle, evals, and trust signals."
narrate "Let's push v1.1.0 of the cluster-doctor with an upgraded model."

separator

cat > "$TMPDIR/agent-v2.yaml" << 'YAML'
kind: agent
identity:
  name: acme/cluster-doctor
  version: "1.1.0"
  title: Cluster Doctor
  description: >-
    AI-powered Kubernetes cluster diagnostics agent (v1.1). Now uses
    granite-3.2 for improved reasoning and adds network policy analysis.
artifacts:
  - oci: "ghcr.io/acme/cluster-doctor:1.1.0"
    digest: "sha256:c3ab8ff13720e8ad9047dd39466b3c8974e592c2fa383d4a3960714caef0c4f2"
metadata:
  labels:
    team: platform-engineering
    tier: production
  tags: ["kubernetes", "diagnostics", "sre", "incident-response", "network-policy"]
  category: infrastructure/diagnostics
  authors:
    - name: Platform Engineering Team
      email: platform@acme.dev
  license: Apache-2.0
  repository:
    url: https://github.com/acme/cluster-doctor
    source: github
capabilities:
  protocols: ["mcp", "a2a"]
  inputModalities: ["text"]
  outputModalities: ["text"]
  streaming: true
  multiTurn: true
bom:
  models:
    - name: granite-3.2-8b-instruct
      provider: ibm
      role: primary
    - name: llama-3.1-70b-instruct
      provider: meta
      role: fallback
  tools:
    - name: acme/kubernetes-mcp
      version: ">=2.0.0"
    - name: acme/prometheus-mcp
      version: ">=1.0.0"
  skills:
    - name: acme/k8s-troubleshooting
      version: ">=1.0.0"
YAML

info "Pushing acme/cluster-doctor@1.1.0..."
run $ARCTL push agents "$TMPDIR/agent-v2.yaml"

separator

narrate "Now we have two versions. Let's list them both:"
run $ARCTL list agents

separator

narrate "v1.0.0 is published. v1.1.0 is in draft, ready to go through"
narrate "its own evaluation and promotion cycle independently."

echo ""
pause

# ============================================================================
#  ACT 10 — REST API & WEB UI
# ============================================================================

act "ACT 10: REST API & Web UI"

narrate "Everything agentctl does is powered by the REST API."
narrate "Any HTTP client can interact with the registry directly."
narrate ""
narrate "The API follows a consistent response envelope pattern:"
narrate "  { data: ..., _meta: { requestId, timestamp }, pagination: { ... } }"

separator

narrate "GET all agents via curl:"
echo -e "\n${CYAN}  \$ curl -s http://localhost:$SERVER_PORT/api/v1/agents | jq .${NC}"
curl -s "http://localhost:$SERVER_PORT/api/v1/agents" | python3 -m json.tool 2>/dev/null | sed 's/^/  /' | head -30
echo -e "  ${DIM}... (truncated)${NC}"
echo ""

separator

narrate "Cross-kind search via the API:"
echo -e "\n${CYAN}  \$ curl -s 'http://localhost:$SERVER_PORT/api/v1/search?q=kubernetes' | jq .${NC}"
curl -s "http://localhost:$SERVER_PORT/api/v1/search?q=kubernetes" | python3 -m json.tool 2>/dev/null | sed 's/^/  /' | head -30
echo -e "  ${DIM}... (truncated)${NC}"
echo ""

separator

narrate "The web UI is available at http://localhost:$SERVER_PORT"
narrate "It provides a visual dashboard with:"
narrate "  - Tabbed views for agents, skills, and MCP servers"
narrate "  - Search bar for cross-kind discovery"
narrate "  - Status badges (draft, evaluated, approved, published)"
narrate "  - Color-coded kind indicators"
narrate "  - Click-through to detailed artifact views"

echo ""
pause

# ============================================================================
#  ACT 11 — CLEANUP COMMANDS
# ============================================================================

act "ACT 11: Cleanup — Delete Draft Artifacts"

narrate "Draft artifacts CAN be deleted (only drafts — immutability applies"
narrate "to everything from 'evaluated' onward)."
narrate ""
narrate "Let's clean up the draft v1.1.0 we pushed in Act 9:"

separator

run $ARCTL delete agents acme/cluster-doctor 1.1.0

narrate "Verify it's gone:"
run $ARCTL list agents

separator

info "Only the published v1.0.0 remains."

echo ""
pause

# ============================================================================
#  EPILOGUE
# ============================================================================

echo ""
echo -e "${BLUE}${BOLD}┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓${NC}"
echo -e "${BLUE}${BOLD}┃  DEMO COMPLETE                                                             ┃${NC}"
echo -e "${BLUE}${BOLD}┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛${NC}"
echo ""
echo -e "  ${BOLD}What we covered:${NC}"
echo ""
echo -e "  ${GREEN}ACT 1${NC}   Built agentctl and started a registry server"
echo -e "  ${GREEN}ACT 2${NC}   Pushed 4 artifacts: 1 agent, 1 skill, 2 MCP servers"
echo -e "  ${GREEN}ACT 3${NC}   Discovered artifacts via list, search, and get"
echo -e "  ${GREEN}ACT 4${NC}   Attached 3 eval records (safety, red-team, functional)"
echo -e "  ${GREEN}ACT 5${NC}   Promoted through lifecycle: draft -> evaluated -> approved -> published"
echo -e "  ${GREEN}ACT 6${NC}   Inspected artifact with eval summary and promotion history"
echo -e "  ${GREEN}ACT 7${NC}   Resolved the dependency graph from the BOM"
echo -e "  ${GREEN}ACT 8${NC}   Verified immutability guardrails (can't delete published artifacts)"
echo -e "  ${GREEN}ACT 9${NC}   Pushed a second version (independent lifecycle)"
echo -e "  ${GREEN}ACT 10${NC}  Demonstrated REST API and web UI access"
echo -e "  ${GREEN}ACT 11${NC}  Cleaned up draft artifacts"
echo ""
echo -e "  ${BOLD}Available commands:${NC}"
echo ""
echo -e "  ${CYAN}agentctl init${NC}           Auto-generate manifests from your project (LLM-powered)"
echo -e "  ${CYAN}agentctl push${NC}           Push artifacts from YAML manifests"
echo -e "  ${CYAN}agentctl list${NC}           List artifacts by kind with filters"
echo -e "  ${CYAN}agentctl search${NC}         Cross-kind full-text search"
echo -e "  ${CYAN}agentctl get${NC}            Fetch full artifact details"
echo -e "  ${CYAN}agentctl eval attach${NC}    Attach evaluation records"
echo -e "  ${CYAN}agentctl eval list${NC}      List evaluation records"
echo -e "  ${CYAN}agentctl promote${NC}        Promote through lifecycle stages"
echo -e "  ${CYAN}agentctl inspect${NC}        View status, evals, promotion history"
echo -e "  ${CYAN}agentctl deps${NC}           Resolve dependency graph"
echo -e "  ${CYAN}agentctl delete${NC}         Delete draft artifacts"
echo -e "  ${CYAN}agentctl config${NC}         Manage CLI configuration"
echo -e "  ${CYAN}agentctl server start${NC}   Start the registry server"
echo ""
separator
echo ""
echo -e "  ${BOLD}Server is still running at:${NC}"
echo -e "    API: ${CYAN}http://localhost:$SERVER_PORT/api/v1${NC}"
echo -e "    UI:  ${CYAN}http://localhost:$SERVER_PORT${NC}"
echo ""
echo -e "  ${DIM}Try:  curl -s http://localhost:$SERVER_PORT/api/v1/agents | jq .${NC}"
echo -e "  ${DIM}Stop: Press Ctrl+C${NC}"
echo ""

# Keep server running
wait
