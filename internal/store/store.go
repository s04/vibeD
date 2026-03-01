package store

import (
	"context"

	"github.com/maxkorbacher/vibed/pkg/api"
)

// ArtifactStore persists artifact metadata and state.
type ArtifactStore interface {
	// Create stores a new artifact. Returns ErrAlreadyExists if name is taken.
	Create(ctx context.Context, artifact *api.Artifact) error

	// Get retrieves an artifact by ID. Returns ErrNotFound if not found.
	Get(ctx context.Context, id string) (*api.Artifact, error)

	// GetByName retrieves an artifact by name. Returns ErrNotFound if not found.
	GetByName(ctx context.Context, name string) (*api.Artifact, error)

	// List returns all artifacts, optionally filtered by status.
	List(ctx context.Context, statusFilter string) ([]api.ArtifactSummary, error)

	// Update replaces the artifact record. Returns ErrNotFound if not found.
	Update(ctx context.Context, artifact *api.Artifact) error

	// Delete removes an artifact by ID. Returns ErrNotFound if not found.
	Delete(ctx context.Context, id string) error
}
