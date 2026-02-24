# Standards-Based Schema Redesign

**Date:** 2026-02-24
**Status:** Proposed
**Authors:** Adel Zaalouk
**Context:** Tiger Team feedback from Matt Prahl, Varsha, Morgan Foster on A2A AgentCard adoption

## Motivation

The agent registry currently defines its own identity and capability schema for agents, MCP servers, and skills. Three open standards now cover this space:

| Artifact kind | Standard | Steward |
|---|---|---|
| Agent | [A2A AgentCard](https://github.com/google-a2a/A2A) | Google, AAIF/Linux Foundation |
| MCP Server | [MCP server.json](https://github.com/modelcontextprotocol/registry) | Anthropic, AAIF/Linux Foundation |
| Skill | [Agent Skills (SKILL.md)](https://agentskills.io/specification) | Anthropic (30+ platform adoption) |

The registry should not reinvent identity and capability schemas. It should adopt these standards as the native format for each artifact kind and focus on what no standard provides: governance, lifecycle promotion, evaluation, supply chain visibility, and trust signals.

This aligns directly with the Tiger Team proposal: "each agent version IS an A2A AgentCard."

## Design Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Integration pattern | Envelope model | Standard doc inside, governance outside. No field duplication. Clean import/export. |
| Storage | Embed inline + origin URL | Self-contained snapshots with verifiability. Re-fetch origin to detect drift. |
| Naming | Keep namespace/name | Simpler than reverse-DNS. Map to/from reverse-DNS on MCP server.json import/export. |
| BOM scope | Registry layer only | Standard docs own identity/capabilities. Registry owns dependency graph. |
| Migration | Backward compatible | Legacy format accepted for one major version. Migration command provided. |

## Architecture

```
+-------------------------------------------------------------+
|                  Registry Governance Envelope                |
|                                                              |
|  +--------------------------------------------------------+  |
|  |  Standard Identity Document                            |  |
|  |                                                        |  |
|  |  agent      -> A2A AgentCard                           |  |
|  |  mcp-server -> MCP server.json                         |  |
|  |  skill      -> Agent Skills (SKILL.md)                 |  |
|  +--------------------------------------------------------+  |
|                                                              |
|  + namespace           (registry scoping)                    |
|  + artifacts[]         (OCI content references)              |
|  + bom                 (dependency graph / supply chain)     |
|  + promotionStatus     (draft->evaluated->...->archived)     |
|  + evalRecords[]       (external evaluation signals)         |
|  + provenance          (SLSA, Sigstore, SBOM)                |
|  + trustSignals        (composite quality score)             |
|  + metadata            (labels, tags, category, license)     |
|  + evalBaseline        (quality gates for promotion)         |
|  + promotionHistory[]  (audit trail)                         |
|                                                              |
+-------------------------------------------------------------+
```

The standard document answers "what is this thing and how do I use it?"
The governance envelope answers "should I trust it, what does it need, and who says it's ready?"

## Governance Envelope Fields

### 1. namespace
Registry scoping for multi-tenancy. The fully qualified registry identifier is `{namespace}/{name-from-standard-doc}`. None of the three standards handle multi-tenant registry scoping.

### 2. artifacts[]
OCI content references (container images, skill content archives). Points to where the runnable bits live. A2A AgentCard has `supportedInterfaces[].url` (a running endpoint, not a pullable image). MCP server.json has `packages[]` (npm/pypi/oci install targets). The registry's `artifacts[]` adds digest pinning and platform constraints.

### 3. bom
Bill of Materials declaring all dependencies: models, tools, skills, prompts, orchestration framework, external services. No standard covers "this agent needs Claude Sonnet, the GitHub MCP server, and the code-review skill to function."

### 4. promotionStatus
Six-state lifecycle: `draft -> evaluated -> approved -> published -> deprecated -> archived`. Content becomes immutable after `evaluated`. A2A has no lifecycle concept. MCP Registry has `active/deprecated/deleted` (three states, no evaluation gate).

### 5. evalRecords[]
Results from external evaluation tools (Garak, eval-hub, lm-eval-harness, custom CI). The registry accepts but does not run evaluations. Multiple records per version, covering safety, functional, red-team, and performance categories.

### 6. provenance
SLSA levels, Sigstore signatures, in-toto attestations, SBOM references. Complements A2A AgentCard `signatures[]` (card-level JWS) with build-level provenance (source commit, build platform, material list, transparency log).

### 7. trustSignals
Composite 0.0-1.0 score from six weighted components: provenance verification, eval results, code health, publisher reputation, community engagement, operational metrics.

### 8. metadata
Labels, tags, category, authors, license, repository URL. Where the standard doc already declares a field (e.g., AgentCard `documentationUrl`, SKILL.md `license`), the standard doc is the source of truth. Registry metadata fills gaps.

### 9. evalBaseline
Minimum evaluation requirements for promotion. Quality gates that no standard provides.

### 10. promotionHistory[]
Timestamped audit trail of every promotion event.

## Artifact Schema

### Agent

```yaml
kind: agent
agentCard:                              # A2A AgentCard (verbatim)
  name: "Recipe Agent"
  description: "Agent that helps with recipes"
  version: "1.0.0"
  supportedInterfaces:
    - url: "https://api.example.com/a2a/v1"
      protocolBinding: "JSONRPC"
      protocolVersion: "1.0"
  provider:
    organization: "Acme Corp"
    url: "https://acme.example.com"
  capabilities:
    streaming: true
    pushNotifications: false
  defaultInputModes: ["text/plain", "application/json"]
  defaultOutputModes: ["text/plain", "application/json"]
  skills:
    - id: "find-recipe"
      name: "Find Recipe"
      description: "Finds recipes based on ingredients"
      tags: ["cooking", "recipes"]
      examples: ["Find me a pasta recipe"]
  securitySchemes:
    bearerAuth:
      httpAuthSecurityScheme:
        scheme: "Bearer"
        bearerFormat: "JWT"
  signatures: []
agentCardOrigin: "https://recipe-agent.example.com/.well-known/agent-card.json"

# Registry governance envelope
namespace: "acme"
artifacts:
  - oci: "ghcr.io/acme/recipe-agent:1.0.0"
    digest: "sha256:abc123..."
bom:
  models:
    - name: "claude-sonnet-4-20250514"
      provider: "anthropic"
      role: "primary"
  tools:
    - name: "acme/recipe-db-mcp"
      version: ">=1.0.0"
  orchestration:
    framework: "adk"
    pattern: "tool-use-loop"
metadata:
  labels: { team: "ai-platform" }
  tags: ["cooking", "nlp"]
  category: "conversational"
  license: "Apache-2.0"
  repository:
    url: "https://github.com/acme/recipe-agent"
provenance: { ... }
evalBaseline: { ... }
```

### MCP Server

```yaml
kind: mcp-server
serverJson:                             # MCP server.json (verbatim)
  name: "io.github.acme/weather-mcp"
  description: "Weather data via MCP"
  version: "2.1.0"
  title: "Weather MCP Server"
  repository:
    url: "https://github.com/acme/weather-mcp"
    source: "github"
  packages:
    - registryType: "npm"
      identifier: "@acme/weather-mcp"
      version: "2.1.0"
      transport: { type: "stdio" }
  remotes:
    - type: "streamable-http"
      url: "https://weather-mcp.example.com/mcp"
  icons:
    - src: "https://example.com/weather-icon.png"
      mimeType: "image/png"
serverJsonOrigin: "https://registry.modelcontextprotocol.io/..."

# Registry governance envelope
namespace: "acme"
artifacts:
  - oci: "ghcr.io/acme/weather-mcp:2.1.0"
bom:
  runtimeDependencies:
    - name: "node"
      version: ">=18.0.0"
      type: "runtime"
  externalServices:
    - name: "OpenWeatherMap API"
      type: "api"
      required: true
metadata:
  tags: ["weather", "data"]
  category: "data-analysis"
provenance: { ... }
```

### Skill

```yaml
kind: skill
skillMd:                                # Agent Skills SKILL.md (parsed)
  name: "code-review"
  description: "Performs thorough code reviews with security focus"
  license: "Apache-2.0"
  compatibility: "Claude Code, Cursor, VS Code"
  metadata:
    author: "Jane Doe"
    version: "1.2.0"
  body: |
    ## Overview
    Review code changes for security, correctness, and style...
skillMdOrigin: "https://github.com/acme/skills/tree/main/code-review"

# Registry governance envelope
namespace: "acme"
artifacts:
  - oci: "ghcr.io/acme/skills/code-review:1.2.0"
bom:
  toolRequirements:
    - name: "github"
      required: true
      operations: ["list_pull_requests", "add_comment"]
  modelRequirements:
    - capability: "code-analysis"
      minContextWindow: 100000
metadata:
  tags: ["code-review", "security"]
  category: "coding"
provenance: { ... }
```

## Field Extraction and Indexing

The registry extracts fields from standard documents at push time for indexing and search. The standard doc is stored verbatim; extracted fields are indexed separately, not duplicated into registry fields.

| Extracted field | Agent (AgentCard) | MCP Server (server.json) | Skill (SKILL.md) |
|---|---|---|---|
| Display name | `name` | `title` or `name` | `name` (frontmatter) |
| Version | `version` | `version` | `metadata.version` |
| Description | `description` | `description` | `description` (frontmatter) |
| Search text | `skills[].name/description/tags` | `title` + `description` | `description` + `body` keywords |

## Validation Rules

Each kind validates its standard document against the corresponding spec:

| Kind | Required fields (per standard) |
|---|---|
| `agent` | `name`, `version`, `description`, `supportedInterfaces`, `capabilities`, `defaultInputModes`, `defaultOutputModes`, `skills` |
| `mcp-server` | `name`, `version`, `description` |
| `skill` | `name`, `description` (frontmatter) |

The registry enforces what the standard requires and nothing more for the standard document. Governance fields have their own validation (namespace, OCI references, BOM semver ranges) unchanged from the current spec.

## CLI Changes

### Push accepts standard documents directly

```bash
# Agent: push A2A AgentCard JSON
agentctl push agents agent-card.json \
  --namespace acme \
  --oci ghcr.io/acme/recipe-agent:1.0.0

# MCP Server: push server.json
agentctl push mcp-servers server.json \
  --namespace acme \
  --oci ghcr.io/acme/weather-mcp:2.1.0

# Skill: push SKILL.md directory
agentctl push skills ./code-review/ \
  --namespace acme \
  --oci ghcr.io/acme/skills/code-review:1.2.0

# Full registry manifest still works
agentctl push agents full-manifest.yaml
```

### Init generates standard document + envelope

```bash
agentctl init --path ./my-agent --image ghcr.io/acme/my-agent:1.0.0
# Output: agent-card.json (valid A2A AgentCard) + manifest.yaml (registry envelope)
```

### Export standard document

```bash
agentctl get agents acme/recipe-agent 1.0.0 --format a2a
# Outputs pure A2A AgentCard JSON

agentctl get mcp-servers acme/weather-mcp 2.1.0 --format server-json
# Outputs pure MCP server.json

agentctl get skills acme/code-review 1.2.0 --format skill-md
# Outputs SKILL.md content
```

### Import from standard source

```bash
# Import from running agent's well-known endpoint
agentctl import --from-a2a https://agent.example.com/.well-known/agent-card.json \
  --namespace acme --oci ghcr.io/acme/my-agent:1.0.0

# Import from MCP Registry
agentctl import --from-mcp-registry io.github.acme/weather-mcp \
  --namespace acme

# Import from skills repo
agentctl import --from-skills https://github.com/acme/skills/tree/main/code-review \
  --namespace acme
```

### Unchanged commands

`promote`, `eval attach`, `eval list`, `inspect`, `deps`, `search`, `list`, `delete`, `config`, `server start` -- all operate on governance fields, not the standard document.

## API Changes

### Modified endpoints

**POST `/{kind}`** -- accepts two payload formats:

1. Full envelope (standard doc + governance fields in one JSON body)
2. Just the standard document (with namespace and OCI ref as query params)

**GET `/{kind}/{ns}/{name}/versions/{ver}`** -- response includes the standard document under `agentCard`, `serverJson`, or `skillMd` field.

### New endpoints

**GET `/{kind}/{ns}/{name}/versions/{ver}/export`** -- returns the pure standard document with no governance wrapping. This is what deployment systems call to get the A2A card for `/.well-known/agent-card.json`.

**POST `/{kind}/import`** -- accepts a source URL, fetches the standard document, validates it, wraps it in a governance envelope, and creates a draft artifact.

### Unchanged endpoints

| Endpoint | Reason |
|---|---|
| `POST /{kind}/{ns}/{name}/versions/{ver}/promote` | Operates on promotionStatus |
| `POST /{kind}/{ns}/{name}/versions/{ver}/evals` | Writes evalRecords |
| `GET /{kind}/{ns}/{name}/versions/{ver}/evals` | Reads evalRecords |
| `GET /{kind}/{ns}/{name}/versions/{ver}/inspect` | Reads governance summary |
| `GET /{kind}/{ns}/{name}/versions/{ver}/dependencies` | Reads BOM |
| `DELETE /{kind}/{ns}/{name}/versions/{ver}` | Deletes draft |
| `GET /search?q=...` | Indexes from standard doc + metadata |
| `GET /ping` | Health check |

## Migration Path

1. **Backward compatible.** `POST /{kind}` accepts both the new envelope format and the legacy format. Legacy pushes auto-generate a minimal standard document from existing `identity` and `capabilities` fields.

2. **Migration command.** `agentctl migrate {kind} {name} {version}` reads existing identity/capabilities from a draft artifact and generates the corresponding standard document.

3. **Deprecation timeline.** Legacy `identity` + `capabilities` schema accepted for one major version. After v2.0, all pushes require the envelope format with a standard document.

4. **JSON Schema updates.** `registry-artifact.schema.json` gains `agentCard`, `serverJson`, `skillMd` fields. The `identity` field becomes optional (legacy mode only). New schemas reference external standard schemas via `$ref`.

## What This Enables

| Tiger Team feedback | How addressed |
|---|---|
| "Each agent version is an A2A AgentCard" (Matt) | AgentCard is the native identity format, stored verbatim |
| "Cross-entity linking to prompts and models" (Matt) | BOM in governance envelope + AgentCard skills |
| "Intercept AgentCard declarations -> upsert to registry" (Matt) | `POST /agents/import` fetches and wraps cards |
| "Intercept `/.well-known/agent.json` -> register" (Matt) | `agentctl import --from-a2a <url>` |
| "Agent card signatures for attestation" (Morgan) | AgentCard `signatures[]` preserved, complemented by registry provenance |
| "Tying agent card to running workload cryptographically" (Morgan) | A2A JWS signatures + registry SLSA/Sigstore provenance |
| "Kagenti auto-discovery without MLflow SDK" (Matt) | `GET /export` returns pure AgentCard, servable at `/.well-known/` |
| "Robust attestation story for each agent" (Morgan) | Three layers: A2A card signatures + registry provenance + eval attestations |

## Standards References

- A2A AgentCard: `specification/a2a.proto` in [google-a2a/A2A](https://github.com/google-a2a/A2A)
- MCP server.json: [modelcontextprotocol/registry](https://github.com/modelcontextprotocol/registry), schema at `docs/reference/server-json/draft/server.schema.json`
- Agent Skills: [agentskills.io/specification](https://agentskills.io/specification), reference at [anthropics/skills](https://github.com/anthropics/skills)
- A2A Protocol: [a2a-protocol.org/latest/specification](https://a2a-protocol.org/latest/specification/)
- MCP Registry API: OpenAPI spec at `docs/reference/api/openapi.yaml` in the MCP registry repo
