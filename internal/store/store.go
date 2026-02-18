package store

import (
	"context"
	"errors"

	"github.com/agentoperations/agent-registry/internal/model"
)

var (
	ErrNotFound      = errors.New("artifact not found")
	ErrAlreadyExists = errors.New("artifact already exists")
	ErrImmutable     = errors.New("artifact is immutable in current status")
)

// Store defines the database-agnostic storage interface.
// The registry is a metadata store — it stores artifacts and accepts signals
// (eval records) from external tools. It does not compute trust scores.
type Store interface {
	// Transaction support
	InTransaction(ctx context.Context, fn func(ctx context.Context) error) error

	// Artifact CRUD
	CreateArtifact(ctx context.Context, artifact *model.RegistryArtifact) (int64, error)
	GetArtifact(ctx context.Context, kind model.Kind, name, version string) (*model.RegistryArtifact, int64, error)
	GetLatestArtifact(ctx context.Context, kind model.Kind, name string) (*model.RegistryArtifact, int64, error)
	ListArtifacts(ctx context.Context, filter *model.ArtifactFilter, limit, offset int) ([]*model.RegistryArtifact, int, error)
	ListVersions(ctx context.Context, kind model.Kind, name string) ([]*model.RegistryArtifact, error)
	DeleteArtifact(ctx context.Context, kind model.Kind, name, version string) error
	UpdateArtifactBody(ctx context.Context, kind model.Kind, name, version string, artifact *model.RegistryArtifact) error

	// Status management
	SetStatus(ctx context.Context, kind model.Kind, name, version string, status model.Status) error
	UpdateLatestFlag(ctx context.Context, kind model.Kind, name string) error

	// Eval records (external signals)
	CreateEvalRecord(ctx context.Context, artifactID int64, eval *model.EvalRecord) error
	ListEvalRecords(ctx context.Context, artifactID int64, filter *model.EvalFilter) ([]*model.EvalRecord, error)

	// Promotion log
	LogPromotion(ctx context.Context, artifactID int64, from, to model.Status, comment string) error
	ListPromotions(ctx context.Context, artifactID int64) ([]model.PromotionEntry, error)

	// Search
	SearchArtifacts(ctx context.Context, query string, kinds []model.Kind, limit, offset int) ([]*model.RegistryArtifact, int, error)

	// Lifecycle
	Close() error
}
