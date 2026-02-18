package model

import (
	"encoding/json"
	"strings"
	"time"
)

// Kind represents the artifact kind discriminator.
type Kind string

const (
	KindAgent     Kind = "agent"
	KindSkill     Kind = "skill"
	KindMCPServer Kind = "mcp-server"
)

func ParseKind(s string) (Kind, bool) {
	switch strings.ToLower(s) {
	case "agent", "agents":
		return KindAgent, true
	case "skill", "skills":
		return KindSkill, true
	case "mcp-server", "mcp-servers":
		return KindMCPServer, true
	}
	return "", false
}

func (k Kind) Plural() string {
	switch k {
	case KindAgent:
		return "agents"
	case KindSkill:
		return "skills"
	case KindMCPServer:
		return "mcp-servers"
	}
	return string(k) + "s"
}

// Status represents the promotion lifecycle status.
type Status string

const (
	StatusDraft      Status = "draft"
	StatusEvaluated  Status = "evaluated"
	StatusApproved   Status = "approved"
	StatusPublished  Status = "published"
	StatusDeprecated Status = "deprecated"
	StatusArchived   Status = "archived"
)

// Identity uniquely identifies an artifact.
type Identity struct {
	Name        string `json:"name" yaml:"name"`
	Version     string `json:"version" yaml:"version"`
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
}

// OCIReference points to content in an OCI registry.
type OCIReference struct {
	OCI       string    `json:"oci" yaml:"oci"`
	MediaType string    `json:"mediaType,omitempty" yaml:"mediaType,omitempty"`
	Digest    string    `json:"digest,omitempty" yaml:"digest,omitempty"`
	Platform  *Platform `json:"platform,omitempty" yaml:"platform,omitempty"`
}

type Platform struct {
	OS           string `json:"os,omitempty" yaml:"os,omitempty"`
	Architecture string `json:"architecture,omitempty" yaml:"architecture,omitempty"`
}

// Author describes an artifact author.
type Author struct {
	Name  string `json:"name" yaml:"name"`
	Email string `json:"email,omitempty" yaml:"email,omitempty"`
	URL   string `json:"url,omitempty" yaml:"url,omitempty"`
}

// Repository describes a source code repository.
type Repository struct {
	URL    string `json:"url" yaml:"url"`
	Source string `json:"source,omitempty" yaml:"source,omitempty"`
}

// Metadata contains artifact metadata.
type Metadata struct {
	Labels        map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Tags          []string          `json:"tags,omitempty" yaml:"tags,omitempty"`
	Category      string            `json:"category,omitempty" yaml:"category,omitempty"`
	Authors       []Author          `json:"authors,omitempty" yaml:"authors,omitempty"`
	License       string            `json:"license,omitempty" yaml:"license,omitempty"`
	Repository    *Repository       `json:"repository,omitempty" yaml:"repository,omitempty"`
	WebsiteURL    string            `json:"websiteUrl,omitempty" yaml:"websiteUrl,omitempty"`
	Documentation string            `json:"documentation,omitempty" yaml:"documentation,omitempty"`
}

// RegistryArtifact is the top-level polymorphic artifact model.
// Kind-specific fields are stored as json.RawMessage to preserve them without loss.
type RegistryArtifact struct {
	Identity  Identity       `json:"identity" yaml:"identity"`
	Kind      Kind           `json:"kind" yaml:"kind"`
	Artifacts []OCIReference `json:"artifacts" yaml:"artifacts"`
	Metadata  *Metadata      `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Kind-specific extensions (agent, skill, mcp-server)
	Capabilities json.RawMessage `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
	Runtime      json.RawMessage `json:"runtime,omitempty" yaml:"runtime,omitempty"`
	Content      json.RawMessage `json:"content,omitempty" yaml:"content,omitempty"`
	Transport    json.RawMessage `json:"transport,omitempty" yaml:"transport,omitempty"`
	Tools        json.RawMessage `json:"tools,omitempty" yaml:"tools,omitempty"`
	Resources    json.RawMessage `json:"resources,omitempty" yaml:"resources,omitempty"`
	Prompts      json.RawMessage `json:"prompts,omitempty" yaml:"prompts,omitempty"`
	BOM          json.RawMessage `json:"bom,omitempty" yaml:"bom,omitempty"`
	Provenance   json.RawMessage `json:"provenance,omitempty" yaml:"provenance,omitempty"`
	EvalBaseline json.RawMessage `json:"evalBaseline,omitempty" yaml:"evalBaseline,omitempty"`
	TrustSignals json.RawMessage `json:"trustSignals,omitempty" yaml:"trustSignals,omitempty"`

	// Registry-managed fields (not submitted by publisher)
	Status      Status `json:"status,omitempty" yaml:"status,omitempty"`
	IsLatest    bool   `json:"isLatest,omitempty" yaml:"isLatest,omitempty"`
	ContentHash string `json:"contentHash,omitempty" yaml:"contentHash,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty" yaml:"updatedAt,omitempty"`
	PublishedAt string `json:"publishedAt,omitempty" yaml:"publishedAt,omitempty"`
}

// Namespace extracts the namespace from the fully qualified name.
func (a *RegistryArtifact) Namespace() string {
	parts := strings.SplitN(a.Identity.Name, "/", 2)
	if len(parts) == 2 {
		return parts[0]
	}
	return "_default"
}

// ArtifactFilter holds query parameters for listing artifacts.
type ArtifactFilter struct {
	Kind      Kind
	Namespace string
	Status    Status
	Category  string
	Tags      []string
	Search    string
}

// ArtifactResponse wraps an artifact with registry metadata.
type ArtifactResponse struct {
	*RegistryArtifact
	RegistryMeta *RegistryMeta `json:"_registryMeta,omitempty"`
}

type RegistryMeta struct {
	Status      Status `json:"status"`
	IsLatest    bool   `json:"isLatest"`
	PublishedAt string `json:"publishedAt,omitempty"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// ResponseEnvelope is the standard API response wrapper.
type ResponseEnvelope struct {
	Data       interface{}   `json:"data"`
	Meta       *ResponseMeta `json:"_meta,omitempty"`
	Pagination *Pagination   `json:"pagination,omitempty"`
}

type ResponseMeta struct {
	RequestID string `json:"requestId"`
	Timestamp string `json:"timestamp"`
	Registry  string `json:"registry"`
}

type Pagination struct {
	NextCursor string `json:"nextCursor"`
	Count      int    `json:"count"`
	Total      int    `json:"total,omitempty"`
}

func NewResponseMeta(requestID string) *ResponseMeta {
	return &ResponseMeta{
		RequestID: requestID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Registry:  "localhost",
	}
}

// ProblemDetail is an RFC 7807 error response.
type ProblemDetail struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance,omitempty"`
}
