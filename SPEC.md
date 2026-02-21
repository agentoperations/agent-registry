# Agent Registry Specification v1.0

**Status:** Draft
**Date:** 2026-02-16
**Authors:** Agent Registry Contributors

---

## Table of Contents

1. [Core Concepts](#1-core-concepts)
2. [Registry Artifact Schema](#2-registry-artifact-schema)
3. [Bill of Materials (BOM)](#3-bill-of-materials-bom)
4. [Version Model](#4-version-model)
5. [Promotion Lifecycle](#5-promotion-lifecycle)
6. [REST API](#6-rest-api)
7. [OCI Integration](#7-oci-integration)
8. [Eval Results Interface](#8-eval-results-interface)
9. [Provenance and Supply Chain](#9-provenance-and-supply-chain)
10. [Trust Signals](#10-trust-signals)
11. [Access Control](#11-access-control)
12. [Federation](#12-federation)

---

## 1. Core Concepts

### 1.1 Overview

The Agent Registry is a centralized registry for securely curating, discovering, deploying, and managing agentic infrastructure. It provides governance and control over AI artifacts, empowering developers to build and deploy AI applications with confidence.

This specification defines the data model, API surface, lifecycle management, and trust mechanisms for three artifact kinds: **agents**, **skills**, and **MCP servers**.

### 1.2 Terminology

The following terms are used throughout this specification with precise meaning.

#### 1.2.1 Registry Artifact

A **Registry Artifact** is the supertype for all items stored in the Agent Registry. Every artifact has a common identity envelope (name, version, description) and a `kind` discriminator field. The three artifact kinds are:

* **agent** -- An autonomous or semi-autonomous AI system that uses models, tools, and skills to accomplish tasks.
* **skill** -- A reusable prompt-based capability that can be composed into agents or used directly by AI-powered IDEs.
* **mcp-server** -- A Model Context Protocol server that exposes tools, resources, and prompts to AI agents and clients.

Registry Artifacts are **metadata records** that describe and reference stored content. They are not the running instances themselves. An artifact in the registry describes *what* something is, *where* its runnable content lives (typically in an OCI registry), and *how* to configure it.

#### 1.2.2 Agent

An **Agent** is a registry artifact of kind `agent`. Agents are autonomous or semi-autonomous AI systems that combine one or more models, tools (via MCP servers), skills, and orchestration logic to accomplish tasks. An agent artifact describes the agent's capabilities, runtime requirements, and its bill of materials.

Agents are distinguished from skills in that they have their own runtime (container, process) and can operate independently. They may consume skills and MCP servers as dependencies.

#### 1.2.3 Skill

A **Skill** is a registry artifact of kind `skill`. Skills are reusable, prompt-based capabilities defined primarily through a `SKILL.md` file. They do not have their own runtime -- they are consumed by agents or AI-powered IDEs that execute the skill's instructions.

Skills may declare dependencies on tools (provided by MCP servers) but do not run MCP servers themselves. Their content is textual: prompt templates, reference documents, scripts, and trigger phrases.

#### 1.2.4 MCP Server

An **MCP Server** is a registry artifact of kind `mcp-server`. MCP servers implement the [Model Context Protocol](https://spec.modelcontextprotocol.io/) and expose tools, resources, and prompts to AI clients. They are the building blocks of agent capabilities.

MCP server artifacts describe the transport mechanism (stdio, streamable-http, SSE), available tools, resources, prompts, and packaging information.

#### 1.2.5 Version

A **Version** is a specific release of an artifact, identified by a [Semantic Versioning 2.0.0](https://semver.org/) string. Each artifact can have multiple versions. Versions are immutable once their promotion status reaches `evaluated` or higher.

#### 1.2.6 Bill of Materials (BOM)

A **Bill of Materials** (BOM) is a structured declaration of an artifact's dependencies. The BOM is kind-aware: an agent BOM declares model, tool, skill, prompt, and orchestration dependencies; a skill BOM declares tool and model requirements; an MCP server BOM declares runtime and external service dependencies.

#### 1.2.7 Promotion Status

A **Promotion Status** indicates the lifecycle stage of an artifact version. The six states are:

| Status | Description |
|---|---|
| `draft` | Initial state. Artifact is mutable and only visible to its owner. |
| `evaluated` | Artifact has passed evaluation. Content becomes immutable. |
| `approved` | Artifact has been reviewed and approved by an approver. |
| `published` | Artifact is publicly visible and discoverable. |
| `deprecated` | Artifact is still available but marked as superseded. |
| `archived` | Artifact is no longer available for new consumption. |

#### 1.2.8 Trust Signal

A **Trust Signal** is a composite quality indicator derived from multiple weighted components including provenance verification, evaluation results, code health, publisher reputation, community engagement, and operational metrics. Trust signals produce a normalized score between 0.0 and 1.0.

#### 1.2.9 Registry Namespace

A **Registry Namespace** is a scoping mechanism for artifact names. Namespaces follow a reverse-domain notation (e.g., `io.github.username`, `com.example.org`). The fully qualified name of an artifact is `{namespace}/{name}`.

Namespace rules:
* Must start and end with an alphanumeric character.
* May contain letters, digits, dots (`.`), and hyphens (`-`).
* Minimum 2 characters.
* Ownership is established through one of the supported verification methods (DNS, GitHub, OIDC).

#### 1.2.10 Agent Card

An **Agent Card** is a human-readable summary view of an agent artifact, suitable for display in UIs, catalogs, and search results. It includes the agent's name, title, description, version, trust score, category, and key capabilities. Agent Cards are derived from the artifact schema and are not stored separately.

#### 1.2.11 OCI Reference

An **OCI Reference** is a pointer to a container image or artifact stored in an OCI-compliant registry. The format follows the OCI Distribution Specification:

```
{registry}/{repository}:{tag}
{registry}/{repository}@{digest}
```

The Agent Registry uses OCI references in the `artifacts` field to locate the runnable content (container images, skill content archives) associated with each artifact version.

### 1.3 Design Principles

1. **Artifacts are metadata, not payloads.** The registry stores structured metadata that references content in OCI registries. This separates governance from storage.

2. **Kind polymorphism.** All artifact kinds share a common identity and metadata envelope. The `kind` field discriminates behavior, and kind-specific extensions provide specialized fields.

3. **Progressive disclosure.** The schema is tiered so that simple use cases require only a few fields (Tier 0), while production deployments can leverage the full schema (Tier 2).

4. **Immutability after evaluation.** Once an artifact version has been evaluated, its content and identity fields become immutable. This provides auditability and reproducibility.

5. **OCI-native.** The registry is an index over OCI artifacts, not a replacement for OCI registries. All binary content is stored in OCI-compliant registries.

---

## 2. Registry Artifact Schema

### 2.1 Schema Overview

The Registry Artifact schema is a polymorphic schema with a shared base envelope and kind-specific extensions. It uses three progressive disclosure tiers to balance simplicity with completeness.

```
RegistryArtifact
├── identity          (Tier 0 - required)
├── kind              (Tier 0 - required)
├── artifacts         (Tier 0 - at least one required)
├── metadata          (Tier 1 - optional)
├── capabilities      (Tier 1 - kind-specific, optional)
├── runtime           (Tier 1 - kind-specific, optional)
├── bom               (Tier 1 - optional, Tier 2 - required)
├── content           (Tier 1 - skill-specific, optional)
├── transport         (Tier 1 - mcp-server-specific, optional)
├── tools             (Tier 1 - mcp-server-specific, optional)
├── resources         (Tier 1 - mcp-server-specific, optional)
├── prompts           (Tier 1 - mcp-server-specific, optional)
├── provenance        (Tier 2 - optional)
├── evalBaseline      (Tier 2 - optional)
└── trustSignals      (Tier 2 - optional)
```

### 2.2 Tier 0: Minimal Artifact (6 Required Fields)

Tier 0 is the minimum viable artifact. It requires exactly six fields that establish the artifact's identity and provide at least one OCI reference for its content.

```yaml
# Tier 0: Minimal agent artifact
identity:
  name: "acme/my-agent"
  version: "1.0.0"
  title: "My Agent"
  description: "A helpful assistant agent."
kind: agent
artifacts:
  - oci: "ghcr.io/acme/my-agent:1.0.0"
```

#### 2.2.1 `identity` (required)

The identity block uniquely identifies the artifact.

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Fully qualified artifact name in `{namespace}/{name}` format. See [Section 1.2.9](#129-registry-namespace) for namespace rules. |
| `version` | string | Yes | Semantic version string conforming to [Semver 2.0.0](https://semver.org/). See [Section 4](#4-version-model). |
| `title` | string | Yes | Human-readable display title. Maximum 128 characters. |
| `description` | string | Yes | Brief description of the artifact's purpose. Maximum 512 characters. |

**Name validation rules by kind:**

* **agent**: The name part (after the namespace separator) must start with a letter and contain only letters and digits. Minimum 2 characters. Must not be a Python keyword.
* **mcp-server**: The name part must start and end with an alphanumeric character. May contain letters, digits, dots (`.`), underscores (`_`), and hyphens (`-`). Minimum 2 characters.
* **skill**: The name part may contain letters, digits, underscores (`_`), and hyphens (`-`).

#### 2.2.2 `kind` (required)

The kind discriminator field. Must be one of:

| Value | Description |
|---|---|
| `agent` | An autonomous AI agent. |
| `skill` | A reusable prompt-based capability. |
| `mcp-server` | A Model Context Protocol server. |

The `kind` field determines which kind-specific extensions are valid for the artifact.

#### 2.2.3 `artifacts` (required, at least one entry)

An array of OCI references pointing to the artifact's stored content.

```yaml
artifacts:
  - oci: "ghcr.io/acme/my-agent:1.0.0"
    mediaType: "application/vnd.oci.image.manifest.v1+json"
    digest: "sha256:abc123..."
    platform:
      os: "linux"
      architecture: "amd64"
```

| Field | Type | Required | Description |
|---|---|---|---|
| `oci` | string | Yes | OCI reference in `{registry}/{repository}:{tag}` or `{registry}/{repository}@{digest}` format. |
| `mediaType` | string | No | OCI media type of the artifact. Defaults vary by kind (see [Section 7](#7-oci-integration)). |
| `digest` | string | No | Content-addressable digest (e.g., `sha256:...`). Recommended for immutable references. |
| `platform` | object | No | Target platform constraints. |
| `platform.os` | string | No | Target operating system (e.g., `linux`, `darwin`, `windows`). |
| `platform.architecture` | string | No | Target CPU architecture (e.g., `amd64`, `arm64`). |

### 2.3 Tier 1: Standard Artifact

Tier 1 adds metadata, kind-specific capabilities, runtime configuration, and optional BOM.

#### 2.3.1 `metadata` (optional)

General-purpose metadata about the artifact.

```yaml
metadata:
  labels:
    environment: "production"
    team: "ai-platform"
  tags:
    - "nlp"
    - "chatbot"
    - "customer-support"
  category: "conversational"
  authors:
    - name: "Jane Doe"
      email: "jane@example.com"
      url: "https://github.com/janedoe"
  license: "Apache-2.0"
  repository:
    url: "https://github.com/acme/my-agent"
    source: "github"
  websiteUrl: "https://my-agent.example.com"
  documentation: "https://docs.example.com/my-agent"
```

| Field | Type | Required | Description |
|---|---|---|---|
| `labels` | map[string]string | No | Key-value pairs for organizational classification. Keys must match `[a-zA-Z0-9._-/]+`, max 63 chars. Values max 256 chars. Maximum 64 labels. |
| `tags` | string[] | No | Freeform tags for discovery. Each tag max 64 chars, lowercase alphanumeric with hyphens. Maximum 32 tags. |
| `category` | string | No | Primary category. One of: `conversational`, `coding`, `data-analysis`, `research`, `automation`, `security`, `devops`, `creative`, `education`, `other`. |
| `authors` | Author[] | No | List of artifact authors/maintainers. |
| `authors[].name` | string | Yes | Author display name. |
| `authors[].email` | string | No | Author email address. |
| `authors[].url` | string | No | Author URL (homepage, GitHub profile). |
| `license` | string | No | SPDX license identifier (e.g., `Apache-2.0`, `MIT`). |
| `repository` | object | No | Source code repository. |
| `repository.url` | string | Yes | Repository URL. |
| `repository.source` | string | No | Repository hosting platform (e.g., `github`, `gitlab`, `bitbucket`). |
| `websiteUrl` | string | No | Project website URL. |
| `documentation` | string | No | Documentation URL. |

#### 2.3.2 `capabilities` (optional, kind-specific)

Capabilities describe what the artifact can do. The structure varies by kind.

##### Agent Capabilities

```yaml
# kind: agent
capabilities:
  protocols:
    - "mcp"
    - "a2a"
    - "openai-chat"
  inputModalities:
    - "text"
    - "image"
    - "audio"
  outputModalities:
    - "text"
    - "image"
  streaming: true
  multiTurn: true
  contextWindow: 128000
  maxOutputTokens: 4096
```

| Field | Type | Required | Description |
|---|---|---|---|
| `protocols` | string[] | No | Communication protocols supported. Known values: `mcp`, `a2a`, `openai-chat`, `anthropic-messages`, `custom`. |
| `inputModalities` | string[] | No | Input types accepted. Known values: `text`, `image`, `audio`, `video`, `file`, `structured-data`. |
| `outputModalities` | string[] | No | Output types produced. Same value set as `inputModalities`. |
| `streaming` | boolean | No | Whether the agent supports streaming responses. Default: `false`. |
| `multiTurn` | boolean | No | Whether the agent supports multi-turn conversations. Default: `false`. |
| `contextWindow` | integer | No | Maximum context window size in tokens. |
| `maxOutputTokens` | integer | No | Maximum output tokens per response. |

##### Skill Capabilities

Skills do not have a `capabilities` section. Their capabilities are implicitly defined by their content (SKILL.md, trigger phrases, tool requirements).

##### MCP Server Capabilities

MCP server capabilities are described through the `tools`, `resources`, and `prompts` top-level fields rather than a `capabilities` block. See [Section 2.3.6](#236-mcp-server-specific-fields).

#### 2.3.3 `runtime` (optional, kind-specific)

Runtime configuration describes how to run the artifact. **Skills do not have a `runtime` section** -- they are consumed by a host agent or IDE.

##### Agent Runtime

```yaml
# kind: agent
runtime:
  env:
    - name: "MODEL_PROVIDER"
      description: "LLM provider to use"
      required: true
      default: "anthropic"
    - name: "MODEL_NAME"
      description: "Model identifier"
      required: true
      default: "claude-sonnet-4-20250514"
    - name: "API_KEY"
      description: "Provider API key"
      required: true
      sensitive: true
  ports:
    - containerPort: 8080
      protocol: "TCP"
      description: "HTTP API endpoint"
  healthCheck:
    httpGet:
      path: "/health"
      port: 8080
    initialDelaySeconds: 10
    periodSeconds: 30
    timeoutSeconds: 5
    failureThreshold: 3
  resources:
    requests:
      cpu: "500m"
      memory: "512Mi"
    limits:
      cpu: "2000m"
      memory: "4Gi"
      gpu: "1"
      gpuType: "nvidia-a10g"
  command: ["python", "-m", "my_agent"]
  args: ["--port", "8080"]
  replicas:
    min: 1
    max: 10
```

| Field | Type | Required | Description |
|---|---|---|---|
| `env` | EnvVar[] | No | Environment variables for configuration. |
| `env[].name` | string | Yes | Variable name. |
| `env[].description` | string | No | Human-readable description. |
| `env[].required` | boolean | No | Whether the variable must be set. Default: `false`. |
| `env[].default` | string | No | Default value if not provided. |
| `env[].sensitive` | boolean | No | If `true`, value should be treated as a secret. Default: `false`. |
| `ports` | Port[] | No | Network ports exposed by the artifact. |
| `ports[].containerPort` | integer | Yes | Port number inside the container. |
| `ports[].protocol` | string | No | Protocol (`TCP`, `UDP`). Default: `TCP`. |
| `ports[].description` | string | No | Purpose of this port. |
| `healthCheck` | HealthCheck | No | Health check configuration. |
| `healthCheck.httpGet` | object | No | HTTP health check. |
| `healthCheck.httpGet.path` | string | Yes | HTTP path to check. |
| `healthCheck.httpGet.port` | integer | Yes | Port to check. |
| `healthCheck.initialDelaySeconds` | integer | No | Seconds before first check. Default: `0`. |
| `healthCheck.periodSeconds` | integer | No | Seconds between checks. Default: `10`. |
| `healthCheck.timeoutSeconds` | integer | No | Seconds before timeout. Default: `1`. |
| `healthCheck.failureThreshold` | integer | No | Failures before marking unhealthy. Default: `3`. |
| `resources` | Resources | No | Compute resource requirements. |
| `resources.requests` | ResourceSpec | No | Minimum resources required. |
| `resources.limits` | ResourceSpec | No | Maximum resources allowed. |
| `resources.requests.cpu` | string | No | CPU request (e.g., `"500m"`, `"2"`). |
| `resources.requests.memory` | string | No | Memory request (e.g., `"512Mi"`, `"4Gi"`). |
| `resources.limits.cpu` | string | No | CPU limit. |
| `resources.limits.memory` | string | No | Memory limit. |
| `resources.limits.gpu` | string | No | GPU count (e.g., `"1"`). |
| `resources.limits.gpuType` | string | No | GPU type (e.g., `"nvidia-a10g"`, `"nvidia-h100"`). |
| `command` | string[] | No | Container entrypoint command. |
| `args` | string[] | No | Arguments to the entrypoint command. |
| `replicas` | object | No | Replica scaling configuration. |
| `replicas.min` | integer | No | Minimum replicas. Default: `1`. |
| `replicas.max` | integer | No | Maximum replicas. Default: `1`. |

##### MCP Server Runtime

MCP server runtime is described through the `transport` field. See [Section 2.3.6](#236-mcp-server-specific-fields).

#### 2.3.4 `bom` (optional in Tier 1, required in Tier 2)

The Bill of Materials. See [Section 3](#3-bill-of-materials-bom) for the full specification.

#### 2.3.5 Skill-Specific Fields

Skills have a `content` block instead of `runtime` and `capabilities`.

```yaml
# kind: skill
content:
  entrypoint: "SKILL.md"
  scripts:
    - path: "scripts/setup.sh"
      description: "Environment setup script"
      interpreter: "bash"
    - path: "scripts/validate.py"
      description: "Output validation"
      interpreter: "python3"
  references:
    - title: "API Documentation"
      url: "https://docs.example.com/api"
      type: "documentation"
    - title: "Style Guide"
      path: "references/style-guide.md"
      type: "guide"
  assets:
    - path: "templates/output.jinja2"
      mediaType: "text/plain"
      description: "Output formatting template"
    - path: "examples/sample-input.json"
      mediaType: "application/json"
      description: "Example input for testing"
  triggerPhrases:
    - "review this code"
    - "analyze the pull request"
    - "check for security issues"
    - "suggest improvements"
```

| Field | Type | Required | Description |
|---|---|---|---|
| `content.entrypoint` | string | Yes | Path to the primary skill definition file (typically `SKILL.md`). This file contains the prompt instructions that define the skill's behavior. |
| `content.scripts` | Script[] | No | Supporting scripts used by the skill. |
| `content.scripts[].path` | string | Yes | Relative path to the script file. |
| `content.scripts[].description` | string | No | What the script does. |
| `content.scripts[].interpreter` | string | No | Script interpreter (e.g., `bash`, `python3`, `node`). |
| `content.references` | Reference[] | No | External or bundled reference materials. |
| `content.references[].title` | string | Yes | Reference title. |
| `content.references[].url` | string | No | External URL (mutually exclusive with `path`). |
| `content.references[].path` | string | No | Bundled file path (mutually exclusive with `url`). |
| `content.references[].type` | string | No | Reference type: `documentation`, `guide`, `example`, `specification`, `other`. |
| `content.assets` | Asset[] | No | Static assets bundled with the skill. |
| `content.assets[].path` | string | Yes | Relative path to the asset. |
| `content.assets[].mediaType` | string | No | MIME type of the asset. |
| `content.assets[].description` | string | No | Asset description. |
| `content.triggerPhrases` | string[] | No | Natural language phrases that should activate this skill. Used for skill discovery and routing. Maximum 20 phrases, each max 256 characters. |

#### 2.3.6 MCP Server-Specific Fields

MCP servers have `transport`, `tools`, `resources`, and `prompts` top-level fields that align with the [MCP Registry specification](https://spec.modelcontextprotocol.io/).

##### Transport

```yaml
# kind: mcp-server
transport:
  type: "streamable-http"
  url: "https://mcp.example.com/v1"
  authorization:
    type: "bearer"
    tokenUrl: "https://auth.example.com/token"
    scopes:
      - "mcp:read"
      - "mcp:write"
```

| Field | Type | Required | Description |
|---|---|---|---|
| `transport.type` | string | Yes | Transport type. One of: `stdio`, `streamable-http`, `sse`. |
| `transport.url` | string | Conditional | Server URL. Required when type is `streamable-http` or `sse`. |
| `transport.authorization` | object | No | Authorization configuration for remote transports. |
| `transport.authorization.type` | string | No | Auth type: `bearer`, `api-key`, `oauth2`. |
| `transport.authorization.tokenUrl` | string | No | OAuth2 token endpoint URL. |
| `transport.authorization.scopes` | string[] | No | Required OAuth2 scopes. |

For `stdio` transport, the execution details come from the artifact's OCI packaging:

```yaml
transport:
  type: "stdio"
  command: "npx"
  args:
    - "-y"
    - "@example/mcp-server"
  env:
    - name: "API_KEY"
      required: true
      sensitive: true
```

| Field | Type | Required | Description |
|---|---|---|---|
| `transport.command` | string | Conditional | Command to start the server. Required for `stdio` transport. |
| `transport.args` | string[] | No | Arguments for the command. |
| `transport.env` | EnvVar[] | No | Environment variables (same schema as agent runtime env). |

##### Tools

```yaml
tools:
  - name: "read_file"
    description: "Read the contents of a file"
    inputSchema:
      type: "object"
      properties:
        path:
          type: "string"
          description: "Path to the file to read"
      required:
        - "path"
  - name: "write_file"
    description: "Write content to a file"
    inputSchema:
      type: "object"
      properties:
        path:
          type: "string"
          description: "Path to write to"
        content:
          type: "string"
          description: "Content to write"
      required:
        - "path"
        - "content"
```

| Field | Type | Required | Description |
|---|---|---|---|
| `tools` | Tool[] | No | Tools exposed by the MCP server. |
| `tools[].name` | string | Yes | Tool identifier. Must be unique within the server. |
| `tools[].description` | string | Yes | Human-readable description of the tool's purpose. |
| `tools[].inputSchema` | object | No | JSON Schema describing the tool's input parameters. |

##### Resources

```yaml
resources:
  - uri: "file:///{path}"
    name: "File Contents"
    description: "Contents of files on the filesystem"
    mimeType: "text/plain"
  - uri: "db:///{table}"
    name: "Database Table"
    description: "Rows from a database table"
    mimeType: "application/json"
```

| Field | Type | Required | Description |
|---|---|---|---|
| `resources` | Resource[] | No | Resources exposed by the MCP server. |
| `resources[].uri` | string | Yes | URI template for the resource. |
| `resources[].name` | string | Yes | Human-readable resource name. |
| `resources[].description` | string | No | Resource description. |
| `resources[].mimeType` | string | No | MIME type of the resource content. |

##### Prompts

```yaml
prompts:
  - name: "summarize"
    description: "Summarize a document"
    arguments:
      - name: "document"
        description: "The document to summarize"
        required: true
      - name: "style"
        description: "Summary style (brief, detailed, bullet-points)"
        required: false
```

| Field | Type | Required | Description |
|---|---|---|---|
| `prompts` | Prompt[] | No | Prompts exposed by the MCP server. |
| `prompts[].name` | string | Yes | Prompt identifier. |
| `prompts[].description` | string | No | What the prompt does. |
| `prompts[].arguments` | Argument[] | No | Prompt arguments. |
| `prompts[].arguments[].name` | string | Yes | Argument name. |
| `prompts[].arguments[].description` | string | No | Argument description. |
| `prompts[].arguments[].required` | boolean | No | Whether the argument is required. Default: `false`. |

### 2.4 Tier 2: Production Artifact

Tier 2 adds fields required for production-grade governance: mandatory BOM, provenance, evaluation baselines, and trust signals.

#### 2.4.1 `provenance` (optional)

Supply chain provenance information. See [Section 9](#9-provenance-and-supply-chain) for the full specification.

```yaml
provenance:
  buildType: "https://slsa.dev/provenance/v1"
  builder:
    id: "https://github.com/actions/runner"
  metadata:
    buildInvocationId: "github-actions-run-12345"
    buildStartedOn: "2026-01-15T10:00:00Z"
    buildFinishedOn: "2026-01-15T10:05:23Z"
  attestation:
    intoto:
      predicateType: "https://slsa.dev/provenance/v1"
    sigstore:
      bundleUrl: "https://rekor.sigstore.dev/api/v1/log/entries/abc123"
```

#### 2.4.2 `evalBaseline` (optional)

Evaluation baseline requirements. See [Section 8](#8-eval-results-interface) for the full specification.

```yaml
evalBaseline:
  minimumScore: 0.7
  requiredBenchmarks:
    - "accuracy"
    - "latency"
  evaluationCadence: "on-publish"
```

#### 2.4.3 `trustSignals` (optional)

Computed trust signals. See [Section 10](#10-trust-signals) for the full specification.

```yaml
trustSignals:
  overallScore: 0.85
  components:
    provenance: 0.95
    evaluation: 0.80
    codeHealth: 0.90
    publisherReputation: 0.75
    community: 0.70
    operational: 0.85
  lastUpdated: "2026-01-15T12:00:00Z"
```

### 2.5 Complete Examples

#### 2.5.1 Complete Agent Artifact (Tier 2)

```yaml
identity:
  name: "acme/customer-support-agent"
  version: "2.1.0"
  title: "Customer Support Agent"
  description: "An AI agent that handles customer support inquiries using knowledge bases and ticketing systems."
kind: agent
artifacts:
  - oci: "ghcr.io/acme/customer-support-agent:2.1.0"
    digest: "sha256:a1b2c3d4e5f6..."
    platform:
      os: "linux"
      architecture: "amd64"
metadata:
  labels:
    team: "customer-experience"
    environment: "production"
  tags:
    - "customer-support"
    - "chatbot"
    - "ticketing"
  category: "conversational"
  authors:
    - name: "ACME AI Team"
      email: "ai-team@acme.com"
  license: "Apache-2.0"
  repository:
    url: "https://github.com/acme/customer-support-agent"
    source: "github"
capabilities:
  protocols:
    - "openai-chat"
    - "mcp"
  inputModalities:
    - "text"
    - "image"
  outputModalities:
    - "text"
  streaming: true
  multiTurn: true
  contextWindow: 200000
runtime:
  env:
    - name: "ANTHROPIC_API_KEY"
      required: true
      sensitive: true
    - name: "TICKETING_API_URL"
      required: true
      default: "https://tickets.acme.com/api"
  ports:
    - containerPort: 8080
      protocol: "TCP"
      description: "Agent API"
  healthCheck:
    httpGet:
      path: "/health"
      port: 8080
    initialDelaySeconds: 15
    periodSeconds: 30
  resources:
    requests:
      cpu: "1000m"
      memory: "2Gi"
    limits:
      cpu: "4000m"
      memory: "8Gi"
bom:
  models:
    - name: "claude-sonnet-4-20250514"
      provider: "anthropic"
      role: "primary"
      contextWindow: 200000
  tools:
    - name: "acme/ticketing-mcp"
      version: ">=1.0.0"
      required: true
    - name: "acme/knowledge-base-mcp"
      version: ">=2.0.0"
      required: true
  skills:
    - name: "acme/empathetic-response"
      version: ">=1.0.0"
  prompts:
    - name: "system-prompt"
      source: "inline"
      role: "system"
  orchestration:
    framework: "adk"
    pattern: "tool-use-loop"
provenance:
  buildType: "https://slsa.dev/provenance/v1"
  builder:
    id: "https://github.com/actions/runner"
  attestation:
    sigstore:
      bundleUrl: "https://rekor.sigstore.dev/api/v1/log/entries/xyz789"
evalBaseline:
  minimumScore: 0.8
  requiredBenchmarks:
    - "customer-satisfaction"
    - "response-accuracy"
    - "latency-p95"
trustSignals:
  overallScore: 0.88
  components:
    provenance: 0.95
    evaluation: 0.85
    codeHealth: 0.92
    publisherReputation: 0.80
    community: 0.75
    operational: 0.90
  lastUpdated: "2026-01-15T12:00:00Z"
```

#### 2.5.2 Complete Skill Artifact (Tier 1)

```yaml
identity:
  name: "acme/code-review"
  version: "1.2.0"
  title: "Code Review Skill"
  description: "Performs thorough code reviews with security analysis and best-practice suggestions."
kind: skill
artifacts:
  - oci: "ghcr.io/acme/code-review-skill:1.2.0"
metadata:
  tags:
    - "code-review"
    - "security"
    - "best-practices"
  category: "coding"
  authors:
    - name: "ACME DevTools"
  license: "MIT"
content:
  entrypoint: "SKILL.md"
  scripts:
    - path: "scripts/lint-check.sh"
      description: "Run linters before review"
      interpreter: "bash"
  references:
    - title: "OWASP Top 10"
      url: "https://owasp.org/www-project-top-ten/"
      type: "specification"
  assets:
    - path: "templates/review-report.md"
      mediaType: "text/markdown"
      description: "Review report template"
  triggerPhrases:
    - "review this code"
    - "check this PR"
    - "security review"
    - "code quality check"
bom:
  toolRequirements:
    - name: "filesystem"
      description: "Read source files"
      required: true
    - name: "github"
      description: "Access PR information"
      required: false
  modelRequirements:
    - capability: "code-analysis"
      minContextWindow: 100000
```

#### 2.5.3 Complete MCP Server Artifact (Tier 1)

```yaml
identity:
  name: "acme/filesystem-server"
  version: "3.0.1"
  title: "Filesystem MCP Server"
  description: "Provides file system access tools for AI agents via the Model Context Protocol."
kind: mcp-server
artifacts:
  - oci: "ghcr.io/acme/filesystem-mcp:3.0.1"
    digest: "sha256:def456..."
  - oci: "registry.npmjs.org/@acme/filesystem-mcp:3.0.1"
metadata:
  tags:
    - "filesystem"
    - "file-access"
    - "mcp"
  category: "devops"
  authors:
    - name: "ACME Platform"
  license: "MIT"
  repository:
    url: "https://github.com/acme/filesystem-mcp"
    source: "github"
transport:
  type: "stdio"
  command: "npx"
  args:
    - "-y"
    - "@acme/filesystem-mcp"
  env:
    - name: "ALLOWED_PATHS"
      description: "Comma-separated list of allowed directory paths"
      required: true
tools:
  - name: "read_file"
    description: "Read the contents of a file at the specified path"
    inputSchema:
      type: "object"
      properties:
        path:
          type: "string"
          description: "Absolute path to the file"
      required:
        - "path"
  - name: "write_file"
    description: "Write content to a file at the specified path"
    inputSchema:
      type: "object"
      properties:
        path:
          type: "string"
          description: "Absolute path to the file"
        content:
          type: "string"
          description: "Content to write"
      required:
        - "path"
        - "content"
  - name: "list_directory"
    description: "List the contents of a directory"
    inputSchema:
      type: "object"
      properties:
        path:
          type: "string"
          description: "Absolute path to the directory"
      required:
        - "path"
resources:
  - uri: "file:///{path}"
    name: "File Resource"
    description: "Direct access to file contents as a resource"
    mimeType: "application/octet-stream"
prompts:
  - name: "explore_directory"
    description: "Generate a prompt to explore and summarize directory contents"
    arguments:
      - name: "path"
        description: "Directory path to explore"
        required: true
      - name: "depth"
        description: "How many levels deep to explore"
        required: false
bom:
  runtimeDependencies:
    - name: "node"
      version: ">=18.0.0"
      type: "runtime"
  externalServices: []
```

### 2.6 Schema Validation Rules

The following validation rules apply across all tiers and kinds:

1. **Unique identity**: The combination of `identity.name` + `identity.version` must be unique within a registry instance.

2. **Kind consistency**: Fields specific to one kind must not appear on artifacts of another kind. For example, a `skill` artifact must not have a `runtime` section, and an `agent` artifact must not have a `content.entrypoint` field.

3. **Version format**: The `identity.version` field must conform to Semantic Versioning 2.0.0.

4. **OCI reference format**: Each entry in `artifacts` must contain a valid OCI reference string.

5. **Tier progression**: Tier 2 fields (provenance, evalBaseline, trustSignals) are ignored by the registry unless the artifact also satisfies all Tier 1 requirements for its kind. The `bom` field is optional at Tier 1 but required at Tier 2.

6. **String length limits**:
   * `identity.name`: maximum 255 characters (including namespace).
   * `identity.title`: maximum 128 characters.
   * `identity.description`: maximum 512 characters.
   * `metadata.tags[]`: maximum 64 characters each.
   * `metadata.labels` keys: maximum 63 characters.
   * `metadata.labels` values: maximum 256 characters.

7. **Array size limits**:
   * `artifacts`: maximum 10 entries.
   * `metadata.tags`: maximum 32 entries.
   * `metadata.labels`: maximum 64 entries.
   * `metadata.authors`: maximum 16 entries.
   * `tools`: maximum 500 entries.
   * `resources`: maximum 200 entries.
   * `prompts`: maximum 100 entries.
   * `content.triggerPhrases`: maximum 20 entries.

---

## 3. Bill of Materials (BOM)

### 3.1 Overview

The Bill of Materials (BOM) provides a structured, machine-readable declaration of an artifact's dependencies. The BOM schema is kind-aware: each artifact kind has a different set of applicable dependency categories.

The BOM serves multiple purposes:
* **Dependency resolution**: Enables tooling to verify that all required dependencies are available before deployment.
* **Supply chain visibility**: Provides transparency into what models, tools, and services an artifact uses.
* **Compatibility checking**: Allows the registry to verify that declared dependencies are compatible.
* **Security auditing**: Enables scanning for known vulnerabilities in dependencies.

### 3.2 Agent BOM

Agent BOMs are the most comprehensive, covering five primary subsections plus supplementary declarations for memory, session management, and identity.

```yaml
# kind: agent
bom:
  # 3.2.1 Models
  models:
    - name: "claude-sonnet-4-20250514"
      provider: "anthropic"
      role: "primary"
      contextWindow: 200000
      maxOutputTokens: 8192
      costTier: "standard"
    - name: "text-embedding-3-large"
      provider: "openai"
      role: "embedding"
      dimensions: 3072

  # 3.2.2 Tools
  tools:
    - name: "acme/github-mcp"
      version: ">=2.0.0, <3.0.0"
      required: true
      capabilities:
        - "create_issue"
        - "list_pull_requests"
    - name: "acme/slack-mcp"
      version: ">=1.5.0"
      required: false

  # 3.2.3 Skills
  skills:
    - name: "acme/code-review"
      version: ">=1.0.0"
    - name: "acme/summarization"
      version: "^2.0.0"

  # 3.2.4 Prompts
  prompts:
    - name: "system-prompt"
      source: "inline"
      role: "system"
      hash: "sha256:abc123..."
    - name: "few-shot-examples"
      source: "file"
      path: "prompts/few-shot.md"
      role: "few-shot"
      hash: "sha256:def456..."

  # 3.2.5 Orchestration
  orchestration:
    framework: "adk"
    frameworkVersion: ">=1.0.0"
    pattern: "tool-use-loop"
    maxIterations: 50
    parallelToolCalls: true
    fallbackBehavior: "graceful-degradation"

  # Supplementary
  memory:
    type: "vector-store"
    provider: "chromadb"
    persistenceRequired: true
  session:
    type: "stateful"
    maxDuration: "24h"
    storageBackend: "redis"
  identity:
    supportsMultiTenant: true
    authMethods:
      - "oauth2"
      - "api-key"
```

#### 3.2.1 Models

Declares the AI models an agent depends on.

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Model identifier as recognized by the provider. |
| `provider` | string | Yes | Model provider (e.g., `anthropic`, `openai`, `google`, `meta`, `local`). |
| `role` | string | Yes | How the model is used. One of: `primary`, `secondary`, `embedding`, `classification`, `routing`, `evaluation`. |
| `contextWindow` | integer | No | Context window size in tokens. |
| `maxOutputTokens` | integer | No | Maximum output tokens. |
| `costTier` | string | No | Cost classification: `free`, `low`, `standard`, `premium`. |
| `version` | string | No | Specific model version constraint. |
| `fineTuned` | boolean | No | Whether this is a fine-tuned model. Default: `false`. |
| `fineTuneBase` | string | No | Base model used for fine-tuning. |

#### 3.2.2 Tools

Declares MCP server dependencies.

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Fully qualified MCP server name (namespace/name). |
| `version` | string | Yes | Version constraint (semver range). |
| `required` | boolean | No | Whether the tool is required for the agent to function. Default: `true`. |
| `capabilities` | string[] | No | Specific tool names required from this server. If omitted, all tools may be used. |

#### 3.2.3 Skills

Declares skill dependencies.

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Fully qualified skill name. |
| `version` | string | Yes | Version constraint (semver range). |

#### 3.2.4 Prompts

Declares prompt dependencies with integrity hashes.

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Prompt identifier. |
| `source` | string | Yes | Source type: `inline`, `file`, `registry`, `url`. |
| `path` | string | No | File path (when source is `file`). |
| `url` | string | No | URL (when source is `url`). |
| `role` | string | No | Prompt role: `system`, `few-shot`, `template`, `guardrail`. |
| `hash` | string | No | Content integrity hash (e.g., `sha256:...`). |

#### 3.2.5 Orchestration

Declares the orchestration framework and pattern.

| Field | Type | Required | Description |
|---|---|---|---|
| `framework` | string | Yes | Orchestration framework identifier (e.g., `adk`, `langchain`, `crewai`, `autogen`, `custom`). |
| `frameworkVersion` | string | No | Framework version constraint. |
| `pattern` | string | No | Orchestration pattern: `tool-use-loop`, `chain-of-thought`, `react`, `plan-and-execute`, `multi-agent`, `custom`. |
| `maxIterations` | integer | No | Maximum orchestration loop iterations. |
| `parallelToolCalls` | boolean | No | Whether the agent supports parallel tool execution. Default: `false`. |
| `fallbackBehavior` | string | No | Behavior on failure: `fail-fast`, `graceful-degradation`, `retry`, `human-in-the-loop`. |

#### 3.2.6 Supplementary: Memory

| Field | Type | Required | Description |
|---|---|---|---|
| `memory.type` | string | No | Memory type: `vector-store`, `key-value`, `graph`, `none`. |
| `memory.provider` | string | No | Memory provider (e.g., `chromadb`, `pinecone`, `redis`). |
| `memory.persistenceRequired` | boolean | No | Whether memory must persist across sessions. Default: `false`. |

#### 3.2.7 Supplementary: Session

| Field | Type | Required | Description |
|---|---|---|---|
| `session.type` | string | No | Session type: `stateless`, `stateful`, `persistent`. |
| `session.maxDuration` | string | No | Maximum session duration (Go duration format, e.g., `24h`, `30m`). |
| `session.storageBackend` | string | No | Session storage backend (e.g., `redis`, `postgres`, `memory`). |

#### 3.2.8 Supplementary: Identity

| Field | Type | Required | Description |
|---|---|---|---|
| `identity.supportsMultiTenant` | boolean | No | Whether the agent supports multi-tenant operation. Default: `false`. |
| `identity.authMethods` | string[] | No | Authentication methods the agent can use: `oauth2`, `api-key`, `mtls`, `saml`. |

### 3.3 Skill BOM

Skill BOMs are simpler, declaring tool requirements and model requirements without specifying concrete providers.

```yaml
# kind: skill
bom:
  dependencies:
    - name: "acme/markdown-utils"
      version: ">=1.0.0"
      type: "skill"

  toolRequirements:
    - name: "filesystem"
      description: "Read and write files"
      required: true
      operations:
        - "read_file"
        - "write_file"
    - name: "github"
      description: "Access GitHub APIs"
      required: false
      operations:
        - "create_issue"
        - "list_pull_requests"

  modelRequirements:
    - capability: "code-analysis"
      minContextWindow: 100000
      preferredProviders:
        - "anthropic"
        - "openai"
    - capability: "text-generation"
      minContextWindow: 50000
```

| Field | Type | Required | Description |
|---|---|---|---|
| `dependencies` | Dependency[] | No | Other skills or libraries this skill depends on. |
| `dependencies[].name` | string | Yes | Dependency name. |
| `dependencies[].version` | string | Yes | Version constraint. |
| `dependencies[].type` | string | No | Dependency type: `skill`, `library`. Default: `skill`. |
| `toolRequirements` | ToolReq[] | No | Abstract tool requirements (not tied to specific MCP servers). |
| `toolRequirements[].name` | string | Yes | Logical tool category name. |
| `toolRequirements[].description` | string | No | What the tool is needed for. |
| `toolRequirements[].required` | boolean | No | Whether the tool is required. Default: `true`. |
| `toolRequirements[].operations` | string[] | No | Specific operations needed. |
| `modelRequirements` | ModelReq[] | No | Abstract model capability requirements. |
| `modelRequirements[].capability` | string | Yes | Required capability: `code-analysis`, `text-generation`, `reasoning`, `vision`, `embedding`. |
| `modelRequirements[].minContextWindow` | integer | No | Minimum context window needed. |
| `modelRequirements[].preferredProviders` | string[] | No | Preferred model providers. |

### 3.4 MCP Server BOM

MCP server BOMs declare runtime dependencies, external service requirements, and optional model dependencies.

```yaml
# kind: mcp-server
bom:
  runtimeDependencies:
    - name: "node"
      version: ">=18.0.0"
      type: "runtime"
    - name: "python"
      version: ">=3.11"
      type: "runtime"
    - name: "@modelcontextprotocol/sdk"
      version: ">=1.0.0"
      type: "library"

  externalServices:
    - name: "PostgreSQL"
      type: "database"
      required: true
      connectionEnv: "DATABASE_URL"
    - name: "Redis"
      type: "cache"
      required: false
      connectionEnv: "REDIS_URL"
    - name: "GitHub API"
      type: "api"
      required: true
      baseUrl: "https://api.github.com"
      authEnv: "GITHUB_TOKEN"

  modelDependencies:
    - name: "text-embedding-3-small"
      provider: "openai"
      purpose: "Semantic search over resources"
      required: false
```

| Field | Type | Required | Description |
|---|---|---|---|
| `runtimeDependencies` | RuntimeDep[] | No | Runtime and library dependencies. |
| `runtimeDependencies[].name` | string | Yes | Dependency name. |
| `runtimeDependencies[].version` | string | Yes | Version constraint. |
| `runtimeDependencies[].type` | string | Yes | Dependency type: `runtime`, `library`, `system`. |
| `externalServices` | ExternalSvc[] | No | External services required. |
| `externalServices[].name` | string | Yes | Service name. |
| `externalServices[].type` | string | Yes | Service type: `database`, `cache`, `api`, `queue`, `storage`, `other`. |
| `externalServices[].required` | boolean | No | Whether the service is required. Default: `true`. |
| `externalServices[].connectionEnv` | string | No | Environment variable name for the connection string. |
| `externalServices[].baseUrl` | string | No | Service base URL (for APIs). |
| `externalServices[].authEnv` | string | No | Environment variable name for authentication. |
| `modelDependencies` | ModelDep[] | No | AI model dependencies. |
| `modelDependencies[].name` | string | Yes | Model identifier. |
| `modelDependencies[].provider` | string | Yes | Model provider. |
| `modelDependencies[].purpose` | string | No | What the model is used for. |
| `modelDependencies[].required` | boolean | No | Whether the model is required. Default: `true`. |

### 3.5 BOM Validation Rules

1. **Version constraints**: All version fields in BOM entries must be valid semver ranges (e.g., `>=1.0.0`, `^2.0.0`, `>=1.0.0, <3.0.0`, `1.2.3`).

2. **Circular dependency detection**: The registry MUST reject artifacts whose BOM creates a circular dependency chain.

3. **Resolvability**: When a BOM references another registry artifact (tool or skill), the referenced artifact MUST exist in the registry or be marked as `required: false`.

4. **Hash integrity**: When prompt hashes are provided, the registry SHOULD verify them against the actual content when possible.

5. **Kind consistency**: Agent BOMs must not contain `toolRequirements` or `modelRequirements` (skill BOM fields). Skill BOMs must not contain `models`, `tools`, `skills`, `prompts`, or `orchestration` (agent BOM fields).

---

## 4. Version Model

### 4.1 Semantic Versioning

All artifact versions MUST conform to [Semantic Versioning 2.0.0](https://semver.org/). The version string format is:

```
MAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]
```

Where:
* **MAJOR**: Incremented for incompatible API or behavior changes.
* **MINOR**: Incremented for backward-compatible new functionality.
* **PATCH**: Incremented for backward-compatible bug fixes.
* **PRERELEASE**: Optional pre-release identifier (e.g., `-alpha.1`, `-beta.2`, `-rc.1`).
* **BUILD**: Optional build metadata (e.g., `+build.123`). Ignored for version precedence.

### 4.2 Version Immutability

Artifact versions become immutable based on their promotion status:

| Promotion Status | Mutable Fields | Immutable Fields |
|---|---|---|
| `draft` | All fields | None |
| `evaluated` | `metadata.labels`, `metadata.tags` | `identity`, `kind`, `artifacts`, `capabilities`, `runtime`, `bom`, `content`, `transport`, `tools`, `resources`, `prompts` |
| `approved` | `metadata.labels`, `metadata.tags` | Same as evaluated |
| `published` | `metadata.labels`, `metadata.tags` | Same as evaluated |
| `deprecated` | `metadata.labels`, `metadata.tags` | Same as evaluated |
| `archived` | None | All fields |

Once an artifact version reaches `evaluated` status, its core content (identity, kind, artifacts, BOM, and kind-specific fields) becomes immutable. This guarantees that evaluated, approved, and published versions are exactly what was tested.

### 4.3 Latest Version Pointer

Each artifact maintains a `latest` pointer that references the highest-precedence version according to semver ordering rules. The latest pointer follows these rules:

1. **Pre-release exclusion**: Pre-release versions (e.g., `1.0.0-alpha.1`) are NOT eligible for the `latest` pointer unless no stable version exists.

2. **Published preference**: If any version is `published`, the `latest` pointer MUST point to the highest published stable version. If no versions are published, it points to the highest version regardless of status.

3. **Automatic update**: When a new version is created or a version's status changes, the registry MUST recalculate the `latest` pointer.

4. **Explicit query**: Clients can request the latest version using the special version string `latest` in API calls.

### 4.4 Pre-release Versions

Pre-release versions have lower precedence than their associated release version:

```
1.0.0-alpha.1 < 1.0.0-alpha.2 < 1.0.0-beta.1 < 1.0.0-rc.1 < 1.0.0
```

Pre-release versions:
* Are always visible in version listings.
* Are excluded from the `latest` pointer by default.
* Can be promoted through the full lifecycle.
* Follow the same immutability rules as stable versions.

### 4.5 Version Limits

To prevent abuse and ensure registry performance:

* **Maximum versions per artifact**: 10,000 versions per unique artifact name.
* **Maximum pre-release versions**: No separate limit; counts toward the 10,000 total.
* **Minimum version**: The first version of an artifact may be any valid semver string (it does not need to start at `0.0.1` or `1.0.0`).

When the version limit is reached, the registry MUST return an error with status code 409 Conflict and a message directing the user to request a limit increase.

### 4.6 Version Ordering

Versions are ordered according to Semantic Versioning 2.0.0 precedence rules:

1. Major, minor, and patch versions are compared numerically.
2. Pre-release versions have lower precedence than the associated normal version.
3. Pre-release precedence is determined by comparing each dot-separated identifier left to right.
4. Build metadata is ignored for precedence.

### 4.7 Version Constraint Syntax

BOM dependencies and API queries use version constraint syntax:

| Syntax | Meaning | Example |
|---|---|---|
| `1.2.3` | Exact version | Only `1.2.3` |
| `>=1.2.3` | Greater than or equal | `1.2.3`, `1.3.0`, `2.0.0` |
| `<2.0.0` | Less than | `1.9.9`, `0.1.0` |
| `>=1.0.0, <2.0.0` | Range | `1.0.0` through `1.x.x` |
| `^1.2.3` | Compatible with | `>=1.2.3, <2.0.0` |
| `~1.2.3` | Approximately | `>=1.2.3, <1.3.0` |
| `*` | Any version | All versions |
| `latest` | Latest stable | Highest non-prerelease version |

---

## 5. Promotion Lifecycle

### 5.1 Overview

Every artifact version moves through a promotion lifecycle that governs its visibility, mutability, and required quality gates. The lifecycle ensures that artifacts are progressively validated before reaching consumers.

### 5.2 Lifecycle States

```
draft --> evaluated --> approved --> published --> deprecated --> archived
  |                                                    |
  +----------------------------------------------------+
                    (can deprecate from any state >= published)
```

#### 5.2.1 Draft

* **Entry**: Automatic on artifact creation.
* **Visibility**: Only visible to the artifact owner and namespace admins.
* **Mutability**: All fields are mutable. The artifact can be updated freely.
* **Allowed transitions**: `evaluated`.
* **Requirements to exit**: At least one OCI artifact reference must be valid and reachable.

#### 5.2.2 Evaluated

* **Entry**: After evaluation results are submitted and meet the artifact's baseline (if defined).
* **Visibility**: Visible to the artifact owner, namespace admins, and designated approvers.
* **Mutability**: Core content becomes immutable. Only `metadata.labels` and `metadata.tags` can be modified.
* **Allowed transitions**: `approved`, `draft` (rejection -- creates a new draft version).
* **Requirements to exit**:
  * At least one `EvalRecord` must be associated with this version.
  * If `evalBaseline` is defined, the overall score must meet `minimumScore`.
  * All `requiredBenchmarks` must have results.

#### 5.2.3 Approved

* **Entry**: An approver (human or automated) has reviewed the evaluated artifact.
* **Visibility**: Visible to the artifact owner, namespace admins, approvers, and viewers with explicit access.
* **Mutability**: Same as evaluated.
* **Allowed transitions**: `published`.
* **Requirements to exit**: An authorized approver must explicitly approve the artifact.
* **Role required**: `approver` or `admin`.

#### 5.2.4 Published

* **Entry**: An admin or publisher promotes the approved artifact to public visibility.
* **Visibility**: Publicly visible. Appears in search results and listings.
* **Mutability**: Same as evaluated.
* **Allowed transitions**: `deprecated`.
* **Requirements to exit**: A publisher or admin must explicitly publish the artifact.
* **Role required**: `publisher` or `admin`.

#### 5.2.5 Deprecated

* **Entry**: The publisher or admin marks the artifact as superseded.
* **Visibility**: Still publicly visible but marked as deprecated. Search results may down-rank deprecated artifacts.
* **Mutability**: Same as evaluated. Additionally, a `deprecation` notice can be added.
* **Allowed transitions**: `archived`.
* **Additional fields**:

```yaml
deprecation:
  reason: "Superseded by v3.0.0 with improved accuracy"
  alternative: "acme/customer-support-agent@3.0.0"
  deadline: "2026-06-01T00:00:00Z"
```

| Field | Type | Required | Description |
|---|---|---|---|
| `deprecation.reason` | string | Yes | Why the artifact is deprecated. |
| `deprecation.alternative` | string | No | Recommended replacement artifact (name@version). |
| `deprecation.deadline` | string | No | RFC3339 timestamp after which the artifact may be archived. |

#### 5.2.6 Archived

* **Entry**: After the deprecation deadline passes, or by admin action.
* **Visibility**: Not visible in public listings or search. Can be retrieved by direct reference only.
* **Mutability**: Fully immutable. No fields can be modified.
* **Allowed transitions**: None. Archival is terminal.
* **Requirements to exit**: N/A (terminal state).

### 5.3 Promotion API

Promotion transitions are triggered via the REST API. See [Section 6](#6-rest-api) for full API details.

```
POST /v1/{kind}s/{name}/versions/{version}/promote
Content-Type: application/json

{
  "targetStatus": "evaluated",
  "evalRecordId": "eval-abc123",
  "comment": "All benchmarks pass with score 0.92"
}
```

### 5.4 Promotion Gates

Each transition may have configurable gates:

| Transition | Default Gate | Configurable |
|---|---|---|
| draft -> evaluated | EvalRecord exists and meets baseline | Yes (custom eval providers) |
| evaluated -> approved | Approver role required | Yes (auto-approve policies) |
| approved -> published | Publisher role required | Yes (auto-publish policies) |
| published -> deprecated | Publisher or admin role | No |
| deprecated -> archived | Admin role or deadline passed | Yes (auto-archive policies) |

### 5.5 Simplified Lifecycle

For registries that do not require the full promotion lifecycle, a simplified two-state model is supported:

```
draft --> published --> deprecated --> archived
```

In simplified mode:
* The `evaluated` and `approved` states are skipped.
* Any user with `publisher` role can promote directly from `draft` to `published`.
* Evaluation and approval are not required.
* This is the default mode for local development registries.

The registry configuration determines which lifecycle mode is active:

```yaml
# Registry configuration
lifecycle:
  mode: "full"      # "full" or "simplified"
  autoArchiveDays: 365  # Days after deprecation before auto-archive
```

---

## 6. REST API

### 6.1 Overview

The Agent Registry exposes a unified REST API where the artifact kind is encoded as a path segment. This provides a consistent interface across all artifact types while allowing kind-specific operations.

**Base URL**: `{registry-url}/v1`

**API versioning**: The API is versioned via the URL path prefix (`/v1`). Breaking changes will increment the version number.

### 6.2 Response Envelope

All API responses use a standard envelope:

```json
{
  "data": { },
  "_meta": {
    "requestId": "req-abc123",
    "timestamp": "2026-01-15T12:00:00Z",
    "registry": "registry.example.com"
  },
  "pagination": {
    "nextCursor": "cursor-xyz",
    "count": 30,
    "total": 142
  }
}
```

| Field | Type | Description |
|---|---|---|
| `data` | object or array | The response payload. Single objects for detail endpoints, arrays for list endpoints. |
| `_meta` | object | Request metadata. |
| `_meta.requestId` | string | Unique request identifier for tracing. |
| `_meta.timestamp` | string | RFC3339 timestamp of the response. |
| `_meta.registry` | string | Registry hostname. |
| `pagination` | object | Present only on list endpoints. |
| `pagination.nextCursor` | string | Opaque cursor for the next page. Empty string when no more results. |
| `pagination.count` | integer | Number of items in this response. |
| `pagination.total` | integer | Total number of matching items (optional, may be omitted for performance). |

### 6.3 Error Response

Error responses follow [RFC 7807](https://datatracker.ietf.org/doc/html/rfc7807) Problem Details:

```json
{
  "type": "https://agentregistry.dev/errors/not-found",
  "title": "Not Found",
  "status": 404,
  "detail": "Artifact 'acme/my-agent' version '1.0.0' not found.",
  "instance": "/v1/agents/acme%2Fmy-agent/versions/1.0.0"
}
```

### 6.4 Authentication

API authentication uses Bearer tokens in the `Authorization` header:

```
Authorization: Bearer <registry-jwt-token>
```

Tokens are obtained through one of the authentication methods described in [Section 11](#11-access-control). Public read operations (listing published artifacts, getting published artifact details) do not require authentication.

### 6.5 CRUD Operations

The following endpoints use `{kind}` as a path variable representing the pluralized artifact kind: `agents`, `skills`, `mcp-servers`.

#### 6.5.1 List Artifacts

```
GET /v1/{kind}
```

Query parameters:

| Parameter | Type | Default | Description |
|---|---|---|---|
| `cursor` | string | `""` | Pagination cursor from previous response. |
| `limit` | integer | `30` | Items per page. Range: 1-100. |
| `search` | string | `""` | Substring search on artifact name. |
| `semantic_search` | boolean | `false` | Enable semantic (vector) search. Requires `search` parameter. |
| `semantic_threshold` | float | `0.0` | Maximum cosine distance for semantic search results. |
| `version` | string | `""` | Version filter. Use `latest` for latest versions only, or an exact version string. |
| `updated_since` | string | `""` | RFC3339 timestamp. Only return artifacts updated after this time. |
| `category` | string | `""` | Filter by category. |
| `tag` | string | `""` | Filter by tag. May be specified multiple times. |
| `status` | string | `""` | Filter by promotion status. Admin only. |

**Response**: `200 OK`

```json
{
  "data": [
    {
      "identity": { "name": "acme/my-agent", "version": "1.0.0", "..." : "..." },
      "kind": "agent",
      "_meta": {
        "io.agentregistry/official": {
          "status": "published",
          "publishedAt": "2026-01-10T12:00:00Z",
          "updatedAt": "2026-01-15T12:00:00Z",
          "isLatest": true,
          "published": true
        },
        "ai.agentregistry/semantic": {
          "score": 0.95
        }
      }
    }
  ],
  "pagination": {
    "nextCursor": "cursor-abc",
    "count": 30
  }
}
```

#### 6.5.2 Get Artifact

```
GET /v1/{kind}/{name}/versions/{version}
```

Path parameters:

| Parameter | Type | Description |
|---|---|---|
| `name` | string | URL-encoded fully qualified artifact name (e.g., `acme%2Fmy-agent`). |
| `version` | string | Exact version string or `latest`. |

**Response**: `200 OK`

```json
{
  "data": {
    "identity": { "..." : "..." },
    "kind": "agent",
    "artifacts": [ "..." ],
    "metadata": { "..." },
    "capabilities": { "..." },
    "runtime": { "..." },
    "bom": { "..." }
  },
  "_meta": {
    "requestId": "req-123",
    "timestamp": "2026-01-15T12:00:00Z"
  }
}
```

#### 6.5.3 Create / Update Artifact

```
POST /v1/{kind}/publish
```

Creates a new artifact version or updates a draft version. The request body is the full artifact schema.

**Request body**: Registry Artifact schema (see [Section 2](#2-registry-artifact-schema)).

**Response**: `201 Created` (new version) or `200 OK` (updated draft).

**Authentication**: Required. The caller must have `push` permission for the artifact's namespace.

#### 6.5.4 Push Artifact (Create Unpublished)

```
POST /v1/{kind}/push
```

Creates a new artifact version in `draft` status. Semantically identical to `POST /v1/{kind}/publish` but makes the intent explicit.

**Authentication**: Required. `push` permission.

#### 6.5.5 Delete Artifact Version

```
DELETE /v1/{kind}/{name}/versions/{version}
```

Permanently removes an artifact version. Only `draft` versions can be deleted. Versions in any other status must be archived instead.

**Response**: `200 OK`

**Authentication**: Required. `delete` permission.

#### 6.5.6 List Versions

```
GET /v1/{kind}/{name}/versions
```

Returns all versions of an artifact, ordered by semver precedence (newest first).

**Response**: `200 OK` with array of artifact versions.

### 6.6 Promotion Endpoints

#### 6.6.1 Promote Artifact

```
POST /v1/{kind}/{name}/versions/{version}/promote
```

Transitions an artifact version to the next promotion status.

**Request body**:

```json
{
  "targetStatus": "published",
  "comment": "Approved after security review"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `targetStatus` | string | Yes | Target promotion status. Must be a valid transition from the current status. |
| `comment` | string | No | Human-readable comment for the audit trail. |
| `evalRecordId` | string | Conditional | Required when transitioning to `evaluated`. References the EvalRecord. |

**Response**: `200 OK`

#### 6.6.2 Publish Artifact (Shortcut)

```
POST /v1/{kind}/{name}/versions/{version}/publish
```

Shortcut to set an artifact's published flag to `true`. In simplified lifecycle mode, this is the primary promotion endpoint.

**Response**: `200 OK`

#### 6.6.3 Unpublish Artifact (Shortcut)

```
POST /v1/{kind}/{name}/versions/{version}/unpublish
```

Removes an artifact from public listings without changing its promotion status.

**Response**: `200 OK`

#### 6.6.4 Deprecate Artifact

```
POST /v1/{kind}/{name}/versions/{version}/deprecate
```

**Request body**:

```json
{
  "reason": "Superseded by v3.0.0",
  "alternative": "acme/customer-support-agent@3.0.0",
  "deadline": "2026-06-01T00:00:00Z"
}
```

**Response**: `200 OK`

### 6.7 Evaluation Endpoints

#### 6.7.1 Submit Eval Result

```
POST /v1/{kind}/{name}/versions/{version}/evals
```

Submits an evaluation record for an artifact version. See [Section 8](#8-eval-results-interface).

**Request body**: `EvalRecord` schema.

**Response**: `201 Created`

#### 6.7.2 List Eval Results

```
GET /v1/{kind}/{name}/versions/{version}/evals
```

Returns all evaluation records for an artifact version.

Query parameters:

| Parameter | Type | Description |
|---|---|---|
| `benchmark` | string | Filter by benchmark name. |
| `evaluator` | string | Filter by evaluator identity. |
| `cursor` | string | Pagination cursor. |
| `limit` | integer | Items per page (1-100, default 30). |

**Response**: `200 OK`

#### 6.7.3 Get Eval Result

```
GET /v1/{kind}/{name}/versions/{version}/evals/{evalId}
```

Returns a specific evaluation record.

**Response**: `200 OK`

### 6.8 Provenance Endpoints

#### 6.8.1 Submit Provenance

```
POST /v1/{kind}/{name}/versions/{version}/provenance
```

Submits or updates provenance information for an artifact version. See [Section 9](#9-provenance-and-supply-chain).

**Request body**: Provenance schema.

**Response**: `201 Created`

#### 6.8.2 Get Provenance

```
GET /v1/{kind}/{name}/versions/{version}/provenance
```

Returns the provenance information for an artifact version.

**Response**: `200 OK`

#### 6.8.3 Verify Provenance

```
POST /v1/{kind}/{name}/versions/{version}/provenance/verify
```

Triggers a provenance verification check. The registry validates attestations, signatures, and OCI digests.

**Response**: `200 OK`

```json
{
  "data": {
    "verified": true,
    "checks": [
      { "name": "sigstore-signature", "passed": true, "detail": "Valid Sigstore bundle" },
      { "name": "oci-digest", "passed": true, "detail": "Digest matches" },
      { "name": "slsa-provenance", "passed": true, "detail": "SLSA v1 provenance verified" }
    ],
    "verifiedAt": "2026-01-15T12:05:00Z"
  }
}
```

### 6.9 Trust Signal Endpoints

#### 6.9.1 Get Trust Signals

```
GET /v1/{kind}/{name}/versions/{version}/trust
```

Returns the computed trust signals for an artifact version. See [Section 10](#10-trust-signals).

**Response**: `200 OK`

```json
{
  "data": {
    "overallScore": 0.85,
    "components": {
      "provenance": { "score": 0.95, "weight": 0.25, "detail": "SLSA L3 verified" },
      "evaluation": { "score": 0.80, "weight": 0.25, "detail": "3 benchmarks passed" },
      "codeHealth": { "score": 0.90, "weight": 0.20, "detail": "No known vulnerabilities" },
      "publisherReputation": { "score": 0.75, "weight": 0.15, "detail": "Verified publisher" },
      "community": { "score": 0.70, "weight": 0.10, "detail": "142 downloads, 12 dependents" },
      "operational": { "score": 0.85, "weight": 0.05, "detail": "99.5% uptime" }
    },
    "lastUpdated": "2026-01-15T12:00:00Z"
  }
}
```

#### 6.9.2 Recalculate Trust Signals

```
POST /v1/{kind}/{name}/versions/{version}/trust/recalculate
```

Triggers a recalculation of trust signals. Admin only.

**Response**: `202 Accepted`

### 6.10 Discovery Endpoints

#### 6.10.1 Search (Cross-Kind)

```
GET /v1/search
```

Searches across all artifact kinds.

Query parameters:

| Parameter | Type | Description |
|---|---|---|
| `q` | string | Search query (required). |
| `kind` | string | Filter by kind (`agent`, `skill`, `mcp-server`). May be specified multiple times. |
| `semantic` | boolean | Enable semantic search. Default: `false`. |
| `category` | string | Filter by category. |
| `tag` | string | Filter by tag. |
| `min_trust` | float | Minimum trust score (0.0-1.0). |
| `cursor` | string | Pagination cursor. |
| `limit` | integer | Items per page (1-100, default 30). |

**Response**: `200 OK`

```json
{
  "data": [
    {
      "identity": { "name": "acme/my-agent", "version": "2.1.0", "..." : "..." },
      "kind": "agent",
      "score": 0.95,
      "highlights": {
        "description": "An AI agent that handles <em>customer support</em> inquiries"
      }
    }
  ],
  "pagination": { "..." }
}
```

#### 6.10.2 Categories

```
GET /v1/categories
```

Returns the list of available categories with artifact counts.

**Response**: `200 OK`

```json
{
  "data": [
    { "name": "conversational", "count": 42 },
    { "name": "coding", "count": 38 },
    { "name": "data-analysis", "count": 25 }
  ]
}
```

#### 6.10.3 Tags

```
GET /v1/tags
```

Returns popular tags with usage counts.

Query parameters:

| Parameter | Type | Description |
|---|---|---|
| `prefix` | string | Filter tags by prefix (autocomplete). |
| `limit` | integer | Number of tags to return (default 50, max 200). |

**Response**: `200 OK`

### 6.11 Skill Content Endpoints

#### 6.11.1 Get Skill Content

```
GET /v1/skills/{name}/versions/{version}/content
```

Returns the skill's entrypoint content (SKILL.md) directly.

**Response**: `200 OK` with `Content-Type: text/markdown`

#### 6.11.2 Get Skill Asset

```
GET /v1/skills/{name}/versions/{version}/content/{path}
```

Returns a specific asset from the skill's content bundle.

**Response**: `200 OK` with appropriate `Content-Type`.

### 6.12 Dependency Graph Endpoints

#### 6.12.1 Get Dependency Graph

```
GET /v1/{kind}/{name}/versions/{version}/dependencies
```

Returns the dependency graph for an artifact.

Query parameters:

| Parameter | Type | Description |
|---|---|---|
| `depth` | integer | Maximum depth to traverse (default 1, max 10). |
| `direction` | string | `downstream` (what this depends on) or `upstream` (what depends on this). Default: `downstream`. |

**Response**: `200 OK`

```json
{
  "data": {
    "root": {
      "name": "acme/my-agent",
      "version": "2.1.0",
      "kind": "agent"
    },
    "edges": [
      {
        "from": "acme/my-agent@2.1.0",
        "to": "acme/github-mcp@2.3.0",
        "type": "tool",
        "required": true
      },
      {
        "from": "acme/my-agent@2.1.0",
        "to": "acme/code-review@1.2.0",
        "type": "skill",
        "required": false
      }
    ],
    "nodes": [
      {
        "name": "acme/github-mcp",
        "version": "2.3.0",
        "kind": "mcp-server",
        "status": "published",
        "trustScore": 0.92
      },
      {
        "name": "acme/code-review",
        "version": "1.2.0",
        "kind": "skill",
        "status": "published",
        "trustScore": 0.88
      }
    ]
  }
}
```

#### 6.12.2 Check Dependency Compatibility

```
POST /v1/{kind}/{name}/versions/{version}/dependencies/check
```

Validates that all declared BOM dependencies can be resolved and are compatible.

**Response**: `200 OK`

```json
{
  "data": {
    "compatible": true,
    "resolved": [
      { "name": "acme/github-mcp", "requested": ">=2.0.0", "resolved": "2.3.0" },
      { "name": "acme/code-review", "requested": ">=1.0.0", "resolved": "1.2.0" }
    ],
    "warnings": [],
    "errors": []
  }
}
```

### 6.13 Federation Endpoints

See [Section 12](#12-federation) for federation-specific API details.

#### 6.13.1 List Federated Registries

```
GET /v1/federation/peers
```

#### 6.13.2 Search Federated

```
GET /v1/federation/search
```

### 6.14 README Endpoints

#### 6.14.1 Upload README

```
PUT /v1/{kind}/{name}/versions/{version}/readme
Content-Type: text/markdown

# My Agent

This agent provides...
```

**Response**: `200 OK`

#### 6.14.2 Get README

```
GET /v1/{kind}/{name}/versions/{version}/readme
```

**Response**: `200 OK` with `Content-Type: text/markdown` or `text/html`.

#### 6.14.3 Get Latest README

```
GET /v1/{kind}/{name}/readme
```

Returns the README for the latest version.

### 6.15 Embedding Endpoints

#### 6.15.1 Upsert Embedding

```
PUT /v1/{kind}/{name}/versions/{version}/embedding
```

Stores or updates the semantic embedding for an artifact version. Used by the indexer for semantic search.

**Request body**:

```json
{
  "vector": [0.1, 0.2, ...],
  "provider": "openai",
  "model": "text-embedding-3-large",
  "dimensions": 3072,
  "checksum": "sha256:abc123"
}
```

**Response**: `200 OK`

**Authentication**: Required. Admin only.

#### 6.15.2 Get Embedding Metadata

```
GET /v1/{kind}/{name}/versions/{version}/embedding
```

Returns metadata about the stored embedding (without the vector payload).

**Response**: `200 OK`

### 6.16 Health and Admin Endpoints

#### 6.16.1 Health Check

```
GET /v1/health
```

**Response**: `200 OK`

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime": "72h15m30s"
}
```

#### 6.16.2 OpenAPI Specification

```
GET /v1/openapi.json
```

Returns the auto-generated OpenAPI 3.1 specification for the registry API.

### 6.17 Admin API

Admin endpoints are served under a separate path prefix (e.g., `/admin/v1`) and require admin authentication. They provide the same CRUD operations as the public API but without the published-only filter, plus additional management operations.

```
GET    /admin/v1/{kind}                              # List all (including unpublished)
POST   /admin/v1/{kind}                              # Create artifact (admin)
GET    /admin/v1/{kind}/{name}/versions/{version}     # Get any version
DELETE /admin/v1/{kind}/{name}/versions/{version}     # Delete any version
POST   /admin/v1/{kind}/{name}/versions/{version}/publish    # Publish
POST   /admin/v1/{kind}/{name}/versions/{version}/unpublish  # Unpublish
```

---

## 7. OCI Integration

### 7.1 Overview

The Agent Registry acts as a **metadata index over OCI-compliant registries**. Binary content (container images, skill content archives, model weights) is stored in standard OCI registries. The Agent Registry stores structured metadata that references this content via OCI references and digests.

This separation of concerns provides several benefits:
* Leverages existing OCI infrastructure (Docker Hub, GitHub Container Registry, Amazon ECR, etc.).
* Avoids duplicating storage and distribution capabilities.
* Allows the registry to focus on governance, discovery, and trust.
* Enables multi-registry artifact storage (an artifact can reference content in multiple OCI registries).

### 7.2 OCI Annotations

The Agent Registry uses a dedicated annotation namespace for OCI manifests and layers. All annotations are prefixed with `ai.agentregistry.`.

#### 7.2.1 Manifest Annotations

| Annotation | Type | Description |
|---|---|---|
| `ai.agentregistry.kind` | string | Artifact kind: `agent`, `skill`, `mcp-server`. |
| `ai.agentregistry.name` | string | Fully qualified artifact name. |
| `ai.agentregistry.version` | string | Semantic version. |
| `ai.agentregistry.title` | string | Human-readable title. |
| `ai.agentregistry.description` | string | Brief description. |
| `ai.agentregistry.registry` | string | Source registry URL. |
| `ai.agentregistry.created` | string | RFC3339 creation timestamp. |
| `ai.agentregistry.authors` | string | Comma-separated author names. |
| `ai.agentregistry.license` | string | SPDX license identifier. |
| `ai.agentregistry.source` | string | Source repository URL. |
| `ai.agentregistry.trust-score` | string | Numeric trust score (0.0-1.0). |
| `ai.agentregistry.promotion-status` | string | Current promotion status. |
| `ai.agentregistry.bom-digest` | string | Digest of the BOM layer. |

#### 7.2.2 Example OCI Manifest

```json
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "config": {
    "mediaType": "application/vnd.ai.agentregistry.agent.config.v1+json",
    "digest": "sha256:config-digest...",
    "size": 1024
  },
  "layers": [
    {
      "mediaType": "application/vnd.ai.agentregistry.agent.content.v1.tar+gzip",
      "digest": "sha256:layer1-digest...",
      "size": 52428800
    },
    {
      "mediaType": "application/vnd.ai.agentregistry.bom.v1+json",
      "digest": "sha256:bom-digest...",
      "size": 4096,
      "annotations": {
        "ai.agentregistry.layer.purpose": "bom"
      }
    }
  ],
  "annotations": {
    "ai.agentregistry.kind": "agent",
    "ai.agentregistry.name": "acme/my-agent",
    "ai.agentregistry.version": "2.1.0",
    "ai.agentregistry.title": "My Agent",
    "org.opencontainers.image.created": "2026-01-15T12:00:00Z",
    "org.opencontainers.image.source": "https://github.com/acme/my-agent"
  }
}
```

### 7.3 Kind-Specific Media Types

Each artifact kind has its own set of OCI media types.

#### 7.3.1 Agent Media Types

| Media Type | Description |
|---|---|
| `application/vnd.ai.agentregistry.agent.config.v1+json` | Agent configuration layer (artifact schema). |
| `application/vnd.ai.agentregistry.agent.content.v1.tar+gzip` | Agent content (application code, etc.). |
| `application/vnd.ai.agentregistry.bom.v1+json` | Bill of Materials (shared across kinds). |
| `application/vnd.ai.agentregistry.provenance.v1+json` | Provenance attestation (shared across kinds). |
| `application/vnd.ai.agentregistry.eval.v1+json` | Evaluation results (shared across kinds). |

#### 7.3.2 Skill Media Types

| Media Type | Description |
|---|---|
| `application/vnd.ai.agentregistry.skill.config.v1+json` | Skill configuration layer (artifact schema). |
| `application/vnd.ai.agentregistry.skill.content.v1.tar+gzip` | Skill content bundle (SKILL.md, scripts, assets). |
| `application/vnd.ai.agentregistry.bom.v1+json` | Bill of Materials. |

#### 7.3.3 MCP Server Media Types

| Media Type | Description |
|---|---|
| `application/vnd.ai.agentregistry.mcp-server.config.v1+json` | MCP server configuration layer. |
| `application/vnd.ai.agentregistry.mcp-server.content.v1.tar+gzip` | MCP server content. |
| `application/vnd.ai.agentregistry.bom.v1+json` | Bill of Materials. |

### 7.4 BOM as OCI Layer

The Bill of Materials is stored as a dedicated OCI layer with the media type `application/vnd.ai.agentregistry.bom.v1+json`. This allows tools to extract and inspect the BOM without downloading the full artifact content.

The BOM layer contains the JSON-serialized BOM object as defined in [Section 3](#3-bill-of-materials-bom).

### 7.5 Skill Content as OCI Layer

Skill content (SKILL.md, scripts, assets, references) is packaged as a tar+gzip archive stored as an OCI layer. The archive structure mirrors the `content` field paths:

```
skill-content.tar.gz
├── SKILL.md                    # Entrypoint
├── scripts/
│   ├── setup.sh
│   └── validate.py
├── references/
│   └── style-guide.md
├── templates/
│   └── output.jinja2
└── examples/
    └── sample-input.json
```

### 7.6 Periodic Verification

The registry SHOULD periodically verify that OCI references in stored artifacts remain valid and accessible. The verification process:

1. **Digest verification**: For artifacts with digest references, verify that the digest still resolves to the expected content.

2. **Tag resolution**: For tag-based references, verify that the tag still exists and resolve its current digest.

3. **Availability check**: Verify that the OCI registry hosting the content is accessible.

4. **Staleness detection**: Flag artifacts whose OCI references have not been verified within a configurable period (default: 7 days).

Verification results are recorded and influence the trust signal computation (see [Section 10](#10-trust-signals)).

```yaml
# Registry verification configuration
verification:
  enabled: true
  intervalHours: 24
  timeoutSeconds: 30
  maxConcurrent: 10
  staleThresholdDays: 7
```

### 7.7 OCI Push/Pull Workflow

#### 7.7.1 Publishing an Artifact

```
1. Developer builds artifact content
2. Developer packages content as OCI image/artifact
3. Developer pushes to OCI registry (e.g., ghcr.io)
4. Developer registers artifact metadata with Agent Registry
   - Includes OCI reference in the `artifacts` field
5. Agent Registry validates OCI reference is reachable
6. Agent Registry stores metadata and indexes for discovery
```

#### 7.7.2 Consuming an Artifact

```
1. Consumer discovers artifact via Agent Registry search/browse
2. Consumer retrieves artifact metadata (including OCI references)
3. Consumer pulls content directly from OCI registry
4. Consumer configures and deploys the artifact
```

---

## 8. Eval Results Interface

### 8.1 Overview

The Evaluation Results Interface provides a standardized way to submit, store, query, and aggregate evaluation results for registry artifacts. Evaluation results are a key input to the promotion lifecycle (required for the `draft -> evaluated` transition) and the trust signal computation.

### 8.2 EvalRecord Schema

An `EvalRecord` represents a single evaluation run against a specific artifact version. The schema is intentionally provider-agnostic: any evaluation framework (eval-hub, lm-evaluation-harness, Garak, RAGAS, or custom pipelines) can submit results as long as they conform to this envelope.

```json
{
  "id": "eval-abc123",
  "artifactName": "acme/my-agent",
  "artifactVersion": "2.1.0",
  "artifactKind": "agent",
  "category": "safety",
  "provider": {
    "name": "garak",
    "version": "0.9.0",
    "url": "https://github.com/NVIDIA/garak"
  },
  "benchmark": {
    "name": "toxicity",
    "version": "1.0.0",
    "suite": "garak-safety",
    "description": "Measures toxic content generation rates under adversarial probing"
  },
  "evaluator": {
    "name": "acme-eval-pipeline",
    "version": "3.2.0",
    "type": "automated",
    "identity": "https://github.com/acme/eval-pipeline"
  },
  "results": {
    "overallScore": 0.87,
    "metrics": {
      "accuracy": { "value": 0.92, "unit": "ratio", "higher_is_better": true },
      "latency_p50": { "value": 1.2, "unit": "seconds", "higher_is_better": false },
      "latency_p95": { "value": 3.8, "unit": "seconds", "higher_is_better": false },
      "token_efficiency": { "value": 0.78, "unit": "ratio", "higher_is_better": true },
      "safety_score": { "value": 0.99, "unit": "ratio", "higher_is_better": true }
    },
    "perTask": [
      {
        "taskId": "greeting-flow",
        "score": 0.95,
        "metrics": {
          "accuracy": { "value": 0.98 },
          "latency_p50": { "value": 0.8 }
        }
      },
      {
        "taskId": "complex-troubleshooting",
        "score": 0.72,
        "metrics": {
          "accuracy": { "value": 0.80 },
          "latency_p50": { "value": 2.1 }
        }
      }
    ]
  },
  "context": {
    "environment": "staging",
    "hardware": "4x NVIDIA A10G",
    "modelVersion": "claude-sonnet-4-20250514",
    "datasetSize": 500,
    "datasetHash": "sha256:dataset-hash...",
    "startedAt": "2026-01-15T10:00:00Z",
    "completedAt": "2026-01-15T10:45:00Z",
    "parameters": {
      "temperature": 0.7,
      "maxTokens": 4096
    }
  },
  "attestation": {
    "signedBy": "eval-pipeline@acme.com",
    "signature": "base64-encoded-signature...",
    "signatureAlgorithm": "ed25519",
    "certificateChain": ["base64-cert-1...", "base64-cert-2..."]
  },
  "createdAt": "2026-01-15T10:50:00Z"
}
```

### 8.3 Schema Fields

#### 8.3.1 Top-Level Fields

| Field | Type | Required | Description |
|---|---|---|---|
| `id` | string | Generated | Unique identifier for this evaluation record. Generated by the registry. |
| `artifactName` | string | Yes | Fully qualified artifact name. |
| `artifactVersion` | string | Yes | Artifact version evaluated. |
| `artifactKind` | string | Yes | Artifact kind. |
| `category` | string | No | Evaluation category. See [Section 8.3.2](#832-evaluation-category). Default: `functional`. |
| `provider` | Provider | No | Evaluation provider/framework. See [Section 8.3.3](#833-provider). |
| `benchmark` | Benchmark | Yes | Benchmark identification. |
| `evaluator` | Evaluator | Yes | Who/what performed the evaluation. |
| `results` | Results | Yes | Evaluation results. |
| `context` | Context | No | Evaluation context and environment. |
| `attestation` | Attestation | No | Cryptographic attestation of results. |
| `createdAt` | string | Generated | RFC3339 timestamp when the record was created. |

#### 8.3.2 Evaluation Category

The `category` field classifies the type of evaluation. This drives how results are aggregated and how they contribute to trust signals (see [Section 10](#10-trust-signals)).

| Value | Description |
|---|---|
| `functional` | Standard capability and accuracy benchmarks (e.g., HumanEval, SWE-bench, MMLU). |
| `safety` | Safety and alignment evaluations including toxicity, bias, and harmful content detection. |
| `red-team` | Adversarial testing — prompt injection, jailbreaking, PII leakage, and other attack surface probing. |
| `performance` | Latency, throughput, resource utilization, and cost efficiency. |
| `custom` | Provider-defined or organization-specific evaluation types. |

The registry does not prescribe which evaluation frameworks map to which categories. A Garak run might produce `safety` or `red-team` records depending on the benchmark. An eval-hub job might produce records across multiple categories in a single run.

#### 8.3.3 Provider

The `provider` block identifies the evaluation framework that produced the results. This is optional — the registry accepts results from any source — but when present, it enables provider-aware aggregation and filtering.

```yaml
provider:
  name: "eval-hub"
  version: "1.2.0"
  url: "https://github.com/eval-hub/eval-hub"
```

| Field | Type | Required | Description |
|---|---|---|---|
| `provider.name` | string | Yes | Provider identifier (e.g., `eval-hub`, `garak`, `lm-evaluation-harness`, `ragas`, `custom`). |
| `provider.version` | string | No | Provider version. |
| `provider.url` | string | No | URL for the provider project or documentation. |

The registry treats providers as opaque — it does not validate provider names against a fixed list. Any string is accepted. Well-known providers include:

* `eval-hub` — Unified evaluation orchestrator supporting multiple frameworks
* `garak` — LLM vulnerability scanner and red-teaming framework
* `lm-evaluation-harness` — EleutherAI's evaluation harness for language models
* `ragas` — RAG evaluation framework
* `guidellm` — Structured output evaluation

#### 8.3.4 Benchmark

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Benchmark identifier (e.g., `accuracy`, `latency`, `customer-satisfaction`, `toxicity`, `prompt-injection`). |
| `version` | string | No | Benchmark version. |
| `suite` | string | No | Benchmark suite name (groups related benchmarks). |
| `description` | string | No | What this benchmark measures. |

#### 8.3.5 Evaluator

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Evaluator identifier. |
| `version` | string | No | Evaluator version. |
| `type` | string | Yes | Evaluator type: `automated`, `human`, `hybrid`, `llm-as-judge`. |
| `identity` | string | No | Verifiable identity (URL, email). |

#### 8.3.6 Results

| Field | Type | Required | Description |
|---|---|---|---|
| `overallScore` | float | Yes | Aggregate score normalized to 0.0-1.0. |
| `metrics` | map[string]Metric | No | Named metrics with values and metadata. |
| `perTask` | TaskResult[] | No | Per-task breakdown of results. |

#### 8.3.7 Metric

| Field | Type | Required | Description |
|---|---|---|---|
| `value` | float | Yes | Metric value. |
| `unit` | string | No | Unit of measurement (e.g., `ratio`, `seconds`, `tokens`, `percent`). |
| `higher_is_better` | boolean | No | Whether higher values indicate better performance. Default: `true`. |

#### 8.3.8 TaskResult

| Field | Type | Required | Description |
|---|---|---|---|
| `taskId` | string | Yes | Task identifier. |
| `score` | float | Yes | Task score (0.0-1.0). |
| `metrics` | map[string]Metric | No | Task-specific metrics. |
| `detail` | string | No | Human-readable detail or explanation. |

#### 8.3.9 Context

| Field | Type | Required | Description |
|---|---|---|---|
| `environment` | string | No | Execution environment (e.g., `production`, `staging`, `ci`). |
| `hardware` | string | No | Hardware description. |
| `modelVersion` | string | No | Specific model version used during evaluation. |
| `datasetSize` | integer | No | Number of evaluation samples. |
| `datasetHash` | string | No | Hash of the evaluation dataset for reproducibility. |
| `startedAt` | string | No | RFC3339 timestamp of evaluation start. |
| `completedAt` | string | No | RFC3339 timestamp of evaluation completion. |
| `parameters` | map[string]any | No | Evaluation parameters (temperature, sampling, etc.). |

#### 8.3.10 Attestation

| Field | Type | Required | Description |
|---|---|---|---|
| `signedBy` | string | Yes | Identity of the signer. |
| `signature` | string | Yes | Base64-encoded signature over the results. |
| `signatureAlgorithm` | string | Yes | Signature algorithm (e.g., `ed25519`, `ecdsa-p256`). |
| `certificateChain` | string[] | No | Certificate chain for signature verification. |

### 8.4 Aggregation

When multiple evaluation records exist for an artifact version, the registry computes aggregate statistics:

```json
{
  "aggregation": {
    "recordCount": 5,
    "latestScore": 0.87,
    "averageScore": 0.84,
    "minScore": 0.72,
    "maxScore": 0.92,
    "trend": "improving",
    "trendConfidence": 0.85,
    "benchmarkCoverage": {
      "total": 3,
      "covered": 3,
      "benchmarks": {
        "accuracy": { "latest": 0.92, "count": 5 },
        "latency": { "latest": 0.78, "count": 3 },
        "safety": { "latest": 0.99, "count": 2 }
      }
    }
  }
}
```

#### 8.4.1 Trend Detection

The registry detects evaluation trends across versions:

| Trend | Condition |
|---|---|
| `improving` | Score increased in the last 3+ consecutive evaluations. |
| `stable` | Score variance is within 5% over the last 3+ evaluations. |
| `declining` | Score decreased in the last 3+ consecutive evaluations. |
| `volatile` | Score variance exceeds 15% over the last 3+ evaluations. |
| `insufficient-data` | Fewer than 3 evaluation records exist. |

### 8.5 Eval Baseline

An artifact can define minimum evaluation requirements:

```yaml
evalBaseline:
  minimumScore: 0.7
  requiredBenchmarks:
    - "accuracy"
    - "safety"
  requiredCategories:
    - "functional"
    - "safety"
  requiredEvaluatorTypes:
    - "automated"
  minimumDatasetSize: 100
  maximumLatency:
    p95: 5.0
    unit: "seconds"
```

| Field | Type | Required | Description |
|---|---|---|---|
| `minimumScore` | float | No | Minimum overall score required (0.0-1.0). Default: `0.0`. |
| `requiredBenchmarks` | string[] | No | Benchmark names that must have results. |
| `requiredCategories` | string[] | No | Evaluation categories that must be covered (e.g., `["functional", "safety"]`). When set, promotion to `evaluated` requires at least one eval record in each listed category. |
| `requiredEvaluatorTypes` | string[] | No | Required evaluator types. |
| `minimumDatasetSize` | integer | No | Minimum number of evaluation samples. |
| `maximumLatency` | object | No | Maximum acceptable latency thresholds. |
| `evaluationCadence` | string | No | When evaluation is required: `on-publish`, `weekly`, `monthly`, `manual`. Default: `on-publish`. |

---

## 9. Provenance and Supply Chain

### 9.1 Overview

The Agent Registry supports supply chain security through provenance tracking, attestation verification, and software bill of materials (SBOM) integration. This section defines how provenance information is captured, stored, verified, and presented for registry artifacts.

Supply chain security for AI artifacts presents unique challenges beyond traditional software:
* **Model selection provenance**: Why was a particular model chosen, and who authorized it?
* **Prompt authorship**: Who wrote the system prompts, and have they been reviewed for safety?
* **Tool verification**: Are the MCP servers in the BOM legitimate and uncompromised?
* **Data lineage**: What training or evaluation data influenced the artifact?

### 9.2 SLSA Framework Integration

The Agent Registry aligns with the [Supply-chain Levels for Software Artifacts (SLSA)](https://slsa.dev/) framework. Artifacts can declare their SLSA compliance level.

#### 9.2.1 SLSA Levels

| Level | Requirements | Agent Registry Support |
|---|---|---|
| SLSA L0 | No provenance | Default for draft artifacts. |
| SLSA L1 | Provenance exists, stating how the artifact was built. | Supported via `provenance.buildType` and `provenance.builder`. |
| SLSA L2 | Signed provenance, generated by a hosted build service. | Supported via `provenance.attestation`. |
| SLSA L3 | Hardened build platform with tamper-resistant provenance. | Supported via Sigstore integration and build platform verification. |

#### 9.2.2 Provenance Schema

```yaml
provenance:
  slsaLevel: 2
  buildType: "https://slsa.dev/provenance/v1"
  builder:
    id: "https://github.com/actions/runner"
    version: "2.311.0"
    builderDependencies:
      - uri: "https://github.com/actions/runner-images"
        digest:
          sha256: "abc123..."
  invocation:
    configSource:
      uri: "https://github.com/acme/my-agent"
      digest:
        sha256: "source-digest..."
      entryPoint: ".github/workflows/build.yml"
    parameters: {}
    environment:
      os: "ubuntu-22.04"
      arch: "amd64"
  metadata:
    buildInvocationId: "github-actions-run-12345"
    buildStartedOn: "2026-01-15T10:00:00Z"
    buildFinishedOn: "2026-01-15T10:05:23Z"
    completeness:
      parameters: true
      environment: true
      materials: true
    reproducible: false
  materials:
    - uri: "https://github.com/acme/my-agent"
      digest:
        sha256: "src-digest..."
    - uri: "ghcr.io/base-images/python:3.12"
      digest:
        sha256: "base-image-digest..."
```

| Field | Type | Required | Description |
|---|---|---|---|
| `slsaLevel` | integer | No | Declared SLSA compliance level (0-3). |
| `buildType` | string | Yes | URI identifying the build type schema. |
| `builder` | object | Yes | Identifies the build platform. |
| `builder.id` | string | Yes | URI identifying the builder. |
| `builder.version` | string | No | Builder version. |
| `builder.builderDependencies` | Material[] | No | Dependencies of the build platform itself. |
| `invocation` | object | No | How the build was invoked. |
| `invocation.configSource` | Material | No | Source of the build configuration. |
| `invocation.configSource.entryPoint` | string | No | Entry point within the config source (e.g., workflow file). |
| `invocation.parameters` | map | No | Build parameters. |
| `invocation.environment` | map | No | Build environment variables. |
| `metadata` | object | No | Build metadata. |
| `metadata.buildInvocationId` | string | No | Unique identifier for the build invocation. |
| `metadata.buildStartedOn` | string | No | RFC3339 build start time. |
| `metadata.buildFinishedOn` | string | No | RFC3339 build end time. |
| `metadata.completeness` | object | No | Which fields are guaranteed complete. |
| `metadata.reproducible` | boolean | No | Whether the build is reproducible. Default: `false`. |
| `materials` | Material[] | No | Input materials used in the build. |

#### 9.2.3 Material

| Field | Type | Required | Description |
|---|---|---|---|
| `uri` | string | Yes | URI of the material. |
| `digest` | map[string]string | Yes | Content digests (algorithm -> hex value). |

### 9.3 In-toto Attestation

The registry supports [in-toto](https://in-toto.io/) attestation format for supply chain verification.

```json
{
  "_type": "https://in-toto.io/Statement/v1",
  "subject": [
    {
      "name": "ghcr.io/acme/my-agent",
      "digest": {
        "sha256": "artifact-digest..."
      }
    }
  ],
  "predicateType": "https://slsa.dev/provenance/v1",
  "predicate": {
    "buildDefinition": {
      "buildType": "https://github.com/actions/runner",
      "externalParameters": {},
      "internalParameters": {},
      "resolvedDependencies": []
    },
    "runDetails": {
      "builder": {
        "id": "https://github.com/actions/runner"
      },
      "metadata": {
        "invocationId": "github-actions-run-12345"
      }
    }
  }
}
```

### 9.4 Sigstore Integration

The Agent Registry integrates with [Sigstore](https://www.sigstore.dev/) for keyless signing and transparency logging.

#### 9.4.1 Signing

Artifact publishers can sign their artifacts using Sigstore's keyless signing flow:

1. Publisher authenticates with an OIDC provider (GitHub, Google, etc.).
2. Sigstore issues a short-lived certificate binding the publisher's identity to a signing key.
3. The artifact (or its provenance attestation) is signed.
4. The signature and certificate are recorded in the Rekor transparency log.

#### 9.4.2 Attestation Storage

```yaml
attestation:
  intoto:
    predicateType: "https://slsa.dev/provenance/v1"
    statement: "base64-encoded-statement..."
  sigstore:
    bundleUrl: "https://rekor.sigstore.dev/api/v1/log/entries/abc123"
    logIndex: 12345678
    signedEntryTimestamp: "base64-timestamp..."
    certificate: "base64-cert..."
    chain: ["base64-intermediate-cert..."]
```

| Field | Type | Required | Description |
|---|---|---|---|
| `attestation.intoto` | object | No | In-toto attestation. |
| `attestation.intoto.predicateType` | string | Yes | Predicate type URI. |
| `attestation.intoto.statement` | string | Yes | Base64-encoded in-toto statement. |
| `attestation.sigstore` | object | No | Sigstore verification data. |
| `attestation.sigstore.bundleUrl` | string | Yes | Rekor transparency log entry URL. |
| `attestation.sigstore.logIndex` | integer | No | Rekor log index. |
| `attestation.sigstore.signedEntryTimestamp` | string | No | Signed entry timestamp from Rekor. |
| `attestation.sigstore.certificate` | string | No | Base64-encoded signing certificate. |
| `attestation.sigstore.chain` | string[] | No | Certificate chain. |

#### 9.4.3 Verification

When verifying Sigstore attestations, the registry performs:

1. **Certificate validation**: Verify the signing certificate against the Sigstore root of trust.
2. **Identity binding**: Verify that the certificate's identity matches the artifact publisher.
3. **Transparency log**: Verify the signature exists in the Rekor transparency log.
4. **Timestamp validation**: Verify the signature was created during the certificate's validity period.
5. **Content integrity**: Verify the signed content matches the artifact's OCI digest.

### 9.5 SBOM Integration

The registry supports standard Software Bill of Materials formats for traditional software dependency tracking.

#### 9.5.1 Supported Formats

| Format | Version | Media Type |
|---|---|---|
| SPDX | 2.3 | `application/spdx+json` |
| CycloneDX | 1.5 | `application/vnd.cyclonedx+json` |

#### 9.5.2 SBOM Storage

SBOMs are stored as OCI layers alongside the artifact content:

```json
{
  "mediaType": "application/vnd.ai.agentregistry.sbom.spdx.v1+json",
  "digest": "sha256:sbom-digest...",
  "size": 8192,
  "annotations": {
    "ai.agentregistry.layer.purpose": "sbom",
    "ai.agentregistry.sbom.format": "spdx",
    "ai.agentregistry.sbom.version": "2.3"
  }
}
```

#### 9.5.3 SBOM API

```
PUT /v1/{kind}/{name}/versions/{version}/sbom
Content-Type: application/spdx+json
```

```
GET /v1/{kind}/{name}/versions/{version}/sbom
Accept: application/spdx+json, application/vnd.cyclonedx+json
```

### 9.6 Agent-Specific Provenance Extensions

AI artifacts have unique provenance considerations beyond traditional software.

#### 9.6.1 Prompt Authorship

```yaml
promptProvenance:
  prompts:
    - name: "system-prompt"
      author: "jane@acme.com"
      reviewedBy:
        - "security-team@acme.com"
        - "ethics-board@acme.com"
      reviewedAt: "2026-01-10T09:00:00Z"
      hash: "sha256:prompt-hash..."
      safetyReview:
        status: "approved"
        reviewer: "ethics-board@acme.com"
        notes: "Reviewed for bias, toxicity, and prompt injection resistance"
```

#### 9.6.2 Model Selection Provenance

```yaml
modelProvenance:
  models:
    - name: "claude-sonnet-4-20250514"
      selectionRationale: "Best accuracy/cost tradeoff for customer support workload"
      selectedBy: "ai-team@acme.com"
      selectedAt: "2026-01-05T14:00:00Z"
      alternatives:
        - name: "gpt-4o"
          rejectionReason: "Higher latency, lower accuracy on internal benchmarks"
      complianceReview:
        status: "approved"
        reviewer: "compliance@acme.com"
        framework: "EU AI Act"
```

#### 9.6.3 Tool Verification

```yaml
toolProvenance:
  tools:
    - name: "acme/github-mcp"
      version: "2.3.0"
      verifiedAt: "2026-01-14T16:00:00Z"
      verifiedBy: "security-team@acme.com"
      verificationMethod: "code-review"
      trustScore: 0.92
      knownVulnerabilities: []
```

### 9.7 Provenance Verification API

```
POST /v1/{kind}/{name}/versions/{version}/provenance/verify
```

**Response**:

```json
{
  "data": {
    "verified": true,
    "slsaLevel": 2,
    "checks": [
      {
        "name": "build-provenance",
        "passed": true,
        "detail": "Valid SLSA v1 provenance from GitHub Actions"
      },
      {
        "name": "sigstore-signature",
        "passed": true,
        "detail": "Valid Sigstore bundle, Rekor log index 12345678"
      },
      {
        "name": "oci-digest-match",
        "passed": true,
        "detail": "Artifact digest matches provenance subject"
      },
      {
        "name": "publisher-identity",
        "passed": true,
        "detail": "Certificate identity matches namespace owner"
      },
      {
        "name": "sbom-present",
        "passed": true,
        "detail": "SPDX 2.3 SBOM present with 42 packages"
      },
      {
        "name": "prompt-review",
        "passed": true,
        "detail": "All prompts reviewed and approved"
      },
      {
        "name": "model-compliance",
        "passed": true,
        "detail": "Model selection reviewed for EU AI Act compliance"
      }
    ],
    "verifiedAt": "2026-01-15T12:05:00Z"
  }
}
```

---

## 10. Trust Signals

### 10.1 Overview

Trust Signals provide a composite quality indicator for registry artifacts. The trust score is a weighted aggregate of six components, each measuring a different aspect of artifact trustworthiness. The overall score is normalized to a range of 0.0 to 1.0.

Trust signals are computed by the registry and updated periodically. They are informational and do not gate any lifecycle transitions by default (though registries can configure minimum trust score requirements for promotion).

### 10.2 Composite Score Formula

```
overallScore = sum(component_score[i] * weight[i]) for i in components
```

The six components and their default weights:

| Component | Weight | Description |
|---|---|---|
| Provenance | 0.25 | Supply chain verification and attestation quality. |
| Evaluation | 0.25 | Evaluation results and benchmark coverage. |
| Code Health | 0.20 | Vulnerability scanning, dependency freshness, code quality. |
| Publisher Reputation | 0.15 | Publisher's track record and verification status. |
| Community | 0.10 | Downloads, dependents, community engagement. |
| Operational | 0.05 | Runtime availability and reliability (for remote artifacts). |

The weights sum to 1.0. Registry administrators can customize the weights, but the sum must always equal 1.0.

### 10.3 Component Scoring

#### 10.3.1 Provenance Score (Weight: 0.25)

Measures the completeness and strength of supply chain provenance.

| Factor | Points | Max |
|---|---|---|
| SLSA L0 (no provenance) | 0.0 | |
| SLSA L1 (provenance exists) | 0.3 | |
| SLSA L2 (signed provenance) | 0.6 | |
| SLSA L3 (hardened build) | 0.8 | |
| Sigstore attestation present | +0.1 | 1.0 |
| SBOM present | +0.05 | 1.0 |
| Prompt authorship documented | +0.05 | 1.0 |
| Model selection provenance | +0.05 | 1.0 |
| All OCI digests verified | +0.05 | 1.0 |

**Score calculation**: Base SLSA score + applicable bonuses, capped at 1.0.

#### 10.3.2 Evaluation Score (Weight: 0.25)

Measures the breadth and quality of evaluation results, including safety and adversarial testing. The registry considers eval records across all categories (`functional`, `safety`, `red-team`, `performance`, `custom`) when computing this score.

| Factor | Points | Max |
|---|---|---|
| No evaluation records | 0.0 | |
| At least one eval record (any category) | 0.2 | |
| Meets evalBaseline minimumScore | +0.15 | |
| All requiredBenchmarks covered | +0.15 | |
| Safety evaluation present (category: `safety`) | +0.15 | |
| Red-team evaluation present (category: `red-team`) | +0.15 | |
| Multiple evaluator types | +0.05 | |
| Trend is `improving` or `stable` | +0.05 | |
| Attestation on eval records | +0.1 | 1.0 |

**Score calculation**: Sum of applicable factors, capped at 1.0.

Safety and red-team evaluations are distinct: a `safety` eval measures alignment and harm prevention under normal use (e.g., toxicity detection, bias measurement), while a `red-team` eval measures resilience under adversarial attack (e.g., prompt injection, jailbreaking, PII extraction). Both are valued because an artifact can pass safety benchmarks while still being vulnerable to adversarial probing.

Results from any provider are accepted. For example, a Garak red-teaming run submitted with `category: "red-team"` and `provider.name: "garak"` contributes the same trust signal points as a custom red-team pipeline — the registry scores based on category coverage, not provider identity.

#### 10.3.3 Code Health Score (Weight: 0.20)

Measures software quality and security posture.

| Factor | Points | Max |
|---|---|---|
| No known vulnerabilities (critical) | 0.4 | |
| No known vulnerabilities (high) | +0.2 | |
| No known vulnerabilities (medium) | +0.1 | |
| Dependencies up to date | +0.1 | |
| License declared and compatible | +0.1 | |
| Repository linked and accessible | +0.1 | 1.0 |

**Scoring details**:
* Vulnerability data is sourced from SBOM scanning (if SBOM present) or OCI image scanning.
* "Up to date" means all declared dependencies have versions within their current major version.
* License compatibility is checked against a configurable allowlist.

#### 10.3.4 Publisher Reputation Score (Weight: 0.15)

Measures the trustworthiness of the artifact publisher.

| Factor | Points | Max |
|---|---|---|
| Anonymous / unverified publisher | 0.0 | |
| Namespace verified (DNS or GitHub) | 0.4 | |
| Publisher has 5+ published artifacts | +0.1 | |
| Publisher has 20+ published artifacts | +0.1 | |
| No artifacts deprecated for security | +0.1 | |
| Publisher account age > 6 months | +0.1 | |
| Publisher is organizational account | +0.1 | |
| Publisher has signed CLA/agreement | +0.1 | 1.0 |

#### 10.3.5 Community Score (Weight: 0.10)

Measures community adoption and engagement.

| Factor | Points | Max |
|---|---|---|
| > 10 downloads (last 30 days) | 0.1 | |
| > 100 downloads (last 30 days) | 0.2 | |
| > 1,000 downloads (last 30 days) | 0.3 | |
| > 10,000 downloads (last 30 days) | 0.4 | |
| 1+ dependents (other artifacts) | +0.2 | |
| 5+ dependents | +0.1 | |
| Repository stars > 10 | +0.1 | |
| Repository stars > 100 | +0.1 | |
| Active maintenance (commit in 90 days) | +0.1 | 1.0 |

#### 10.3.6 Operational Score (Weight: 0.05)

Measures runtime reliability. Only applicable to artifacts with remote transport (MCP servers with HTTP/SSE, deployed agents).

| Factor | Points | Max |
|---|---|---|
| No operational data | 0.5 (neutral) | |
| Uptime > 99.9% (30 days) | 1.0 | |
| Uptime > 99.0% | 0.8 | |
| Uptime > 95.0% | 0.5 | |
| Uptime < 95.0% | 0.2 | |
| Health check responding | +0.0 (included in uptime) | 1.0 |

For artifacts without remote transport (stdio MCP servers, skills), the operational score defaults to 0.5 (neutral) so it does not unfairly penalize or boost the overall score.

### 10.4 Trust Signal Schema

```json
{
  "overallScore": 0.85,
  "components": {
    "provenance": {
      "score": 0.95,
      "weight": 0.25,
      "weightedContribution": 0.2375,
      "factors": [
        { "name": "slsa-level", "value": "L2", "points": 0.6 },
        { "name": "sigstore-attestation", "value": "present", "points": 0.1 },
        { "name": "sbom", "value": "spdx-2.3", "points": 0.05 },
        { "name": "prompt-authorship", "value": "documented", "points": 0.05 },
        { "name": "model-provenance", "value": "documented", "points": 0.05 },
        { "name": "oci-digests", "value": "verified", "points": 0.05 }
      ]
    },
    "evaluation": {
      "score": 0.90,
      "weight": 0.25,
      "weightedContribution": 0.225,
      "factors": [
        { "name": "eval-records", "value": "8 records", "points": 0.2 },
        { "name": "baseline-met", "value": "true", "points": 0.15 },
        { "name": "benchmarks-covered", "value": "3/3", "points": 0.15 },
        { "name": "safety-eval-present", "value": "garak toxicity + bias", "points": 0.15 },
        { "name": "red-team-eval-present", "value": "garak prompt-injection + pii-leakage", "points": 0.15 },
        { "name": "evaluator-diversity", "value": "automated+human", "points": 0.05 },
        { "name": "trend", "value": "stable", "points": 0.05 }
      ]
    },
    "codeHealth": {
      "score": 0.90,
      "weight": 0.20,
      "weightedContribution": 0.18,
      "factors": [
        { "name": "critical-vulns", "value": "0", "points": 0.4 },
        { "name": "high-vulns", "value": "0", "points": 0.2 },
        { "name": "medium-vulns", "value": "0", "points": 0.1 },
        { "name": "deps-current", "value": "true", "points": 0.1 },
        { "name": "license", "value": "Apache-2.0", "points": 0.1 }
      ]
    },
    "publisherReputation": {
      "score": 0.75,
      "weight": 0.15,
      "weightedContribution": 0.1125,
      "factors": [
        { "name": "namespace-verified", "value": "github", "points": 0.4 },
        { "name": "artifact-count", "value": "12", "points": 0.2 },
        { "name": "no-security-deprecations", "value": "true", "points": 0.1 },
        { "name": "account-age", "value": "2 years", "points": 0.1 }
      ]
    },
    "community": {
      "score": 0.70,
      "weight": 0.10,
      "weightedContribution": 0.07,
      "factors": [
        { "name": "downloads-30d", "value": "2500", "points": 0.3 },
        { "name": "dependents", "value": "3", "points": 0.2 },
        { "name": "repo-stars", "value": "45", "points": 0.1 },
        { "name": "active-maintenance", "value": "true", "points": 0.1 }
      ]
    },
    "operational": {
      "score": 0.85,
      "weight": 0.05,
      "weightedContribution": 0.0425,
      "factors": [
        { "name": "uptime-30d", "value": "99.2%", "points": 0.8 }
      ]
    }
  },
  "lastUpdated": "2026-01-15T12:00:00Z",
  "nextUpdate": "2026-01-16T12:00:00Z"
}
```

### 10.5 Trust Signal Configuration

Registry administrators can customize trust signal computation:

```yaml
trustSignals:
  enabled: true
  updateIntervalHours: 24
  weights:
    provenance: 0.25
    evaluation: 0.25
    codeHealth: 0.20
    publisherReputation: 0.15
    community: 0.10
    operational: 0.05
  thresholds:
    minimumForPublish: 0.0      # No minimum by default
    warningBelow: 0.5           # Show warning in UI
    displayPrecision: 2         # Decimal places
  vulnerabilityScanning:
    enabled: true
    provider: "trivy"
    scanOnPublish: true
```

### 10.6 Trust Score Display

Trust scores are presented with a tiered badge system for quick visual assessment:

| Score Range | Badge | Color |
|---|---|---|
| 0.90 - 1.00 | Excellent | Green |
| 0.75 - 0.89 | Good | Blue |
| 0.50 - 0.74 | Fair | Yellow |
| 0.25 - 0.49 | Low | Orange |
| 0.00 - 0.24 | Minimal | Red |

---

## 11. Access Control

### 11.1 Overview

The Agent Registry implements role-based access control (RBAC) with five predefined roles. Access control is enforced at the namespace level, with global admin privileges available for registry-wide operations.

### 11.2 Roles

| Role | Description | Scope |
|---|---|---|
| `anonymous` | Unauthenticated access. Read-only access to published artifacts. | Global |
| `viewer` | Authenticated read access. Can view published artifacts and their own drafts. | Namespace or global |
| `publisher` | Can create, update, and publish artifacts within their namespace(s). | Namespace |
| `approver` | Can review and approve artifacts for publication. | Namespace |
| `admin` | Full access to all operations across all namespaces. | Global |

### 11.3 Permission Matrix

| Operation | Anonymous | Viewer | Publisher | Approver | Admin |
|---|---|---|---|---|---|
| List published artifacts | Yes | Yes | Yes | Yes | Yes |
| Get published artifact details | Yes | Yes | Yes | Yes | Yes |
| Search published artifacts | Yes | Yes | Yes | Yes | Yes |
| List all artifacts (incl. drafts) | No | Own only | Namespace | Namespace | All |
| Create artifact (push) | No | No | Namespace | No | All |
| Update draft artifact | No | No | Own | No | All |
| Delete draft artifact | No | No | Own | No | All |
| Submit eval results | No | No | Namespace | Namespace | All |
| Promote to evaluated | No | No | Namespace | Namespace | All |
| Promote to approved | No | No | No | Namespace | All |
| Promote to published | No | No | Namespace | Namespace | All |
| Deprecate artifact | No | No | Namespace | Namespace | All |
| Archive artifact | No | No | No | No | All |
| Manage namespace members | No | No | No | No | All |
| Configure registry | No | No | No | No | All |
| Delete any artifact | No | No | No | No | All |
| Manage federation | No | No | No | No | All |

### 11.4 Authentication Methods

The registry supports multiple authentication methods, each producing a session with associated permissions.

#### 11.4.1 GitHub Access Token (`github-at`)

Authentication using a GitHub personal access token. The token is used to verify the user's GitHub identity and derive namespace permissions.

```
POST /v1/auth/github
Content-Type: application/json

{
  "github_token": "ghp_abc123..."
}
```

**Response**:

```json
{
  "registry_token": "eyJ...",
  "expires_at": 1737000000
}
```

Namespace mapping: GitHub username `janedoe` -> namespace `io.github.janedoe/*`.

#### 11.4.2 GitHub Actions OIDC (`github-oidc`)

Keyless authentication from GitHub Actions workflows using OIDC tokens.

```
POST /v1/auth/github-oidc
Content-Type: application/json

{
  "oidc_token": "eyJ..."
}
```

The OIDC token's `sub` claim (e.g., `repo:acme/my-agent:ref:refs/heads/main`) is used to derive permissions.

#### 11.4.3 Generic OIDC (`oidc`)

Authentication using any OIDC-compliant identity provider.

```
POST /v1/auth/oidc
Content-Type: application/json

{
  "oidc_token": "eyJ...",
  "provider": "https://accounts.google.com"
}
```

#### 11.4.4 DNS Verification (`dns`)

Namespace ownership verification through DNS TXT records.

```
POST /v1/auth/dns
Content-Type: application/json

{
  "namespace": "com.example",
  "challenge": "agentregistry-verify=abc123..."
}
```

The registry verifies that the DNS TXT record `_agentregistry.example.com` contains the challenge string.

#### 11.4.5 HTTP Verification (`http`)

Namespace ownership verification through an HTTP well-known endpoint.

```
POST /v1/auth/http
Content-Type: application/json

{
  "namespace": "com.example",
  "challenge": "abc123..."
}
```

The registry verifies that `https://example.com/.well-known/agentregistry-verify` returns the challenge string.

#### 11.4.6 No Authentication (`none`)

For local development and testing only. All operations are permitted without authentication.

**WARNING**: This mode MUST NOT be used in production deployments.

### 11.5 Namespace Ownership

Namespace ownership is established through one of the verification methods above. Once verified, the namespace owner has `publisher` permissions for all artifacts under that namespace.

#### 11.5.1 Namespace Mapping Rules

| Verification Method | Namespace Pattern | Example |
|---|---|---|
| GitHub (personal) | `io.github.{username}/*` | `io.github.janedoe/my-agent` |
| GitHub (organization) | `io.github.{org}/*` | `io.github.acme/my-agent` |
| DNS | `{reversed-domain}/*` | `com.example/my-agent` |
| HTTP | `{reversed-domain}/*` | `com.example/my-agent` |
| OIDC | Configured per provider | Depends on identity mapping |

#### 11.5.2 Namespace Delegation

Namespace owners can delegate permissions to other users:

```
POST /admin/v1/namespaces/{namespace}/members
Content-Type: application/json

{
  "identity": "github:janedoe",
  "role": "publisher",
  "permissions": [
    { "action": "push", "resource": "io.github.acme/*" },
    { "action": "publish", "resource": "io.github.acme/*" }
  ]
}
```

### 11.6 JWT Token Structure

Registry authentication produces a short-lived JWT token (default: 5 minutes) with the following claims:

```json
{
  "iss": "agent-registry",
  "sub": "github:janedoe",
  "iat": 1737000000,
  "exp": 1737000300,
  "nbf": 1737000000,
  "auth_method": "github-at",
  "auth_method_sub": "janedoe",
  "permissions": [
    { "action": "push", "resource": "io.github.janedoe/*" },
    { "action": "publish", "resource": "io.github.janedoe/*" },
    { "action": "delete", "resource": "io.github.janedoe/*" },
    { "action": "read", "resource": "*" }
  ]
}
```

Tokens are signed using Ed25519 (EdDSA algorithm).

### 11.7 Permission Resolution

Permissions are evaluated in the following order:

1. **Global admin check**: If the session has a permission with `resource: "*"`, all operations are allowed.

2. **System session check**: Internal system sessions (e.g., indexer, reconciler) bypass all authorization checks.

3. **Public action check**: Read operations on published artifacts are allowed for all sessions (including anonymous).

4. **Permission matching**: For other operations, the session's permissions are checked against the requested resource using pattern matching:
   * `*` matches all resources.
   * `io.github.janedoe/*` matches all resources under that namespace.
   * `io.github.janedoe/my-agent` matches that specific resource.

5. **Blocked namespace check**: Before issuing tokens, the registry checks the namespace against a denylist of blocked namespaces.

### 11.8 Blocked Namespaces

The registry maintains a denylist of namespaces that are prohibited from publishing. This is used to prevent abuse, impersonation, and trademark violations.

```go
// Example blocked namespaces
BlockedNamespaces = []string{
    "io.github.spammer",
    "com.evil-domain",
}
```

Attempts to authenticate with a blocked namespace result in an error:

```
"Your namespace is blocked. Raise an issue at https://github.com/agentregistry-dev/agentregistry if you think this is a mistake."
```

---

## 12. Federation

### 12.1 Overview

Federation allows multiple Agent Registry instances to discover and share artifacts. Federated registries can search across peer registries, synchronize metadata, and establish trust relationships.

Federation is designed to support:
* **Multi-team deployments**: Teams run their own registries but need to discover artifacts from other teams.
* **Public/private split**: Organizations run a private registry but consume from a public registry.
* **Geographic distribution**: Registries in different regions synchronize for low-latency access.
* **Ecosystem growth**: Independent registries form a network for broader artifact discovery.

### 12.2 Well-Known Discovery Endpoint

Every Agent Registry instance SHOULD expose a discovery endpoint at:

```
GET /.well-known/agent-registry.json
```

**Response**:

```json
{
  "version": "1.0",
  "registryUrl": "https://registry.example.com",
  "apiVersion": "v1",
  "apiBase": "https://registry.example.com/v1",
  "federation": {
    "enabled": true,
    "peersEndpoint": "/v1/federation/peers",
    "searchEndpoint": "/v1/federation/search",
    "syncEndpoint": "/v1/federation/sync"
  },
  "capabilities": {
    "kinds": ["agent", "skill", "mcp-server"],
    "authentication": ["github-at", "github-oidc", "oidc", "dns"],
    "search": {
      "semantic": true,
      "fullText": true
    },
    "trustSignals": true,
    "provenance": true,
    "evaluation": true
  },
  "contact": {
    "name": "Registry Admin",
    "email": "admin@example.com"
  }
}
```

### 12.3 Peer Registration

Registries can register each other as peers to enable cross-registry operations.

#### 12.3.1 Register Peer

```
POST /v1/federation/peers
Content-Type: application/json
Authorization: Bearer <admin-token>

{
  "url": "https://other-registry.example.com",
  "name": "ACME Internal Registry",
  "description": "Internal registry for ACME engineering teams",
  "trustLevel": "trusted",
  "syncPolicy": {
    "enabled": true,
    "direction": "pull",
    "intervalMinutes": 60,
    "kinds": ["agent", "skill", "mcp-server"],
    "namespaceFilter": ["com.acme.*"],
    "minimumTrustScore": 0.5
  }
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `url` | string | Yes | Peer registry base URL. |
| `name` | string | Yes | Human-readable peer name. |
| `description` | string | No | Peer description. |
| `trustLevel` | string | No | Trust classification: `untrusted`, `basic`, `trusted`, `verified`. Default: `basic`. |
| `syncPolicy` | object | No | Synchronization policy for this peer. |
| `syncPolicy.enabled` | boolean | No | Whether sync is enabled. Default: `false`. |
| `syncPolicy.direction` | string | No | Sync direction: `pull`, `push`, `bidirectional`. Default: `pull`. |
| `syncPolicy.intervalMinutes` | integer | No | Sync interval in minutes. Default: `60`. |
| `syncPolicy.kinds` | string[] | No | Artifact kinds to sync. Default: all kinds. |
| `syncPolicy.namespaceFilter` | string[] | No | Namespace patterns to sync (glob). Default: `["*"]`. |
| `syncPolicy.minimumTrustScore` | float | No | Minimum trust score for synced artifacts. Default: `0.0`. |

#### 12.3.2 List Peers

```
GET /v1/federation/peers
```

**Response**:

```json
{
  "data": [
    {
      "id": "peer-abc123",
      "url": "https://other-registry.example.com",
      "name": "ACME Internal Registry",
      "trustLevel": "trusted",
      "status": "healthy",
      "lastSync": "2026-01-15T12:00:00Z",
      "artifactCount": 142,
      "latencyMs": 45
    }
  ]
}
```

#### 12.3.3 Remove Peer

```
DELETE /v1/federation/peers/{peerId}
```

### 12.4 Pull-Based Synchronization

The primary synchronization model is pull-based: a registry periodically fetches metadata from its peers.

#### 12.4.1 Sync Protocol

```
GET /v1/federation/sync
```

Query parameters:

| Parameter | Type | Description |
|---|---|---|
| `since` | string | RFC3339 timestamp. Only return artifacts updated after this time. |
| `kind` | string | Filter by kind. |
| `namespace` | string | Filter by namespace pattern (glob). |
| `cursor` | string | Pagination cursor. |
| `limit` | integer | Items per page (max 100). |

**Response**:

```json
{
  "data": [
    {
      "identity": { "name": "acme/my-agent", "version": "2.1.0", "..." : "..." },
      "kind": "agent",
      "status": "published",
      "updatedAt": "2026-01-15T12:00:00Z",
      "trustSignals": { "overallScore": 0.85 },
      "sourceRegistry": "https://registry.example.com"
    }
  ],
  "pagination": {
    "nextCursor": "cursor-xyz",
    "count": 50
  },
  "syncMetadata": {
    "serverTimestamp": "2026-01-15T12:05:00Z",
    "totalChanges": 142
  }
}
```

#### 12.4.2 Sync Process

1. **Discovery**: The syncing registry fetches `/.well-known/agent-registry.json` from the peer.
2. **Incremental fetch**: Using the `since` parameter, only artifacts updated since the last sync are fetched.
3. **Filter application**: Sync policy filters (kinds, namespaces, trust score) are applied.
4. **Metadata storage**: Synced artifact metadata is stored with a `sourceRegistry` annotation.
5. **Conflict resolution**: If the same artifact exists in multiple registries, the version with the highest trust score wins. If scores are equal, the most recently updated version wins.
6. **Verification**: The syncing registry MAY verify OCI references from the peer to ensure content availability.

#### 12.4.3 Sync Status

```
GET /v1/federation/peers/{peerId}/sync-status
```

**Response**:

```json
{
  "data": {
    "peerId": "peer-abc123",
    "lastSyncStarted": "2026-01-15T12:00:00Z",
    "lastSyncCompleted": "2026-01-15T12:02:30Z",
    "lastSyncStatus": "success",
    "artifactsSynced": 15,
    "artifactsSkipped": 3,
    "errors": [],
    "nextSyncScheduled": "2026-01-15T13:00:00Z"
  }
}
```

### 12.5 Cross-Registry Search

Federated search allows querying across all peer registries from a single endpoint.

```
GET /v1/federation/search
```

Query parameters: Same as the standard search endpoint ([Section 6.10.1](#6101-search-cross-kind)), plus:

| Parameter | Type | Description |
|---|---|---|
| `registries` | string | Comma-separated list of peer IDs to search. Default: all peers. |
| `include_local` | boolean | Include local registry results. Default: `true`. |
| `timeout_ms` | integer | Maximum time to wait for peer responses. Default: `5000`. |

**Response**:

```json
{
  "data": [
    {
      "identity": { "name": "acme/my-agent", "version": "2.1.0" },
      "kind": "agent",
      "score": 0.95,
      "sourceRegistry": {
        "url": "https://registry.example.com",
        "name": "ACME Registry",
        "trustLevel": "trusted"
      }
    },
    {
      "identity": { "name": "other-org/similar-agent", "version": "1.0.0" },
      "kind": "agent",
      "score": 0.82,
      "sourceRegistry": {
        "url": "https://other-registry.example.com",
        "name": "Other Registry",
        "trustLevel": "basic"
      }
    }
  ],
  "searchMetadata": {
    "registriesQueried": 3,
    "registriesResponded": 3,
    "registriesTimedOut": 0,
    "totalResults": 42
  }
}
```

### 12.6 Trust Policies

Federated registries establish trust policies that govern how artifacts from peers are treated.

#### 12.6.1 Trust Levels

| Level | Description | Permissions |
|---|---|---|
| `untrusted` | Unknown registry. Search only, no sync. | Search results shown with warning. |
| `basic` | Known registry. Metadata sync allowed. | Synced artifacts shown but not auto-promoted. |
| `trusted` | Verified registry. Full sync with filtering. | Synced artifacts eligible for local promotion. |
| `verified` | Mutually authenticated registry. Full interop. | Synced artifacts inherit trust signals. |

#### 12.6.2 Trust Policy Configuration

```yaml
federation:
  defaultTrustLevel: "basic"
  policies:
    - peerPattern: "*.internal.example.com"
      trustLevel: "verified"
      autoSync: true
      inheritTrustSignals: true
    - peerPattern: "registry.public-example.com"
      trustLevel: "trusted"
      autoSync: true
      minimumTrustScore: 0.7
      allowedNamespaces: ["com.public-example.*"]
    - peerPattern: "*"
      trustLevel: "basic"
      autoSync: false
```

#### 12.6.3 Trust Signal Inheritance

When syncing artifacts from a `verified` peer, trust signals can be inherited:

* The peer's trust score is preserved but annotated with the source.
* The local registry may apply a discount factor (e.g., `inherited_score = peer_score * 0.9`).
* Provenance and evaluation records from the peer are included in trust computation.
* Operational scores are not inherited (they must be measured locally).

### 12.7 Federation Security

#### 12.7.1 Peer Authentication

Peer-to-peer communication is authenticated using mutual TLS or pre-shared API keys:

```yaml
federation:
  authentication:
    method: "mtls"  # or "api-key"
    certFile: "/etc/agentregistry/federation.crt"
    keyFile: "/etc/agentregistry/federation.key"
    caCertFile: "/etc/agentregistry/federation-ca.crt"
```

#### 12.7.2 Data Integrity

All synced metadata includes a content hash that the receiving registry MUST verify:

```json
{
  "artifact": { "..." },
  "integrity": {
    "hash": "sha256:metadata-hash...",
    "algorithm": "sha256",
    "signedBy": "peer-registry.example.com",
    "signature": "base64-signature..."
  }
}
```

#### 12.7.3 Rate Limiting

Federated endpoints enforce rate limits to prevent abuse:

| Endpoint | Rate Limit |
|---|---|
| `/v1/federation/search` | 100 requests/minute per peer |
| `/v1/federation/sync` | 10 requests/minute per peer |
| `/v1/federation/peers` | 10 requests/minute |

---

## Appendix A: JSON Schema References

The normative JSON Schema files for the Registry Artifact schema, BOM, EvalRecord, and Provenance are maintained in the `schemas/` directory of the specification repository:

* `schemas/registry-artifact.schema.json` -- Base registry artifact schema
* `schemas/agent.schema.json` -- Agent kind extensions
* `schemas/skill.schema.json` -- Skill kind extensions
* `schemas/mcp-server.schema.json` -- MCP server kind extensions
* `schemas/bom.schema.json` -- Bill of Materials schema
* `schemas/eval-record.schema.json` -- Evaluation record schema
* `schemas/provenance.schema.json` -- Provenance schema
* `schemas/trust-signals.schema.json` -- Trust signals schema

## Appendix B: Media Type Registry

| Media Type | Description |
|---|---|
| `application/vnd.ai.agentregistry.agent.config.v1+json` | Agent configuration |
| `application/vnd.ai.agentregistry.agent.content.v1.tar+gzip` | Agent content archive |
| `application/vnd.ai.agentregistry.skill.config.v1+json` | Skill configuration |
| `application/vnd.ai.agentregistry.skill.content.v1.tar+gzip` | Skill content archive |
| `application/vnd.ai.agentregistry.mcp-server.config.v1+json` | MCP server configuration |
| `application/vnd.ai.agentregistry.mcp-server.content.v1.tar+gzip` | MCP server content archive |
| `application/vnd.ai.agentregistry.bom.v1+json` | Bill of Materials |
| `application/vnd.ai.agentregistry.provenance.v1+json` | Provenance attestation |
| `application/vnd.ai.agentregistry.eval.v1+json` | Evaluation results |
| `application/vnd.ai.agentregistry.sbom.spdx.v1+json` | SPDX SBOM |
| `application/vnd.ai.agentregistry.sbom.cyclonedx.v1+json` | CycloneDX SBOM |

## Appendix C: OCI Annotation Namespace

All Agent Registry OCI annotations use the prefix `ai.agentregistry.`. See [Section 7.2](#72-oci-annotations) for the complete list.

## Appendix D: Glossary

| Term | Definition |
|---|---|
| **A2A** | Agent-to-Agent protocol for inter-agent communication. |
| **ADK** | Agent Development Kit, a framework for building AI agents. |
| **Artifact** | A metadata record in the registry describing an agent, skill, or MCP server. |
| **BOM** | Bill of Materials; structured dependency declaration. |
| **CycloneDX** | An OWASP standard for SBOM. |
| **EdDSA** | Edwards-curve Digital Signature Algorithm (Ed25519). |
| **EvalRecord** | A structured evaluation result for an artifact version. |
| **Federation** | The ability for registries to discover and share artifacts. |
| **In-toto** | A framework for securing software supply chains. |
| **MCP** | Model Context Protocol; a protocol for AI tool integration. |
| **Namespace** | A scoping mechanism for artifact names (reverse-domain notation). |
| **OCI** | Open Container Initiative; standards for container images and distribution. |
| **OIDC** | OpenID Connect; an authentication protocol. |
| **Promotion** | The lifecycle transition of an artifact version through states. |
| **Rekor** | Sigstore's transparency log for software supply chain. |
| **SBOM** | Software Bill of Materials. |
| **Semver** | Semantic Versioning 2.0.0. |
| **Sigstore** | A suite of tools for signing and verifying software artifacts. |
| **SLSA** | Supply-chain Levels for Software Artifacts; a security framework. |
| **SPDX** | Software Package Data Exchange; an SBOM standard. |
| **Trust Signal** | A composite quality indicator derived from multiple components. |

---

*End of Agent Registry Specification v1.0*
