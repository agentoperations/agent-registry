package model

import "fmt"

// PromotionRequest is the request body for promoting an artifact.
type PromotionRequest struct {
	TargetStatus Status `json:"targetStatus"`
	Comment      string `json:"comment,omitempty"`
	// For deprecation
	Reason      string `json:"reason,omitempty"`
	Alternative string `json:"alternative,omitempty"`
}

// ValidTransitions defines the allowed status transitions.
var ValidTransitions = map[Status][]Status{
	StatusDraft:      {StatusEvaluated},
	StatusEvaluated:  {StatusApproved},
	StatusApproved:   {StatusPublished},
	StatusPublished:  {StatusDeprecated},
	StatusDeprecated: {StatusArchived},
}

// IsValidTransition checks if the transition is allowed.
func IsValidTransition(from, to Status) bool {
	targets, ok := ValidTransitions[from]
	if !ok {
		return false
	}
	for _, t := range targets {
		if t == to {
			return true
		}
	}
	return false
}

// PromotionGateError is returned when promotion gate requirements are not met.
type PromotionGateError struct {
	From    Status
	To      Status
	Message string
}

func (e *PromotionGateError) Error() string {
	return fmt.Sprintf("promotion gate %s->%s: %s", e.From, e.To, e.Message)
}

// InspectResult is a summary view of an artifact's status, eval coverage, and promotion history.
type InspectResult struct {
	Artifact    *RegistryArtifact `json:"artifact"`
	EvalSummary *EvalSummary      `json:"evalSummary"`
	Promotions  []PromotionEntry  `json:"promotions"`
}

type EvalSummary struct {
	TotalRecords    int            `json:"totalRecords"`
	Categories      map[string]int `json:"categories"`
	LatestScore     float64        `json:"latestScore,omitempty"`
	AverageScore    float64        `json:"averageScore,omitempty"`
}

type PromotionEntry struct {
	From      Status `json:"from"`
	To        Status `json:"to"`
	Comment   string `json:"comment,omitempty"`
	Timestamp string `json:"timestamp"`
}

// SearchResult wraps an artifact with relevance info for search.
type SearchResult struct {
	Artifact *RegistryArtifact `json:"artifact"`
	Score    float64           `json:"score,omitempty"`
}

// DependencyGraph represents a resolved dependency tree.
type DependencyGraph struct {
	Artifact     DependencyNode   `json:"artifact"`
	Dependencies []DependencyNode `json:"dependencies"`
}

type DependencyNode struct {
	Name              string           `json:"name"`
	Kind              Kind             `json:"kind"`
	Version           string           `json:"version"`
	VersionConstraint string           `json:"versionConstraint,omitempty"`
	Resolved          bool             `json:"resolved"`
	Dependencies      []DependencyNode `json:"dependencies,omitempty"`
}
