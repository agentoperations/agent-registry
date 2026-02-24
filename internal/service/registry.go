package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/agentoperations/agent-registry/internal/model"
	"github.com/agentoperations/agent-registry/internal/store"
	"github.com/google/uuid"
)

type registryService struct {
	store store.Store
}

func New(s store.Store) RegistryService {
	return &registryService{store: s}
}

func (s *registryService) CreateArtifact(ctx context.Context, kind model.Kind, a *model.RegistryArtifact) (*model.RegistryArtifact, error) {
	a.Kind = kind
	a.Status = model.StatusDraft

	// If a standard document is present, extract identity fields from it
	if a.HasStandardDocument() {
		if err := a.ExtractIdentity(); err != nil {
			return nil, fmt.Errorf("extract identity from standard document: %w", err)
		}
	}

	// Validate: must have name and version after extraction
	if a.Identity.Name == "" || a.Identity.Version == "" {
		return nil, fmt.Errorf("artifact must have name and version (via identity or standard document)")
	}

	if _, err := s.store.CreateArtifact(ctx, a); err != nil {
		return nil, err
	}
	if err := s.store.UpdateLatestFlag(ctx, kind, a.Identity.Name); err != nil {
		return nil, fmt.Errorf("update latest flag: %w", err)
	}

	artifact, _, err := s.store.GetArtifact(ctx, kind, a.Identity.Name, a.Identity.Version)
	if err != nil {
		return nil, err
	}
	return artifact, nil
}

func (s *registryService) GetArtifact(ctx context.Context, kind model.Kind, name, version string) (*model.RegistryArtifact, error) {
	if version == "" || version == "latest" {
		a, _, err := s.store.GetLatestArtifact(ctx, kind, name)
		return a, err
	}
	a, _, err := s.store.GetArtifact(ctx, kind, name, version)
	return a, err
}

func (s *registryService) ListArtifacts(ctx context.Context, kind model.Kind, filter *model.ArtifactFilter, limit, offset int) ([]*model.RegistryArtifact, int, error) {
	if filter == nil {
		filter = &model.ArtifactFilter{}
	}
	filter.Kind = kind
	if limit <= 0 {
		limit = 30
	}
	return s.store.ListArtifacts(ctx, filter, limit, offset)
}

func (s *registryService) ListVersions(ctx context.Context, kind model.Kind, name string) ([]*model.RegistryArtifact, error) {
	return s.store.ListVersions(ctx, kind, name)
}

func (s *registryService) DeleteArtifact(ctx context.Context, kind model.Kind, name, version string) error {
	return s.store.DeleteArtifact(ctx, kind, name, version)
}

func (s *registryService) PromoteArtifact(ctx context.Context, kind model.Kind, name, version string, req *model.PromotionRequest) (*model.RegistryArtifact, error) {
	a, id, err := s.store.GetArtifact(ctx, kind, name, version)
	if err != nil {
		return nil, err
	}

	if !model.IsValidTransition(a.Status, req.TargetStatus) {
		return nil, &model.PromotionGateError{
			From:    a.Status,
			To:      req.TargetStatus,
			Message: fmt.Sprintf("invalid transition from %s to %s", a.Status, req.TargetStatus),
		}
	}

	if err := s.store.SetStatus(ctx, kind, name, version, req.TargetStatus); err != nil {
		return nil, err
	}
	if err := s.store.LogPromotion(ctx, id, a.Status, req.TargetStatus, req.Comment); err != nil {
		return nil, fmt.Errorf("log promotion: %w", err)
	}

	return s.GetArtifact(ctx, kind, name, version)
}

func (s *registryService) SubmitEval(ctx context.Context, kind model.Kind, name, version string, eval *model.EvalRecord) (*model.EvalRecord, error) {
	_, id, err := s.store.GetArtifact(ctx, kind, name, version)
	if err != nil {
		return nil, err
	}

	eval.ID = uuid.New().String()
	eval.ArtifactName = name
	eval.ArtifactVersion = version
	eval.ArtifactKind = kind
	if eval.Category == "" {
		eval.Category = model.EvalCategoryFunctional
	}

	if err := s.store.CreateEvalRecord(ctx, id, eval); err != nil {
		return nil, err
	}

	return eval, nil
}

func (s *registryService) ListEvals(ctx context.Context, kind model.Kind, name, version string, filter *model.EvalFilter) ([]*model.EvalRecord, error) {
	_, id, err := s.store.GetArtifact(ctx, kind, name, version)
	if err != nil {
		return nil, err
	}
	return s.store.ListEvalRecords(ctx, id, filter)
}

func (s *registryService) Inspect(ctx context.Context, kind model.Kind, name, version string) (*model.InspectResult, error) {
	a, id, err := s.store.GetArtifact(ctx, kind, name, version)
	if err != nil {
		return nil, err
	}

	// Eval summary
	evals, err := s.store.ListEvalRecords(ctx, id, nil)
	if err != nil {
		return nil, err
	}
	summary := &model.EvalSummary{
		TotalRecords: len(evals),
		Categories:   map[string]int{},
	}
	var totalScore float64
	for _, e := range evals {
		summary.Categories[string(e.Category)]++
		totalScore += e.Results.OverallScore
		summary.LatestScore = e.Results.OverallScore
	}
	if len(evals) > 0 {
		summary.AverageScore = totalScore / float64(len(evals))
	}

	// Promotion history
	promotions, err := s.store.ListPromotions(ctx, id)
	if err != nil {
		return nil, err
	}

	return &model.InspectResult{
		Artifact:    a,
		EvalSummary: summary,
		Promotions:  promotions,
	}, nil
}

func (s *registryService) Search(ctx context.Context, query string, kinds []model.Kind, limit, offset int) ([]*model.RegistryArtifact, int, error) {
	if limit <= 0 {
		limit = 30
	}
	return s.store.SearchArtifacts(ctx, query, kinds, limit, offset)
}

func (s *registryService) GetDependencies(ctx context.Context, kind model.Kind, name, version string) (*model.DependencyGraph, error) {
	a, _, err := s.store.GetArtifact(ctx, kind, name, version)
	if err != nil {
		return nil, err
	}

	graph := &model.DependencyGraph{
		Artifact: model.DependencyNode{
			Name:     a.Identity.Name,
			Kind:     a.Kind,
			Version:  a.Identity.Version,
			Resolved: true,
		},
	}

	if len(a.BOM) == 0 {
		return graph, nil
	}

	// Parse BOM based on kind
	switch kind {
	case model.KindAgent:
		graph.Dependencies = s.resolveAgentBOM(ctx, a.BOM)
	case model.KindSkill:
		graph.Dependencies = s.resolveSkillBOM(ctx, a.BOM)
	}

	return graph, nil
}

func (s *registryService) ExportStandardDoc(ctx context.Context, kind model.Kind, name, version string) (json.RawMessage, string, error) {
	a, err := s.GetArtifact(ctx, kind, name, version)
	if err != nil {
		return nil, "", err
	}
	doc := a.StandardDocument()
	if len(doc) == 0 {
		return nil, "", fmt.Errorf("artifact has no standard document")
	}
	contentType := "application/json"
	return doc, contentType, nil
}

func (s *registryService) resolveAgentBOM(ctx context.Context, bomRaw json.RawMessage) []model.DependencyNode {
	var bom struct {
		Tools  []struct{ Name, Version string } `json:"tools"`
		Skills []struct{ Name, Version string } `json:"skills"`
		Models []struct{ Name, Provider string } `json:"models"`
	}
	if err := json.Unmarshal(bomRaw, &bom); err != nil {
		return nil
	}

	var deps []model.DependencyNode
	for _, t := range bom.Tools {
		node := model.DependencyNode{Name: t.Name, Kind: model.KindMCPServer, VersionConstraint: t.Version}
		if a, _, err := s.store.GetLatestArtifact(ctx, model.KindMCPServer, t.Name); err == nil {
			node.Version = a.Identity.Version
			node.Resolved = true
		}
		deps = append(deps, node)
	}
	for _, sk := range bom.Skills {
		node := model.DependencyNode{Name: sk.Name, Kind: model.KindSkill, VersionConstraint: sk.Version}
		if a, _, err := s.store.GetLatestArtifact(ctx, model.KindSkill, sk.Name); err == nil {
			node.Version = a.Identity.Version
			node.Resolved = true
		}
		deps = append(deps, node)
	}
	for _, m := range bom.Models {
		deps = append(deps, model.DependencyNode{
			Name:     m.Name,
			Kind:     "model",
			Version:  m.Provider,
			Resolved: true, // models are external, always "resolved"
		})
	}
	return deps
}

func (s *registryService) resolveSkillBOM(ctx context.Context, bomRaw json.RawMessage) []model.DependencyNode {
	var bom struct {
		Dependencies []struct{ Name, Version string } `json:"dependencies"`
	}
	if err := json.Unmarshal(bomRaw, &bom); err != nil {
		return nil
	}
	var deps []model.DependencyNode
	for _, d := range bom.Dependencies {
		node := model.DependencyNode{Name: d.Name, Kind: model.KindSkill, VersionConstraint: d.Version}
		if a, _, err := s.store.GetLatestArtifact(ctx, model.KindSkill, d.Name); err == nil {
			node.Version = a.Identity.Version
			node.Resolved = true
		}
		deps = append(deps, node)
	}
	return deps
}
