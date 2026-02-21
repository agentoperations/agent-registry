package service

import (
	"context"

	"github.com/agentoperations/agent-registry/internal/model"
)

// RegistryService defines the business logic interface.
// The registry is a metadata store with a promotion lifecycle.
// Eval records are accepted from external tools — the registry stores them, not computes them.
type RegistryService interface {
	CreateArtifact(ctx context.Context, kind model.Kind, artifact *model.RegistryArtifact) (*model.RegistryArtifact, error)
	GetArtifact(ctx context.Context, kind model.Kind, name, version string) (*model.RegistryArtifact, error)
	ListArtifacts(ctx context.Context, kind model.Kind, filter *model.ArtifactFilter, limit, offset int) ([]*model.RegistryArtifact, int, error)
	ListVersions(ctx context.Context, kind model.Kind, name string) ([]*model.RegistryArtifact, error)
	DeleteArtifact(ctx context.Context, kind model.Kind, name, version string) error

	PromoteArtifact(ctx context.Context, kind model.Kind, name, version string, req *model.PromotionRequest) (*model.RegistryArtifact, error)

	SubmitEval(ctx context.Context, kind model.Kind, name, version string, eval *model.EvalRecord) (*model.EvalRecord, error)
	ListEvals(ctx context.Context, kind model.Kind, name, version string, filter *model.EvalFilter) ([]*model.EvalRecord, error)

	Inspect(ctx context.Context, kind model.Kind, name, version string) (*model.InspectResult, error)

	Search(ctx context.Context, query string, kinds []model.Kind, limit, offset int) ([]*model.RegistryArtifact, int, error)

	GetDependencies(ctx context.Context, kind model.Kind, name, version string) (*model.DependencyGraph, error)
}
