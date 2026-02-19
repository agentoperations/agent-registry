# Agent Registry Spec — Project Instructions

## What This Project Is

A vendor-neutral, portable Agent Registry specification and implementation. The registry handles three first-class artifact types: **agents**, **skills**, and **MCP servers** — with unified versioning, promotion lifecycle, trust signals, and discovery API.

## Spec-First Development

The specification is the source of truth. Always reference it before implementing.

### Spec Files (committed as foundation)

* `SPEC.md` — Full 12-section specification (3,773 lines). **Read the relevant section before implementing any component.**
* `schemas/` — 8 JSON Schema files (draft 2020-12) for validation
* `examples/` — 7 YAML examples across all tiers and artifact kinds
* `api/openapi.yaml` — OpenAPI 3.1 REST API spec (2,246 lines)

### Spec Section Map

When implementing a feature, read the corresponding SPEC.md section first:

| Feature Area | SPEC.md Section | Lines |
|---|---|---|
| Data model, terminology | Section 1: Core Concepts | ~26-128 |
| Artifact schemas, tiers, kind extensions | Section 2: Registry Artifact Schema | ~130-942 |
| Dependencies (models, tools, skills, prompts) | Section 3: Bill of Materials | ~944-1240 |
| Semver, immutability, version rules | Section 4: Version Model | ~1242-1334 |
| draft→evaluated→approved→published→deprecated→archived | Section 5: Promotion Lifecycle | ~1336-1467 |
| REST endpoints, response envelope | Section 6: REST API | ~1469-2170 |
| OCI annotations, media types, layers | Section 7: OCI Integration | ~2172-2344 |
| EvalRecord schema, aggregation | Section 8: Eval Results | ~2346-2573 |
| SLSA, Sigstore, SBOM, provenance | Section 9: Provenance | ~2575-2903 |
| Composite trust score, 6 components | Section 10: Trust Signals | ~2905-3147 |
| Roles, permissions, namespace ownership | Section 11: Access Control | ~3149-3373 |
| Peer discovery, sync, cross-registry search | Section 12: Federation | ~3375-3704 |

### JSON Schemas

* `schemas/registry-artifact.schema.json` — Base with `kind` discriminator (oneOf agent/skill/mcp-server)
* `schemas/agent.schema.json` — Agent: capabilities, runtime, agent BOM ref
* `schemas/skill.schema.json` — Skill: content (entrypoint, scripts, references, assets, triggerPhrases), no runtime
* `schemas/mcp-server.schema.json` — MCP server: transport, tools, resources, prompts
* `schemas/bom.schema.json` — Kind-aware BOM (AgentBOM, SkillBOM, MCPServerBOM via $defs)
* `schemas/eval-record.schema.json` — Benchmark, evaluator, results, context, attestation
* `schemas/provenance.schema.json` — SLSA, in-toto, Sigstore, SBOM, agent-specific extensions
* `schemas/trust-signals.schema.json` — 6-component weighted composite score

## Key Design Decisions

* **Artifacts are metadata, not payloads.** Registry stores structured metadata referencing OCI registries. Never store binary content in the registry itself.
* **Three artifact kinds** share identity/metadata/versioning/promotion/trust. They differ in BOM structure and runtime contract:
  * Agents have `runtime` (runnable containers)
  * Skills have `content` (prompt bundles, FROM scratch, not runnable)
  * MCP servers have `transport`/`tools`/`resources`/`prompts`
* **Progressive disclosure tiers**: Tier 0 (6 fields), Tier 1 (metadata+capabilities), Tier 2 (full governance)
* **BOM captures dependency graph**: agents→skills→MCP servers with transitive resolution and deduplication
* **Immutability after evaluation**: content locked once status >= `evaluated`
* **OCI-native**: `ai.agentregistry.*` annotation namespace, kind-specific media types
* **Promotion lifecycle**: draft→evaluated→approved→published→deprecated→archived with configurable gates

## Relationship Model

The BOM is how we capture relationships between artifacts:

* **Agent BOM** declares: models (with role/provider/pinning), tools (MCP servers with version constraints + specific capabilities used), skills (by name + version), prompts (with content hashes), orchestration config, memory, session, identity
* **Skill BOM** declares: dependencies (other skills), toolRequirements (abstract — not tied to specific MCP servers), modelRequirements (advisory capabilities)
* **MCP Server BOM** declares: runtimeDependencies, externalServices, modelDependencies
* **Dependency graph API**: `GET /api/v1/{kind}/{name}/versions/{version}/dependencies` resolves the full transitive tree with deduplication and `referencedBy` tracking

## Trust Signals — 6 Components

Composite score (0-1) from: Provenance (0.25), Evaluation (0.25), Code Health (0.20), Publisher Reputation (0.15), Community (0.10), Operational (0.05). Weights are transparent and configurable per registry instance.

## Inspiration Codebase

The `agentregistry` repo at `~/go/src/github.com/agentregistry` was used for inspiration during spec design. It contains Go-based implementations of similar concepts (AgentJSON, SkillJSON, ServerJSON, auth, deployment). Reference it for patterns and ideas, but this project is independent.

## Implementation Notes

* Language: TBD (to be decided when implementation starts)
* The OpenAPI spec at `api/openapi.yaml` defines the full REST API surface — use it to generate server stubs if desired
* All JSON schemas validate with draft 2020-12 and use `$id` URIs under `https://agentregistry.dev/schemas/`
* Examples in `examples/` are validated against the schemas and can be used as test fixtures
