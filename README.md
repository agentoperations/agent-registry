# Agent Registry

A vendor-neutral, agent-framework-agnostic registry for AI agents, skills, and MCP servers.

Built on open standards: [A2A AgentCard](https://github.com/google-a2a/A2A) for agents, [MCP server.json](https://github.com/modelcontextprotocol/registry) for MCP servers, and [Agent Skills](https://agentskills.io) for skills.

Stores metadata alongside OCI registries. Accepts evaluation signals from external tools. Enforces a promotion lifecycle. Works with any agent framework. Does not store binaries or compute trust scores.

![Architecture](docs/architecture.png)

## Quick Start

```bash
go build -o agentctl ./cmd/agentctl
./agentctl server start
open http://localhost:8080
```

### Demo

[![Demo](docs/screenshots/demo-thumbnail.png)](https://github.com/agentoperations/agent-registry/releases/download/v0.1.0/demo-preview.mp4)

[Full narrated walkthrough (2 min)](https://github.com/agentoperations/agent-registry/releases/download/v0.1.0/agent-registry-demo.mp4)

### Presentation

Slide deck covering the problem, architecture, and workflow:

[View presentation](https://agentoperations.github.io/agent-registry/presentation.html)

### Web UI

The server embeds a catalog UI at the root URL. Browse artifacts by kind, filter by status, search across all kinds, and click through to see identity, OCI references, eval records, promotion history, and resolved dependencies.

![Catalog view](docs/screenshots/ui-catalog.png)

![Detail panel — eval records, promotion history, dependencies](docs/screenshots/ui-detail.png)

## How It Works

```
You build an agent        agentctl does the rest            Others find it
──────────────────        ────────────────────              ──────────────

docker push image    ──>  agentctl init --path .            agentctl search "k8s"
                          (LLM reads your code,             agentctl inspect ...
                           generates manifest)              agentctl deps ...
                                                            agentctl get ... -> docker pull
                     ──>  agentctl push agents m.yaml
                          (draft — private, mutable)

                     ──>  agentctl eval attach ...          Evals are optional.
                          (external tools submit results)   Any tool can POST /evals.

                     ──>  agentctl promote --to evaluated   Content locks here.
                     ──>  agentctl promote --to approved
                     ──>  agentctl promote --to published   Now discoverable.
```

### Step by step

**Generate a manifest.** Point `agentctl init` at your project. An LLM scans source code, Dockerfile, dependencies, and README to produce a complete YAML manifest. No manual YAML writing.

```bash
agentctl config set init.provider openai
agentctl config set init.model gpt-4o-mini
agentctl init --path ./my-agent --image ghcr.io/acme/my-agent:1.0.0 -o manifest.yaml
```

Works with Anthropic (Messages API), OpenAI (Chat Completions, Responses API), or any compatible endpoint (Ollama, vLLM, LiteLLM) via `--base-url`.

**Push it.** Lands in `draft` status.

```bash
agentctl push agents manifest.yaml
```

**Attach eval records (optional).** The registry stores results from external tools. It does not run evaluations.

```bash
agentctl eval attach agents acme/my-agent 1.0.0 \
    --category safety --provider garak --benchmark toxicity --score 0.96
```

**Promote.** Each step is a valid transition. Content becomes immutable after `evaluated`.

```bash
agentctl promote agents acme/my-agent 1.0.0 --to evaluated
agentctl promote agents acme/my-agent 1.0.0 --to approved
agentctl promote agents acme/my-agent 1.0.0 --to published
```

**Discover, inspect, deploy.**

```bash
agentctl search "kubernetes diagnostics"
agentctl inspect agents acme/my-agent 1.0.0
agentctl deps agents acme/my-agent 1.0.0
agentctl get agents acme/my-agent 1.0.0   # read OCI ref, then docker pull
```

## CLI

```
agentctl config init                     Interactive LLM provider setup
agentctl config set <key> <value>        Set config value
agentctl config show                     Show config

agentctl init -p ./project -o out.yaml   Generate manifest from code (LLM)
agentctl push <kind> <file>              Register artifact (draft)
agentctl push <kind> <file> --namespace <ns> --oci <ref>   Push standard doc (AgentCard, server.json, SKILL.md)
agentctl get <kind> <name> [version] [--format a2a|server-json|skill-md]     Get artifact details
agentctl list <kind>                     List artifacts
agentctl delete <kind> <name> <version>  Delete draft
agentctl promote <kind> <name> <ver> --to <status>
agentctl eval attach <kind> <name> <ver> --category --provider --benchmark --score
agentctl eval list <kind> <name> <ver>
agentctl inspect <kind> <name> <ver>     Status + evals + promotion history
agentctl deps <kind> <name> <ver>        Dependency graph from BOM
agentctl search <query>                  Full-text search across all kinds
agentctl import --from-a2a <url> --namespace --oci   Import agent from A2A AgentCard URL
agentctl server start [--port] [--db]    Start server (UI at /, API at /api/v1)
```

`<kind>` is `agents`, `skills`, or `mcp-servers`.

## API

All under `/api/v1`. Responses wrapped in `{ "data": ..., "_meta": {...}, "pagination": {...} }`. Errors follow RFC 7807.

| Method | Path | |
|--------|------|-|
| GET | `/{kind}` | List |
| POST | `/{kind}` | Create (draft) |
| GET | `/{kind}/{ns}/{name}/versions/{ver}` | Get |
| DELETE | `/{kind}/{ns}/{name}/versions/{ver}` | Delete (draft only) |
| POST | `/{kind}/{ns}/{name}/versions/{ver}/promote` | Promote |
| POST | `/{kind}/{ns}/{name}/versions/{ver}/evals` | Submit eval |
| GET | `/{kind}/{ns}/{name}/versions/{ver}/evals` | List evals |
| GET | `/{kind}/{ns}/{name}/versions/{ver}/inspect` | Inspect |
| GET | `/{kind}/{ns}/{name}/versions/{ver}/dependencies` | Deps |
| GET | `/search?q=...` | Search |
| GET | `/ping` | Health |

## Deployment

### Local

```bash
go build -o agentctl ./cmd/agentctl
./agentctl server start --port 8080
# API at http://localhost:8080/api/v1, UI at http://localhost:8080
```

### Container

```bash
make image                    # build image (multi-stage, distroless runtime)
make image push               # build and push to quay.io
```

The image is `quay.io/azaalouk/agent-registry`. Override with `IMAGE_REPO` and `IMAGE_TAG`:

```bash
make image push IMAGE_REPO=quay.io/myorg IMAGE_TAG=v0.2.0
```

### Kubernetes

```bash
make deploy OVERLAY=k8s
```

This creates a namespace `agent-registry` with a Deployment, Service, and health probes. Access via port-forward:

```bash
kubectl -n agent-registry port-forward svc/agent-registry 8080:8080
open http://localhost:8080
```

### OpenShift

```bash
make deploy                   # OVERLAY=openshift is the default
```

This creates the namespace, Deployment, Service, and a TLS Route. Get the URL:

```bash
oc -n agent-registry get route agent-registry -o jsonpath='{.spec.host}'
```

The deployment runs as non-root with a read-only root filesystem, passes the `restricted` SCC without any grants.

### Tear down

```bash
make undeploy                 # removes all resources
```

### Makefile targets

| Target | Description |
|--------|-------------|
| `make build` | Compile binary locally |
| `make image` | Build container image |
| `make push` | Push to container registry |
| `make deploy` | Create namespace and apply manifests |
| `make undeploy` | Delete all deployed resources |
| `make clean` | Remove local build artifacts |

## Project Layout

```
cmd/agentctl/       Entry point (thin — embeds UI, calls cli package)
cmd/server/         Standalone server entry point
internal/
  cli/              Cobra commands (HTTP calls + output formatting only)
  handler/          HTTP handlers (parse request, call service, write response)
  service/          Business logic (promotion state machine, BOM resolution)
  store/            Storage interface + SQLite (swappable to Postgres)
  model/            Go types matching spec schemas
  server/           Chi router, middleware, serves embedded UI
pkg/client/         HTTP client (used by CLI, usable by any Go program)
ui/index.html       Catalog UI (embedded in binary, served at /)
schemas/            JSON Schemas (spec source of truth)
api/openapi.yaml    OpenAPI 3.1 spec
deploy/
  Dockerfile        Multi-stage build (distroless runtime)
  k8s/base/         Deployment, Service, Kustomization
  k8s/overlays/     openshift (+ Route) and k8s overlays
Makefile            build, image, push, deploy, undeploy
scripts/            Demo setup/teardown scripts
docs/               Presentation, architecture diagrams, screenshots
```

## Design

**Metadata, not payloads.** The registry indexes OCI artifacts. Binary content stays in ghcr.io / quay.io / ECR.

**Three artifact kinds** share identity, versioning, and promotion lifecycle. They differ in BOM structure: agents declare models + tools + skills, skills declare tool requirements, MCP servers declare transport + tools.

**Evals are external signals.** The registry accepts eval records from any tool (Garak, eval-hub, lm-eval-harness, custom CI). It does not run evaluations or compute trust scores. Trust scoring is a separate concern.

**Promotion is a state machine.** `draft -> evaluated -> approved -> published -> deprecated -> archived`. No hardcoded gates. Policy layers can be added externally.

**CLI is a thin client.** All logic is server-side. Any entry point (CLI, curl, UI, CI pipeline) behaves identically.

**Store is swappable.** SQLite for dev, Postgres for production. Implement the `Store` interface.

## Config

```
~/.config/agentctl/config.yaml

AGENTCTL_SERVER       Registry URL (default http://localhost:8080)
ANTHROPIC_API_KEY     For init with Anthropic
OPENAI_API_KEY        For init with OpenAI
```
