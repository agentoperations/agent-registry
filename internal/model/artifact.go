package model

import (
	"encoding/json"
	"fmt"
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

	// Standard identity documents (envelope model — one per kind)
	AgentCard        json.RawMessage `json:"agentCard,omitempty" yaml:"agentCard,omitempty"`
	AgentCardOrigin  string          `json:"agentCardOrigin,omitempty" yaml:"agentCardOrigin,omitempty"`
	ServerJson       json.RawMessage `json:"serverJson,omitempty" yaml:"serverJson,omitempty"`
	ServerJsonOrigin string          `json:"serverJsonOrigin,omitempty" yaml:"serverJsonOrigin,omitempty"`
	SkillMd          json.RawMessage `json:"skillMd,omitempty" yaml:"skillMd,omitempty"`
	SkillMdOrigin    string          `json:"skillMdOrigin,omitempty" yaml:"skillMdOrigin,omitempty"`

	Metadata *Metadata `json:"metadata,omitempty" yaml:"metadata,omitempty"`

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
// If the name has no slash, treats the entire name as the namespace
// (supports --namespace flag where name is set to just the namespace).
func (a *RegistryArtifact) Namespace() string {
	parts := strings.SplitN(a.Identity.Name, "/", 2)
	if len(parts) == 2 {
		return parts[0]
	}
	if a.Identity.Name != "" {
		return a.Identity.Name
	}
	return "_default"
}

// ExtractIdentity populates Identity fields from the standard document.
func (a *RegistryArtifact) ExtractIdentity() error {
	switch a.Kind {
	case KindAgent:
		return a.extractFromAgentCard()
	case KindMCPServer:
		return a.extractFromServerJson()
	case KindSkill:
		return a.extractFromSkillMd()
	}
	return fmt.Errorf("unknown kind: %s", a.Kind)
}

func (a *RegistryArtifact) extractFromAgentCard() error {
	if len(a.AgentCard) == 0 {
		return nil
	}
	var card struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(a.AgentCard, &card); err != nil {
		return fmt.Errorf("parse agentCard: %w", err)
	}
	if card.Name == "" || card.Version == "" {
		return fmt.Errorf("agentCard missing required fields: name, version")
	}
	ns := a.Namespace()
	a.Identity.Title = card.Name
	a.Identity.Description = card.Description
	a.Identity.Version = card.Version
	if !strings.Contains(a.Identity.Name, "/") || a.Identity.Name == ns {
		a.Identity.Name = ns + "/" + strings.ToLower(strings.ReplaceAll(card.Name, " ", "-"))
	}
	return nil
}

func (a *RegistryArtifact) extractFromServerJson() error {
	if len(a.ServerJson) == 0 {
		return nil
	}
	var sj struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(a.ServerJson, &sj); err != nil {
		return fmt.Errorf("parse serverJson: %w", err)
	}
	if sj.Name == "" || sj.Version == "" {
		return fmt.Errorf("serverJson missing required fields: name, version")
	}
	a.Identity.Version = sj.Version
	a.Identity.Description = sj.Description
	if sj.Title != "" {
		a.Identity.Title = sj.Title
	} else {
		a.Identity.Title = sj.Name
	}
	// If identity.name doesn't have a namespace prefix, use server.json name
	if !strings.Contains(a.Identity.Name, "/") {
		a.Identity.Name = sj.Name
	}
	return nil
}

func (a *RegistryArtifact) extractFromSkillMd() error {
	if len(a.SkillMd) == 0 {
		return nil
	}
	var sm struct {
		Name        string            `json:"name"`
		Description string            `json:"description"`
		Metadata    map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(a.SkillMd, &sm); err != nil {
		return fmt.Errorf("parse skillMd: %w", err)
	}
	if sm.Name == "" {
		return fmt.Errorf("skillMd missing required field: name")
	}
	ns := a.Namespace()
	a.Identity.Title = sm.Name
	a.Identity.Description = sm.Description
	if v, ok := sm.Metadata["version"]; ok {
		a.Identity.Version = v
	}
	if !strings.Contains(a.Identity.Name, "/") || a.Identity.Name == ns {
		a.Identity.Name = ns + "/" + sm.Name
	}
	return nil
}

// StandardDocument returns the standard doc for this artifact's kind.
func (a *RegistryArtifact) StandardDocument() json.RawMessage {
	switch a.Kind {
	case KindAgent:
		return a.AgentCard
	case KindMCPServer:
		return a.ServerJson
	case KindSkill:
		return a.SkillMd
	}
	return nil
}

// HasStandardDocument returns true if a standard doc is present.
func (a *RegistryArtifact) HasStandardDocument() bool {
	return len(a.StandardDocument()) > 0
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
