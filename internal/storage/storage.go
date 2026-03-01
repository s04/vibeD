package storage

import "context"

// StorageRef points to a stored artifact's source code on disk or remotely.
type StorageRef struct {
	Backend   string `json:"backend"`    // "local" or "github"
	LocalPath string `json:"local_path"` // Absolute path on local filesystem
	RemoteRef string `json:"remote_ref"` // GitHub commit SHA or similar
}

// Storage handles persistence of artifact source code and deployment manifests.
type Storage interface {
	// StoreSource writes source files for an artifact.
	// files maps relative file path to file content.
	StoreSource(ctx context.Context, artifactID string, files map[string]string) (*StorageRef, error)

	// StoreManifest writes deployment manifests for an artifact.
	// manifests maps filename to content bytes.
	StoreManifest(ctx context.Context, artifactID string, manifests map[string][]byte) error

	// GetSourcePath returns the local filesystem path to an artifact's source code.
	// For remote backends, this may trigger a download.
	GetSourcePath(ctx context.Context, artifactID string) (string, error)

	// Delete removes all stored data for an artifact.
	Delete(ctx context.Context, artifactID string) error
}
